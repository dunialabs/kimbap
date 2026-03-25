package socket

import (
	"net/http/httptest"
	"testing"
)

func TestIsAllowedSocketOrigin(t *testing.T) {
	t.Run("allows same host origin", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com/socket.io/", nil)
		req.Host = "api.example.com:8080"
		if !isAllowedSocketOrigin(req, "https://api.example.com") {
			t.Fatal("expected same host origin to be allowed")
		}
	})

	t.Run("rejects different host origin", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com/socket.io/", nil)
		req.Host = "api.example.com"
		if isAllowedSocketOrigin(req, "https://evil.example.com") {
			t.Fatal("expected different host origin to be rejected")
		}
	})
}
