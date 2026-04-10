package main

import (
	"fmt"
	"os"
	"strings"
)

func ensureDirectOutputPathSafe(rawPath string) error {
	trimmed := strings.TrimSpace(rawPath)
	if trimmed == "" {
		return nil
	}
	info, err := os.Lstat(trimmed)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect output path %q: %w", trimmed, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to write through symlinked output path: %s", trimmed)
	}
	return nil
}
