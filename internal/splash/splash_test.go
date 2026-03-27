package splash

import (
	"strings"
	"testing"
)

func TestRenderDefaultsAndBranding(t *testing.T) {
	out := Render(Options{})
	if !strings.Contains(out, "k i m b a p") {
		t.Fatalf("render output missing brand title: %q", out)
	}
	if !strings.Contains(out, string(ModeEmbedded)) {
		t.Fatalf("default mode should be embedded: %q", out)
	}
	if !strings.Contains(out, "vault") {
		t.Fatalf("render output missing vault status line")
	}
}

func TestRenderConnectedModeIncludesServer(t *testing.T) {
	out := Render(Options{
		Version:     "1.2.3",
		Mode:        ModeConnected,
		VaultStatus: "ready",
		Server:      "https://auth.example.com",
	})
	if !strings.Contains(out, "connected") {
		t.Fatalf("expected connected mode in render output")
	}
	if !strings.Contains(out, "https://auth.example.com") {
		t.Fatalf("expected server URL in render output")
	}
	if !strings.Contains(out, "1.2.3") {
		t.Fatalf("expected version in render output")
	}
}

func TestGetCellOutsideShapeReturnsNil(t *testing.T) {
	if v := getCell(2.0, 2.0); v != nil {
		t.Fatalf("expected nil cell outside splash shape, got %+v", v)
	}
	if v := getCell(0.0, 0.0); v == nil {
		t.Fatal("expected non-nil cell in splash center")
	}
}
