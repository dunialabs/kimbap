package connectors

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dunialabs/kimbap-core/internal/security"
)

type ConnectorStore interface {
	Save(ctx context.Context, state *ConnectorState) error
	Get(ctx context.Context, tenantID, name string) (*ConnectorState, error)
	List(ctx context.Context, tenantID string) ([]ConnectorState, error)
	Delete(ctx context.Context, tenantID, name string) error
}

type Manager struct {
	configs map[string]ConnectorConfig
	store   ConnectorStore

	mu      sync.RWMutex
	pending map[string]pendingLogin

	refreshMu  sync.Mutex
	refreshing map[string]*refreshResult
}

type DeviceFlowResult struct {
	VerificationURL string
	UserCode        string
	ExpiresIn       int
	Interval        int

	deviceCode string
}

type refreshResult struct {
	done chan struct{}
	err  error
}

type pendingLogin struct {
	deviceCode string
	interval   int
	expiresAt  time.Time
}

func NewManager(store ConnectorStore) *Manager {
	return &Manager{
		configs:    map[string]ConnectorConfig{},
		store:      store,
		pending:    map[string]pendingLogin{},
		refreshing: map[string]*refreshResult{},
	}
}

func (m *Manager) RegisterConfig(cfg ConnectorConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configs[cfg.Name] = cfg
}

func (m *Manager) Login(ctx context.Context, tenantID, name string) (*DeviceFlowResult, error) {
	cfg, err := m.configFor(name)
	if err != nil {
		return nil, err
	}

	result, err := DeviceCodeRequest(cfg)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	state, _ := m.store.Get(ctx, tenantID, name)
	if state == nil {
		state = &ConnectorState{Name: name, TenantID: tenantID, Provider: cfg.Provider, CreatedAt: now}
	}
	state.Provider = cfg.Provider
	state.Status = StatusPending
	state.UpdatedAt = now
	if err := m.saveState(ctx, state); err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.pending[m.pendingKey(tenantID, name)] = pendingLogin{
		deviceCode: result.deviceCode,
		interval:   result.Interval,
		expiresAt:  now.Add(time.Duration(result.ExpiresIn) * time.Second),
	}
	m.mu.Unlock()

	return result, nil
}

func (m *Manager) CompleteLogin(ctx context.Context, tenantID, name string, code string) error {
	cfg, err := m.configFor(name)
	if err != nil {
		return err
	}

	m.mu.RLock()
	pending, hasPending := m.pending[m.pendingKey(tenantID, name)]
	m.mu.RUnlock()

	deviceCode := strings.TrimSpace(code)
	interval := 5

	if deviceCode == "" {
		if !hasPending {
			return errors.New("no pending login found")
		}
		if !pending.expiresAt.IsZero() && time.Now().After(pending.expiresAt) {
			return errors.New("pending login has expired")
		}
		deviceCode = pending.deviceCode
		if pending.interval > 0 {
			interval = pending.interval
		}
	} else {
		if !hasPending {
			return errors.New("no pending login found; initiate login first")
		}
		if pending.deviceCode != deviceCode {
			return errors.New("device code does not match pending login")
		}
		if !pending.expiresAt.IsZero() && time.Now().After(pending.expiresAt) {
			return errors.New("pending login has expired")
		}
		if pending.interval > 0 {
			interval = pending.interval
		}
	}

	token, err := PollForToken(cfg, deviceCode, interval, 10*time.Minute)
	if err != nil {
		return err
	}

	state, err := m.store.Get(ctx, tenantID, name)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if state == nil {
		state = &ConnectorState{Name: name, TenantID: tenantID, Provider: cfg.Provider, CreatedAt: now}
	}
	state.Provider = cfg.Provider
	state.AccessToken = token.AccessToken
	if token.RefreshToken != "" {
		state.RefreshToken = token.RefreshToken
	}
	if token.Scope != "" {
		state.Scopes = strings.Fields(token.Scope)
	} else {
		state.Scopes = append([]string(nil), cfg.Scopes...)
	}
	if token.ExpiresIn > 0 {
		expiresAt := now.Add(time.Duration(token.ExpiresIn) * time.Second)
		state.ExpiresAt = &expiresAt
	} else {
		state.ExpiresAt = nil
	}
	state.Status = deriveStatus(state)
	state.UpdatedAt = now
	if err := m.saveState(ctx, state); err != nil {
		return err
	}

	m.mu.Lock()
	delete(m.pending, m.pendingKey(tenantID, name))
	m.mu.Unlock()

	return nil
}

func (m *Manager) Refresh(ctx context.Context, tenantID, name string) error {
	cfg, err := m.configFor(name)
	if err != nil {
		return err
	}

	state, err := m.store.Get(ctx, tenantID, name)
	if err != nil {
		return err
	}
	if state == nil {
		return errors.New("connector state not found")
	}

	refreshToken, err := m.decryptToken(state.RefreshToken)
	if err != nil {
		return fmt.Errorf("decrypt refresh token: %w", err)
	}
	if strings.TrimSpace(refreshToken) == "" {
		state.Status = StatusReauthNeeded
		state.UpdatedAt = time.Now().UTC()
		if saveErr := m.store.Save(ctx, state); saveErr != nil {
			return saveErr
		}
		return errors.New("refresh token is missing")
	}

	token, err := RefreshAccessToken(cfg, refreshToken)
	if err != nil {
		state.Status = StatusReauthNeeded
		state.UpdatedAt = time.Now().UTC()
		_ = m.store.Save(ctx, state)
		return err
	}

	now := time.Now().UTC()
	state.AccessToken = token.AccessToken
	if token.RefreshToken != "" {
		state.RefreshToken = token.RefreshToken
	}
	if token.Scope != "" {
		state.Scopes = strings.Fields(token.Scope)
	}
	if token.ExpiresIn > 0 {
		expiresAt := now.Add(time.Duration(token.ExpiresIn) * time.Second)
		state.ExpiresAt = &expiresAt
	} else {
		state.ExpiresAt = nil
	}
	state.Status = deriveStatus(state)
	state.UpdatedAt = now
	state.LastRefresh = &now

	return m.saveState(ctx, state)
}

func (m *Manager) GetAccessToken(ctx context.Context, tenantID, name string) (string, error) {
	state, err := m.store.Get(ctx, tenantID, name)
	if err != nil {
		return "", err
	}
	if state == nil {
		return "", errors.New("connector state not found")
	}

	if state.ExpiresAt != nil && time.Now().After(*state.ExpiresAt) {
		if err := m.refreshOnce(ctx, tenantID, name); err != nil {
			return "", err
		}
		state, err = m.store.Get(ctx, tenantID, name)
		if err != nil {
			return "", err
		}
		if state == nil {
			return "", errors.New("connector state not found")
		}
	}

	accessToken, err := m.decryptToken(state.AccessToken)
	if err != nil {
		return "", fmt.Errorf("decrypt access token: %w", err)
	}
	if accessToken == "" {
		return "", errors.New("access token is empty")
	}

	return accessToken, nil
}

// refreshOnce ensures only one refresh runs per connector at a time.
// Concurrent callers wait for the first refresh to complete and reuse the result.
func (m *Manager) refreshOnce(ctx context.Context, tenantID, name string) error {
	key := tenantID + "::" + name

	m.refreshMu.Lock()
	if r, ok := m.refreshing[key]; ok {
		m.refreshMu.Unlock()
		select {
		case <-r.done:
			return r.err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	r := &refreshResult{done: make(chan struct{})}
	m.refreshing[key] = r
	m.refreshMu.Unlock()

	r.err = m.Refresh(ctx, tenantID, name)

	m.refreshMu.Lock()
	delete(m.refreshing, key)
	m.refreshMu.Unlock()
	close(r.done)

	return r.err
}

func (m *Manager) List(ctx context.Context, tenantID string) ([]ConnectorState, error) {
	items, err := m.store.List(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	for i := range items {
		items[i].Status = deriveStatus(&items[i])
		items[i].AccessToken = ""
		items[i].RefreshToken = ""
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})

	return items, nil
}

func (m *Manager) Status(ctx context.Context, tenantID, name string) (*ConnectorState, error) {
	state, err := m.store.Get(ctx, tenantID, name)
	if err != nil {
		return nil, err
	}
	if state == nil {
		return nil, errors.New("connector state not found")
	}

	copyState := *state
	copyState.Status = deriveStatus(&copyState)
	copyState.AccessToken = ""
	copyState.RefreshToken = ""

	return &copyState, nil
}

func (m *Manager) configFor(name string) (ConnectorConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cfg, ok := m.configs[name]
	if !ok {
		return ConnectorConfig{}, fmt.Errorf("connector config %q is not registered", name)
	}
	if strings.TrimSpace(cfg.ClientID) == "" {
		return ConnectorConfig{}, errors.New("connector client_id is required")
	}
	return cfg, nil
}

func (m *Manager) saveState(ctx context.Context, state *ConnectorState) error {
	copyState := *state

	encryptedAccess, err := m.encryptToken(copyState.AccessToken)
	if err != nil {
		return err
	}
	encryptedRefresh, err := m.encryptToken(copyState.RefreshToken)
	if err != nil {
		return err
	}
	copyState.AccessToken = encryptedAccess
	copyState.RefreshToken = encryptedRefresh

	return m.store.Save(ctx, &copyState)
}

func (m *Manager) encryptToken(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	key := connectorEncryptionKey()
	if key == "" {
		return "", errors.New("connector encryption key is not configured")
	}
	return security.EncryptData(value, key)
}

func (m *Manager) decryptToken(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	key := connectorEncryptionKey()
	if key == "" {
		return "", errors.New("connector encryption key is not configured")
	}
	return security.DecryptDataFromString(value, key)
}

func connectorEncryptionKey() string {
	return strings.TrimSpace(os.Getenv("KIMBAP_CONNECTOR_ENCRYPTION_KEY"))
}

func deriveStatus(state *ConnectorState) ConnectorStatus {
	if state == nil {
		return StatusPending
	}
	if state.Status == StatusReauthNeeded {
		return StatusReauthNeeded
	}
	if strings.TrimSpace(state.AccessToken) == "" {
		return StatusPending
	}
	if state.ExpiresAt == nil {
		return StatusHealthy
	}

	now := time.Now().UTC()
	if now.After(*state.ExpiresAt) {
		return StatusOldExpired
	}
	if now.Add(5 * time.Minute).After(*state.ExpiresAt) {
		return StatusExpiring
	}
	return StatusHealthy
}

func (m *Manager) pendingKey(tenantID, name string) string {
	return tenantID + "::" + name
}
