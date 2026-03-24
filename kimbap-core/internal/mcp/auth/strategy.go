package auth

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"
)

const (
	authHTTPTimeout = 8 * time.Second
	expiryBuffer    = 5 * time.Minute
)

type TokenInfo struct {
	AccessToken string `json:"accessToken"`
	ExpiresIn   int64  `json:"expiresIn"`
	ExpiresAt   int64  `json:"expiresAt"`
}

type AuthStrategy interface {
	GetInitialToken() (*TokenInfo, error)
	RefreshToken() (*TokenInfo, error)
	GetCurrentOAuthConfig() map[string]interface{}
	MarkConfigAsPersisted()
}

type strategyState struct {
	mu            sync.RWMutex
	refreshMu     sync.Mutex
	configChanged bool
}

func getStringValue(config map[string]interface{}, key string) string {
	v, ok := config[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func getInt64Value(config map[string]interface{}, key string) (int64, bool) {
	v, ok := config[key]
	if !ok || v == nil {
		return 0, false
	}
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int32:
		return int64(n), true
	case int64:
		return n, true
	case float64:
		return int64(n), true
	case json.Number:
		i, err := n.Int64()
		if err == nil {
			return i, true
		}
		f, err := n.Float64()
		if err == nil {
			return int64(f), true
		}
	case string:
		i, err := strconv.ParseInt(n, 10, 64)
		if err == nil {
			return i, true
		}
		f, err := strconv.ParseFloat(n, 64)
		if err == nil {
			return int64(f), true
		}
	}
	return 0, false
}
