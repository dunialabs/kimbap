package core

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type MappingEntry struct {
	OriginalRequestID   any
	ProxyRequestID      string
	DownstreamRequestID any
	ServerID            string
	Timestamp           time.Time
	Method              string
}

type RequestIDMapper struct {
	sessionID string

	mu sync.RWMutex

	clientToProxy     map[string]string
	proxyToClient     map[string]any
	proxyToDownstream map[string]*MappingEntry
	downstreamToProxy map[string]string

	stopCh chan struct{}

	ttl             time.Duration
	cleanupInterval time.Duration
}

func NewRequestIDMapper(sessionID string) *RequestIDMapper {
	m := &RequestIDMapper{
		sessionID:         sessionID,
		clientToProxy:     make(map[string]string),
		proxyToClient:     make(map[string]any),
		proxyToDownstream: make(map[string]*MappingEntry),
		downstreamToProxy: make(map[string]string),
		stopCh:            make(chan struct{}),
		ttl:               5 * time.Minute,
		cleanupInterval:   time.Minute,
	}
	go m.runCleanupLoop()
	return m
}

func (m *RequestIDMapper) RegisterClientRequest(originalRequestID any, method string, serverID string) string {
	key := requestIDKey(originalRequestID)

	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.clientToProxy[key]; ok {
		return existing
	}

	proxyRequestID := fmt.Sprintf("%s:%v:%d", m.sessionID, originalRequestID, time.Now().UnixMilli())
	m.clientToProxy[key] = proxyRequestID
	m.proxyToClient[proxyRequestID] = originalRequestID
	m.proxyToDownstream[proxyRequestID] = &MappingEntry{
		OriginalRequestID: originalRequestID,
		ProxyRequestID:    proxyRequestID,
		ServerID:          serverID,
		Timestamp:         time.Now(),
		Method:            method,
	}
	return proxyRequestID
}

func (m *RequestIDMapper) RegisterDownstreamMapping(proxyRequestID string, downstreamRequestID any, serverID string) {
	downstreamKey := downstreamRequestIDKey(downstreamRequestID, serverID)

	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.proxyToDownstream[proxyRequestID]
	if !ok {
		return
	}

	entry.DownstreamRequestID = downstreamRequestID
	entry.ServerID = serverID
	m.downstreamToProxy[downstreamKey] = proxyRequestID
}

func (m *RequestIDMapper) GetProxyRequestID(originalRequestID any) (string, bool) {
	key := requestIDKey(originalRequestID)
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.clientToProxy[key]
	return value, ok
}

func (m *RequestIDMapper) GetOriginalRequestID(proxyRequestID string) (any, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.proxyToClient[proxyRequestID]
	return value, ok
}

func (m *RequestIDMapper) GetProxyRequestIDFromDownstream(downstreamRequestID any, serverID string) (string, bool) {
	key := downstreamRequestIDKey(downstreamRequestID, serverID)
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.downstreamToProxy[key]
	return value, ok
}

func (m *RequestIDMapper) GetOriginalRequestIDFromDownstream(downstreamRequestID any, serverID string) (any, bool) {
	proxyID, ok := m.GetProxyRequestIDFromDownstream(downstreamRequestID, serverID)
	if !ok {
		return nil, false
	}
	return m.GetOriginalRequestID(proxyID)
}

func (m *RequestIDMapper) GetMappingEntry(proxyRequestID string) (*MappingEntry, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entry, ok := m.proxyToDownstream[proxyRequestID]
	if !ok {
		return nil, false
	}
	copy := *entry
	return &copy, true
}

func (m *RequestIDMapper) RemoveMapping(proxyRequestID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.proxyToDownstream[proxyRequestID]
	if !ok {
		return
	}

	originalKey := requestIDKey(entry.OriginalRequestID)
	delete(m.clientToProxy, originalKey)
	delete(m.proxyToClient, proxyRequestID)
	delete(m.proxyToDownstream, proxyRequestID)
	if entry.DownstreamRequestID != nil && entry.ServerID != "" {
		downstreamKey := downstreamRequestIDKey(entry.DownstreamRequestID, entry.ServerID)
		delete(m.downstreamToProxy, downstreamKey)
	}
}

func (m *RequestIDMapper) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clientToProxy = make(map[string]string)
	m.proxyToClient = make(map[string]any)
	m.proxyToDownstream = make(map[string]*MappingEntry)
	m.downstreamToProxy = make(map[string]string)
}

func (m *RequestIDMapper) Destroy() {
	select {
	case <-m.stopCh:
	default:
		close(m.stopCh)
	}
	m.Clear()
}

func (m *RequestIDMapper) runCleanupLoop() {
	ticker := time.NewTicker(m.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanupExpired()
		case <-m.stopCh:
			return
		}
	}
}

func (m *RequestIDMapper) cleanupExpired() {
	cutoff := time.Now().Add(-m.ttl)

	m.mu.Lock()
	defer m.mu.Unlock()

	for proxyID, entry := range m.proxyToDownstream {
		if entry.Timestamp.Before(cutoff) {
			originalKey := requestIDKey(entry.OriginalRequestID)
			delete(m.clientToProxy, originalKey)
			delete(m.proxyToClient, proxyID)
			delete(m.proxyToDownstream, proxyID)
			if entry.DownstreamRequestID != nil && entry.ServerID != "" {
				delete(m.downstreamToProxy, downstreamRequestIDKey(entry.DownstreamRequestID, entry.ServerID))
			}
		}
	}

	log.Debug().Str("sessionId", m.sessionID).Int("remaining", len(m.proxyToDownstream)).Msg("request ID mapper cleanup complete")
}

func requestIDKey(v any) string {
	switch id := v.(type) {
	case nil:
		return "null"
	case string:
		return "s:" + id
	case json.Number:
		return "n:" + id.String()
	case int:
		return "n:" + strconv.FormatInt(int64(id), 10)
	case int8:
		return "n:" + strconv.FormatInt(int64(id), 10)
	case int16:
		return "n:" + strconv.FormatInt(int64(id), 10)
	case int32:
		return "n:" + strconv.FormatInt(int64(id), 10)
	case int64:
		return "n:" + strconv.FormatInt(id, 10)
	case uint:
		return "n:" + strconv.FormatUint(uint64(id), 10)
	case uint8:
		return "n:" + strconv.FormatUint(uint64(id), 10)
	case uint16:
		return "n:" + strconv.FormatUint(uint64(id), 10)
	case uint32:
		return "n:" + strconv.FormatUint(uint64(id), 10)
	case uint64:
		return "n:" + strconv.FormatUint(id, 10)
	case float32:
		return "n:" + strconv.FormatFloat(float64(id), 'g', -1, 32)
	case float64:
		return "n:" + strconv.FormatFloat(id, 'g', -1, 64)
	default:
		return fmt.Sprintf("%T:%v", v, v)
	}
}

func downstreamRequestIDKey(v any, serverID string) string {
	return requestIDKey(v) + ":" + serverID
}
