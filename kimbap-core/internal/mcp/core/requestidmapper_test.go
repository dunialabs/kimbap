package core

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRequestIDKey_TypeSafety(t *testing.T) {
	t.Run("string and int IDs produce different keys", func(t *testing.T) {
		stringKey := requestIDKey("1")
		intKey := requestIDKey(1)
		if stringKey == intKey {
			t.Fatalf("string '1' and int 1 produced same key: %s", stringKey)
		}
		if stringKey != "s:1" {
			t.Fatalf("expected s:1 for string, got %s", stringKey)
		}
		if intKey != "n:1" {
			t.Fatalf("expected n:1 for int, got %s", intKey)
		}
	})

	t.Run("json.Number and int produce same numeric key", func(t *testing.T) {
		jsonNumKey := requestIDKey(json.Number("42"))
		intKey := requestIDKey(42)
		if jsonNumKey != intKey {
			t.Fatalf("json.Number('42') = %s, int(42) = %s", jsonNumKey, intKey)
		}
	})

	t.Run("float64 integer value matches int key", func(t *testing.T) {
		floatKey := requestIDKey(float64(1))
		intKey := requestIDKey(1)
		if floatKey != intKey {
			t.Fatalf("float64(1) = %s, int(1) = %s", floatKey, intKey)
		}
	})

	t.Run("nil produces null key", func(t *testing.T) {
		if got := requestIDKey(nil); got != "null" {
			t.Fatalf("expected null, got %s", got)
		}
	})
}

func TestRequestIDMapper_Lifecycle(t *testing.T) {
	t.Run("register and lookup round-trip", func(t *testing.T) {
		mapper := NewRequestIDMapper("sess-1")
		defer mapper.Destroy()

		proxyID := mapper.RegisterClientRequest(42, "tools/call", "server-1")
		got, ok := mapper.GetProxyRequestID(42)
		if !ok || got != proxyID {
			t.Fatalf("expected %s, got %s (ok=%v)", proxyID, got, ok)
		}

		original, ok := mapper.GetOriginalRequestID(proxyID)
		if !ok || original != 42 {
			t.Fatalf("expected original 42, got %v (ok=%v)", original, ok)
		}
	})

	t.Run("string and int IDs use distinct map keys", func(t *testing.T) {
		mapper := NewRequestIDMapper("sess-2")
		defer mapper.Destroy()

		proxyStr := mapper.RegisterClientRequest("1", "tools/call", "s1")
		time.Sleep(2 * time.Millisecond)
		proxyInt := mapper.RegisterClientRequest(1, "tools/call", "s1")

		if proxyStr == proxyInt {
			t.Fatalf("string '1' and int 1 got same proxy ID")
		}

		gotStr, ok := mapper.GetProxyRequestID("1")
		if !ok || gotStr != proxyStr {
			t.Fatalf("string '1' lookup failed: got %s (ok=%v)", gotStr, ok)
		}

		gotInt, ok := mapper.GetProxyRequestID(1)
		if !ok || gotInt != proxyInt {
			t.Fatalf("int 1 lookup failed: got %s (ok=%v)", gotInt, ok)
		}
	})

	t.Run("downstream mapping round-trip", func(t *testing.T) {
		mapper := NewRequestIDMapper("sess-3")
		defer mapper.Destroy()

		proxyID := mapper.RegisterClientRequest(10, "tools/call", "srv-a")
		mapper.RegisterDownstreamMapping(proxyID, 99, "srv-a")

		got, ok := mapper.GetProxyRequestIDFromDownstream(99, "srv-a")
		if !ok || got != proxyID {
			t.Fatalf("downstream lookup failed: got %s (ok=%v)", got, ok)
		}

		original, ok := mapper.GetOriginalRequestIDFromDownstream(99, "srv-a")
		if !ok || original != 10 {
			t.Fatalf("expected 10, got %v", original)
		}
	})

	t.Run("RemoveMapping cleans all directions", func(t *testing.T) {
		mapper := NewRequestIDMapper("sess-4")
		defer mapper.Destroy()

		proxyID := mapper.RegisterClientRequest(5, "tools/call", "srv-b")
		mapper.RegisterDownstreamMapping(proxyID, 55, "srv-b")

		mapper.RemoveMapping(proxyID)

		if _, ok := mapper.GetProxyRequestID(5); ok {
			t.Fatal("client->proxy mapping should be removed")
		}
		if _, ok := mapper.GetOriginalRequestID(proxyID); ok {
			t.Fatal("proxy->client mapping should be removed")
		}
		if _, ok := mapper.GetProxyRequestIDFromDownstream(55, "srv-b"); ok {
			t.Fatal("downstream->proxy mapping should be removed")
		}
	})

	t.Run("Destroy stops cleanup goroutine and clears maps", func(t *testing.T) {
		mapper := NewRequestIDMapper("sess-5")
		mapper.RegisterClientRequest(1, "test", "srv")

		mapper.Destroy()

		if _, ok := mapper.GetProxyRequestID(1); ok {
			t.Fatal("expected maps to be cleared after Destroy")
		}
	})

	t.Run("expired entries are cleaned up", func(t *testing.T) {
		mapper := NewRequestIDMapper("sess-6")
		defer mapper.Destroy()
		mapper.ttl = 50 * time.Millisecond

		mapper.RegisterClientRequest(1, "test", "srv")

		time.Sleep(100 * time.Millisecond)
		mapper.cleanupExpired()

		if _, ok := mapper.GetProxyRequestID(1); ok {
			t.Fatal("expired entry should have been cleaned up")
		}
	})
}
