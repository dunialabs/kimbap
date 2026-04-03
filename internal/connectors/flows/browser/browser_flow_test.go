package browser

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
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

func TestRunBrowserFlow_RejectsCustomRedirectURIWithNonLoopbackHost(t *testing.T) {
	_, err := RunBrowserFlow(context.Background(), BrowserFlowConfig{
		AuthEndpoint:  "https://auth.example.com/authorize",
		TokenEndpoint: "https://auth.example.com/token",
		ClientID:      "client-id",
		RedirectURI:   "http://example.com/callback",
		NoOpen:        true,
	}, io.Discard)
	if err == nil {
		t.Fatal("expected redirect URI validation error")
	}
	if !strings.Contains(err.Error(), "must be loopback") {
		t.Fatalf("expected loopback validation error, got %v", err)
	}
}

func TestRunBrowserFlow_AssignsListenerPortToCustomRedirectURIWithoutPort(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse token form: %v", err)
		}
		redirectURI := r.PostForm.Get("redirect_uri")
		parsedRedirect, err := url.Parse(redirectURI)
		if err != nil {
			t.Fatalf("parse redirect_uri: %v", err)
		}
		if parsedRedirect.Port() == "" {
			t.Fatalf("redirect_uri %q should include assigned listener port", redirectURI)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"portless-token"}`))
	}))
	defer tokenSrv.Close()

	writer := newSyncWriter()
	resultCh := make(chan struct {
		res *BrowserFlowResult
		err error
	}, 1)

	go func() {
		var nilCtx context.Context
		res, runErr := RunBrowserFlow(nilCtx, BrowserFlowConfig{
			AuthEndpoint:  "https://auth.example.com/authorize",
			TokenEndpoint: tokenSrv.URL,
			ClientID:      "client-id",
			RedirectURI:   "http://127.0.0.1/oauth/callback",
			NoOpen:        true,
			Timeout:       3 * time.Second,
		}, writer)
		resultCh <- struct {
			res *BrowserFlowResult
			err error
		}{res: res, err: runErr}
	}()

	authURL, err := writer.waitForAuthURL(2 * time.Second)
	if err != nil {
		t.Fatalf("wait for auth URL: %v", err)
	}
	parsedAuth, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("parse auth URL: %v", err)
	}
	redirectURI := parsedAuth.Query().Get("redirect_uri")
	parsedRedirect, err := url.Parse(redirectURI)
	if err != nil {
		t.Fatalf("parse redirect uri from auth URL: %v", err)
	}
	if parsedRedirect.Port() == "" {
		t.Fatalf("auth URL redirect_uri %q should include assigned port", redirectURI)
	}
	state := parsedAuth.Query().Get("state")
	if state == "" {
		t.Fatalf("auth URL state should be present: %s", authURL)
	}

	resp, err := http.Get(redirectURI + "?code=callback-code&state=" + url.QueryEscape(state))
	if err != nil {
		t.Fatalf("call assigned redirect URI: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		t.Fatalf("callback response status = %d, want 200", resp.StatusCode)
	}
	_ = resp.Body.Close()

	result := <-resultCh
	if result.err != nil {
		t.Fatalf("RunBrowserFlow() error = %v", result.err)
	}
	if result.res == nil || result.res.AccessToken != "portless-token" {
		t.Fatalf("unexpected browser flow result: %+v", result.res)
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
		if got := r.PostForm.Get("client_secret"); got != "client-secret" {
			t.Fatalf("client_secret = %q, want client-secret", got)
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

func TestNewLoopbackCallbackServerSetsTimeouts(t *testing.T) {
	srv := newLoopbackCallbackServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	if srv.ReadHeaderTimeout != 5*time.Second {
		t.Fatalf("expected read header timeout 5s, got %s", srv.ReadHeaderTimeout)
	}
	if srv.ReadTimeout != 30*time.Second {
		t.Fatalf("expected read timeout 30s, got %s", srv.ReadTimeout)
	}
	if srv.WriteTimeout != 30*time.Second {
		t.Fatalf("expected write timeout 30s, got %s", srv.WriteTimeout)
	}
	if srv.IdleTimeout != 30*time.Second {
		t.Fatalf("expected idle timeout 30s, got %s", srv.IdleTimeout)
	}
}

func TestExchangeAuthorizationCode_UsesBasicAuthWhenConfigured(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); !strings.HasPrefix(got, "Basic ") {
			t.Fatalf("expected basic auth header, got %q", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := r.PostForm.Get("client_secret"); got != "" {
			t.Fatalf("client_secret should be omitted from body for basic auth, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token-basic"}`))
	}))
	defer srv.Close()

	result, err := exchangeAuthorizationCode(context.Background(), BrowserFlowConfig{
		TokenEndpoint: srv.URL,
		ClientID:      "client-id",
		ClientSecret:  "client-secret",
		AuthMethod:    "basic",
	}, "http://127.0.0.1:8181/callback", "auth-code", "verifier-abc")
	if err != nil {
		t.Fatalf("exchangeAuthorizationCode() error = %v", err)
	}
	if result.AccessToken != "token-basic" {
		t.Fatalf("access token = %q, want token-basic", result.AccessToken)
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
	if got := q.Get("client_id"); got != "client-id" {
		t.Fatalf("auth URL client_id = %q, want client-id", got)
	}
	if got := q.Get("response_type"); got != "code" {
		t.Fatalf("auth URL response_type = %q, want code", got)
	}
	if got := q.Get("code_challenge_method"); got != "S256" {
		t.Fatalf("auth URL code_challenge_method = %q, want S256", got)
	}
	if strings.TrimSpace(q.Get("code_challenge")) == "" {
		t.Fatal("auth URL code_challenge should be non-empty")
	}
	if got := q.Get("scope"); got != "read" {
		t.Fatalf("auth URL scope = %q, want read", got)
	}

	callbackURL := redirectURI + "?code=callback-code&state=" + url.QueryEscape(state)
	resp, err := http.Get(callbackURL)
	if err != nil {
		t.Fatalf("call loopback callback URL: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("callback response status = %d, want 200", resp.StatusCode)
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
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("callback response status = %d, want 400 for wrong state", resp.StatusCode)
	}
	_ = resp.Body.Close()

	runErr := <-resultCh
	if runErr == nil {
		t.Fatal("expected error after wrong state callback")
	}
	if !strings.Contains(runErr.Error(), "oauth state mismatch") {
		t.Fatalf("expected oauth state mismatch error, got: %v", runErr)
	}
}

func TestRunBrowserFlow_TimesOutWithoutCallback(t *testing.T) {
	_, err := RunBrowserFlow(context.Background(), BrowserFlowConfig{
		AuthEndpoint:  "https://auth.example.com/authorize",
		TokenEndpoint: "https://auth.example.com/token",
		ClientID:      "client-id",
		NoOpen:        true,
		Timeout:       300 * time.Millisecond,
	}, io.Discard)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("unexpected timeout error: %v", err)
	}
}

func TestRunBrowserFlow_CallbackOAuthErrorReturned(t *testing.T) {
	var tokenCalls int32
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&tokenCalls, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"should-not-be-used"}`))
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

	resp, err := http.Get(redirectURI + "?error=access_denied&error_description=user+denied")
	if err != nil {
		t.Fatalf("call loopback callback URL: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("callback response status = %d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()

	runErr := <-resultCh
	if runErr == nil {
		t.Fatal("expected callback oauth error")
	}
	if !strings.Contains(runErr.Error(), "access_denied: user denied") {
		t.Fatalf("unexpected callback error: %v", runErr)
	}
	if got := atomic.LoadInt32(&tokenCalls); got != 0 {
		t.Fatalf("token endpoint should not be called on callback oauth error, got %d call(s)", got)
	}
}

func TestRunBrowserFlow_CallbackMissingCodeOrState(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{name: "missing state", query: "code=callback-code"},
		{name: "missing code", query: "state=callback-state"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var tokenCalls int32
			tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				atomic.AddInt32(&tokenCalls, 1)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"access_token":"should-not-be-used"}`))
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

			resp, err := http.Get(redirectURI + "?" + tc.query)
			if err != nil {
				t.Fatalf("call loopback callback URL: %v", err)
			}
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("callback response status = %d, want 400", resp.StatusCode)
			}
			_ = resp.Body.Close()

			runErr := <-resultCh
			if runErr == nil {
				t.Fatal("expected callback missing code/state error")
			}
			if !strings.Contains(runErr.Error(), "callback missing code or state") {
				t.Fatalf("unexpected callback error: %v", runErr)
			}
			if got := atomic.LoadInt32(&tokenCalls); got != 0 {
				t.Fatalf("token endpoint should not be called on malformed callback, got %d call(s)", got)
			}
		})
	}
}

func TestRunBrowserFlow_CustomRedirectURIPathAndPort(t *testing.T) {
	const maxAttempts = 8
	var lastBindErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("allocate free port: %v", err)
		}
		port := listener.Addr().(*net.TCPAddr).Port
		_ = listener.Close()

		err = runCustomRedirectURIFlowAttempt(t, port)
		if err == nil {
			return
		}
		if errors.Is(err, ErrLoopbackListener) {
			lastBindErr = err
			continue
		}
		t.Fatalf("custom redirect flow attempt %d failed: %v", attempt, err)
	}

	t.Fatalf("custom redirect flow failed after %d attempts due loopback bind race: %v", maxAttempts, lastBindErr)
}

func runCustomRedirectURIFlowAttempt(t *testing.T, port int) error {
	t.Helper()

	customRedirectURI := fmt.Sprintf("http://127.0.0.1:%d/oauth/callback", port)

	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse token form: %v", err)
		}
		if got := r.PostForm.Get("redirect_uri"); got != customRedirectURI {
			t.Fatalf("redirect_uri in token exchange = %q, want %q", got, customRedirectURI)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"custom-redirect-token"}`))
	}))
	defer tokenSrv.Close()

	writer := newSyncWriter()
	resultCh := make(chan struct {
		res *BrowserFlowResult
		err error
	}, 1)

	go func() {
		res, runErr := RunBrowserFlow(context.Background(), BrowserFlowConfig{
			AuthEndpoint:  "https://auth.example.com/authorize",
			TokenEndpoint: tokenSrv.URL,
			ClientID:      "client-id",
			RedirectURI:   customRedirectURI,
			NoOpen:        true,
			Timeout:       3 * time.Second,
		}, writer)
		resultCh <- struct {
			res *BrowserFlowResult
			err error
		}{res: res, err: runErr}
	}()

	authURL, err := writer.waitForAuthURL(2 * time.Second)
	if err != nil {
		select {
		case result := <-resultCh:
			if errors.Is(result.err, ErrLoopbackListener) {
				return result.err
			}
			if result.err != nil {
				return fmt.Errorf("wait for auth URL failed: %w (flow error: %v)", err, result.err)
			}
		default:
		}
		return fmt.Errorf("wait for auth URL failed: %w", err)
	}

	parsedAuth, err := url.Parse(authURL)
	if err != nil {
		return fmt.Errorf("parse auth URL: %w", err)
	}
	q := parsedAuth.Query()
	if got := q.Get("redirect_uri"); got != customRedirectURI {
		return fmt.Errorf("auth URL redirect_uri = %q, want %q", got, customRedirectURI)
	}
	state := q.Get("state")
	if state == "" {
		return fmt.Errorf("auth URL state should be present: %s", authURL)
	}

	wrongPathURL := fmt.Sprintf("http://127.0.0.1:%d/callback?code=wrong&state=%s", port, url.QueryEscape(state))
	wrongResp, err := http.Get(wrongPathURL)
	if err != nil {
		return fmt.Errorf("call wrong callback path: %w", err)
	}
	if wrongResp.StatusCode != http.StatusNotFound {
		_ = wrongResp.Body.Close()
		return fmt.Errorf("wrong callback path status = %d, want 404", wrongResp.StatusCode)
	}
	_ = wrongResp.Body.Close()

	resp, err := http.Get(customRedirectURI + "?code=custom-code&state=" + url.QueryEscape(state))
	if err != nil {
		return fmt.Errorf("call custom callback URL: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return fmt.Errorf("custom callback response status = %d, want 200", resp.StatusCode)
	}
	_ = resp.Body.Close()

	result := <-resultCh
	if result.err != nil {
		return result.err
	}
	if result.res == nil || result.res.AccessToken != "custom-redirect-token" {
		return fmt.Errorf("unexpected custom redirect result: %+v", result.res)
	}

	return nil
}
