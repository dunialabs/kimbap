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
	safeDialer := &net.Dialer{Timeout: 10 * time.Second}
	safeTransport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, splitErr := net.SplitHostPort(addr)
			if splitErr != nil {
				return nil, fmt.Errorf("invalid dial address %q: %w", addr, splitErr)
			}
			addrs, resolveErr := net.DefaultResolver.LookupIPAddr(ctx, host)
			if resolveErr != nil {
				return nil, fmt.Errorf("resolve %q: %w", host, resolveErr)
			}
			if len(addrs) == 0 {
				return nil, fmt.Errorf("no address resolved for %q", host)
			}
			for _, a := range addrs {
				ip := a.IP
				if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
					if !opts.allowLoopbackHTTP || !ip.IsLoopback() {
						return nil, fmt.Errorf("resolved host %q to private/loopback address %s: SSRF protection rejected", host, ip)
					}
				}
			}
			return safeDialer.DialContext(ctx, network, net.JoinHostPort(addrs[0].IP.String(), port))
		},
	}
	return &http.Client{
		Timeout:   defaultRemoteFetchTimeout,
		Transport: safeTransport,
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
			if scheme == "http" && opts.allowLoopbackHTTP && !isLoopbackHost(r.URL.Hostname()) {
				return fmt.Errorf("redirect to non-https URL %q rejected", r.URL)
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

