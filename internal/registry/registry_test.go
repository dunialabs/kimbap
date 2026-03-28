package registry

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dunialabs/kimbap/skills"
)

const validHTTPManifestYAML = `name: test-service
version: 1.0.0
description: test service
base_url: https://api.example.com
auth:
  type: header
  header_name: Authorization
  credential_ref: test.token
actions:
  ping:
    method: GET
    path: /ping
    description: ping
    risk:
      level: low
`

func TestErrNotFoundErrorMessage(t *testing.T) {
	errWithRegistry := (&ErrNotFound{Name: "github", Registry: "official"}).Error()
	if errWithRegistry != `service "github" not found in registry "official"` {
		t.Fatalf("unexpected error message: %q", errWithRegistry)
	}

	errWithoutRegistry := (&ErrNotFound{Name: "github"}).Error()
	if errWithoutRegistry != `service "github" not found` {
		t.Fatalf("unexpected error message: %q", errWithoutRegistry)
	}
}

func TestParseGitHubRefParsesOwnerRepoServiceAndSubdir(t *testing.T) {
	owner, repo, serviceName, subdir, err := ParseGitHubRef("github:acme/tools/services/github")
	if err != nil {
		t.Fatalf("ParseGitHubRef() error = %v", err)
	}
	if owner != "acme" || repo != "tools" || serviceName != "github" || subdir != "services" {
		t.Fatalf("unexpected parse result: owner=%q repo=%q service=%q subdir=%q", owner, repo, serviceName, subdir)
	}

	owner, repo, serviceName, subdir, err = ParseGitHubRef("github:acme/tools/services/github.yaml")
	if err != nil {
		t.Fatalf("ParseGitHubRef() with .yaml error = %v", err)
	}
	if serviceName != "github" {
		t.Fatalf("serviceName = %q, want github", serviceName)
	}
	if subdir != "services" {
		t.Fatalf("subdir = %q, want services", subdir)
	}

	if _, _, _, _, err := ParseGitHubRef("github:bad/ref"); err == nil {
		t.Fatal("expected invalid ref to fail")
	}
}

func TestGitHubRegistryResolveAndName(t *testing.T) {
	githubYAML := strings.Replace(validHTTPManifestYAML, "name: test-service", "name: github", 1)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/acme/tools/main/services/github.yaml" {
			t.Fatalf("request path = %q", got)
		}
		_, _ = w.Write([]byte(githubYAML))
	}))
	defer srv.Close()

	r := NewGitHubRegistry("acme", "tools", "main", "services")
	r.client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				if strings.HasPrefix(addr, "raw.githubusercontent.com:") {
					return (&net.Dialer{}).DialContext(ctx, network, srv.Listener.Addr().String())
				}
				return (&net.Dialer{}).DialContext(ctx, network, addr)
			},
		},
	}

	manifest, source, err := r.Resolve(context.Background(), "github")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if manifest.Name != "github" {
		t.Fatalf("manifest name = %q, want github", manifest.Name)
	}
	if source != "github:acme/tools/services:github" {
		t.Fatalf("source = %q", source)
	}
	if r.Name() != "github:acme/tools" {
		t.Fatalf("Name() = %q", r.Name())
	}
}

func TestFetchManifestFromURLRejectsInsecureURLAndHandlesStatus(t *testing.T) {
	if _, err := fetchManifestFromURL(context.Background(), http.DefaultClient, "http://insecure.example.com/test.yaml"); err == nil {
		t.Fatal("expected insecure URL to be rejected")
	}

	notFoundSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer notFoundSrv.Close()

	_, err := fetchManifestFromURL(context.Background(), notFoundSrv.Client(), notFoundSrv.URL+"/missing.yaml")
	var nf *ErrNotFound
	if !errors.As(err, &nf) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	badSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer badSrv.Close()

	if _, err := fetchManifestFromURL(context.Background(), badSrv.Client(), badSrv.URL+"/service.yaml"); err == nil {
		t.Fatal("expected non-200 status to fail")
	}
}

func TestFetchManifestFromURLSuccessAndSizeLimit(t *testing.T) {
	successSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(validHTTPManifestYAML))
	}))
	defer successSrv.Close()

	manifest, err := fetchManifestFromURL(context.Background(), successSrv.Client(), successSrv.URL+"/service.yaml")
	if err != nil {
		t.Fatalf("fetchManifestFromURL() error = %v", err)
	}
	if manifest.Name != "test-service" {
		t.Fatalf("manifest name = %q, want test-service", manifest.Name)
	}

	oversizedSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, strings.Repeat("a", (1<<20)+2))
	}))
	defer oversizedSrv.Close()

	if _, err := fetchManifestFromURL(context.Background(), oversizedSrv.Client(), oversizedSrv.URL+"/big.yaml"); err == nil {
		t.Fatal("expected oversized manifest to fail")
	}
}

func TestEmbeddedRegistryResolveAndList(t *testing.T) {
	registry := NewEmbeddedRegistry()
	names, err := registry.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(names) == 0 {
		t.Fatal("expected embedded registry to list official services")
	}

	known := names[0]
	manifest, source, err := registry.Resolve(context.Background(), known)
	if err != nil {
		t.Fatalf("Resolve(%q) error = %v", known, err)
	}
	if manifest.Name != known {
		t.Fatalf("manifest name = %q, want %q", manifest.Name, known)
	}
	if source != "official:"+known {
		t.Fatalf("source = %q, want official:%s", source, known)
	}

	_, _, err = registry.Resolve(context.Background(), "definitely-not-a-real-service")
	var notFound *ErrNotFound
	if !errors.As(err, &notFound) {
		t.Fatalf("expected ErrNotFound from embedded registry, got %v", err)
	}

	if _, err := skills.Get(known); err != nil {
		t.Fatalf("skills.Get(%q) should succeed for listed embedded service: %v", known, err)
	}
}
