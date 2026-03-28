package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newAgentProfileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "profile",
		Short:  "Removed — use 'kimbap agents' instead",
		Hidden: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("'kimbap profile' has been removed.\n\nUse these commands instead:\n  kimbap agents setup --with-profiles   (was: profile install)\n  kimbap agents status                  (was: profile list)")
		},
	}
	// Override help so --help shows the deprecation error, not old help text.
	cmd.SetHelpFunc(func(c *cobra.Command, _ []string) {
		fmt.Fprintln(c.ErrOrStderr(), c.RunE(c, nil))
	})
	return cmd
}
