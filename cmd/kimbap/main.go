package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	cobra.EnableTraverseRunHooks = true
	os.Args = rewriteArgsForExecutableAlias(os.Args)
	if err := rootCmd.Execute(); err != nil {
		printErrorHint(err)
		os.Exit(mapErrorToExitCode(err))
	}
}
