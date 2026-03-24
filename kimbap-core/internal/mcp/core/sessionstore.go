package core

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	mcptypes "github.com/dunialabs/kimbap-core/internal/mcp/types"
	"github.com/dunialabs/kimbap-core/internal/types"
)

const (
	maxSessionsPerUser = 10
	maxGlobalSessions  = 1000
)

type SessionStore struct {
	mu sync.RWMutex

	sessions       map[string]*ClientSession
	proxySessions  map[string]*ProxySession
	userSessions   map[string]map[string]struct{}
	eventStores    map[string]*PersistentEventStore
	sessionLoggers map[string]SessionLogger

	eventRepo EventRepository
	notifier  SocketNotifier

	totalCreated  int64
	cleanupTicker *time.Ticker
	cleanupStop   chan struct{}
	stopped       atomic.Bool
}

var (
	sessionStoreInstance *SessionStore
	sessionStoreOnce     sync.Once
)

func SessionStoreInstance() *SessionStore {
	sessionStoreOnce.Do(func() {
		sessionStoreInstance = &SessionStore{
			sessions:       map[string]*ClientSession{},
			proxySessions:  map[string]*ProxySession{},
			userSessions:   map[string]map[string]struct{}{},
			eventStores:    map[string]*PersistentEventStore{},
			sessionLoggers: map[string]SessionLogger{},
		}
		sessionStoreInstance.startCleanupTimer()
	})
	return sessionStoreInstance
}

func (s *SessionStore) SetEventRepository(repo EventRepository) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.eventRepo = repo
}

func (s *SessionStore) SetNotifier(notifier SocketNotifier) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notifier = notifier
}

func (s *SessionStore) CreateSession(ctx context.Context, sessionID, userID, token string, authContext mcptypes.AuthContext, logger SessionLogger) (*ClientSession, error) {
	clientSession := NewClientSession(sessionID, userID, token, authContext)
	clientSession.LastUserInfoRefresh = time.Now().UnixMilli()
	s.mu.RLock()
	eventRepo := s.eventRepo
	s.mu.RUnlock()
	eventStore := NewPersistentEventStore(sessionID, userID, eventRepo)
	logger.LogSessionLifecycle(types.MCPEventLogTypeSessionInit, "")

	proxySession := NewProxySession(sessionID, userID, clientSession, logger, eventStore, s.removeSingleSession)
	clientSession.SetProxySession(proxySession)

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.sessions) >= maxGlobalSessions {
		return nil, fmt.Errorf("global session limit reached (%d)", maxGlobalSessions)
	}
	if userSess, ok := s.userSessions[userID]; ok && len(userSess) >= maxSessionsPerUser {
		return nil, fmt.Errorf("per-user session limit reached (%d)", maxSessionsPerUser)
	}

	s.sessions[sessionID] = clientSession
	s.proxySessions[sessionID] = proxySession
	s.eventStores[sessionID] = eventStore
	s.sessionLoggers[sessionID] = logger
	if _, ok := s.userSessions[userID]; !ok {
		s.userSessions[userID] = map[string]struct{}{}
	}
	s.userSessions[userID][sessionID] = struct{}{}
	atomic.AddInt64(&s.totalCreated, 1)

	_ = ctx
	return clientSession, nil
}

func (s *SessionStore) GetSession(sessionID string) *ClientSession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessions[sessionID]
}

func (s *SessionStore) GetProxySession(sessionID string) *ProxySession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.proxySessions[sessionID]
}

func (s *SessionStore) GetAllProxySessions() []*ProxySession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*ProxySession, 0, len(s.proxySessions))
	for _, session := range s.proxySessions {
		if session != nil {
			out = append(out, session)
		}
	}
	return out
}

func (s *SessionStore) GetEventStore(sessionID string) *PersistentEventStore {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.eventStores[sessionID]
}

func (s *SessionStore) GetSessionLogger(sessionID string) SessionLogger {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessionLoggers[sessionID]
}

func (s *SessionStore) GetAllSessions() []*ClientSession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*ClientSession, 0, len(s.sessions))
	for _, session := range s.sessions {
		out = append(out, session)
	}
	return out
}

func (s *SessionStore) GetUserSessions(userID string) []*ClientSession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*ClientSession, 0)
	for sessionID := range s.userSessions[userID] {
		if session := s.sessions[sessionID]; session != nil {
			out = append(out, session)
		}
	}
	return out
}

func (s *SessionStore) GetSessionsUsingServer(serverID string) []*ClientSession {
	if serverID == "" {
		return []*ClientSession{}
	}
	sessions := s.GetAllSessions()
	out := make([]*ClientSession, 0, len(sessions))
	for _, session := range sessions {
		if session == nil {
			continue
		}
		if session.CanAccessServer(serverID) {
			out = append(out, session)
		}
	}
	return out
}

func (s *SessionStore) GetUserFirstSession(userID string) *ClientSession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for sessionID := range s.userSessions[userID] {
		return s.sessions[sessionID]
	}
	return nil
}

func (s *SessionStore) RemoveSession(sessionID string, reason mcptypes.DisconnectReason, removeAllForUser bool) {
	var userID string
	s.mu.RLock()
	session := s.sessions[sessionID]
	if session != nil {
		userID = session.UserID
	}
	s.mu.RUnlock()
	if session == nil {
		return
	}
	if removeAllForUser {
		s.RemoveAllUserSessions(userID, reason)
		return
	}
	s.removeSingleSession(sessionID, reason)
}

func (s *SessionStore) TerminateSession(sessionID string) (err error) {
	defer func() {
		s.mu.Lock()
		delete(s.sessions, sessionID)
		delete(s.proxySessions, sessionID)
		s.mu.Unlock()

		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("terminate session %s: %v", sessionID, recovered)
		}
	}()

	s.RemoveSession(sessionID, mcptypes.DisconnectReasonClientDisconnect, false)
	return nil
}

func (s *SessionStore) removeSingleSession(sessionID string, reason mcptypes.DisconnectReason) {
	shouldCloseTemporaryServers := false
	s.mu.Lock()
	session := s.sessions[sessionID]
	proxySession := s.proxySessions[sessionID]
	eventStore := s.eventStores[sessionID]
	delete(s.sessions, sessionID)
	delete(s.proxySessions, sessionID)
	delete(s.eventStores, sessionID)
	delete(s.sessionLoggers, sessionID)
	if session != nil {
		delete(s.userSessions[session.UserID], sessionID)
		if len(s.userSessions[session.UserID]) == 0 {
			delete(s.userSessions, session.UserID)
			shouldCloseTemporaryServers = true
		}
	}
	s.mu.Unlock()

	if proxySession != nil {
		proxySession.Cleanup(context.Background())
	}

	if eventStore != nil {
		eventStore.Stop()
	}

	GlobalRequestRouterInstance().CleanupSessionNotifications(sessionID)
	if session != nil {
		session.Close(reason)
		if shouldCloseTemporaryServers {
			ServerManagerInstance().CloseUserTemporaryServers(context.Background(), session.UserID)
		}
	}
}

func (s *SessionStore) RemoveAllUserSessions(userID string, reason mcptypes.DisconnectReason) {
	s.mu.RLock()
	userSessionIDs := s.userSessions[userID]
	sessionIDs := make([]string, 0, len(userSessionIDs))
	for sessionID := range userSessionIDs {
		sessionIDs = append(sessionIDs, sessionID)
	}
	s.mu.RUnlock()

	for _, sessionID := range sessionIDs {
		s.removeSingleSession(sessionID, reason)
	}
}

func (s *SessionStore) RemoveAllSessions(reason mcptypes.DisconnectReason) {
	for _, session := range s.GetAllSessions() {
		if session == nil {
			continue
		}
		s.removeSingleSession(session.SessionID, reason)
	}
}

func (s *SessionStore) GetActiveSessionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, session := range s.sessions {
		if session != nil && session.IsSSEConnected() {
			count++
		}
	}

	return count
}

func (s *SessionStore) cleanupExpiredSessions() {
	now := time.Now()
	type cleanupCandidate struct {
		sessionID string
		session   *ClientSession
	}

	s.mu.RLock()
	candidates := make([]cleanupCandidate, 0, len(s.sessions))
	for sessionID, session := range s.sessions {
		if session == nil {
			continue
		}
		candidates = append(candidates, cleanupCandidate{sessionID: sessionID, session: session})
	}
	s.mu.RUnlock()

	s.mu.RLock()
	notifier := s.notifier
	s.mu.RUnlock()

	notifiedUsers := map[string]bool{}
	for _, candidate := range candidates {
		if candidate.session.IsExpired(now, 0) {
			userID := candidate.session.UserID
			if notifier != nil && !notifiedUsers[userID] {
				notifier.NotifyUserExpired(userID)
				notifiedUsers[userID] = true
			}
			s.RemoveSession(candidate.sessionID, mcptypes.DisconnectReasonUserExpired, true)
			continue
		}
		if candidate.session.IsInactive(now, 10*time.Minute) {
			s.RemoveSession(candidate.sessionID, mcptypes.DisconnectReasonSessionTimeout, false)
		}
	}
}

func (s *SessionStore) startCleanupTimer() {
	s.cleanupTicker = time.NewTicker(5 * time.Minute)
	s.cleanupStop = make(chan struct{})
	go func() {
		for {
			select {
			case <-s.cleanupTicker.C:
				s.cleanupExpiredSessions()
			case <-s.cleanupStop:
				return
			}
		}
	}()
}

func (s *SessionStore) Stop() {
	if s.stopped.CompareAndSwap(false, true) {
		s.mu.RLock()
		eventStores := make([]*PersistentEventStore, 0, len(s.eventStores))
		for _, eventStore := range s.eventStores {
			if eventStore != nil {
				eventStores = append(eventStores, eventStore)
			}
		}
		s.mu.RUnlock()

		if s.cleanupTicker != nil {
			s.cleanupTicker.Stop()
		}
		if s.cleanupStop != nil {
			close(s.cleanupStop)
		}
		for _, eventStore := range eventStores {
			eventStore.Stop()
		}
	}
}

func (s *SessionStore) TotalCreated() int64 {
	return atomic.LoadInt64(&s.totalCreated)
}

func (s *SessionStore) UpdateUserPreferences(userID string, prefs mcptypes.Permissions, comparator CapabilitiesComparator) {
	updatedPrefs := cloneSessionPermissions(prefs)
	for _, session := range s.GetUserSessions(userID) {
		if session == nil {
			continue
		}

		oldPrefs := cloneSessionPermissions(session.GetUserPreferences())
		toolsChanged := true
		resourcesChanged := true
		promptsChanged := true
		if comparator != nil {
			toolsChanged, resourcesChanged, promptsChanged = comparator.ComparePermissions(oldPrefs, updatedPrefs)
		}

		session.UpdateUserPreferences(cloneSessionPermissions(updatedPrefs))
		proxy := session.GetProxySession()
		if proxy == nil {
			continue
		}
		if toolsChanged {
			proxy.SendToolsListChangedToClient()
		}
		if resourcesChanged {
			proxy.SendResourcesListChangedToClient()
		}
		if promptsChanged {
			proxy.SendPromptsListChangedToClient()
		}
	}
}

func cloneSessionPermissions(in mcptypes.Permissions) mcptypes.Permissions {
	if in == nil {
		return mcptypes.Permissions{}
	}

	raw, err := json.Marshal(in)
	if err != nil {
		out := make(mcptypes.Permissions, len(in))
		for serverID, cfg := range in {
			out[serverID] = cfg
		}
		return out
	}

	out := mcptypes.Permissions{}
	if err := json.Unmarshal(raw, &out); err != nil || out == nil {
		out = make(mcptypes.Permissions, len(in))
		for serverID, cfg := range in {
			out[serverID] = cfg
		}
	}
	return out
}
