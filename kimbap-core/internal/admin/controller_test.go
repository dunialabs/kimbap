package admin

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	types "github.com/dunialabs/kimbap-core/internal/types"
)

func TestIsLoopbackRequest(t *testing.T) {
	t.Run("accepts loopback remote without forwarded headers", func(t *testing.T) {
		req := httptest.NewRequest("POST", "http://example.com/admin", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		if !isLoopbackRequest(req) {
			t.Fatal("expected loopback request to be accepted")
		}
	})

	t.Run("rejects spoofed first XFF value on loopback proxy", func(t *testing.T) {
		req := httptest.NewRequest("POST", "http://example.com/admin", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		req.Header.Set("X-Forwarded-For", "127.0.0.1, 198.51.100.25")
		if isLoopbackRequest(req) {
			t.Fatal("expected spoofed forwarded request to be rejected")
		}
	})

	t.Run("rejects non-loopback remote even with loopback forwarded header", func(t *testing.T) {
		req := httptest.NewRequest("POST", "http://example.com/admin", nil)
		req.RemoteAddr = "198.51.100.25:1234"
		req.Header.Set("X-Forwarded-For", "127.0.0.1")
		if isLoopbackRequest(req) {
			t.Fatal("expected non-loopback remote request to be rejected")
		}
	})
}

func TestIsLoopbackHost(t *testing.T) {
	t.Run("accepts localhost host", func(t *testing.T) {
		if !isLoopbackHost("localhost:8080") {
			t.Fatal("expected localhost host to be loopback")
		}
	})

	t.Run("accepts loopback ip host", func(t *testing.T) {
		if !isLoopbackHost("127.0.0.1:8080") {
			t.Fatal("expected loopback ip host to be loopback")
		}
	})

	t.Run("rejects external host", func(t *testing.T) {
		if isLoopbackHost("api.example.com") {
			t.Fatal("expected external host to be non-loopback")
		}
	})
}

func TestIsPublicAdminActionSensitiveActionsRequireToken(t *testing.T) {
	if isPublicAdminAction(types.AdminActionCountUsers) {
		t.Fatal("expected count users action to require token")
	}
	if isPublicAdminAction(types.AdminActionCountServers) {
		t.Fatal("expected count servers action to require token")
	}
	if isPublicAdminAction(types.AdminActionGetProxy) {
		t.Fatal("expected get proxy action to require token")
	}
	if isPublicAdminAction(types.AdminActionGetOwner) {
		t.Fatal("expected get owner action to require token")
	}
	if isPublicAdminAction(types.AdminActionCreateProxy) {
		t.Fatal("expected create proxy action to require token")
	}
	if isPublicAdminAction(types.AdminActionDeleteProxy) {
		t.Fatal("expected delete proxy action to require token")
	}
	if isPublicAdminAction(types.AdminActionCreateUser) {
		t.Fatal("expected create user action to require token")
	}
}

func TestIsOwnerOnlyAdminActionIncludesDeleteProxy(t *testing.T) {
	if !isOwnerOnlyAdminAction(types.AdminActionDeleteProxy) {
		t.Fatal("expected delete proxy action to be owner-only")
	}
}

func TestHandleAdminRequestRejectsTrailingJSONPayload(t *testing.T) {
	controller := &Controller{}
	body := fmt.Sprintf(`{"action":%d,"data":{}}{"extra":1}`, types.AdminActionCountUsers)
	req := httptest.NewRequest(http.MethodPost, "http://example.com/admin", strings.NewReader(body))
	rr := httptest.NewRecorder()

	controller.HandleAdminRequest(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "Invalid admin request format") {
		t.Fatalf("expected invalid format error, got %s", rr.Body.String())
	}
}
