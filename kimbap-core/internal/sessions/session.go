package sessions

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dunialabs/kimbap-core/internal/auth"
)

const (
	sessionTokenPrefix = "kss_"
	defaultSessionTTL  = 15 * time.Minute
)

var (
	ErrInvalidSession = errors.New("invalid session token")
	ErrExpiredSession = errors.New("expired session token")
)

type SessionToken struct {
	ID          string
	PrincipalID string
	TenantID    string
	AgentName   string
	Scopes      []string
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

type SessionService struct {
	mu     sync.RWMutex
	tokens map[string]SessionToken
	ttl    time.Duration
}

func NewSessionService(ttl time.Duration) *SessionService {
	return &SessionService{ttl: ttl}
}

func (s *SessionService) Exchange(_ context.Context, principal *auth.Principal) (*SessionToken, string, error) {
	if principal == nil || strings.TrimSpace(principal.ID) == "" {
		return nil, "", errors.New("principal is required")
	}
	if strings.TrimSpace(principal.TenantID) == "" {
		return nil, "", errors.New("tenant id is required")
	}

	rawSession, err := randomToken(sessionTokenPrefix, 32)
	if err != nil {
		return nil, "", err
	}
	sessionID, err := randomToken("sst_", 16)
	if err != nil {
		return nil, "", err
	}

	now := time.Now().UTC()
	ttl := s.ttl
	if ttl <= 0 {
		ttl = defaultSessionTTL
	}

	session := SessionToken{
		ID:          sessionID,
		PrincipalID: principal.ID,
		TenantID:    principal.TenantID,
		AgentName:   principal.AgentName,
		Scopes:      append([]string(nil), principal.Scopes...),
		CreatedAt:   now,
		ExpiresAt:   now.Add(ttl),
	}

	s.ensureMap()
	hash := hashSession(rawSession)
	s.mu.Lock()
	s.tokens[hash] = session
	s.mu.Unlock()

	copySession := session
	return &copySession, rawSession, nil
}

func (s *SessionService) Validate(_ context.Context, rawSession string) (*SessionToken, error) {
	if !strings.HasPrefix(rawSession, sessionTokenPrefix) {
		return nil, ErrInvalidSession
	}

	hash := hashSession(rawSession)
	s.mu.RLock()
	session, ok := s.tokens[hash]
	s.mu.RUnlock()
	if !ok {
		return nil, ErrInvalidSession
	}
	if time.Now().UTC().After(session.ExpiresAt) {
		return nil, ErrExpiredSession
	}

	copySession := session
	return &copySession, nil
}

func (s *SessionService) ensureMap() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tokens == nil {
		s.tokens = map[string]SessionToken{}
	}
}

func hashSession(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func randomToken(prefix string, n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	return prefix + hex.EncodeToString(b), nil
}
