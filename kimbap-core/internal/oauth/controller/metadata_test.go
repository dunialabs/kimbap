package controller

import "testing"

func TestPublicURL(t *testing.T) {
	t.Run("uses canonical public URL when configured", func(t *testing.T) {
		t.Setenv("KIMBAP_PUBLIC_BASE_URL", "https://public.example.com/")
		got := publicURL()
		if got != "https://public.example.com" {
			t.Fatalf("expected canonical public url, got %s", got)
		}
	})

	t.Run("ignores canonical public URL with userinfo", func(t *testing.T) {
		t.Setenv("KIMBAP_PUBLIC_BASE_URL", "https://user:pass@public.example.com")
		got := publicURL()
		if got != "" {
			t.Fatalf("expected invalid canonical URL to be ignored, got %s", got)
		}
	})

	t.Run("returns empty when not configured", func(t *testing.T) {
		t.Setenv("KIMBAP_PUBLIC_BASE_URL", "")
		got := publicURL()
		if got != "" {
			t.Fatalf("expected empty when KIMBAP_PUBLIC_BASE_URL is unset, got %s", got)
		}
	})

	t.Run("rejects non-http scheme", func(t *testing.T) {
		t.Setenv("KIMBAP_PUBLIC_BASE_URL", "ftp://public.example.com")
		got := publicURL()
		if got != "" {
			t.Fatalf("expected non-http scheme to be rejected, got %s", got)
		}
	})
}
