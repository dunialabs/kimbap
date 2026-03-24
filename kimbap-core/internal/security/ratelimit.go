package security

import (
	"sync"
	"time"
)

type rateLimitEntry struct {
	count       int
	windowStart time.Time
	lastSeen    time.Time
}

type RateLimitService struct {
	mu        sync.Mutex
	entries   map[string]*rateLimitEntry
	window    time.Duration
	stopCh    chan struct{}
	closeOnce sync.Once
}

func NewRateLimitService() *RateLimitService {
	s := &RateLimitService{
		entries: make(map[string]*rateLimitEntry),
		window:  time.Minute,
		stopCh:  make(chan struct{}),
	}
	go s.cleanupLoop()
	return s
}

func (s *RateLimitService) Close() {
	s.closeOnce.Do(func() {
		close(s.stopCh)
	})
}

func (s *RateLimitService) CheckRateLimit(userID string, limit int) (allowed bool, remaining int, resetTime time.Time) {
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.entries[userID]
	if !ok {
		entry = &rateLimitEntry{
			count:       1,
			windowStart: now,
			lastSeen:    now,
		}
		s.entries[userID] = entry
		return true, limit - entry.count, entry.windowStart.Add(s.window)
	}

	if now.Sub(entry.windowStart) >= s.window {
		entry.count = 1
		entry.windowStart = now
		entry.lastSeen = now
		return true, limit - entry.count, entry.windowStart.Add(s.window)
	}

	entry.lastSeen = now
	if entry.count >= limit {
		return false, 0, entry.windowStart.Add(s.window)
	}

	entry.count++
	return true, limit - entry.count, entry.windowStart.Add(s.window)
}

func (s *RateLimitService) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.cleanupExpired()
		}
	}
}

func (s *RateLimitService) cleanupExpired() {
	threshold := time.Now().Add(-2 * s.window)

	s.mu.Lock()
	defer s.mu.Unlock()

	for userID, entry := range s.entries {
		if entry.lastSeen.Before(threshold) {
			delete(s.entries, userID)
		}
	}
}
