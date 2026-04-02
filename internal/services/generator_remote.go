package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"gopkg.in/yaml.v3"
)

const maxOpenAPISpecBytes int64 = 10 << 20

func GenerateFromOpenAPIURL(ctx context.Context, rawURL string) (*ServiceManifest, error) {
	return GenerateFromOpenAPIURLWithOptions(ctx, rawURL, OpenAPIGenerateOptions{})
}

func GenerateFromOpenAPIURLWithOptions(ctx context.Context, rawURL string, opts OpenAPIGenerateOptions) (*ServiceManifest, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	parsed, err := parseOpenAPISpecURL(rawURL)
	if err != nil {
		return nil, err
	}
	body, fetchedURL, err := fetchOpenAPISpec(ctx, parsed, rawURL)
	if err != nil {
		return nil, err
	}
	body = resolveServerURL(body, fetchedURL.String())
	return GenerateFromOpenAPIWithOptions(body, opts)
}

func parseOpenAPISpecURL(rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("invalid OpenAPI URL %q: %w", rawURL, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme %q for OpenAPI spec", parsed.Scheme)
	}
	if strings.EqualFold(strings.TrimSpace(parsed.Scheme), "http") && !isLoopbackHost(parsed.Hostname()) {
		return nil, fmt.Errorf("insecure URL %q rejected: use https:// for OpenAPI sources", rawURL)
	}
	return parsed, nil
}

func fetchOpenAPISpec(ctx context.Context, parsed *url.URL, rawURL string) ([]byte, *url.URL, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("build request for %q: %w", rawURL, err)
	}

	initialScheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	client := &http.Client{
		Timeout: defaultRemoteFetchTimeout,
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			scheme := strings.ToLower(strings.TrimSpace(r.URL.Scheme))
			if scheme != "http" && scheme != "https" {
				return fmt.Errorf("redirect to unsupported URL scheme %q rejected", r.URL.Scheme)
			}
			if initialScheme == "https" && scheme == "http" {
				return fmt.Errorf("redirect to insecure URL %q rejected: use https:// for OpenAPI sources", r.URL)
			}
			if scheme == "http" && !isLoopbackHost(r.URL.Hostname()) {
				return fmt.Errorf("redirect to insecure URL %q rejected: use https:// for OpenAPI sources", r.URL)
			}
			return nil
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch OpenAPI spec from %q: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, nil, fmt.Errorf("fetch OpenAPI spec from %q: got HTTP %d", rawURL, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxOpenAPISpecBytes+1))
	if err != nil {
		return nil, nil, fmt.Errorf("read OpenAPI spec from %q: %w", rawURL, err)
	}
	if int64(len(body)) > maxOpenAPISpecBytes {
		return nil, nil, fmt.Errorf("OpenAPI spec from %q exceeds %d bytes", rawURL, maxOpenAPISpecBytes)
	}

	return body, resp.Request.URL, nil
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

func resolveServerURL(specBytes []byte, fetchedURL string) []byte {
	root, err := parseOpenAPIRoot(specBytes)
	if err != nil {
		return specBytes
	}

	servers := anySliceAt(root, "servers")
	if len(servers) == 0 {
		return specBytes
	}

	firstServer, ok := servers[0].(map[string]any)
	if !ok {
		return specBytes
	}

	rawServerURL := strings.TrimSpace(stringAt(firstServer, "url"))
	if rawServerURL == "" {
		return specBytes
	}

	parsedServerURL, err := url.Parse(rawServerURL)
	if err != nil || parsedServerURL.IsAbs() {
		return specBytes
	}

	parsedFetchedURL, err := url.Parse(strings.TrimSpace(fetchedURL))
	if err != nil || parsedFetchedURL.Scheme == "" || parsedFetchedURL.Host == "" {
		return specBytes
	}

	resolvedServerURL := parsedFetchedURL.ResolveReference(parsedServerURL)
	if resolvedServerURL.Scheme == "" || resolvedServerURL.Host == "" {
		return specBytes
	}

	firstServer["url"] = resolvedServerURL.String()

	var out []byte
	if json.Valid(specBytes) {
		out, err = json.Marshal(root)
	} else {
		out, err = yaml.Marshal(root)
	}
	if err != nil {
		return specBytes
	}

	return out
}
