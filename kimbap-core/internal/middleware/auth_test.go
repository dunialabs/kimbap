package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractAuthToken(t *testing.T) {
	t.Run("returns empty token without error when auth header missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/path", nil)

		token, err := ExtractAuthToken(req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if token != "" {
			t.Fatalf("expected empty token, got %q", token)
		}
	})

	t.Run("returns empty token without error for non-bearer authorization", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/path", nil)
		req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")

		token, err := ExtractAuthToken(req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if token != "" {
			t.Fatalf("expected empty token, got %q", token)
		}
	})

	t.Run("accepts case-insensitive bearer scheme", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/path", nil)
		req.Header.Set("Authorization", "bearer test-token")

		token, err := ExtractAuthToken(req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if token != "test-token" {
			t.Fatalf("expected test-token, got %q", token)
		}
	})

	t.Run("ignores query token when authorization is non-bearer", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/path?token=query-token&api_key=legacy", nil)
		req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")

		token, err := ExtractAuthToken(req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if token != "" {
			t.Fatalf("expected empty token, got %q", token)
		}
	})

	t.Run("returns error for empty bearer token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/path", nil)
		req.Header.Set("Authorization", "Bearer ")

		_, err := ExtractAuthToken(req)
		if err == nil {
			t.Fatal("expected error for empty bearer token")
		}
	})
}

func TestSetWWWAuthenticateIfUnauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/path", nil)

	t.Run("does not set header for forbidden status", func(t *testing.T) {
		rr := httptest.NewRecorder()
		setWWWAuthenticateIfUnauthorized(rr, req, http.StatusForbidden, "invalid_token", "forbidden")

		if got := rr.Header().Get("WWW-Authenticate"); got != "" {
			t.Fatalf("expected no WWW-Authenticate header, got %q", got)
		}
	})

	t.Run("sets header for unauthorized status", func(t *testing.T) {
		rr := httptest.NewRecorder()
		setWWWAuthenticateIfUnauthorized(rr, req, http.StatusUnauthorized, "invalid_token", "invalid token")

		if got := rr.Header().Get("WWW-Authenticate"); got == "" {
			t.Fatal("expected WWW-Authenticate header to be set")
		}
	})
}

func TestAuthErrorWriters(t *testing.T) {
	t.Run("writeJSONError writes generic error shape", func(t *testing.T) {
		rr := httptest.NewRecorder()
		writeJSONError(rr, http.StatusUnauthorized, "missing token")

		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rr.Code)
		}

		var body map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("invalid json: %v", err)
		}
		errObj, ok := body["error"].(map[string]any)
		if !ok {
			t.Fatalf("expected error object")
		}
		if errObj["code"] != float64(http.StatusUnauthorized) {
			t.Fatalf("expected code=401, got %#v", errObj["code"])
		}
		if errObj["message"] != "missing token" {
			t.Fatalf("expected message=missing token, got %#v", errObj["message"])
		}
	})

	t.Run("writeUserAuthError writes success/error shape", func(t *testing.T) {
		rr := httptest.NewRecorder()
		writeUserAuthError(rr, http.StatusForbidden, "user disabled")

		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected status 403, got %d", rr.Code)
		}

		var body map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("invalid json: %v", err)
		}
		if body["success"] != false {
			t.Fatalf("expected success=false, got %#v", body["success"])
		}
		errObj, ok := body["error"].(map[string]any)
		if !ok {
			t.Fatalf("expected error object")
		}
		if errObj["code"] != float64(1003) {
			t.Fatalf("expected code=1003, got %#v", errObj["code"])
		}
		if errObj["message"] != "user disabled" {
			t.Fatalf("expected message=user disabled, got %#v", errObj["message"])
		}
	})

}

func TestDecodePermissions(t *testing.T) {
	t.Run("rejects top-level null", func(t *testing.T) {
		_, err := decodePermissions(json.RawMessage("null"))
		if err == nil {
			t.Fatal("expected error for null permissions")
		}
	})

	t.Run("accepts empty object", func(t *testing.T) {
		perms, err := decodePermissions(json.RawMessage("{}"))
		if err != nil {
			t.Fatalf("expected no error for empty permissions object, got %v", err)
		}
		if perms == nil {
			t.Fatal("expected non-nil permissions map")
		}
		if len(perms) != 0 {
			t.Fatalf("expected empty permissions map, got len=%d", len(perms))
		}
	})
}

func TestBuildWWWAuthenticateHeaderUsesCanonicalAndTrustedForwardedSource(t *testing.T) {
	t.Run("uses canonical base when configured", func(t *testing.T) {
		t.Setenv("KIMBAP_PUBLIC_BASE_URL", "https://public.example.com/")
		req := httptest.NewRequest(http.MethodGet, "http://service.local/api/v1/actions", nil)
		req.RemoteAddr = "198.51.100.10:40000"
		req.Host = "service.local"
		req.Header.Set("X-Forwarded-Host", "evil.example")
		req.Header.Set("X-Forwarded-Proto", "http")

		header := BuildWWWAuthenticateHeader(req, "invalid_token", "Missing Authorization header")
		if !strings.Contains(header, `resource_metadata="https://public.example.com/.well-known/oauth-protected-resource"`) {
			t.Fatalf("expected canonical resource metadata, got %s", header)
		}
	})

	t.Run("rejects canonical base with userinfo", func(t *testing.T) {
		t.Setenv("KIMBAP_PUBLIC_BASE_URL", "https://user:pass@public.example.com")
		req := httptest.NewRequest(http.MethodGet, "http://service.local/api/v1/actions", nil)
		req.RemoteAddr = "198.51.100.10:40000"
		req.Host = "service.local"

		header := BuildWWWAuthenticateHeader(req, "invalid_token", "Missing Authorization header")
		if strings.Contains(header, `resource_metadata=`) {
			t.Fatalf("expected invalid canonical url to be ignored, got %s", header)
		}
	})

	t.Run("ignores forwarded headers from non-loopback source", func(t *testing.T) {
		t.Setenv("KIMBAP_PUBLIC_BASE_URL", "")
		req := httptest.NewRequest(http.MethodGet, "http://service.local/api/v1/actions", nil)
		req.RemoteAddr = "198.51.100.10:40000"
		req.Host = "service.local"
		req.Header.Set("X-Forwarded-Host", "evil.example")
		req.Header.Set("X-Forwarded-Proto", "https")

		header := BuildWWWAuthenticateHeader(req, "invalid_token", "Missing Authorization header")
		if strings.Contains(header, `resource_metadata=`) {
			t.Fatalf("expected non-loopback source without canonical url to omit resource_metadata, got %s", header)
		}
	})
}

type countingTokenValidator struct {
	userID string
	err    error
}

func (v countingTokenValidator) ValidateToken(token string) (string, error) {
	if v.err != nil {
		return "", v.err
	}
	return v.userID, nil
}

type countingUserRepository struct {
	user  *User
	err   error
	calls int
}

func (r *countingUserRepository) FindByUserID(_ context.Context, userID string) (*User, error) {
	r.calls++
	if r.err != nil {
		return nil, r.err
	}
	if r.user == nil {
		return nil, nil
	}
	copied := *r.user
	copied.UserID = userID
	return &copied, nil
}

func TestAuthenticateRequestAvoidsDuplicateUserLookup(t *testing.T) {
	repo := &countingUserRepository{
		user: &User{
			Status:          UserStatusEnabled,
			Permissions:     json.RawMessage(`{}`),
			UserPreferences: json.RawMessage(`{}`),
		},
	}
	mw := NewAuthMiddleware(countingTokenValidator{userID: "user-1"}, nil, repo, nil)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/v1/actions", nil)
	req.Header.Set("Authorization", "Bearer test-token")

	authCtx, err := mw.AuthenticateRequest(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if authCtx == nil {
		t.Fatal("expected auth context")
	}
	if repo.calls != 1 {
		t.Fatalf("expected exactly one user lookup, got %d", repo.calls)
	}
}
