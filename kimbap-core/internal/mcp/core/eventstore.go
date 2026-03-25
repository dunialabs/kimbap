package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dunialabs/kimbap-core/internal/database"
	mcptypes "github.com/dunialabs/kimbap-core/internal/mcp/types"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/rs/zerolog/log"
)

type persistJob struct {
	entity    *database.Event
	eventID   mcptypes.EventID
	streamID  mcptypes.StreamID
	sessionID string
}

type PersistentEventStore struct {
	sessionID string
	userID    string

	mu sync.RWMutex

	enqueueMu sync.Mutex

	cache *lru.Cache[string, mcptypes.CachedEvent]

	repo EventRepository

	persistCh chan persistJob
	stopCh    chan struct{}
	stopOnce  sync.Once
	stopped   atomic.Bool

	eventRetention time.Duration
}

func NewPersistentEventStore(sessionID, userID string, repo EventRepository) *PersistentEventStore {
	cache, _ := lru.New[string, mcptypes.CachedEvent](10000)
	es := &PersistentEventStore{
		sessionID:      sessionID,
		userID:         userID,
		cache:          cache,
		repo:           repo,
		persistCh:      make(chan persistJob, 256),
		stopCh:         make(chan struct{}),
		eventRetention: 7 * 24 * time.Hour,
	}

	go es.persistWorker()

	return es
}

func (s *PersistentEventStore) StoreEvent(ctx context.Context, streamID mcptypes.StreamID, message mcptypes.JSONRPCMessage) (mcptypes.EventID, error) {
	now := time.Now()
	eventID := mcptypes.EventID(fmt.Sprintf("%s_%d_%s", streamID, now.UnixMilli(), randomHex(4)))

	event := mcptypes.CachedEvent{
		EventID:   string(eventID),
		Message:   message,
		Timestamp: now,
		StreamID:  string(streamID),
	}

	s.mu.Lock()
	s.cache.Add(s.cacheKey(streamID, eventID), event)
	s.mu.Unlock()

	if s.repo != nil {
		entity := &database.Event{
			EventID:     string(eventID),
			StreamID:    string(streamID),
			SessionID:   s.sessionID,
			MessageType: message.Method,
			MessageData: mustJSON(message),
			CreatedAt:   now,
			ExpiresAt:   now.Add(s.eventRetention),
		}

		s.enqueueMu.Lock()
		defer s.enqueueMu.Unlock()

		if s.stopped.Load() {
			return eventID, nil
		}

		job := persistJob{entity: entity, eventID: eventID, streamID: streamID, sessionID: s.sessionID}
		select {
		case s.persistCh <- job:
		default:
			timer := time.NewTimer(100 * time.Millisecond)
			select {
			case s.persistCh <- job:
				timer.Stop()
			case <-timer.C:
				log.Error().
					Str("eventId", string(eventID)).
					Str("streamId", string(streamID)).
					Str("eventAction", message.Method).
					Str("messageType", entity.MessageType).
					Str("sessionId", s.sessionID).
					Msg("event persist queue full after timeout, dropping persistence")
			case <-s.stopCh:
				timer.Stop()
			}
		}
	}

	return eventID, nil
}

func (s *PersistentEventStore) persistWorker() {
	for {
		select {
		case job := <-s.persistCh:
			if s.repo == nil {
				continue
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := s.repo.Create(ctx, job.entity); err != nil {
				log.Error().
					Err(err).
					Str("eventId", string(job.eventID)).
					Str("streamId", string(job.streamID)).
					Str("sessionId", job.sessionID).
					Msg("failed to persist event to repository")
			}
			cancel()
		case <-s.stopCh:
			return
		}
	}
}

func (s *PersistentEventStore) Stop() {
	s.stopOnce.Do(func() {
		s.enqueueMu.Lock()
		defer s.enqueueMu.Unlock()

		s.stopped.Store(true)
		close(s.stopCh)
		drainEnd := time.Now().Add(2 * time.Second)
		for {
			remaining := time.Until(drainEnd)
			if remaining <= 0 {
				return
			}
			select {
			case job := <-s.persistCh:
				ctx, cancel := context.WithTimeout(context.Background(), remaining)
				if err := s.repo.Create(ctx, job.entity); err != nil {
					log.Error().Err(err).Str("eventId", string(job.eventID)).Msg("failed to persist event during drain")
				}
				cancel()
			default:
				return
			}
		}
	})
}

func (s *PersistentEventStore) ReplayEventsAfter(ctx context.Context, lastEventID mcptypes.EventID, options mcptypes.ReplayOptions) (mcptypes.StreamID, error) {
	streamID := extractStreamID(lastEventID)
	lastTimestamp := extractTimestampFromEventID(lastEventID)

	if s.repo == nil {
		cacheEvents := s.collectCacheEventsForStream(string(streamID), lastTimestamp)
		for _, event := range cacheEvents {
			if ctx.Err() != nil {
				return streamID, ctx.Err()
			}
			if err := options.Send(ctx, mcptypes.EventID(event.EventID), event.Message); err != nil {
				if ctx.Err() != nil {
					return streamID, ctx.Err()
				}
				log.Error().Err(err).Str("eventId", event.EventID).Msg("Failed to send event during replay")
			}
		}
		return streamID, nil
	}

	dbEvents, err := s.repo.FindByStreamIDAfter(ctx, string(streamID), string(lastEventID))
	if err != nil {
		return streamID, err
	}

	cacheEvents := s.collectCacheEventsForStream(string(streamID), lastTimestamp)
	seen := make(map[string]struct{}, len(dbEvents))

	merged := make([]mcptypes.CachedEvent, 0, len(dbEvents)+len(cacheEvents))
	for _, e := range dbEvents {
		eventSessionID := strings.TrimSpace(e.SessionID)
		if eventSessionID != "" && eventSessionID != strings.TrimSpace(s.sessionID) {
			continue
		}
		msg := mcptypes.JSONRPCMessage{}
		if parseErr := json.Unmarshal([]byte(e.MessageData), &msg); parseErr != nil {
			log.Error().Err(parseErr).Str("eventId", e.EventID).Msg("Failed to parse event during replay")
			continue
		}
		seen[e.EventID] = struct{}{}
		merged = append(merged, mcptypes.CachedEvent{
			EventID:   e.EventID,
			StreamID:  e.StreamID,
			Timestamp: e.CreatedAt,
			Message:   msg,
		})
	}
	for _, ce := range cacheEvents {
		if _, dup := seen[ce.EventID]; !dup {
			merged = append(merged, ce)
		}
	}

	sort.SliceStable(merged, func(i, j int) bool {
		return merged[i].Timestamp.Before(merged[j].Timestamp)
	})

	for _, event := range merged {
		if ctx.Err() != nil {
			return streamID, ctx.Err()
		}
		if err := options.Send(ctx, mcptypes.EventID(event.EventID), event.Message); err != nil {
			if ctx.Err() != nil {
				return streamID, ctx.Err()
			}
			log.Error().Err(err).Str("eventId", event.EventID).Msg("Failed to send event during replay")
		}
	}

	return streamID, nil
}

func (s *PersistentEventStore) collectCacheEventsForStream(streamID string, afterTimestampMs int64) []mcptypes.CachedEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	afterTime := time.UnixMilli(afterTimestampMs)
	prefix := streamID + "::"
	var events []mcptypes.CachedEvent

	for _, key := range s.cache.Keys() {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		event, ok := s.cache.Peek(key)
		if !ok {
			continue
		}
		if event.Timestamp.After(afterTime) {
			events = append(events, event)
		}
	}

	sort.SliceStable(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})

	return events
}

func (s *PersistentEventStore) CleanupStream(streamID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, key := range s.cache.Keys() {
		if strings.HasPrefix(key, streamID+"::") {
			s.cache.Remove(key)
		}
	}
}

func (s *PersistentEventStore) cacheKey(streamID mcptypes.StreamID, eventID mcptypes.EventID) string {
	return string(streamID) + "::" + string(eventID)
}

func randomHex(bytesLen int) string {
	if bytesLen <= 0 {
		bytesLen = 4
	}
	b := make([]byte, bytesLen)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
