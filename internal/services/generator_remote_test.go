package services

import (
	"context"
	"strings"
	"testing"
)

func TestGenerateFromOpenAPIURLRejectsUppercaseHTTPSInsecureHTTP(t *testing.T) {
	_, err := GenerateFromOpenAPIURL(context.Background(), "HTTP://example.com/openapi.yaml")
	if err == nil {
		t.Fatal("expected insecure HTTP URL to be rejected")
	}
	if !strings.Contains(err.Error(), "insecure URL") {
		t.Fatalf("expected insecure URL error, got %v", err)
	}
}
