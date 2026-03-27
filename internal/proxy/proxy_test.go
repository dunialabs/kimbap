package proxy

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dunialabs/kimbap/internal/classifier"
)

func TestProxyHTTPForwardsGETRequest(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Upstream-Method", r.Method)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok:" + r.URL.Path))
	}))
	defer target.Close()

	proxyAddr, stop := startTestProxy(t, nil, UnmatchedPolicyAllow)
	defer stop()

	proxyURL, _ := url.Parse("http://" + proxyAddr)
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}

	resp, err := client.Get(target.URL + "/hello")
	if err != nil {
		t.Fatalf("request via proxy failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if string(body) != "ok:/hello" {
		t.Fatalf("unexpected body: %q", string(body))
	}
}

func TestProxyCONNECTMITMFlow(t *testing.T) {
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("secure:" + r.URL.Path))
	}))
	defer target.Close()

	originalTransport := proxyForwardTransport
	proxyForwardTransport = cloneProxyTransportWithTargetRootCAs(t, target)
	defer func() {
		proxyForwardTransport = originalTransport
	}()

	proxyAddr, stop, ca := startTestProxyWithCA(t, nil, UnmatchedPolicyAllow)
	defer stop()

	pool := x509.NewCertPool()
	if ok := pool.AppendCertsFromPEM(ca.CertPEM); !ok {
		t.Fatal("failed to add proxy CA cert to pool")
	}

	proxyURL, _ := url.Parse("http://" + proxyAddr)
	client := &http.Client{
		Transport: &http.Transport{
			Proxy:           http.ProxyURL(proxyURL),
			TLSClientConfig: &tls.Config{RootCAs: pool},
		},
	}

	resp, err := client.Get(target.URL + "/secure")
	if err != nil {
		t.Fatalf("https via mitm proxy failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if string(body) != "secure:/secure" {
		t.Fatalf("unexpected body: %q", string(body))
	}
}

func cloneProxyTransportWithTargetRootCAs(t *testing.T, target *httptest.Server) *http.Transport {
	t.Helper()
	cloned := proxyForwardTransport.Clone()
	targetTransport, ok := target.Client().Transport.(*http.Transport)
	if !ok || targetTransport.TLSClientConfig == nil || targetTransport.TLSClientConfig.RootCAs == nil {
		t.Fatal("expected target TLS transport with root CAs")
	}
	if cloned.TLSClientConfig == nil {
		cloned.TLSClientConfig = &tls.Config{}
	}
	cloned.TLSClientConfig = cloned.TLSClientConfig.Clone()
	cloned.TLSClientConfig.RootCAs = targetTransport.TLSClientConfig.RootCAs
	return cloned
}

func TestProxyClassificationMatchWithoutRuntime(t *testing.T) {
	targetHit := atomic.Int32{}
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetHit.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	targetURL, _ := url.Parse(target.URL)
	c := classifier.NewClassifier()
	if err := c.AddRule(classifier.Rule{
		ID:          "installed-skill",
		Service:     "svc",
		Action:      "act",
		HostPattern: targetURL.Hostname(),
		PathPattern: "/skill",
		Method:      "GET",
		Priority:    100,
	}); err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	proxyAddr, stop := startTestProxy(t, c, UnmatchedPolicyAllow)
	defer stop()

	proxyURL, _ := url.Parse("http://" + proxyAddr)
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}

	resp, err := client.Get(target.URL + "/skill")
	if err != nil {
		t.Fatalf("request via proxy failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 (runtime unavailable) for matched request without runtime, got %d", resp.StatusCode)
	}
	if targetHit.Load() != 0 {
		t.Fatalf("expected upstream not hit for classified request, got %d", targetHit.Load())
	}
}

func TestProxyUnmatchedRequestPolicyDeny(t *testing.T) {
	targetHit := atomic.Int32{}
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetHit.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	c := classifier.NewClassifier()
	if err := c.AddRule(classifier.Rule{
		ID:          "only-path",
		Service:     "svc",
		Action:      "act",
		HostPattern: "example.com",
		PathPattern: "/only",
		Method:      "GET",
		Priority:    100,
	}); err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	proxyAddr, stop := startTestProxy(t, c, UnmatchedPolicyDeny)
	defer stop()

	proxyURL, _ := url.Parse("http://" + proxyAddr)
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}

	resp, err := client.Get(target.URL + "/different")
	if err != nil {
		t.Fatalf("request via proxy failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for unmatched request, got %d", resp.StatusCode)
	}
	if targetHit.Load() != 0 {
		t.Fatalf("expected unmatched request not forwarded, target hits=%d", targetHit.Load())
	}
}

func TestProxyAddsRequestIDHeader(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Kimbap-Request-ID")
		if id == "" {
			t.Fatal("missing X-Kimbap-Request-ID")
		}
		if len(strings.TrimSpace(id)) < 8 {
			t.Fatalf("request id too short: %q", id)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	proxyAddr, stop := startTestProxy(t, nil, UnmatchedPolicyAllow)
	defer stop()

	proxyURL, _ := url.Parse("http://" + proxyAddr)
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}

	resp, err := client.Get(target.URL + "/id")
	if err != nil {
		t.Fatalf("request via proxy failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestNewProxyHTTPServerTimeoutDefaults(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := newProxyHTTPServer(h)

	if srv.ReadHeaderTimeout != defaultProxyReadHeaderTimeout {
		t.Fatalf("expected ReadHeaderTimeout %s, got %s", defaultProxyReadHeaderTimeout, srv.ReadHeaderTimeout)
	}
	if srv.ReadTimeout != defaultProxyReadTimeout {
		t.Fatalf("expected ReadTimeout %s, got %s", defaultProxyReadTimeout, srv.ReadTimeout)
	}
	if srv.WriteTimeout != defaultProxyWriteTimeout {
		t.Fatalf("expected WriteTimeout %s, got %s", defaultProxyWriteTimeout, srv.WriteTimeout)
	}
	if srv.IdleTimeout != defaultProxyIdleTimeout {
		t.Fatalf("expected IdleTimeout %s, got %s", defaultProxyIdleTimeout, srv.IdleTimeout)
	}
}

func TestExtractProxyInputRejectsOversizedBody(t *testing.T) {
	body := bytes.Repeat([]byte("x"), maxProxyRequestBodyBytes+1)
	req := httptest.NewRequest(http.MethodPost, "http://example.com/path", bytes.NewReader(body))

	_, err := extractProxyInput(req)
	if err == nil {
		t.Fatal("expected oversized body error")
	}
	if !strings.Contains(err.Error(), "exceeded") {
		t.Fatalf("expected exceeded size error, got %v", err)
	}
}

func startTestProxy(t *testing.T, c *classifier.Classifier, policy UnmatchedPolicy) (string, func()) {
	t.Helper()
	addr, stop, _ := startTestProxyWithCA(t, c, policy)
	return addr, stop
}

func startTestProxyWithCA(t *testing.T, c *classifier.Classifier, policy UnmatchedPolicy) (string, func(), *CAConfig) {
	t.Helper()

	ca, err := GenerateCA(t.TempDir())
	if err != nil {
		t.Fatalf("generate ca: %v", err)
	}

	proxy := NewProxyServer("127.0.0.1:0", ca, WithClassifier(c), WithUnmatchedPolicy(policy))
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- proxy.Start(ctx)
	}()

	addr := waitForProxyAddr(t, proxy)
	stop := func() {
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer shutdownCancel()
		_ = proxy.Stop(shutdownCtx)
		select {
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for proxy stop")
		case err := <-errCh:
			if err != nil {
				t.Fatalf("proxy stopped with error: %v", err)
			}
		}
	}

	return addr, stop, ca
}

func waitForProxyAddr(t *testing.T, p *ProxyServer) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		addr := p.Addr()
		if addr != "" {
			if _, port, err := net.SplitHostPort(addr); err == nil && port != "0" {
				return addr
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("proxy address not available")
	return ""
}
