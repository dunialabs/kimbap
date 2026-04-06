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

var errProxyNonJSONMatchedBody = errors.New("matched proxy requests only support empty or JSON request bodies")

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
	DisableCompression:     true,
	MaxIdleConns:           100,
	MaxIdleConnsPerHost:    10,
}

type ProxyOption func(*ProxyServer)

const maxConcurrentConnects = 64

type ProxyServer struct {
	listenAddr string
	ca         *CAConfig
	classifier *classifier.Classifier
	runtime    *runtime.Runtime
	agentToken string
	TenantID   string
	AgentName  string
	connectSem chan struct{}

	unmatchedPolicy UnmatchedPolicy

	server   *http.Server
	listener net.Listener
	mu       sync.RWMutex
}

// bufferedConn wraps a net.Conn and overrides Read to drain a pre-buffered
// reader before falling back to the connection itself. Used to preserve bytes
// already consumed from the TCP stream by a bufio.Reader before TLS handshake.
type bufferedConn struct {
	net.Conn
	r io.Reader
}

type countingReadCloser struct {
	io.ReadCloser
	n int64
}

func (c *countingReadCloser) Read(p []byte) (int, error) {
	n, err := c.ReadCloser.Read(p)
	c.n += int64(n)
	return n, err
}

func (c *bufferedConn) Read(b []byte) (int, error) {
	return c.r.Read(b)
}

func NewProxyServer(addr string, ca *CAConfig, opts ...ProxyOption) *ProxyServer {
	p := &ProxyServer{
		listenAddr:      addr,
		ca:              ca,
		classifier:      classifier.NewClassifier(),
		unmatchedPolicy: UnmatchedPolicyDeny,
		TenantID:        "default",
		AgentName:       "kimbap-proxy",
		connectSem:      make(chan struct{}, maxConcurrentConnects),
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
	p.mu.RLock()
	alreadyStarted := p.listener != nil || p.server != nil
	p.mu.RUnlock()
	if alreadyStarted {
		return errors.New("proxy server already started")
	}

	ln, err := net.Listen("tcp", p.listenAddr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	srv := newProxyHTTPServer(p)

	p.mu.Lock()
	if p.listener != nil || p.server != nil {
		p.mu.Unlock()
		_ = ln.Close()
		return errors.New("proxy server already started")
	}
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
	p.mu.Lock()
	p.server = nil
	p.listener = nil
	p.mu.Unlock()
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
	if p.connectSem != nil {
		select {
		case p.connectSem <- struct{}{}:
			defer func() { <-p.connectSem }()
		default:
			http.Error(w, "too many concurrent connections", http.StatusServiceUnavailable)
			return
		}
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

	var connForTLS net.Conn = clientConn
	if rw.Reader.Buffered() > 0 {
		connForTLS = &bufferedConn{
			Conn: clientConn,
			r:    io.MultiReader(rw.Reader, clientConn),
		}
	}
	tlsConn := tls.Server(connForTLS, &tls.Config{Certificates: []tls.Certificate{*hostCert}})
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
			return
		}
		mitmReq = mitmReq.WithContext(req.Context())
		shouldClose, err := p.processMITMRequest(req.Context(), tlsConn, mitmReq, connectHost, drainBody)
		if err != nil {
			return
		}
		if shouldClose {
			return
		}
	}
}

func writeTunnelErrorAndDrain(tlsConn *tls.Conn, req *http.Request, status int, message string, drainBody func(io.ReadCloser)) bool {
	_ = tlsConn.SetWriteDeadline(time.Now().Add(defaultProxyConnWriteTimeout))
	_ = writePlainErrorResponse(tlsConn, status, message)
	_ = tlsConn.SetWriteDeadline(time.Time{})
	drainBody(req.Body)
	return req.Close
}

func (p *ProxyServer) processMITMRequest(ctx context.Context, tlsConn *tls.Conn, mitmReq *http.Request, connectHost string, drainBody func(io.ReadCloser)) (bool, error) {
	_ = tlsConn.SetReadDeadline(time.Now().Add(defaultProxyReadTimeout))

	if mitmReq.URL == nil {
		mitmReq.URL = &url.URL{}
	}
	mitmReq.URL.Scheme = "https"
	mitmReq.URL.Host = connectHost
	if mitmReq.Host == "" {
		mitmReq.Host = connectHost
	}

	if strings.EqualFold(mitmReq.Header.Get("Expect"), "100-continue") {
		_, _ = fmt.Fprintf(tlsConn, "HTTP/1.1 100 Continue\r\n\r\n")
	}

	if isWebSocketUpgrade(mitmReq) {
		return writeTunnelErrorAndDrain(tlsConn, mitmReq, http.StatusNotImplemented, "websocket is not supported in proxy mode v1", drainBody), nil
	}

	reqID := newRequestID()
	classification := p.classify(mitmReq.Method, connectHost, mitmReq.URL.Path)
	if classification != nil && classification.Matched {
		mitmReq.Header.Set("X-Kimbap-Request-ID", reqID)
		result := p.executeClassifiedRequest(ctx, mitmReq, reqID, classification, connectHost, mitmReq.URL.Path)
		if result.Error != nil && runtimeResultStatus(result) >= 500 {
			return writeTunnelErrorAndDrain(tlsConn, mitmReq, runtimeResultStatus(result), "proxy request failed", drainBody), nil
		}
		_ = tlsConn.SetWriteDeadline(time.Now().Add(defaultProxyConnWriteTimeout))
		if err := writeRuntimeConnResponse(tlsConn, result); err != nil {
			_ = tlsConn.SetWriteDeadline(time.Time{})
			drainBody(mitmReq.Body)
			return false, err
		}
		_ = tlsConn.SetWriteDeadline(time.Time{})
		drainBody(mitmReq.Body)
		return mitmReq.Close, nil
	} else if p.unmatchedPolicy == UnmatchedPolicyDeny {
		return writeTunnelErrorAndDrain(tlsConn, mitmReq, http.StatusForbidden, "proxy request denied", drainBody), nil
	} else {
		mitmReq.Header.Set("X-Kimbap-Request-ID", reqID)
	}

	resp, ferr := p.forwardRequest(mitmReq)
	if ferr != nil {
		return writeTunnelErrorAndDrain(tlsConn, mitmReq, http.StatusBadGateway, "proxy request failed", drainBody), nil
	}

	drainBody(mitmReq.Body)
	_ = tlsConn.SetWriteDeadline(time.Now().Add(defaultProxyConnWriteTimeout))
	writeErr := resp.Write(tlsConn)
	_ = tlsConn.SetWriteDeadline(time.Time{})
	_ = resp.Body.Close()
	if writeErr != nil {
		return false, writeErr
	}

	return mitmReq.Close, nil
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
		httpStatus := http.StatusBadRequest
		if errors.Is(err, errProxyNonJSONMatchedBody) {
			httpStatus = http.StatusUnsupportedMediaType
		}
		return actions.ExecutionResult{
			RequestID:  requestID,
			Status:     actions.StatusError,
			HTTPStatus: httpStatus,
			Error:      actions.NewExecutionError(actions.ErrValidationFailed, err.Error(), httpStatus, false, nil),
		}
	}

	actionName := buildActionName(classification)

	idempotencyKey := strings.TrimSpace(req.Header.Get("Idempotency-Key"))
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
	if req.ContentLength > maxProxyRequestBodyBytes {
		return nil, fmt.Errorf("proxy request body exceeded %d bytes", maxProxyRequestBodyBytes)
	}
	bodyReader := &countingReadCloser{ReadCloser: req.Body}
	defer func() { _ = req.Body.Close() }()
	limitedBody := io.LimitReader(bodyReader, maxProxyRequestBodyBytes+1)
	decoder := json.NewDecoder(limitedBody)

	var body any
	decodeErr := decoder.Decode(&body)
	if decodeErr == nil {
		var extra any
		if err := decoder.Decode(&extra); err != io.EOF {
			decodeErr = err
		}
	}

	if _, err := io.Copy(io.Discard, limitedBody); err != nil {
		return nil, err
	}

	if bodyReader.n > maxProxyRequestBodyBytes {
		return nil, fmt.Errorf("proxy request body exceeded %d bytes", maxProxyRequestBodyBytes)
	}

	if decodeErr != nil {
		if errors.Is(decodeErr, io.EOF) {
			return out, nil
		}
		return nil, errProxyNonJSONMatchedBody
	}

	if body == nil {
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
	outReq.Host = outReq.URL.Host
	clearHopHeaders(outReq.Header)
	outReq.Header.Del("X-Kimbap-Agent-Token")
	for _, h := range []string{"Forwarded", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto", "X-Real-IP"} {
		outReq.Header.Del(h)
	}

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

	if fields := strings.Fields(proxyAuth); len(fields) == 2 && strings.EqualFold(fields[0], "Basic") {
		decoded, err := base64.StdEncoding.DecodeString(fields[1])
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
