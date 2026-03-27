package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/store"
)

func TestWithRuntimeStoreWrapsOpenFailures(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Database.Driver = "unsupported-driver"

	err := withRuntimeStore(cfg, func(_ *store.SQLStore) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected openRuntimeStore failure")
	}
	if !isRuntimeStoreUnavailable(err) {
		t.Fatalf("expected runtime store unavailable classification, got %v", err)
	}
}

func TestWithRuntimeStorePreservesCallbackError(t *testing.T) {
	cfg := config.DefaultConfig()
	tmp := t.TempDir()
	cfg.DataDir = tmp
	cfg.Database.Driver = "sqlite"
	cfg.Database.DSN = filepath.Join(tmp, "kimbap.db")

	sentinel := errors.New("sentinel callback failure")
	err := withRuntimeStore(cfg, func(_ *store.SQLStore) error {
		return sentinel
	})
	if err == nil {
		t.Fatal("expected callback failure")
	}
	if isRuntimeStoreUnavailable(err) {
		t.Fatalf("expected callback error to remain non-store-unavailable, got %v", err)
	}
	if !strings.Contains(err.Error(), "sentinel callback failure") {
		t.Fatalf("expected sentinel error, got %v", err)
	}
}

func TestRunApproveAcceptPreservesDomainErrors(t *testing.T) {
	dataDir := t.TempDir()
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cfgRaw := "data_dir: " + dataDir + "\n" +
		"database:\n" +
		"  driver: sqlite\n" +
		"  dsn: " + filepath.Join(dataDir, "kimbap.db") + "\n"
	if err := os.WriteFile(cfgPath, []byte(cfgRaw), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	prevOpts := opts
	opts = cliOptions{configPath: cfgPath, format: "json"}
	t.Cleanup(func() {
		opts = prevOpts
	})

	err := runApproveAccept("missing-request")
	if err == nil {
		t.Fatal("expected approve failure for missing request")
	}
	if isRuntimeStoreUnavailable(err) {
		t.Fatalf("expected domain failure, got runtime-store-unavailable: %v", err)
	}
	if !strings.Contains(err.Error(), "approve failed") {
		t.Fatalf("expected approve failure context, got %v", err)
	}
}
