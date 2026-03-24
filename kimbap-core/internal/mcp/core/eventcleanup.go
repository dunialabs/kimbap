package core

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type EventCleanupService struct {
	repo EventRepository

	mu sync.Mutex

	interval           time.Duration
	stopCh             chan struct{}
	doneCh             chan struct{}
	running            bool
	lastCleanupTime    *time.Time
	totalCleanedEvents int64
}

func NewEventCleanupService(repo EventRepository) *EventCleanupService {
	service := &EventCleanupService{repo: repo, interval: 24 * time.Hour, stopCh: make(chan struct{})}
	service.Start()
	return service
}

func (s *EventCleanupService) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	interval := s.interval
	stopCh := s.stopCh
	doneCh := make(chan struct{})
	s.doneCh = doneCh
	s.mu.Unlock()

	go func() {
		defer close(doneCh)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				deleted, err := s.CleanupExpiredEvents(context.Background())
				if err != nil {
					log.Error().Err(err).Msg("event cleanup failed")
				} else if deleted > 0 {
					log.Info().Int64("deletedCount", deleted).Msg("cleaned up expired events")
				}
			case <-stopCh:
				return
			}
		}
	}()
}

func (s *EventCleanupService) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.stopCh)
	doneCh := s.doneCh
	s.stopCh = make(chan struct{})
	s.mu.Unlock()
	if doneCh != nil {
		<-doneCh
	}
}

func (s *EventCleanupService) CleanupExpiredEvents(ctx context.Context) (int64, error) {
	if s.repo == nil {
		return 0, nil
	}

	deleted, err := s.repo.DeleteExpired(ctx)
	if err != nil {
		return 0, err
	}

	now := time.Now()
	s.mu.Lock()
	s.lastCleanupTime = &now
	s.totalCleanedEvents += deleted
	s.mu.Unlock()

	return deleted, nil
}

func (s *EventCleanupService) CleanupStream(ctx context.Context, streamID string) (int64, error) {
	if s.repo == nil {
		return 0, nil
	}
	deleted, err := s.repo.DeleteByStreamID(ctx, streamID)
	if err != nil {
		return 0, err
	}
	s.mu.Lock()
	s.totalCleanedEvents += deleted
	s.mu.Unlock()
	return deleted, nil
}
