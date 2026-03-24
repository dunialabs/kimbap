package service

import (
	"bytes"
	"container/list"
	"context"
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

	oauthtypes "github.com/dunialabs/kimbap-core/internal/oauth/types"
)

type cacheEntry struct {
	key       string
	value     oauthtypes.OAuthClientMetadata
	fetchedAt time.Time
}

type ClientMetadataFetcher struct {
	client   *http.Client
	cacheTTL time.Duration
	maxSize  int
	mu       sync.Mutex
	entries  map[string]*list.Element
	order    *list.List
}

const maxClientMetadataBytes int64 = 1 << 20

func NewClientMetadataFetcher() *ClientMetadataFetcher {
	transport := &http.Transport{Proxy: nil, DialContext: metadataDialContext}
	if defaultTransport, ok := http.DefaultTransport.(*http.Transport); ok {
		cloned := defaultTransport.Clone()
		cloned.Proxy = nil
		cloned.DialContext = metadataDialContext
		cloned.DialTLS = nil
		cloned.DialTLSContext = nil
		transport = cloned
	}

	return &ClientMetadataFetcher{
		client: &http.Client{
			Transport: transport,
			Timeout:   5 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		cacheTTL: time.Hour,
		maxSize:  256,
		entries:  make(map[string]*list.Element),
		order:    list.New(),
	}
}

func (f *ClientMetadataFetcher) FetchClientMetadata(rawURL string) (*oauthtypes.OAuthClientMetadata, error) {
	if err := f.validateURL(rawURL); err != nil {
		return nil, err
	}

	if cached := f.getCached(rawURL); cached != nil {
		return cached, nil
	}

	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid_client_metadata: failed to build request")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "kimbap-core/1.0")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("invalid_client_metadata: failed to fetch metadata")
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return nil, fmt.Errorf("invalid_client_metadata: client metadata URL must not redirect")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("invalid_client_metadata: failed to fetch client metadata: HTTP %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(strings.ToLower(contentType), "application/json") {
		return nil, fmt.Errorf("invalid_client_metadata: client metadata must be JSON (application/json)")
	}
	if resp.ContentLength > maxClientMetadataBytes {
		return nil, fmt.Errorf("invalid_client_metadata: metadata too large")
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxClientMetadataBytes+1))
	if err != nil {
		return nil, fmt.Errorf("invalid_client_metadata: failed to read metadata body")
	}
	if int64(len(body)) > maxClientMetadataBytes {
		return nil, fmt.Errorf("invalid_client_metadata: metadata too large")
	}

	var metadata oauthtypes.OAuthClientMetadata
	dec := json.NewDecoder(bytes.NewReader(body))
	if err := dec.Decode(&metadata); err != nil {
		return nil, fmt.Errorf("invalid_client_metadata: invalid JSON body")
	}
	var trailing json.RawMessage
	if err := dec.Decode(&trailing); err == nil {
		return nil, fmt.Errorf("invalid_client_metadata: invalid JSON body")
	} else if !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("invalid_client_metadata: invalid JSON body")
	}

	if err := f.validateMetadata(&metadata); err != nil {
		return nil, err
	}

	f.putCached(rawURL, metadata)
	return &metadata, nil
}

func (f *ClientMetadataFetcher) validateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid_client_metadata: Invalid URL format: %s", err.Error())
	}
	if strings.ToLower(u.Scheme) != "https" {
		return fmt.Errorf("invalid_client_metadata: Client metadata URL must use HTTPS protocol")
	}
	if u.Path == "" || u.Path == "/" {
		return fmt.Errorf("invalid_client_metadata: Client metadata URL pathname cannot be root (\"/\"), must specify a document path")
	}
	if u.Port() != "" && u.Port() != "443" {
		return fmt.Errorf("invalid_client_metadata: Client metadata URL must use default HTTPS port")
	}
	host := strings.TrimSpace(u.Hostname())
	if host == "" {
		return fmt.Errorf("invalid_client_metadata: Client metadata URL must include a valid host")
	}
	if strings.EqualFold(host, "localhost") {
		return fmt.Errorf("invalid_client_metadata: localhost is not allowed")
	}

	if ip := net.ParseIP(host); ip != nil {
		if isBlockedMetadataIP(ip) {
			return fmt.Errorf("invalid_client_metadata: private or local metadata hosts are not allowed")
		}
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	resolved, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil || len(resolved) == 0 {
		return fmt.Errorf("invalid_client_metadata: unable to resolve metadata host")
	}
	for _, addr := range resolved {
		if isBlockedMetadataIP(addr.IP) {
			return fmt.Errorf("invalid_client_metadata: private or local metadata hosts are not allowed")
		}
	}
	return nil
}

func isBlockedMetadataIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	v4 := ip.To4()
	if v4 != nil {
		if v4[0] == 0 || v4[0] == 255 {
			return true
		}
		if v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127 {
			return true
		}
	}
	v6 := ip.To16()
	if v6 != nil && v4 == nil {
		if v6[0]&0xfe == 0xfc {
			return true
		}
		if v6[0] == 0xfe && (v6[1]&0xc0) == 0x80 {
			return true
		}
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalMulticast() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() || ip.IsMulticast()
}

func metadataDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		host = address
		port = "443"
	}
	if strings.TrimSpace(host) == "" {
		return nil, fmt.Errorf("invalid_client_metadata: invalid dial host")
	}

	if ip := net.ParseIP(host); ip != nil {
		if isBlockedMetadataIP(ip) {
			return nil, fmt.Errorf("invalid_client_metadata: private or local metadata hosts are not allowed")
		}
		dialer := &net.Dialer{}
		return dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
	}

	resolved, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil || len(resolved) == 0 {
		return nil, fmt.Errorf("invalid_client_metadata: unable to resolve metadata host")
	}

	dialer := &net.Dialer{}
	var lastErr error
	for _, addr := range resolved {
		if isBlockedMetadataIP(addr.IP) {
			continue
		}
		conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(addr.IP.String(), port))
		if dialErr == nil {
			return conn, nil
		}
		lastErr = dialErr
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("invalid_client_metadata: private or local metadata hosts are not allowed")
}

func (f *ClientMetadataFetcher) validateMetadata(metadata *oauthtypes.OAuthClientMetadata) error {
	if metadata == nil {
		return fmt.Errorf("invalid_client_metadata: metadata is required")
	}
	if len(metadata.RedirectURIs) == 0 {
		return fmt.Errorf("invalid_client_metadata: redirect_uris is required and must be a non-empty array")
	}
	for _, uri := range metadata.RedirectURIs {
		if strings.TrimSpace(uri) == "" {
			return fmt.Errorf("invalid_client_metadata: All redirect_uris must be non-empty strings")
		}
		parsed, err := url.Parse(uri)
		if err != nil || parsed == nil || parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("invalid_redirect_uri: Invalid redirect_uri: %s", uri)
		}
	}

	if len(metadata.GrantTypes) > 0 {
		supported := map[string]bool{"authorization_code": true, "refresh_token": true, "client_credentials": true}
		invalid := make([]string, 0)
		for _, grant := range metadata.GrantTypes {
			if !supported[grant] {
				invalid = append(invalid, grant)
			}
		}
		if len(invalid) > 0 {
			return fmt.Errorf("invalid_client_metadata: unsupported grant_types: %s", strings.Join(invalid, ", "))
		}
	}

	if len(metadata.ResponseTypes) > 0 {
		invalid := make([]string, 0)
		for _, rt := range metadata.ResponseTypes {
			if rt != "code" {
				invalid = append(invalid, rt)
			}
		}
		if len(invalid) > 0 {
			return fmt.Errorf("invalid_client_metadata: unsupported response_types: %s", strings.Join(invalid, ", "))
		}
	}

	if metadata.TokenEndpointAuthMethod != "" {
		switch metadata.TokenEndpointAuthMethod {
		case "client_secret_basic", "client_secret_post", "none":
		default:
			return fmt.Errorf("invalid_client_metadata: unsupported token_endpoint_auth_method: %s", metadata.TokenEndpointAuthMethod)
		}
	}
	if metadata.TokenEndpointAuthMethod == "none" {
		for _, grant := range metadata.GrantTypes {
			if grant == "client_credentials" {
				return fmt.Errorf("invalid_client_metadata: token_endpoint_auth_method 'none' cannot be used with client_credentials grant")
			}
		}
	}
	return nil
}

func (f *ClientMetadataFetcher) getCached(key string) *oauthtypes.OAuthClientMetadata {
	f.mu.Lock()
	defer f.mu.Unlock()
	el, ok := f.entries[key]
	if !ok {
		return nil
	}
	entry, ok := el.Value.(cacheEntry)
	if !ok {
		f.order.Remove(el)
		delete(f.entries, key)
		return nil
	}
	if time.Since(entry.fetchedAt) > f.cacheTTL {
		f.order.Remove(el)
		delete(f.entries, key)
		return nil
	}
	f.order.MoveToFront(el)
	copyVal := entry.value
	return &copyVal
}

func (f *ClientMetadataFetcher) putCached(key string, value oauthtypes.OAuthClientMetadata) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if el, ok := f.entries[key]; ok {
		f.order.MoveToFront(el)
		el.Value = cacheEntry{key: key, value: value, fetchedAt: time.Now()}
		return
	}
	el := f.order.PushFront(cacheEntry{key: key, value: value, fetchedAt: time.Now()})
	f.entries[key] = el
	if len(f.entries) <= f.maxSize {
		return
	}
	back := f.order.Back()
	if back == nil {
		return
	}
	entry, ok := back.Value.(cacheEntry)
	if !ok {
		f.order.Remove(back)
		return
	}
	f.order.Remove(back)
	delete(f.entries, entry.key)
}

func (f *ClientMetadataFetcher) ClearCache(clientMetadataURL string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if strings.TrimSpace(clientMetadataURL) != "" {
		if el, ok := f.entries[clientMetadataURL]; ok {
			f.order.Remove(el)
			delete(f.entries, clientMetadataURL)
		}
		return
	}

	f.entries = make(map[string]*list.Element)
	f.order.Init()
}

func (f *ClientMetadataFetcher) CleanExpiredCache() {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := time.Now()
	for key, el := range f.entries {
		entry, ok := el.Value.(cacheEntry)
		if !ok {
			f.order.Remove(el)
			delete(f.entries, key)
			continue
		}
		if now.Sub(entry.fetchedAt) >= f.cacheTTL {
			f.order.Remove(el)
			delete(f.entries, key)
		}
	}
}

func (f *ClientMetadataFetcher) validateForRegister(metadata *oauthtypes.OAuthClientMetadata) error {
	if metadata == nil {
		return errors.New("invalid_client_metadata: metadata is required")
	}
	return f.validateMetadata(metadata)
}
