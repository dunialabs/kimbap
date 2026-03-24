package controller

import (
	"net/http/httptest"
	"testing"
)

func TestPublicURLForwardedHeaderTrust(t *testing.T) {
	t.Run("uses canonical public URL when configured", func(t *testing.T) {
		t.Setenv("KIMBAP_PUBLIC_BASE_URL", "https://public.example.com/")
		req := httptest.NewRequest("GET", "http://service.local/.well-known/oauth-protected-resource", nil)
		req.RemoteAddr = "198.51.100.10:44321"
		req.Host = "service.local"
		req.Header.Set("X-Forwarded-Host", "evil.example")
		req.Header.Set("X-Forwarded-Proto", "http")

		got := publicURL(req)
		if got != "https://public.example.com" {
			t.Fatalf("expected canonical public url to win, got %s", got)
		}
	})

	t.Run("ignores canonical public URL with userinfo", func(t *testing.T) {
		t.Setenv("KIMBAP_PUBLIC_BASE_URL", "https://user:pass@public.example.com")
		req := httptest.NewRequest("GET", "http://service.local/.well-known/oauth-protected-resource", nil)
		req.RemoteAddr = "198.51.100.10:44321"
		req.Host = "service.local"

		got := publicURL(req)
		if got != "" {
			t.Fatalf("expected invalid canonical URL to be ignored, got %s", got)
		}
	})

	t.Run("ignores forwarded headers from non-loopback remote", func(t *testing.T) {
		t.Setenv("KIMBAP_PUBLIC_BASE_URL", "")
		req := httptest.NewRequest("GET", "http://service.local/.well-known/oauth-protected-resource", nil)
		req.RemoteAddr = "198.51.100.10:44321"
		req.Host = "service.local"
		req.Header.Set("X-Forwarded-Host", "evil.example")
		req.Header.Set("X-Forwarded-Proto", "https")

		got := publicURL(req)
		if got != "" {
			t.Fatalf("expected non-loopback to ignore forwarded headers, got %s", got)
		}
	})

	t.Run("uses forwarded headers from loopback remote", func(t *testing.T) {
		t.Setenv("KIMBAP_PUBLIC_BASE_URL", "")
		req := httptest.NewRequest("GET", "http://service.local/.well-known/oauth-protected-resource", nil)
		req.RemoteAddr = "127.0.0.1:50000"
		req.Host = "service.local"
		req.Header.Set("X-Forwarded-Host", "api.example.com")
		req.Header.Set("X-Forwarded-Proto", "https")

		got := publicURL(req)
		if got != "" {
			t.Fatalf("expected loopback to honor forwarded headers, got %s", got)
		}
	})
}
