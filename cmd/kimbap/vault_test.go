package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadSecretInputRejectsOversizedFile(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "secret.bin")
	payload := strings.Repeat("a", (1<<20)+1)
	if err := os.WriteFile(filePath, []byte(payload), 0o600); err != nil {
		t.Fatalf("write oversized secret file: %v", err)
	}

	_, err := readSecretInput(filePath, false)
	if err == nil {
		t.Fatal("expected oversized file to return error")
	}
	if !strings.Contains(err.Error(), "file payload exceeds") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVaultListFooterEmpty(t *testing.T) {
	hint := "Use 'kimbap link <service> --stdin' to store credentials for a service."
	if !strings.Contains(hint, "kimbap link") {
		t.Fatalf("empty-state footer hint does not reference kimbap link: %q", hint)
	}
}

func TestVaultListFooterNonEmpty(t *testing.T) {
	hint := "Use 'kimbap link list' to see which services are connected."
	if !strings.Contains(hint, "kimbap link list") {
		t.Fatalf("non-empty-state footer hint does not reference kimbap link list: %q", hint)
	}
}
