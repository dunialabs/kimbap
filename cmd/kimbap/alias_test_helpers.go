package main

import (
	"os"
	"path/filepath"
	"testing"
)

func stubAliasLookPathToDir(t *testing.T, dir string) {
	t.Helper()
	orig := aliasLookPath
	aliasLookPath = func(file string) (string, error) {
		if file == "" {
			return "", os.ErrNotExist
		}
		return filepath.Join(dir, file), nil
	}
	t.Cleanup(func() {
		aliasLookPath = orig
	})
}
