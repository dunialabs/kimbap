package services

import (
	"context"
	"encoding/json"
	"fmt"
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
	body, fetchedURL, err := fetchHTTPResource(ctx, parsed.String(), maxOpenAPISpecBytes, "OpenAPI spec", remoteFetchOptions{allowLoopbackHTTP: true})
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
