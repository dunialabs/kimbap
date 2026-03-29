package connectors

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchUserInfoRejectsOversizedResponse(t *testing.T) {
	oversizedLogin := strings.Repeat("a", (1<<20)+32)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"login":"` + oversizedLogin + `"}`))
	}))
	defer server.Close()

	_, err := FetchUserInfo(context.Background(), server.URL, "token")
	if err == nil {
		t.Fatal("expected oversize error, got nil")
	}
	if !strings.Contains(err.Error(), "userinfo response exceeded") {
		t.Fatalf("expected oversize error message, got %v", err)
	}
}
