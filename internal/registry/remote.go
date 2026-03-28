package registry

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dunialabs/kimbap/internal/services"
)

type RemoteRegistry struct {
	name    string
	baseURL string
	client  *http.Client
}

func NewRemoteRegistry(name, baseURL string) *RemoteRegistry {
	return &RemoteRegistry{
		name:    name,
		baseURL: strings.TrimSuffix(strings.TrimSpace(baseURL), "/"),
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if req.URL.Scheme != "https" {
					return fmt.Errorf("redirect to non-https URL %q rejected", req.URL)
				}
				return nil
			},
		},
	}
}

func (r *RemoteRegistry) Name() string { return r.name }

func (r *RemoteRegistry) Resolve(ctx context.Context, name string) (*services.ServiceManifest, string, error) {
	url := r.baseURL + "/" + name + ".yaml"
	manifest, err := fetchManifestFromURL(ctx, r.client, url)
	if err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(manifest.Name) != strings.TrimSpace(name) {
		return nil, "", fmt.Errorf("manifest name %q does not match requested service %q", manifest.Name, name)
	}
	return manifest, "remote:" + url, nil
}

func (r *RemoteRegistry) List(_ context.Context) ([]string, error) {
	return nil, fmt.Errorf("list not supported for remote registry %q", r.name)
}

func fetchManifestFromURL(ctx context.Context, client *http.Client, url string) (*services.ServiceManifest, error) {
	if strings.HasPrefix(strings.ToLower(url), "http://") {
		return nil, fmt.Errorf("insecure URL %q rejected: use https:// to install service manifests", url)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request for %q: %w", url, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch manifest from %q: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, &ErrNotFound{Name: url, Registry: "remote"}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch manifest from %q: HTTP %d", url, resp.StatusCode)
	}

	const maxBytes = 1 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read manifest from %q: %w", url, err)
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("manifest from %q exceeds %d bytes", url, maxBytes)
	}

	manifest, err := services.ParseManifest(body)
	if err != nil {
		return nil, fmt.Errorf("parse manifest from %q: %w", url, err)
	}
	return manifest, nil
}
