package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/kerrors"
)

func printErrorHint(err error) {
	hint := kerrors.GetHint(err)
	docsURL := kerrors.GetDocsURL(err)

	if hint == "" {
		var execErr *actions.ExecutionError
		if errors.As(err, &execErr) {
			hint, docsURL = executionErrorHint(execErr)
		}
	}

	if hint == "" {
		return
	}

	_, _ = fmt.Fprintf(os.Stderr, "\nHint: %s\n", hint)
	if docsURL != "" {
		_, _ = fmt.Fprintf(os.Stderr, "Docs: %s\n", docsURL)
	}
}

func executionErrorHint(execErr *actions.ExecutionError) (hint string, docsURL string) {
	switch execErr.Code {
	case actions.ErrCredentialMissing:
		return kerrors.ErrCredentialMissing.Hint, kerrors.ErrCredentialMissing.DocsURL
	case actions.ErrTokenExpired:
		return "Re-authenticate with 'kimbap link <service>' to refresh the token.", ""
	case actions.ErrConnectorNotLoggedIn:
		return "Run 'kimbap link <service>' to connect via OAuth.", ""
	case actions.ErrApprovalRequired:
		return "This action requires approval. Run 'kimbap approve accept <id>' after requesting.", ""
	case actions.ErrApprovalTimeout:
		return "The approval request timed out. Re-run the command to create a new approval request.", ""
	case actions.ErrActionNotFound:
		return kerrors.ErrActionNotFound.Hint, kerrors.ErrActionNotFound.DocsURL
	case actions.ErrServiceInvalid:
		return kerrors.ErrManifestInvalid.Hint, kerrors.ErrManifestInvalid.DocsURL
	default:
		return "", ""
	}
}
