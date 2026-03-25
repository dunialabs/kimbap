package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	cobra.EnableTraverseRunHooks = true
	if err := rootCmd.Execute(); err != nil {
		os.Exit(mapErrorToExitCode(err))
	}
}
