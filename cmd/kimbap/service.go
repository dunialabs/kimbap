package main

import "github.com/spf13/cobra"

func newServiceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Manage service manifests",
	}

	cmd.AddCommand(newServiceInstallCommand())
	cmd.AddCommand(newServiceListCommand())
	cmd.AddCommand(newServiceSearchCommand())
	cmd.AddCommand(newServiceDescribeCommand())
	cmd.AddCommand(newServiceEnableCommand())
	cmd.AddCommand(newServiceDisableCommand())
	cmd.AddCommand(newServiceRemoveCommand())
	cmd.AddCommand(newServiceUpdateCommand())
	cmd.AddCommand(newServiceOutdatedCommand())
	cmd.AddCommand(newServiceVerifyCommand())
	cmd.AddCommand(newServiceSignCommand())
	cmd.AddCommand(newServiceValidateCommand())
	cmd.AddCommand(newServiceGenerateCommand())
	cmd.AddCommand(newServiceExportAgentSkillCommand())

	return cmd
}
