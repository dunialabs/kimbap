package registry

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dunialabs/kimbap/internal/services"
)

type GitHubRegistry struct {
	owner  string
	repo   string
	branch string
	subdir string
	client *http.Client
}

func ParseGitHubRef(ref string) (owner, repo, serviceName, subdir string, err error) {
	trimmed := strings.TrimSpace(strings.TrimPrefix(ref, "github:"))
	parts := strings.Split(trimmed, "/")
	if len(parts) < 3 {
		return "", "", "", "", fmt.Errorf("invalid github ref %q: expected github:owner/repo/service-name", ref)
	}

	owner = strings.TrimSpace(parts[0])
	repo = strings.TrimSpace(parts[1])
	serviceName = strings.TrimSpace(parts[len(parts)-1])
	if strings.HasSuffix(strings.ToLower(serviceName), ".yaml") {
		serviceName = strings.TrimSuffix(serviceName, ".yaml")
	}

	if len(parts) > 3 {
		subdir = strings.Join(parts[2:len(parts)-1], "/")
	}

	if owner == "" || repo == "" || serviceName == "" {
		return "", "", "", "", fmt.Errorf("invalid github ref %q: owner, repo, and service name are required", ref)
	}

	return owner, repo, serviceName, subdir, nil
}

func NewGitHubRegistry(owner, repo, branch, subdir string) *GitHubRegistry {
	if strings.TrimSpace(branch) == "" {
		branch = "main"
	}
	return &GitHubRegistry{
		owner:  strings.TrimSpace(owner),
		repo:   strings.TrimSpace(repo),
		branch: branch,
		subdir: strings.Trim(strings.TrimSpace(subdir), "/"),
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

func (r *GitHubRegistry) Name() string {
	return fmt.Sprintf("github:%s/%s", r.owner, r.repo)
}

func (r *GitHubRegistry) rawURL(name string) string {
	path := strings.TrimSpace(name) + ".yaml"
	if r.subdir != "" {
		path = r.subdir + "/" + path
	}
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", r.owner, r.repo, r.branch, path)
}

func (r *GitHubRegistry) Resolve(ctx context.Context, name string) (*services.ServiceManifest, string, error) {
	url := r.rawURL(name)
	manifest, err := fetchManifestFromURL(ctx, r.client, url)
	if err != nil {
		return nil, "", err
	}
	source := fmt.Sprintf("github:%s/%s", r.owner, r.repo)
	if r.subdir != "" {
		source += "/" + r.subdir
	}
	return manifest, source + ":" + strings.TrimSpace(name), nil
}

func (r *GitHubRegistry) List(_ context.Context) ([]string, error) {
	return nil, fmt.Errorf("list not supported for GitHub registry")
}
