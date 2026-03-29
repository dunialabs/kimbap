package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultRemoteFetchTimeout = 30 * time.Second

func FetchHTTPResource(ctx context.Context, rawURL string, maxBytes int64, purpose string, requireHTTPS bool) ([]byte, *url.URL, error) {
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
	if requireHTTPS && parsed.Scheme != "https" {
		return nil, nil, fmt.Errorf("insecure URL %q rejected: use https:// for %s", rawURL, purpose)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("build request for %q: %w", rawURL, err)
	}

	client := newRemoteFetchClient(parsed.Scheme, requireHTTPS)

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

func newRemoteFetchClient(initialScheme string, requireHTTPS bool) *http.Client {
	return &http.Client{
		Timeout: defaultRemoteFetchTimeout,
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			if r.URL.Scheme != "http" && r.URL.Scheme != "https" {
				return fmt.Errorf("redirect to unsupported URL scheme %q rejected", r.URL.Scheme)
			}
			if requireHTTPS && r.URL.Scheme != "https" {
				return fmt.Errorf("redirect to non-https URL %q rejected", r.URL)
			}
			if !requireHTTPS && strings.EqualFold(initialScheme, "https") && r.URL.Scheme != "https" {
				return fmt.Errorf("redirect to non-https URL %q rejected", r.URL)
			}
			return nil
		},
	}
}
