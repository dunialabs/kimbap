package core

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dunialabs/kimbap-core/internal/database"
	"github.com/dunialabs/kimbap-core/internal/security"
	"github.com/dunialabs/kimbap-core/internal/types"
)

func TestServerContext_StateManagement(t *testing.T) {
	t.Run("NewServerContext initializes defaults", func(t *testing.T) {
		ctx := NewServerContext(database.Server{ServerID: "srv-1"})

		if ctx.ServerID != "srv-1" {
			t.Fatalf("expected ServerID=srv-1, got %s", ctx.ServerID)
		}
		if ctx.Status != types.ServerStatusOffline {
			t.Fatalf("expected offline status, got %d", ctx.Status)
		}
		if ctx.Capabilities == nil {
			t.Fatalf("expected Capabilities to be initialized")
		}
		if ctx.CapabilitiesConfig.Tools == nil || ctx.CapabilitiesConfig.Resources == nil || ctx.CapabilitiesConfig.Prompts == nil {
			t.Fatalf("expected CapabilitiesConfig maps to be initialized")
		}
		if ctx.LastActive == 0 {
			t.Fatalf("expected LastActive to be initialized")
		}
	})

	t.Run("StatusSnapshot reflects UpdateStatus under lock", func(t *testing.T) {
		ctx := NewServerContext(database.Server{ServerID: "srv-2"})
		ctx.UpdateStatus(types.ServerStatusOnline)

		if got := ctx.StatusSnapshot(); got != types.ServerStatusOnline {
			t.Fatalf("expected status online, got %d", got)
		}
	})

	t.Run("Touch updates LastActive atomically", func(t *testing.T) {
		ctx := NewServerContext(database.Server{ServerID: "srv-3"})
		before := atomic.LoadInt64(&ctx.LastActive)
		time.Sleep(2 * time.Millisecond)
		ctx.Touch()
		after := atomic.LoadInt64(&ctx.LastActive)

		if after <= before {
			t.Fatalf("expected LastActive to increase, before=%d after=%d", before, after)
		}
	})

	t.Run("IsIdle returns true after timeout", func(t *testing.T) {
		ctx := NewServerContext(database.Server{ServerID: "srv-4"})
		atomic.StoreInt64(&ctx.LastActive, time.Now().Add(-2*time.Second).UnixMilli())

		if !ctx.IsIdle(500 * time.Millisecond) {
			t.Fatalf("expected context to be idle")
		}
	})

	t.Run("RecordTimeout increments before threshold", func(t *testing.T) {
		ctx := NewServerContext(database.Server{ServerID: "srv-5"})
		ctx.MaxTimeoutCount = 3

		// Below threshold: RecordTimeout returns false (no reconnect happened)
		if ok := ctx.RecordTimeout(errors.New("timeout-1")); ok {
			t.Fatalf("expected false below threshold (no reconnect)")
		}
		if ok := ctx.RecordTimeout(errors.New("timeout-2")); ok {
			t.Fatalf("expected false below threshold (no reconnect)")
		}
		if ctx.TimeoutCount != 2 {
			t.Fatalf("expected TimeoutCount=2, got %d", ctx.TimeoutCount)
		}
	})

	t.Run("ClearTimeout resets timeout count", func(t *testing.T) {
		ctx := NewServerContext(database.Server{ServerID: "srv-6"})
		_ = ctx.RecordTimeout(errors.New("timeout"))
		ctx.ClearTimeout()

		if ctx.TimeoutCount != 0 {
			t.Fatalf("expected TimeoutCount reset to 0, got %d", ctx.TimeoutCount)
		}
	})

	t.Run("RecordError increments count and stores message", func(t *testing.T) {
		ctx := NewServerContext(database.Server{ServerID: "srv-7"})
		ctx.RecordError("connection failed")

		if ctx.ErrorCount != 1 {
			t.Fatalf("expected ErrorCount=1, got %d", ctx.ErrorCount)
		}
		if ctx.LastError != "connection failed" {
			t.Fatalf("expected LastError to be recorded, got %q", ctx.LastError)
		}
	})
}

func TestExtractLaunchConfigNullNormalizesToEmptyMap(t *testing.T) {
	const userToken = "test-token"
	encrypted, err := security.EncryptData("null", userToken)
	if err != nil {
		t.Fatalf("failed to encrypt payload: %v", err)
	}

	launchConfig, _, err := extractLaunchConfig(encrypted, userToken)
	if err != nil {
		t.Fatalf("extractLaunchConfig returned error: %v", err)
	}
	if launchConfig == nil {
		t.Fatal("expected launchConfig map to be non-nil")
	}
	if len(launchConfig) != 0 {
		t.Fatalf("expected empty launchConfig map, got len=%d", len(launchConfig))
	}
}
