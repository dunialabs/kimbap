package proxy

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/classifier"
	"github.com/dunialabs/kimbap/internal/runtime"
	"github.com/rs/zerolog/log"
)

type UnmatchedPolicy string

const (
	UnmatchedPolicyAllow UnmatchedPolicy = "allow"
	UnmatchedPolicyDeny  UnmatchedPolicy = "deny"
)

const (
	defaultProxyReadHeaderTimeout  = 5 * time.Second
	defaultProxyReadTimeout        = 60 * time.Second
	defaultProxyWriteTimeout       = 60 * time.Second
	defaultProxyIdleTimeout        = 60 * time.Second
	defaultProxyShutdownTimeout    = 5 * time.Second
	defaultProxyHandshakeTimeout   = 10 * time.Second
	defaultProxyConnReadTimeout    = 60 * time.Second
	defaultProxyConnWriteTimeout   = 60 * time.Second
	maxProxyRequestBodyBytes       = 4 << 20
	maxProxyConnRequestHeaderBytes = 64 << 10 // 64 KB — bounds bufio.NewReader on CONNECT tunnel
)

var proxyForwardTransport = &http.Transport{
	Proxy: nil,
	DialContext: (&net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext,
	TLSClientConfig:        &tls.Config{InsecureSkipVerify: false},
	TLSHandshakeTimeout:    10 * time.Second,
	ResponseHeaderTimeout:  30 * time.Second,
	IdleConnTimeout:        90 * time.Second,
	MaxResponseHeaderBytes: 1 << 20,
	ForceAttemptHTTP2:      false,
	DisableCompression:     false,
	MaxIdleConns:           100,
	MaxIdleConnsPerHost:    10,
}

type ProxyOption func(*ProxyServer)

type ProxyServer struct {
	listenAddr string
	ca         *CAConfig
	classifier *classifier.Classifier
	runtime    *runtime.Runtime
	agentToken string
	TenantID   string
	AgentName  string

	unmatchedPolicy UnmatchedPolicy

	server   *http.Server
	listener net.Listener
	mu       sync.RWMutex
}

func NewProxyServer(addr string, ca *CAConfig, opts ...ProxyOption) *ProxyServer {
	p := &ProxyServer{
		listenAddr:      addr,
		ca:              ca,
		classifier:      classifier.NewClassifier(),
		unmatchedPolicy: UnmatchedPolicyDeny,
		TenantID:        "default",
		AgentName:       "kimbap-proxy",
	}
	for _, opt := range opts {
		if opt != nil {
			opt(p)
		}
	}
	return p
}

func WithClassifier(c *classifier.Classifier) ProxyOption {
	return func(p *ProxyServer) {
		if c != nil {
			p.classifier = c
		}
	}
}

func WithRuntime(rt *runtime.Runtime) ProxyOption {
	return func(p *ProxyServer) {
		p.runtime = rt
	}
}

func WithAgentToken(token string) ProxyOption {
	return func(p *ProxyServer) {
		p.agentToken = strings.TrimSpace(token)
	}
}

func WithTenantID(id string) ProxyOption {
	return func(p *ProxyServer) {
		trimmed := strings.TrimSpace(id)
		if trimmed != "" {
			p.TenantID = trimmed
		}
	}
}

func WithAgentName(name string) ProxyOption {
	return func(p *ProxyServer) {
		trimmed := strings.TrimSpace(name)
		if trimmed != "" {
			p.AgentName = trimmed
		}
	}
}

func WithUnmatchedPolicy(policy UnmatchedPolicy) ProxyOption {
	return func(p *ProxyServer) {
		switch policy {
		case UnmatchedPolicyAllow, UnmatchedPolicyDeny:
			p.unmatchedPolicy = policy
		}
	}
}

func (p *ProxyServer) Addr() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.listener != nil {
		return p.listener.Addr().String()
	}
	return p.listenAddr
}

func (p *ProxyServer) Ready() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.listener != nil
}

func (p *ProxyServer) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", p.listenAddr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	srv := newProxyHTTPServer(p)

	p.mu.Lock()
	p.listener = ln
	p.server = srv
	p.mu.Unlock()

	shutdownDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultProxyShutdownTimeout)
			defer cancel()
			_ = p.Stop(shutdownCtx)
		case <-shutdownDone:
		}
	}()

	err = srv.Serve(ln)
	close(shutdownDone)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve proxy: %w", err)
	}
	return nil
}

func (p *ProxyServer) Stop(ctx context.Context) error {
	p.mu.Lock()
	srv := p.server
	p.server = nil
	p.listener = nil
	p.mu.Unlock()
	if srv == nil {
		return nil
	}
	if err := srv.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		_ = srv.Close()
		return err
	}
	return nil
}

func newProxyHTTPServer(handler http.Handler) *http.Server {
	return &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: defaultProxyReadHeaderTimeout,
		ReadTimeout:       defaultProxyReadTimeout,
		WriteTimeout:      defaultProxyWriteTimeout,
		IdleTimeout:       defaultProxyIdleTimeout,
	}
}

func (p *ProxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.EqualFold(r.Method, http.MethodConnect) {
		p.handleConnect(w, r)
		return
	}
	p.handleHTTP(w, r)
}

func isWebSocketUpgrade(req *http.Request) bool {
	return strings.EqualFold(req.Header.Get("Upgrade"), "websocket") &&
		strings.Contains(strings.ToLower(req.Header.Get("Connection")), "upgrade")
}

func (p *ProxyServer) handleHTTP(w http.ResponseWriter, req *http.Request) {
	if isWebSocketUpgrade(req) {
		http.Error(w, "websocket is not supported in proxy mode v1", http.StatusNotImplemented)
		return
	}
	if !p.validateProxyAuth(req) {
		w.Header().Set("Proxy-Authenticate", `Basic realm="kimbap"`)
		http.Error(w, "proxy authentication required", http.StatusProxyAuthRequired)
		return
	}
	reqID := newRequestID()
	targetHost, targetPath := targetFromRequest(req)

	classification := p.classify(req.Method, targetHost, targetPath)
	if classification != nil && classification.Matched {
		req.Header.Set("X-Kimbap-Request-ID", reqID)
		result := p.executeClassifiedRequest(req.Context(), req, reqID, classification, targetHost, targetPath)
		if result.Error != nil && runtimeResultStatus(result) >= 500 {
			http.Error(w, "proxy request failed", runtimeResultStatus(result))
			return
		}
		writeRuntimeHTTPResponse(w, result)
		return
	} else if p.unmatchedPolicy == UnmatchedPolicyDeny {
		http.Error(w, "proxy request denied", http.StatusForbidden)
		return
	} else {
		req.Header.Set("X-Kimbap-Request-ID", reqID)
	}

	resp, err := p.forwardRequest(req)
	if err != nil {
		http.Error(w, "proxy request failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	if written, copyErr := io.Copy(w, resp.Body); copyErr != nil {
		log.Warn().Err(copyErr).Str("requestId", reqID).Int64("bytesWritten", written).Msg("failed to write proxy upstream response body")
	}
}

func (p *ProxyServer) handleConnect(w http.ResponseWriter, req *http.Request) {
	if !p.validateProxyAuth(req) {
		w.Header().Set("Proxy-Authenticate", `Basic realm="kimbap"`)
		http.Error(w, "proxy authentication required", http.StatusProxyAuthRequired)
		return
	}
	if p.ca == nil {
		http.Error(w, "proxy request denied", http.StatusBadGateway)
		return
	}

	host, port := splitHostPort(req.Host)
	if host == "" {
		http.Error(w, "proxy request denied", http.StatusBadRequest)
		return
	}
	connectHost := net.JoinHostPort(host, port)

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "proxy request failed", http.StatusInternalServerError)
		return
	}

	clientConn, rw, err := hj.Hijack()
	if err != nil {
		http.Error(w, "proxy request failed", http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	_ = clientConn.SetWriteDeadline(time.Now().Add(defaultProxyConnWriteTimeout))
	if _, err := rw.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n"); err != nil {
		return
	}
	if err := rw.Flush(); err != nil {
		return
	}
	_ = clientConn.SetWriteDeadline(time.Time{})

	hostCert, err := GenerateHostCert(p.ca, host)
	if err != nil {
		return
	}

	tlsConn := tls.Server(clientConn, &tls.Config{Certificates: []tls.Certificate{*hostCert}})
	defer tlsConn.Close()
	if err := tlsConn.SetDeadline(time.Now().Add(defaultProxyHandshakeTimeout)); err != nil {
		return
	}
	if err := tlsConn.Handshake(); err != nil {
		return
	}
	_ = tlsConn.SetDeadline(time.Time{})

	drainBody := func(body io.ReadCloser) {
		if body == nil {
			return
		}
		_, _ = io.Copy(io.Discard, io.LimitReader(body, maxProxyRequestBodyBytes))
		_ = body.Close()
	}

	reader := bufio.NewReaderSize(tlsConn, int(maxProxyConnRequestHeaderBytes))
	for {
		if err := tlsConn.SetReadDeadline(time.Now().Add(defaultProxyConnReadTimeout)); err != nil {
			return
		}
		mitmReq, err := http.ReadRequest(reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			return
		}
		_ = tlsConn.SetReadDeadline(time.Now().Add(defaultProxyReadTimeout))

		if mitmReq.URL == nil {
			mitmReq.URL = &url.URL{}
		}
		mitmReq.URL.Scheme = "https"
		mitmReq.URL.Host = connectHost
		if mitmReq.Host == "" {
			mitmReq.Host = connectHost
		}

		if isWebSocketUpgrade(mitmReq) {
			_ = tlsConn.SetWriteDeadline(time.Now().Add(defaultProxyConnWriteTimeout))
			_ = writePlainErrorResponse(tlsConn, http.StatusNotImplemented, "websocket is not supported in proxy mode v1")
			_ = tlsConn.SetWriteDeadline(time.Time{})
			drainBody(mitmReq.Body)
			continue
		}

		reqID := newRequestID()
		classification := p.classify(mitmReq.Method, connectHost, mitmReq.URL.Path)
		if classification != nil && classification.Matched {
			mitmReq.Header.Set("X-Kimbap-Request-ID", reqID)
			result := p.executeClassifiedRequest(req.Context(), mitmReq, reqID, classification, connectHost, mitmReq.URL.Path)
			if result.Error != nil && runtimeResultStatus(result) >= 500 {
				_ = tlsConn.SetWriteDeadline(time.Now().Add(defaultProxyConnWriteTimeout))
				_ = writePlainErrorResponse(tlsConn, runtimeResultStatus(result), "proxy request failed")
				_ = tlsConn.SetWriteDeadline(time.Time{})
				drainBody(mitmReq.Body)
				continue
			}
			_ = tlsConn.SetWriteDeadline(time.Now().Add(defaultProxyConnWriteTimeout))
			if err := writeRuntimeConnResponse(tlsConn, result); err != nil {
				_ = tlsConn.SetWriteDeadline(time.Time{})
				drainBody(mitmReq.Body)
				return
			}
			_ = tlsConn.SetWriteDeadline(time.Time{})
			drainBody(mitmReq.Body)
			if mitmReq.Close {
				return
			}
			continue
		} else if p.unmatchedPolicy == UnmatchedPolicyDeny {
			_ = tlsConn.SetWriteDeadline(time.Now().Add(defaultProxyConnWriteTimeout))
			_ = writePlainErrorResponse(tlsConn, http.StatusForbidden, "proxy request denied")
			_ = tlsConn.SetWriteDeadline(time.Time{})
			drainBody(mitmReq.Body)
			continue
		} else {
			mitmReq.Header.Set("X-Kimbap-Request-ID", reqID)
		}

		resp, ferr := p.forwardRequest(mitmReq)
		if ferr != nil {
			_ = tlsConn.SetWriteDeadline(time.Now().Add(defaultProxyConnWriteTimeout))
			_ = writePlainErrorResponse(tlsConn, http.StatusBadGateway, "proxy request failed")
			_ = tlsConn.SetWriteDeadline(time.Time{})
			drainBody(mitmReq.Body)
			continue
		}

		drainBody(mitmReq.Body)
		_ = tlsConn.SetWriteDeadline(time.Now().Add(defaultProxyConnWriteTimeout))
		writeErr := resp.Write(tlsConn)
		_ = tlsConn.SetWriteDeadline(time.Time{})
		_ = resp.Body.Close()
		if writeErr != nil {
			return
		}

		if mitmReq.Close {
			return
		}
	}
}

func (p *ProxyServer) classify(method, host, reqPath string) *classifier.ClassificationResult {
	if p.classifier == nil {
		return &classifier.ClassificationResult{Matched: false, Confidence: "none"}
	}
	return p.classifier.Classify(method, host, reqPath)
}

func (p *ProxyServer) executeClassifiedRequest(ctx context.Context, req *http.Request, requestID string, classification *classifier.ClassificationResult, host, reqPath string) actions.ExecutionResult {
	if p.runtime == nil {
		return actions.ExecutionResult{
			RequestID:  requestID,
			Status:     actions.StatusError,
			HTTPStatus: http.StatusNotImplemented,
			Error: actions.NewExecutionError(
				actions.ErrDownstreamUnavailable,
				"runtime pipeline unavailable",
				http.StatusNotImplemented,
				false,
				nil,
			),
		}
	}

	input, err := extractProxyInput(req)
	if err != nil {
		return actions.ExecutionResult{
			RequestID:  requestID,
			Status:     actions.StatusError,
			HTTPStatus: http.StatusBadRequest,
			Error:      actions.NewExecutionError(actions.ErrValidationFailed, err.Error(), http.StatusBadRequest, false, nil),
		}
	}

	actionName := buildActionName(classification)

	idempotencyKey := strings.TrimSpace(req.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		idempotencyKey = requestID
	}
	result := p.runtime.Execute(ctx, actions.ExecutionRequest{
		RequestID:      requestID,
		IdempotencyKey: idempotencyKey,
		TraceID:        strings.TrimSpace(req.Header.Get("X-Kimbap-Trace-ID")),
		TenantID:       p.proxyTenantID(),
		Principal: actions.Principal{
			ID:        p.proxyPrincipalID(),
			TenantID:  p.proxyTenantID(),
			AgentName: p.proxyAgentName(),
			Type:      "service",
		},
		Action: actions.ActionDefinition{Name: actionName},
		Input:  input,
		Mode:   actions.ModeProxy,
		Classification: &actions.ClassificationInfo{
			Service:       classification.Service,
			ActionName:    actionName,
			Method:        req.Method,
			Path:          reqPath,
			Host:          host,
			MatchedRuleID: classification.RuleID,
			Confidence:    confidenceToFloat(classification.Confidence),
		},
	})
	return result
}

func extractProxyInput(req *http.Request) (map[string]any, error) {
	out := map[string]any{}
	for key, values := range req.URL.Query() {
		switch len(values) {
		case 0:
		case 1:
			out[key] = values[0]
		default:
			items := make([]any, 0, len(values))
			for _, value := range values {
				items = append(items, value)
			}
			out[key] = items
		}
	}

	if req.Body == nil {
		return out, nil
	}
	raw, err := io.ReadAll(io.LimitReader(req.Body, maxProxyRequestBodyBytes+1))
	if err != nil {
		return nil, err
	}
	_ = req.Body.Close()
	if len(raw) > maxProxyRequestBodyBytes {
		return nil, fmt.Errorf("proxy request body exceeded %d bytes", maxProxyRequestBodyBytes)
	}

	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return out, nil
	}

	var body any
	if err := json.Unmarshal(trimmed, &body); err != nil {
		out["raw_body"] = string(trimmed)
		return out, nil
	}

	switch typed := body.(type) {
	case map[string]any:
		for key, value := range typed {
			out[key] = value
		}
	default:
		out["body"] = typed
	}
	return out, nil
}

func buildActionName(classification *classifier.ClassificationResult) string {
	if classification == nil {
		return ""
	}
	action := strings.TrimSpace(classification.Action)
	service := strings.TrimSpace(classification.Service)
	if strings.Contains(action, ".") {
		return action
	}
	if service == "" {
		return action
	}
	if action == "" {
		return service
	}
	return service + "." + action
}

func confidenceToFloat(confidence string) float64 {
	switch strings.ToLower(strings.TrimSpace(confidence)) {
	case "exact":
		return 1.0
	case "pattern":
		return 0.8
	default:
		return 0.0
	}
}

func runtimeResultStatus(result actions.ExecutionResult) int {
	if result.HTTPStatus > 0 {
		return result.HTTPStatus
	}
	if result.Error != nil {
		return http.StatusInternalServerError
	}
	return http.StatusOK
}

func writeRuntimeHTTPResponse(w http.ResponseWriter, result actions.ExecutionResult) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(runtimeResultStatus(result))
	_ = json.NewEncoder(w).Encode(result)
}

func writeRuntimeConnResponse(w io.Writer, result actions.ExecutionResult) error {
	body, err := json.Marshal(result)
	if err != nil {
		return err
	}
	statusCode := runtimeResultStatus(result)
	resp := &http.Response{
		StatusCode: statusCode,
		Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
	resp.Header.Set("Content-Type", "application/json")
	return resp.Write(w)
}

func (p *ProxyServer) forwardRequest(in *http.Request) (*http.Response, error) {
	outReq := in.Clone(in.Context())
	outReq.RequestURI = ""
	clearHopHeaders(outReq.Header)
	outReq.Header.Del("X-Kimbap-Agent-Token")

	resp, err := proxyForwardTransport.RoundTrip(outReq)
	if err != nil {
		return nil, err
	}
	clearHopHeaders(resp.Header)
	return resp, nil
}

func targetFromRequest(req *http.Request) (string, string) {
	host := req.URL.Host
	if host == "" {
		host = req.Host
	}
	if host == "" {
		host = req.URL.Hostname()
	}
	reqPath := req.URL.Path
	if reqPath == "" {
		reqPath = "/"
	}
	return strings.ToLower(host), reqPath
}

func splitHostPort(hostport string) (string, string) {
	h, p, err := net.SplitHostPort(hostport)
	if err == nil {
		return strings.ToLower(strings.Trim(h, "[]")), p
	}
	return strings.ToLower(strings.Trim(hostport, "[]")), "443"
}

func newRequestID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "kimbap-request"
	}
	return hex.EncodeToString(b)
}

func copyHeader(dst, src http.Header) {
	for k, values := range src {
		for _, v := range values {
			dst.Add(k, v)
		}
	}
}

func clearHopHeaders(h http.Header) {
	for _, connVal := range h["Connection"] {
		for _, token := range strings.Split(connVal, ",") {
			h.Del(strings.TrimSpace(token))
		}
	}
	for _, key := range []string{
		"Connection",
		"Proxy-Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",
		"Trailer",
		"Transfer-Encoding",
		"Upgrade",
	} {
		h.Del(key)
	}
}

func (p *ProxyServer) validateProxyAuth(req *http.Request) bool {
	if p.agentToken == "" {
		return true
	}

	if customToken := strings.TrimSpace(req.Header.Get("X-Kimbap-Agent-Token")); customToken != "" {
		if subtle.ConstantTimeCompare([]byte(customToken), []byte(p.agentToken)) == 1 {
			req.Header.Del("X-Kimbap-Agent-Token")
			return true
		}
		return false
	}

	proxyAuth := strings.TrimSpace(req.Header.Get("Proxy-Authorization"))
	if proxyAuth == "" {
		return false
	}

	if subtle.ConstantTimeCompare([]byte(proxyAuth), []byte(p.agentToken)) == 1 {
		return true
	}

	if raw, ok := strings.CutPrefix(proxyAuth, "Basic "); ok {
		raw = strings.TrimSpace(raw)
		decoded, err := base64.StdEncoding.DecodeString(raw)
		if err == nil {
			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) == 2 {
				return subtle.ConstantTimeCompare([]byte(parts[1]), []byte(p.agentToken)) == 1
			}
		}
	}

	return false
}

func (p *ProxyServer) proxyTenantID() string {
	tenantID := strings.TrimSpace(p.TenantID)
	if tenantID == "" {
		return "default"
	}
	return tenantID
}

func (p *ProxyServer) proxyPrincipalID() string {
	if p.agentToken != "" {
		sum := sha256.Sum256([]byte(p.agentToken))
		return "agent:" + hex.EncodeToString(sum[:4])
	}
	return "proxy-agent"
}

func (p *ProxyServer) proxyAgentName() string {
	agentName := strings.TrimSpace(p.AgentName)
	if agentName == "" {
		return "kimbap-proxy"
	}
	return agentName
}

func writePlainErrorResponse(w io.Writer, status int, body string) error {
	resp := &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	resp.Header.Set("Content-Type", "text/plain; charset=utf-8")
	return resp.Write(w)
}
