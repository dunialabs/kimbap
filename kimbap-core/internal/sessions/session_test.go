package sessions

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/dunialabs/kimbap-core/internal/auth"
)

func TestSessionServiceExchangeAndValidate(t *testing.T) {
	svc := NewSessionService(0)
	principal := &auth.Principal{
		ID:        "svc-agent",
		TenantID:  "tenant-a",
		AgentName: "agent-alpha",
		Scopes:    []string{"tools:read"},
	}

	session, raw, err := svc.Exchange(context.Background(), principal)
	if err != nil {
		t.Fatalf("exchange session: %v", err)
	}
	if !strings.HasPrefix(raw, sessionTokenPrefix) {
		t.Fatalf("expected raw session with prefix %q, got %q", sessionTokenPrefix, raw)
	}
	if session == nil {
		t.Fatal("expected session token")
	}

	validated, err := svc.Validate(context.Background(), raw)
	if err != nil {
		t.Fatalf("validate session: %v", err)
	}
	if validated.ID != session.ID {
		t.Fatalf("session id mismatch: got=%q want=%q", validated.ID, session.ID)
	}
}

func TestSessionServiceValidateFailsWhenExpired(t *testing.T) {
	svc := NewSessionService(time.Millisecond)
	principal := &auth.Principal{ID: "svc-agent", TenantID: "tenant-a"}

	_, raw, err := svc.Exchange(context.Background(), principal)
	if err != nil {
		t.Fatalf("exchange session: %v", err)
	}

	time.Sleep(5 * time.Millisecond)
	_, err = svc.Validate(context.Background(), raw)
	if !errors.Is(err, ErrExpiredSession) {
		t.Fatalf("expected ErrExpiredSession, got %v", err)
	}
}
