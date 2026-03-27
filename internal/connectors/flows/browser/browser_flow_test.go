package browser

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

type syncWriter struct {
	mu  sync.Mutex
	buf strings.Builder
	ch  chan struct{}
}

func newSyncWriter() *syncWriter {
	return &syncWriter{ch: make(chan struct{}, 8)}
}

func (w *syncWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	_, _ = w.buf.Write(p)
	w.mu.Unlock()
	select {
	case w.ch <- struct{}{}:
	default:
	}
	return len(p), nil
}

func (w *syncWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

func (w *syncWriter) waitForAuthURL(timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		text := w.String()
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
				return line, nil
			}
		}
		select {
		case <-w.ch:
		case <-time.After(10 * time.Millisecond):
		}
	}
	return "", context.DeadlineExceeded
}

func TestRunBrowserFlow_ValidatesRequiredFields(t *testing.T) {
	tests := []struct {
		name       string
		cfg        BrowserFlowConfig
		wantErrSub string
	}{
		{
			name:       "missing auth endpoint",
			cfg:        BrowserFlowConfig{TokenEndpoint: "https://example.com/token", ClientID: "cid"},
			wantErrSub: "auth endpoint is required",
		},
		{
			name:       "missing token endpoint",
			cfg:        BrowserFlowConfig{AuthEndpoint: "https://example.com/auth", ClientID: "cid"},
			wantErrSub: "token endpoint is required",
		},
		{
			name:       "missing client id",
			cfg:        BrowserFlowConfig{AuthEndpoint: "https://example.com/auth", TokenEndpoint: "https://example.com/token"},
			wantErrSub: "client id is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := RunBrowserFlow(context.Background(), tc.cfg, io.Discard)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tc.wantErrSub) {
				t.Fatalf("error = %v, want substring %q", err, tc.wantErrSub)
			}
		})
	}
}

func TestExchangeAuthorizationCode_SuccessAndRequestFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := r.PostForm.Get("grant_type"); got != "authorization_code" {
			t.Fatalf("grant_type = %q, want authorization_code", got)
		}
		if got := r.PostForm.Get("client_id"); got != "client-id" {
			t.Fatalf("client_id = %q, want client-id", got)
		}
		if got := r.PostForm.Get("code"); got != "auth-code" {
			t.Fatalf("code = %q, want auth-code", got)
		}
		if got := r.PostForm.Get("redirect_uri"); got != "http://127.0.0.1:8181/callback" {
			t.Fatalf("redirect_uri = %q", got)
		}
		if got := r.PostForm.Get("code_verifier"); got != "verifier-abc" {
			t.Fatalf("code_verifier = %q, want verifier-abc", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token-1","refresh_token":"refresh-1","expires_in":3600,"scope":"read write"}`))
	}))
	defer srv.Close()

	result, err := exchangeAuthorizationCode(context.Background(), BrowserFlowConfig{
		TokenEndpoint: srv.URL,
		ClientID:      "client-id",
		ClientSecret:  "client-secret",
		AuthMethod:    "body",
	}, "http://127.0.0.1:8181/callback", "auth-code", "verifier-abc")
	if err != nil {
		t.Fatalf("exchangeAuthorizationCode() error = %v", err)
	}
	if result.AccessToken != "token-1" {
		t.Fatalf("access token = %q, want token-1", result.AccessToken)
	}
	if result.RefreshToken != "refresh-1" {
		t.Fatalf("refresh token = %q, want refresh-1", result.RefreshToken)
	}
	if result.ExpiresIn != 3600 {
		t.Fatalf("expires_in = %d, want 3600", result.ExpiresIn)
	}
}

func TestExchangeAuthorizationCode_OAuthErrorBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"expired code"}`))
	}))
	defer srv.Close()

	_, err := exchangeAuthorizationCode(context.Background(), BrowserFlowConfig{
		TokenEndpoint: srv.URL,
		ClientID:      "client-id",
	}, "http://127.0.0.1:8181/callback", "bad-code", "verifier")
	if err == nil {
		t.Fatal("expected oauth token error")
	}
	if !strings.Contains(err.Error(), "oauth token error: invalid_grant (expired code)") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunBrowserFlow_SuccessViaLoopbackCallback(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse token form: %v", err)
		}
		if got := r.PostForm.Get("grant_type"); got != "authorization_code" {
			t.Fatalf("grant_type = %q, want authorization_code", got)
		}
		if got := r.PostForm.Get("code"); got != "callback-code" {
			t.Fatalf("code = %q, want callback-code", got)
		}
		if got := r.PostForm.Get("code_verifier"); strings.TrimSpace(got) == "" {
			t.Fatal("code_verifier should not be empty")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"flow-token","refresh_token":"flow-refresh","expires_in":1800,"scope":"read"}`))
	}))
	defer tokenSrv.Close()

	writer := newSyncWriter()
	resultCh := make(chan struct {
		res *BrowserFlowResult
		err error
	}, 1)

	go func() {
		res, err := RunBrowserFlow(context.Background(), BrowserFlowConfig{
			AuthEndpoint:  "https://auth.example.com/authorize",
			TokenEndpoint: tokenSrv.URL,
			ClientID:      "client-id",
			ClientSecret:  "client-secret",
			Scopes:        []string{"read"},
			NoOpen:        true,
			Timeout:       3 * time.Second,
		}, writer)
		resultCh <- struct {
			res *BrowserFlowResult
			err error
		}{res: res, err: err}
	}()

	authURL, err := writer.waitForAuthURL(2 * time.Second)
	if err != nil {
		t.Fatalf("wait for auth URL failed: %v\noutput=%s", err, writer.String())
	}

	parsedAuth, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("parse auth URL: %v", err)
	}
	q := parsedAuth.Query()
	redirectURI := q.Get("redirect_uri")
	state := q.Get("state")
	if redirectURI == "" || state == "" {
		t.Fatalf("missing redirect_uri/state in auth URL: %s", authURL)
	}

	callbackURL := redirectURI + "?code=callback-code&state=" + url.QueryEscape(state)
	resp, err := http.Get(callbackURL)
	if err != nil {
		t.Fatalf("call loopback callback URL: %v", err)
	}
	_ = resp.Body.Close()

	result := <-resultCh
	if result.err != nil {
		t.Fatalf("RunBrowserFlow() error = %v\noutput=%s", result.err, writer.String())
	}
	if result.res.AccessToken != "flow-token" {
		t.Fatalf("access token = %q, want flow-token", result.res.AccessToken)
	}
}

func TestRunBrowserFlow_StateMismatchRejected(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"unused"}`))
	}))
	defer tokenSrv.Close()

	writer := newSyncWriter()
	resultCh := make(chan error, 1)

	go func() {
		_, err := RunBrowserFlow(context.Background(), BrowserFlowConfig{
			AuthEndpoint:  "https://auth.example.com/authorize",
			TokenEndpoint: tokenSrv.URL,
			ClientID:      "client-id",
			NoOpen:        true,
			Timeout:       3 * time.Second,
		}, writer)
		resultCh <- err
	}()

	authURL, err := writer.waitForAuthURL(2 * time.Second)
	if err != nil {
		t.Fatalf("wait for auth URL failed: %v", err)
	}
	parsedAuth, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("parse auth URL: %v", err)
	}
	redirectURI := parsedAuth.Query().Get("redirect_uri")
	if redirectURI == "" {
		t.Fatalf("missing redirect_uri in auth URL: %s", authURL)
	}

	resp, err := http.Get(redirectURI + "?code=callback-code&state=wrong-state")
	if err != nil {
		t.Fatalf("call loopback callback URL: %v", err)
	}
	_ = resp.Body.Close()

	runErr := <-resultCh
	if runErr == nil {
		t.Fatal("expected oauth state mismatch error")
	}
	if !strings.Contains(runErr.Error(), "oauth state mismatch") {
		t.Fatalf("unexpected error: %v", runErr)
	}
}

func TestRunBrowserFlow_TimesOutWithoutCallback(t *testing.T) {
	_, err := RunBrowserFlow(context.Background(), BrowserFlowConfig{
		AuthEndpoint:  "https://auth.example.com/authorize",
		TokenEndpoint: "https://auth.example.com/token",
		ClientID:      "client-id",
		NoOpen:        true,
		Timeout:       30 * time.Millisecond,
	}, io.Discard)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("unexpected timeout error: %v", err)
	}
}
