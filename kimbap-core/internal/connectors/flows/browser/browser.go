package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxTokenBodyBytes int64 = 4 << 20

type BrowserFlowConfig struct {
	AuthEndpoint  string
	TokenEndpoint string
	ClientID      string
	ClientSecret  string
	RedirectURI   string
	Scopes        []string
	Port          int
	NoOpen        bool
	Timeout       time.Duration
}

type BrowserFlowResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	Scope        string
	Raw          map[string]any
}

type callbackPayload struct {
	code  string
	state string
	err   error
}

func RunBrowserFlow(ctx context.Context, cfg BrowserFlowConfig, output io.Writer) (*BrowserFlowResult, error) {
	if strings.TrimSpace(cfg.AuthEndpoint) == "" {
		return nil, errors.New("auth endpoint is required")
	}
	if strings.TrimSpace(cfg.TokenEndpoint) == "" {
		return nil, errors.New("token endpoint is required")
	}
	if strings.TrimSpace(cfg.ClientID) == "" {
		return nil, errors.New("client id is required")
	}
	if output == nil {
		output = io.Discard
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	verifier, challenge, err := GeneratePKCE()
	if err != nil {
		return nil, fmt.Errorf("generate pkce: %w", err)
	}
	state, err := GenerateState()
	if err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}

	bindPort := cfg.Port
	if bindPort == 0 && cfg.RedirectURI != "" {
		if redirectURL, parseErr := url.Parse(cfg.RedirectURI); parseErr == nil {
			if p := redirectURL.Port(); p != "" {
				if pi, convErr := strconv.Atoi(p); convErr == nil {
					bindPort = pi
				}
			}
		}
	}

	listener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(bindPort)))
	if err != nil {
		return nil, fmt.Errorf("start loopback listener: %w", err)
	}
	defer listener.Close()

	actualPort := listener.Addr().(*net.TCPAddr).Port
	redirectURI := cfg.RedirectURI
	if redirectURI == "" {
		redirectURI = fmt.Sprintf("http://127.0.0.1:%d/callback", actualPort)
	}

	parsedRedirect, err := url.Parse(redirectURI)
	if err != nil {
		return nil, fmt.Errorf("invalid redirect uri: %w", err)
	}
	callbackPath := parsedRedirect.EscapedPath()
	if callbackPath == "" {
		callbackPath = "/"
	}

	authURL, err := buildAuthorizationURL(cfg, redirectURI, state, challenge)
	if err != nil {
		return nil, err
	}

	_, _ = fmt.Fprintf(output, "Open this URL to continue authentication:\n%s\n", authURL)

	resultCh := make(chan callbackPayload, 1)
	serverErrCh := make(chan error, 1)
	var once sync.Once

	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != callbackPath {
			http.NotFound(w, r)
			return
		}

		query := r.URL.Query()
		if oauthErr := query.Get("error"); oauthErr != "" {
			desc := query.Get("error_description")
			if desc != "" {
				oauthErr = oauthErr + ": " + desc
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("<html><body><h1>Authentication failed</h1><p>You can close this window.</p></body></html>"))
			once.Do(func() {
				resultCh <- callbackPayload{err: errors.New(oauthErr)}
			})
			return
		}

		code := query.Get("code")
		cbState := query.Get("state")
		if code == "" || cbState == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("<html><body><h1>Invalid callback</h1><p>Missing OAuth parameters. You can close this window.</p></body></html>"))
			once.Do(func() {
				resultCh <- callbackPayload{err: errors.New("callback missing code or state")}
			})
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html><body><h1>Authentication complete</h1><p>You can return to the terminal.</p></body></html>"))
		once.Do(func() {
			resultCh <- callbackPayload{code: code, state: cbState}
		})
	})}

	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			serverErrCh <- serveErr
		}
	}()

	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer shutdownCancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if !cfg.NoOpen {
		if err := openBrowser(authURL); err != nil {
			_, _ = fmt.Fprintf(output, "Unable to open browser automatically: %v\n", err)
		}
	}

	var callback callbackPayload
	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, errors.New("browser flow timed out waiting for callback")
		}
		return nil, fmt.Errorf("browser flow canceled: %w", ctx.Err())
	case serveErr := <-serverErrCh:
		return nil, fmt.Errorf("loopback server error: %w", serveErr)
	case callback = <-resultCh:
	}

	if callback.err != nil {
		return nil, callback.err
	}
	if !ValidateState(state, callback.state) {
		return nil, errors.New("oauth state mismatch")
	}

	result, err := exchangeAuthorizationCode(ctx, cfg, redirectURI, callback.code, verifier)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func buildAuthorizationURL(cfg BrowserFlowConfig, redirectURI, state, challenge string) (string, error) {
	parsed, err := url.Parse(cfg.AuthEndpoint)
	if err != nil {
		return "", fmt.Errorf("invalid auth endpoint: %w", err)
	}

	query := parsed.Query()
	query.Set("client_id", cfg.ClientID)
	query.Set("redirect_uri", redirectURI)
	query.Set("response_type", "code")
	query.Set("state", state)
	query.Set("code_challenge", challenge)
	query.Set("code_challenge_method", "S256")
	if len(cfg.Scopes) > 0 {
		query.Set("scope", strings.Join(cfg.Scopes, " "))
	}

	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func exchangeAuthorizationCode(ctx context.Context, cfg BrowserFlowConfig, redirectURI, code, verifier string) (*BrowserFlowResult, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", cfg.ClientID)
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("code_verifier", verifier)
	if strings.TrimSpace(cfg.ClientSecret) != "" {
		form.Set("client_secret", cfg.ClientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.TokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("exchange authorization code: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, maxTokenBodyBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read token response: %w", err)
	}
	if int64(len(body)) > maxTokenBodyBytes {
		return nil, fmt.Errorf("token response exceeded %d bytes", maxTokenBodyBytes)
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		oauthErr := stringFromAny(raw["error"])
		oauthDesc := stringFromAny(raw["error_description"])
		if oauthErr != "" && oauthDesc != "" {
			return nil, fmt.Errorf("oauth token error: %s (%s)", oauthErr, oauthDesc)
		}
		if oauthErr != "" {
			return nil, fmt.Errorf("oauth token error: %s", oauthErr)
		}
		return nil, fmt.Errorf("token endpoint returned status %d", res.StatusCode)
	}

	accessToken := stringFromAny(raw["access_token"])
	if accessToken == "" {
		return nil, errors.New("token response missing access_token")
	}

	return &BrowserFlowResult{
		AccessToken:  accessToken,
		RefreshToken: stringFromAny(raw["refresh_token"]),
		ExpiresIn:    intFromAny(raw["expires_in"]),
		Scope:        stringFromAny(raw["scope"]),
		Raw:          raw,
	}, nil
}

func openBrowser(authURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", authURL)
	case "linux":
		cmd = exec.Command("xdg-open", authURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", authURL)
	default:
		return fmt.Errorf("unsupported platform %q", runtime.GOOS)
	}
	return cmd.Start()
}

func stringFromAny(v any) string {
	s, _ := v.(string)
	return s
}

func intFromAny(v any) int {
	switch value := v.(type) {
	case float64:
		return int(value)
	case int:
		return value
	case int64:
		return int(value)
	case json.Number:
		n, err := value.Int64()
		if err == nil {
			return int(n)
		}
	}
	return 0
}
