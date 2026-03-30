package main

import (
	"os"
	"testing"
)

func TestConfirmShortcutSetupDefaultsYesOnEOF(t *testing.T) {
	oldStdin := os.Stdin
	t.Cleanup(func() { os.Stdin = oldStdin })

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdin pipe: %v", err)
	}
	_ = w.Close()
	os.Stdin = r
	t.Cleanup(func() { _ = r.Close() })

	confirm, confirmErr := confirmShortcutSetup("demo")
	if confirmErr != nil {
		t.Fatalf("confirmShortcutSetup() error: %v", confirmErr)
	}
	if !confirm {
		t.Fatal("expected EOF input to default to yes")
	}
}

func TestConfirmShortcutSetupRespectsNo(t *testing.T) {
	oldStdin := os.Stdin
	t.Cleanup(func() { os.Stdin = oldStdin })

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdin pipe: %v", err)
	}
	if _, err := w.WriteString("n\n"); err != nil {
		t.Fatalf("write stdin input: %v", err)
	}
	_ = w.Close()
	os.Stdin = r
	t.Cleanup(func() { _ = r.Close() })

	confirm, confirmErr := confirmShortcutSetup("demo")
	if confirmErr != nil {
		t.Fatalf("confirmShortcutSetup() error: %v", confirmErr)
	}
	if confirm {
		t.Fatal("expected explicit no input to return false")
	}
}
