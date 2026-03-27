package main

import (
	"fmt"
	"os"

	"github.com/dunialabs/kimbap-core/internal/kerrors"
)

func printErrorHint(err error) {
	hint := kerrors.GetHint(err)
	if hint == "" {
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, "\nHint: %s\n", hint)
	docsURL := kerrors.GetDocsURL(err)
	if docsURL != "" {
		_, _ = fmt.Fprintf(os.Stderr, "Docs: %s\n", docsURL)
	}
}
