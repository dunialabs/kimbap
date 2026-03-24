package repository

import "testing"

func TestNormalizeJSONObjectBytes(t *testing.T) {
	t.Run("converts null to empty object", func(t *testing.T) {
		got := normalizeJSONObjectBytes([]byte("null"))
		if string(got) != "{}" {
			t.Fatalf("expected {}, got %s", string(got))
		}
	})

	t.Run("keeps object payload", func(t *testing.T) {
		got := normalizeJSONObjectBytes([]byte(`{"a":1}`))
		if string(got) != `{"a":1}` {
			t.Fatalf("expected object payload to be unchanged, got %s", string(got))
		}
	})
}
