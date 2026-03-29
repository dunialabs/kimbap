package services

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"

	"gopkg.in/yaml.v3"
)

const maxOpenAPISpecBytes int64 = 10 << 20

func GenerateFromOpenAPIURL(rawURL string) (*ServiceManifest, error) {
	body, fetchedURL, err := FetchHTTPResource(context.Background(), rawURL, maxOpenAPISpecBytes, "OpenAPI spec", false)
	if err != nil {
		return nil, err
	}
	body = resolveServerURL(body, fetchedURL.String())
	return GenerateFromOpenAPI(body)
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
