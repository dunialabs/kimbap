package main

import (
	"strings"
	"testing"
)

func TestServiceGenerateRejectsUppercaseHTTPSource(t *testing.T) {
	cmd := newServiceGenerateCommand()
	cmd.SetArgs([]string{"--openapi", "HTTP://example.com/openapi.yaml"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected insecure HTTP source to be rejected")
	}
	if !strings.Contains(err.Error(), "insecure URL") {
		t.Fatalf("expected insecure URL error, got %v", err)
	}
}

func TestIsServiceHTTPURLHandlesUppercaseScheme(t *testing.T) {
	if !isServiceHTTPURL("HTTPS://example.com/openapi.yaml") {
		t.Fatal("expected uppercase HTTPS scheme to be recognized")
	}
	if !isServiceHTTPURL("HTTP://example.com/openapi.yaml") {
		t.Fatal("expected uppercase HTTP scheme to be recognized")
	}
}
