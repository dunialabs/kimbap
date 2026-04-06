package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSplashShownRecently_NoStampFile(t *testing.T) {
	resetOptsForTest(t)
	opts.dataDir = t.TempDir()

	if splashShownRecently() {
		t.Fatal("expected splashShownRecently to be false without a stamp file")
	}
}

func TestSplashShownRecently_RecentStamp(t *testing.T) {
	resetOptsForTest(t)
	dataDir := t.TempDir()
	opts.dataDir = dataDir
	writeSplashStamp(t, dataDir, strconv.FormatInt(time.Now().Unix(), 10))

	if !splashShownRecently() {
		t.Fatal("expected splashShownRecently to be true for a recent stamp")
	}
}

func TestSplashShownRecently_OldStamp(t *testing.T) {
	resetOptsForTest(t)
	dataDir := t.TempDir()
	opts.dataDir = dataDir
	writeSplashStamp(t, dataDir, strconv.FormatInt(time.Now().Add(-3*time.Hour).Unix(), 10))

	if splashShownRecently() {
		t.Fatal("expected splashShownRecently to be false for an old stamp")
	}
}

func TestSplashShownRecently_FutureStamp(t *testing.T) {
	resetOptsForTest(t)
	dataDir := t.TempDir()
	opts.dataDir = dataDir
	writeSplashStamp(t, dataDir, strconv.FormatInt(time.Now().Add(1*time.Hour).Unix(), 10))

	if splashShownRecently() {
		t.Fatal("expected splashShownRecently to be false for a future stamp")
	}
}

func TestSplashShownRecently_MalformedStamp(t *testing.T) {
	resetOptsForTest(t)
	dataDir := t.TempDir()
	opts.dataDir = dataDir
	writeSplashStamp(t, dataDir, "not-a-number")

	if splashShownRecently() {
		t.Fatal("expected splashShownRecently to be false for a malformed stamp")
	}
}

func TestMarkSplashShown_CreatesStamp(t *testing.T) {
	resetOptsForTest(t)
	dataDir := t.TempDir()
	opts.dataDir = dataDir

	before := time.Now().Unix()
	markSplashShown()
	after := time.Now().Unix()

	raw := readSplashStamp(t, dataDir)
	ts, err := strconv.ParseInt(strings.TrimSpace(string(raw)), 10, 64)
	if err != nil {
		t.Fatalf("expected valid unix timestamp, got error: %v", err)
	}
	if ts < before || ts > after {
		t.Fatalf("expected stamp timestamp between %d and %d, got %d", before, after, ts)
	}
}

func TestMarkSplashShown_CreatesDataDir(t *testing.T) {
	resetOptsForTest(t)
	baseDir := t.TempDir()
	dataDir := filepath.Join(baseDir, "nested", "data")
	opts.dataDir = dataDir

	markSplashShown()

	if _, err := os.Stat(dataDir); err != nil {
		t.Fatalf("expected data dir to be created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, ".splash-stamp")); err != nil {
		t.Fatalf("expected splash stamp file to be created: %v", err)
	}
}

func TestMarkSplashShown_DataDirOverride(t *testing.T) {
	resetOptsForTest(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	dataDir := filepath.Join(t.TempDir(), "override")
	opts.dataDir = dataDir

	markSplashShown()

	overrideStamp := filepath.Join(dataDir, ".splash-stamp")
	if _, err := os.Stat(overrideStamp); err != nil {
		t.Fatalf("expected splash stamp in overridden data dir: %v", err)
	}

	defaultStamp := filepath.Join(home, ".kimbap", ".splash-stamp")
	if _, err := os.Stat(defaultStamp); !os.IsNotExist(err) {
		t.Fatalf("expected no splash stamp in default data dir, stat err=%v", err)
	}
}

func writeSplashStamp(t *testing.T, dataDir string, value string) {
	t.Helper()
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir data dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, ".splash-stamp"), []byte(value), 0o600); err != nil {
		t.Fatalf("write splash stamp: %v", err)
	}
}

func readSplashStamp(t *testing.T, dataDir string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dataDir, ".splash-stamp"))
	if err != nil {
		t.Fatalf("read splash stamp: %v", err)
	}
	return raw
}
