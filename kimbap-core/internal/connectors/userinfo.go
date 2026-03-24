package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type UserInfo struct {
	Email       string
	Login       string
	DisplayName string
}

func (u UserInfo) Principal() string {
	if u.Email != "" {
		return u.Email
	}
	if u.Login != "" {
		return u.Login
	}
	return u.DisplayName
}

func FetchUserInfo(ctx context.Context, endpoint, accessToken string) (*UserInfo, error) {
	if strings.TrimSpace(endpoint) == "" || strings.TrimSpace(accessToken) == "" {
		return nil, fmt.Errorf("endpoint and access token are required")
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	res, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("userinfo endpoint returned HTTP %d", res.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	info := &UserInfo{}
	for _, key := range []string{"email", "mail"} {
		if v, ok := raw[key].(string); ok && v != "" {
			info.Email = v
			break
		}
	}
	for _, key := range []string{"login", "username", "preferred_username", "sub"} {
		if v, ok := raw[key].(string); ok && v != "" {
			info.Login = v
			break
		}
	}
	for _, key := range []string{"name", "display_name", "displayName"} {
		if v, ok := raw[key].(string); ok && v != "" {
			info.DisplayName = v
			break
		}
	}

	if info.Email == "" && info.Login == "" && info.DisplayName == "" {
		return nil, fmt.Errorf("no recognizable identity fields in userinfo response")
	}

	return info, nil
}
