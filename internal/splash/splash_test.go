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

func TestRenderColorProfileANSI256(t *testing.T) {
	out := Render(Options{ColorProfile: ColorProfileANSI256})
	if !strings.Contains(out, "\x1b[38;5;") {
		t.Fatalf("expected ANSI 256 color escapes, got %q", out)
	}
	if strings.Contains(out, "\x1b[38;2;") {
		t.Fatalf("expected truecolor escapes to be converted, got %q", out)
	}
}

func TestRenderColorProfileNone(t *testing.T) {
	out := Render(Options{ColorProfile: ColorProfileNone})
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("expected ANSI escapes to be stripped, got %q", out)
	}
	if !strings.Contains(out, "k i m b a p") {
		t.Fatalf("render output missing brand title: %q", out)
	}
}

func TestRenderLightBackgroundUsesDarkerBrandText(t *testing.T) {
	out := Render(Options{Background: BackgroundToneLight})
	if !strings.Contains(out, rgb(15, 23, 32)+"k i m b a p") {
		t.Fatalf("expected darker title color for light backgrounds, got %q", out)
	}
	if !strings.Contains(out, rgb(234, 224, 202)+"██") {
		t.Fatalf("expected lighter rice palette for light backgrounds, got %q", out)
	}
	if !strings.Contains(out, rgb(12, 52, 24)+"██") {
		t.Fatalf("expected deeper nori outline for light backgrounds, got %q", out)
	}
}

func TestGetCellOutsideShapeReturnsNil(t *testing.T) {
	if v := getCell(2.0, 2.0, darkPalette); v != nil {
		t.Fatalf("expected nil cell outside splash shape, got %+v", v)
	}
	if v := getCell(0.0, 0.0, darkPalette); v == nil {
		t.Fatal("expected non-nil cell in splash center")
	}
}
