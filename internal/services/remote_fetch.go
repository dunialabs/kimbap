package services

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultRemoteFetchTimeout = 30 * time.Second

type remoteFetchOptions struct {
	requireHTTPS      bool
	allowLoopbackHTTP bool
}

func FetchHTTPResource(ctx context.Context, rawURL string, maxBytes int64, purpose string, requireHTTPS bool) ([]byte, *url.URL, error) {
	return fetchHTTPResource(ctx, rawURL, maxBytes, purpose, remoteFetchOptions{requireHTTPS: requireHTTPS})
}

func fetchHTTPResource(ctx context.Context, rawURL string, maxBytes int64, purpose string, opts remoteFetchOptions) ([]byte, *url.URL, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	trimmed := strings.TrimSpace(rawURL)
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, nil, fmt.Errorf("parse %s URL %q: %w", purpose, rawURL, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, nil, fmt.Errorf("unsupported URL scheme %q for %s", parsed.Scheme, purpose)
	}
	if opts.requireHTTPS && parsed.Scheme != "https" {
		return nil, nil, fmt.Errorf("insecure URL %q rejected: use https:// for %s", rawURL, purpose)
	}
	if parsed.Scheme == "http" && opts.allowLoopbackHTTP && !isLoopbackHost(parsed.Hostname()) {
		return nil, nil, fmt.Errorf("insecure URL %q rejected: use https:// for %s", rawURL, purpose)
	}
	if isPrivateOrLoopbackServices(parsed.Hostname()) && !opts.allowLoopbackHTTP {
		return nil, nil, fmt.Errorf("URL %q targets a private/loopback address and is not allowed for %s", rawURL, purpose)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("build request for %q: %w", rawURL, err)
	}

	client := newRemoteFetchClient(parsed, opts)

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch %s from %q: %w", purpose, rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, nil, fmt.Errorf("fetch %s from %q: got HTTP %d", purpose, rawURL, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, nil, fmt.Errorf("read %s from %q: %w", purpose, rawURL, err)
	}
	if int64(len(body)) > maxBytes {
		return nil, nil, fmt.Errorf("%s from %q exceeds %d bytes", purpose, rawURL, maxBytes)
	}

	return body, resp.Request.URL, nil
}

func newRemoteFetchClient(initialURL *url.URL, opts remoteFetchOptions) *http.Client {
	initialScheme := ""
	if initialURL != nil {
		initialScheme = strings.ToLower(strings.TrimSpace(initialURL.Scheme))
	}
	return &http.Client{
		Timeout: defaultRemoteFetchTimeout,
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			scheme := strings.ToLower(strings.TrimSpace(r.URL.Scheme))
			if scheme != "http" && scheme != "https" {
				return fmt.Errorf("redirect to unsupported URL scheme %q rejected", r.URL.Scheme)
			}
			if opts.requireHTTPS && scheme != "https" {
				return fmt.Errorf("redirect to non-https URL %q rejected", r.URL)
			}
			if !opts.requireHTTPS && initialScheme == "https" && scheme != "https" {
				return fmt.Errorf("redirect to non-https URL %q rejected", r.URL)
			}
			if !opts.requireHTTPS && opts.allowLoopbackHTTP && scheme == "http" && !isLoopbackHost(r.URL.Hostname()) {
				return fmt.Errorf("redirect to non-https URL %q rejected", r.URL)
			}
			if isPrivateOrLoopbackServices(r.URL.Hostname()) && !opts.allowLoopbackHTTP {
				return fmt.Errorf("redirect to private/loopback host %q rejected", r.URL.Hostname())
			}
			return nil
		},
	}
}

func isLoopbackHost(host string) bool {
	host = strings.TrimSpace(host)
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func isPrivateOrLoopbackServices(host string) bool {
	host = strings.TrimSpace(host)
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	normalized := host
	if idx := strings.IndexByte(host, '%'); idx >= 0 {
		normalized = host[:idx]
	}
	ip := net.ParseIP(normalized)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}
