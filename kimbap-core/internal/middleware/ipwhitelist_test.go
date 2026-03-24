package middleware

import (
	"net/http/httptest"
	"testing"
)

func TestClientIPFromRequest(t *testing.T) {
	t.Run("extracts from X-Forwarded-For", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.2")
		req.RemoteAddr = "127.0.0.1:1234"

		got := ClientIPFromRequest(req)
		if got != "198.51.100.2" {
			t.Fatalf("expected 198.51.100.2, got %s", got)
		}
	})

	t.Run("extracts from X-Real-IP", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com", nil)
		req.Header.Set("X-Real-IP", "198.51.100.10")
		req.RemoteAddr = "127.0.0.1:1234"

		got := ClientIPFromRequest(req)
		if got != "198.51.100.10" {
			t.Fatalf("expected 198.51.100.10, got %s", got)
		}
	})

	t.Run("extracts from CF-Connecting-IP", func(t *testing.T) {
		t.Setenv("KIMBAP_TRUSTED_PROXY_CIDRS", "")
		t.Setenv("KIMBAP_TRUSTED_CF_CIDRS", "127.0.0.0/8")
		req := httptest.NewRequest("GET", "http://example.com", nil)
		req.Header.Set("CF-Connecting-IP", "198.51.100.11")
		req.RemoteAddr = "127.0.0.1:1234"

		got := ClientIPFromRequest(req)
		if got != "198.51.100.11" {
			t.Fatalf("expected 198.51.100.11, got %s", got)
		}
	})

	t.Run("falls back to RemoteAddr", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com", nil)
		req.RemoteAddr = "192.0.2.42:8080"

		got := ClientIPFromRequest(req)
		if got != "192.0.2.42" {
			t.Fatalf("expected 192.0.2.42, got %s", got)
		}
	})

	t.Run("header priority prefers trusted CF-Connecting-IP over XFF/X-Real-IP", func(t *testing.T) {
		t.Setenv("KIMBAP_TRUSTED_PROXY_CIDRS", "")
		t.Setenv("KIMBAP_TRUSTED_CF_CIDRS", "127.0.0.0/8")
		req := httptest.NewRequest("GET", "http://example.com", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.5")
		req.Header.Set("X-Real-IP", "198.51.100.5")
		req.Header.Set("CF-Connecting-IP", "198.51.100.6")
		req.RemoteAddr = "127.0.0.1:1234"

		got := ClientIPFromRequest(req)
		if got != "198.51.100.6" {
			t.Fatalf("expected priority to select CF-Connecting-IP value, got %s", got)
		}
	})

	t.Run("ignores CF-Connecting-IP when source is not trusted cloudflare", func(t *testing.T) {
		t.Setenv("KIMBAP_TRUSTED_PROXY_CIDRS", "")
		t.Setenv("KIMBAP_TRUSTED_CF_CIDRS", "")
		req := httptest.NewRequest("GET", "http://example.com", nil)
		req.Header.Set("CF-Connecting-IP", "198.51.100.6")
		req.Header.Set("X-Forwarded-For", "203.0.113.5")
		req.RemoteAddr = "127.0.0.1:1234"

		got := ClientIPFromRequest(req)
		if got != "203.0.113.5" {
			t.Fatalf("expected CF-Connecting-IP to be ignored without trusted CF source, got %s", got)
		}
	})

	t.Run("ignores forwarded headers from untrusted remote", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.5")
		req.Header.Set("X-Real-IP", "198.51.100.5")
		req.Header.Set("CF-Connecting-IP", "198.51.100.6")
		req.RemoteAddr = "192.0.2.99:9000"

		got := ClientIPFromRequest(req)
		if got != "192.0.2.99" {
			t.Fatalf("expected untrusted remote addr to be used, got %s", got)
		}
	})

	t.Run("ignores forwarded headers from private non-loopback remote", func(t *testing.T) {
		t.Setenv("KIMBAP_TRUSTED_PROXY_CIDRS", "")
		req := httptest.NewRequest("GET", "http://example.com", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.5")
		req.RemoteAddr = "10.0.0.10:1234"

		got := ClientIPFromRequest(req)
		if got != "10.0.0.10" {
			t.Fatalf("expected private non-loopback remote addr to be used, got %s", got)
		}
	})

	t.Run("uses forwarded headers from configured trusted proxy cidr", func(t *testing.T) {
		t.Setenv("KIMBAP_TRUSTED_PROXY_CIDRS", "10.0.0.0/8")
		req := httptest.NewRequest("GET", "http://example.com", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.5")
		req.RemoteAddr = "10.1.2.3:1234"

		got := ClientIPFromRequest(req)
		if got != "203.0.113.5" {
			t.Fatalf("expected trusted proxy cidr to allow forwarded ip, got %s", got)
		}
	})

	t.Run("uses rightmost valid XFF entry", func(t *testing.T) {
		t.Setenv("KIMBAP_TRUSTED_PROXY_CIDRS", "")
		req := httptest.NewRequest("GET", "http://example.com", nil)
		req.Header.Set("X-Forwarded-For", "malformed, 203.0.113.7")
		req.RemoteAddr = "127.0.0.1:1234"

		got := ClientIPFromRequest(req)
		if got != "203.0.113.7" {
			t.Fatalf("expected rightmost valid XFF entry, got %s", got)
		}
	})

	t.Run("strips trusted proxies from xff tail", func(t *testing.T) {
		t.Setenv("KIMBAP_TRUSTED_PROXY_CIDRS", "10.0.0.0/8")
		req := httptest.NewRequest("GET", "http://example.com", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.9, 10.1.2.3")
		req.RemoteAddr = "10.1.2.4:1234"

		got := ClientIPFromRequest(req)
		if got != "203.0.113.9" {
			t.Fatalf("expected trusted proxy tail to be stripped, got %s", got)
		}
	})

	t.Run("ignores invalid X-Real-IP and falls back", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com", nil)
		req.Header.Set("X-Real-IP", "not-an-ip")
		req.RemoteAddr = "127.0.0.1:1234"

		got := ClientIPFromRequest(req)
		if got != "127.0.0.1" {
			t.Fatalf("expected fallback to remote addr, got %s", got)
		}
	})
}

func TestNormalizeIP(t *testing.T) {
	t.Run("normalizes IPv6-mapped IPv4", func(t *testing.T) {
		got := normalizeIP("::ffff:1.2.3.4")
		if got != "1.2.3.4" {
			t.Fatalf("expected 1.2.3.4, got %s", got)
		}
	})

	t.Run("normalizes loopback ::1 to 127.0.0.1", func(t *testing.T) {
		got := normalizeIP("::1")
		if got != "127.0.0.1" {
			t.Fatalf("expected 127.0.0.1, got %s", got)
		}
	})
}
