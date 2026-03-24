package security

import (
	"fmt"
	"testing"
	"time"
)

func TestRateLimitService_FixedWindowBehavior(t *testing.T) {
	t.Run("first request allowed with expected remaining", func(t *testing.T) {
		svc := NewRateLimitService()
		t.Cleanup(svc.Close)

		allowed, remaining, reset := svc.CheckRateLimit("user-1", 3)
		if !allowed {
			t.Fatalf("expected first request to be allowed")
		}
		if remaining != 2 {
			t.Fatalf("expected remaining=2, got %d", remaining)
		}
		if reset.Before(time.Now()) {
			t.Fatalf("expected reset in the future, got %v", reset)
		}
	})

	t.Run("window fills to limit and next request rejected", func(t *testing.T) {
		svc := NewRateLimitService()
		t.Cleanup(svc.Close)

		for i := 0; i < 3; i++ {
			allowed, _, _ := svc.CheckRateLimit("user-2", 3)
			if !allowed {
				t.Fatalf("expected request %d to be allowed", i+1)
			}
		}

		allowed, remaining, _ := svc.CheckRateLimit("user-2", 3)
		if allowed {
			t.Fatalf("expected request over limit to be rejected")
		}
		if remaining != 0 {
			t.Fatalf("expected remaining=0 when rejected, got %d", remaining)
		}
	})

	t.Run("counter resets after window expires", func(t *testing.T) {
		svc := NewRateLimitService()
		t.Cleanup(svc.Close)

		allowed, _, _ := svc.CheckRateLimit("user-3", 1)
		if !allowed {
			t.Fatalf("expected first request to be allowed")
		}

		svc.mu.Lock()
		svc.entries["user-3"].windowStart = time.Now().Add(-2 * svc.window)
		svc.mu.Unlock()

		allowed, remaining, _ := svc.CheckRateLimit("user-3", 1)
		if !allowed {
			t.Fatalf("expected request after window expiry to be allowed")
		}
		if remaining != 0 {
			t.Fatalf("expected remaining=0 for limit=1, got %d", remaining)
		}
	})

	t.Run("cleanup removes expired entries", func(t *testing.T) {
		svc := NewRateLimitService()
		t.Cleanup(svc.Close)

		_, _, _ = svc.CheckRateLimit("expired-user", 5)

		svc.mu.Lock()
		svc.entries["expired-user"].lastSeen = time.Now().Add(-3 * svc.window)
		svc.mu.Unlock()

		svc.cleanupExpired()

		svc.mu.Lock()
		_, ok := svc.entries["expired-user"]
		svc.mu.Unlock()
		if ok {
			t.Fatalf("expected expired entry to be deleted")
		}
	})

	t.Run("Close is idempotent", func(t *testing.T) {
		svc := NewRateLimitService()
		svc.Close()
		svc.Close()
	})

	t.Run("different users have independent windows", func(t *testing.T) {
		svc := NewRateLimitService()
		t.Cleanup(svc.Close)

		_, _, _ = svc.CheckRateLimit("user-a", 1)
		allowedB, remainingB, _ := svc.CheckRateLimit("user-b", 2)
		if !allowedB {
			t.Fatalf("expected user-b first request to be allowed")
		}
		if remainingB != 1 {
			t.Fatalf("expected user-b remaining=1, got %d", remainingB)
		}

		allowedA, _, _ := svc.CheckRateLimit("user-a", 1)
		if allowedA {
			t.Fatalf("expected user-a second request to be rejected")
		}
	})

	t.Run("changing limit can be isolated by keying user identity", func(t *testing.T) {
		svc := NewRateLimitService()
		t.Cleanup(svc.Close)

		keyAtLimitTwo := fmt.Sprintf("%s|limit:%d", "user-c", 2)
		keyAtLimitFive := fmt.Sprintf("%s|limit:%d", "user-c", 5)

		allowed, remaining, _ := svc.CheckRateLimit(keyAtLimitTwo, 2)
		if !allowed || remaining != 1 {
			t.Fatalf("expected first key to have independent counter, allowed=%v remaining=%d", allowed, remaining)
		}

		allowed, remaining, _ = svc.CheckRateLimit(keyAtLimitFive, 5)
		if !allowed || remaining != 4 {
			t.Fatalf("expected second key to create independent entry, allowed=%v remaining=%d", allowed, remaining)
		}
	})
}
