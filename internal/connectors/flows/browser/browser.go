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
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dunialabs/kimbap/internal/connectors"
)

var ErrLoopbackListener = errors.New("loopback listener unavailable")

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
	AuthMethod    string
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
		return nil, fmt.Errorf("%w: %v", ErrLoopbackListener, err)
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
		cbState := query.Get("state")
		if oauthErr := query.Get("error"); oauthErr != "" {
			if cbState != "" && !ValidateState(state, cbState) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
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
		if code == "" || cbState == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("<html><body><h1>Invalid callback</h1><p>Missing OAuth parameters. You can close this window.</p></body></html>"))
			once.Do(func() {
				resultCh <- callbackPayload{err: errors.New("callback missing code or state")}
			})
			return
		}

		if !ValidateState(state, cbState) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("<html><body><h1>Authentication failed</h1><p>Invalid state parameter. You can close this window.</p></body></html>"))
			once.Do(func() {
				resultCh <- callbackPayload{err: errors.New("oauth state mismatch")}
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

	body, err := connectors.PostFormWithAuth(ctx, cfg.TokenEndpoint, form, cfg.AuthMethod, cfg.ClientID, cfg.ClientSecret)
	if err != nil {
		var httpErr *connectors.OAuthHTTPError
		if errors.As(err, &httpErr) {
			var raw map[string]any
			if json.Unmarshal(httpErr.RawBody, &raw) == nil {
				if oauthErr := stringFromAny(raw["error"]); oauthErr != "" {
					oauthDesc := stringFromAny(raw["error_description"])
					if oauthDesc != "" {
						return nil, fmt.Errorf("oauth token error: %s (%s)", oauthErr, oauthDesc)
					}
					return nil, fmt.Errorf("oauth token error: %s", oauthErr)
				}
			}
		}
		return nil, fmt.Errorf("exchange authorization code: %w", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}

	if oauthErr := stringFromAny(raw["error"]); oauthErr != "" {
		oauthDesc := stringFromAny(raw["error_description"])
		if oauthDesc != "" {
			return nil, fmt.Errorf("oauth token error: %s (%s)", oauthErr, oauthDesc)
		}
		return nil, fmt.Errorf("oauth token error: %s", oauthErr)
	}

	accessToken := stringFromAny(raw["access_token"])
	if accessToken == "" {
		return nil, errors.New("token response missing access_token")
	}

	scope := stringFromAny(raw["scope"])
	if scope == "" {
		scope = strings.Join(cfg.Scopes, " ")
	}
	return &BrowserFlowResult{
		AccessToken:  accessToken,
		RefreshToken: stringFromAny(raw["refresh_token"]),
		ExpiresIn:    intFromAny(raw["expires_in"]),
		Scope:        scope,
		Raw:          raw,
	}, nil
}

func openBrowser(authURL string) error {
	name, args, err := browserOpenCommandForOS(runtime.GOOS, authURL, exec.LookPath, os.Getenv("BROWSER"))
	if err != nil {
		return err
	}
	cmd := exec.Command(name, args...)
	if err := cmd.Start(); err != nil {
		return err
	}
	_ = cmd.Process.Release()
	return nil
}

func browserOpenCommandForOS(goos, authURL string, lookPath func(file string) (string, error), browserEnv string) (string, []string, error) {
	switch goos {
	case "darwin":
		return "open", []string{authURL}, nil
	case "linux":
		if name, args, ok := parseBrowserEnv(browserEnv, authURL); ok {
			return name, args, nil
		}
		if lookPath != nil {
			if _, err := lookPath("xdg-open"); err == nil {
				return "xdg-open", []string{authURL}, nil
			}
		}
		return "", nil, errors.New("xdg-open is not installed (install xdg-utils) and BROWSER is not set")
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", authURL}, nil
	default:
		return "", nil, fmt.Errorf("unsupported platform %q", goos)
	}
}

func parseBrowserEnv(browserEnv, authURL string) (string, []string, bool) {
	browserEnv = strings.TrimSpace(browserEnv)
	if browserEnv == "" {
		return "", nil, false
	}
	choices := splitBrowserChoices(browserEnv)
	for _, choice := range choices {
		parts, err := shellLikeFields(choice)
		if err != nil || len(parts) == 0 {
			continue
		}
		args := make([]string, 0, len(parts))
		replaced := false
		for _, arg := range parts[1:] {
			if strings.Contains(arg, "%s") {
				replaced = true
				args = append(args, strings.ReplaceAll(arg, "%s", authURL))
				continue
			}
			args = append(args, arg)
		}
		if !replaced {
			args = append(args, authURL)
		}
		return parts[0], args, true
	}
	return "", nil, false
}

func splitBrowserChoices(raw string) []string {
	var (
		choices    []string
		buf        strings.Builder
		inSingle   bool
		inDouble   bool
		escapeNext bool
	)
	for _, r := range raw {
		if escapeNext {
			buf.WriteRune(r)
			escapeNext = false
			continue
		}
		switch r {
		case '\\':
			escapeNext = true
			buf.WriteRune(r)
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
			buf.WriteRune(r)
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
			buf.WriteRune(r)
		case ':':
			if !inSingle && !inDouble {
				choice := strings.TrimSpace(buf.String())
				if choice != "" {
					choices = append(choices, choice)
				}
				buf.Reset()
				continue
			}
			buf.WriteRune(r)
		default:
			buf.WriteRune(r)
		}
	}
	if tail := strings.TrimSpace(buf.String()); tail != "" {
		choices = append(choices, tail)
	}
	return choices
}

func shellLikeFields(raw string) ([]string, error) {
	var (
		out        []string
		buf        strings.Builder
		inSingle   bool
		inDouble   bool
		escapeNext bool
	)
	flush := func() {
		if buf.Len() == 0 {
			return
		}
		out = append(out, buf.String())
		buf.Reset()
	}

	for _, r := range raw {
		if escapeNext {
			buf.WriteRune(r)
			escapeNext = false
			continue
		}
		switch r {
		case '\\':
			if inSingle {
				buf.WriteRune(r)
				continue
			}
			escapeNext = true
		case '\'':
			if inDouble {
				buf.WriteRune(r)
				continue
			}
			inSingle = !inSingle
		case '"':
			if inSingle {
				buf.WriteRune(r)
				continue
			}
			inDouble = !inDouble
		case ' ', '\t', '\n':
			if inSingle || inDouble {
				buf.WriteRune(r)
				continue
			}
			flush()
		default:
			buf.WriteRune(r)
		}
	}
	if escapeNext {
		buf.WriteRune('\\')
	}
	if inSingle || inDouble {
		return nil, errors.New("unclosed quote in BROWSER")
	}
	flush()
	return out, nil
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
