package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResolveServiceInstallSourceRejectsUppercaseHTTPURL(t *testing.T) {
	_, _, err := resolveServiceInstallSource(context.Background(), "HTTP://example.com/service.yaml")
	if err == nil {
		t.Fatal("expected insecure URL rejection")
	}
	if !strings.Contains(err.Error(), "insecure URL") {
		t.Fatalf("expected insecure URL error, got %v", err)
	}
}

func TestResolveRegistryServiceByNameRespectsCanceledContext(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(2 * time.Second):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("name: zzzzz-test-service\nversion: 1.0.0\nactions: {}\n"))
		}
	}))
	defer server.Close()

	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfigWithRegistryURL(t, dataDir, servicesDir, server.URL)

	withServiceCLIOpts(t, configPath, func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, _, err := resolveRegistryServiceByName(ctx, "zzzzz-test-service")
		if err == nil {
			t.Fatal("expected cancellation error")
		}
		msg := strings.ToLower(err.Error())
		if !strings.Contains(msg, "context canceled") && !strings.Contains(msg, "deadline exceeded") {
			t.Fatalf("expected context cancellation-related error, got %v", err)
		}
	})
}
