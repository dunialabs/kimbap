package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/dunialabs/kimbap/internal/config"
	"github.com/spf13/cobra"
)

type cliOptions struct {
	configPath     string
	dataDir        string
	logLevel       string
	mode           string
	format         string
	splashColor    string
	jsonInput      string
	idempotencyKey string
	dryRun         bool
	trace          bool
	noSplash       bool
}

const defaultApprovalTTL = 30 * time.Minute

var opts = cliOptions{}

func binaryName() string {
	if len(os.Args) > 0 {
		if name := filepath.Base(os.Args[0]); name != "" {
			return name
		}
	}
	return "kimbap"
}

var rootCmd = &cobra.Command{
	Use:          "kimbap",
	Short:        "Secure action runtime for AI agents",
	Long:         "Kimbap lets AI agents use external services without handling raw credentials.",
	SilenceUsage: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		prescanRawSplashFlags()
		showSplashOnce()
	},
}

var splashShown bool

func init() {
	flags := rootCmd.PersistentFlags()
	flags.StringVar(&opts.configPath, "config", "", "config file path")
	flags.StringVar(&opts.dataDir, "data-dir", "", "data directory (default ~/.kimbap)")
	flags.StringVar(&opts.logLevel, "log-level", "", "log level (debug, info, warn, error)")
	flags.StringVar(&opts.mode, "mode", "", "execution mode (dev, embedded, connected)")
	flags.StringVar(&opts.format, "format", "text", "output format (text, json)")
	flags.StringVar(&opts.splashColor, "splash-color", "", "splash color mode override (auto, truecolor, ansi256, none)")
	flags.BoolVar(&opts.dryRun, "dry-run", false, "validate and preview without executing (returns JSON)")
	flags.BoolVar(&opts.trace, "trace", false, "print execution pipeline trace to stderr")
	flags.BoolVar(&opts.noSplash, "no-splash", false, "disable startup splash output")
	name := binaryName()
	rootCmd.Use = name
	rootCmd.Version = config.CLIVersionDisplay()
	rootCmd.SetVersionTemplate(name + " {{.Version}}\n")
	defaultHelp := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		showSplashOnce()
		defaultHelp(cmd, args)
	})

	rootCmd.AddGroup(
		&cobra.Group{ID: "core", Title: "Core Commands:"},
		&cobra.Group{ID: "setup", Title: "Setup:"},
		&cobra.Group{ID: "management", Title: "Management:"},
		&cobra.Group{ID: "advanced", Title: "Advanced:"},
	)

	addGrouped := func(cmd *cobra.Command, groupID string) {
		cmd.GroupID = groupID
		rootCmd.AddCommand(cmd)
	}
	addGroupedHidden := func(cmd *cobra.Command, groupID string) {
		cmd.GroupID = groupID
		cmd.Hidden = true
		rootCmd.AddCommand(cmd)
	}

	addGrouped(newCallCommand(), "core")
	addGrouped(newLinkCommand(), "core")
	addGrouped(newSearchCommand(), "core")
	addGrouped(newActionsCommand(), "core")
	addGrouped(newAliasCommand(), "core")

	addGrouped(newInitCommand(), "setup")
	addGrouped(newServiceCommand(), "setup")
	addGrouped(newAgentsCommand(), "setup")

	addGrouped(newVaultCommand(), "management")
	addGrouped(newPolicyCommand(), "management")
	addGroupedHidden(newCheckCommand(), "management")
	addGrouped(newDoctorCommand(), "management")
	addGrouped(newStatusCommand(), "management")
	addGrouped(newAuthCommand(), "management")

	addGrouped(newApproveCommand(), "advanced")
	addGrouped(newRunCommand(), "advanced")
	addGrouped(newProxyCommand(), "advanced")
	addGrouped(newServeCommand(), "advanced")

	for _, c := range []*cobra.Command{
		newConnectorCommand(),
		newAgentProfileCommand(),
		newGenerateCommand(),
		newTokenCommand(),
		newAuditCommand(),
		newDaemonCommand(),
		newCompletionCommand(),
	} {
		c.Hidden = true
		rootCmd.AddCommand(c)
	}
}

func newCompletionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for kimbap (also available as kb).

Add to your shell profile:
  bash:       source <(kimbap completion bash)
  zsh:        source <(kimbap completion zsh)
  fish:       kimbap completion fish | source
  powershell: kimbap completion powershell | Out-String | Invoke-Expression`,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		DisableFlagsInUseLine: true,
		RunE: func(_ *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				return rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				return rootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			}
			return nil
		},
	}
}
