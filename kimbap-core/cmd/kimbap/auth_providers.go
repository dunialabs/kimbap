package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

func newAuthProvidersCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "providers",
		Short: "Inspect provider registry metadata",
	}
	cmd.AddCommand(newAuthProvidersListCommand())
	cmd.AddCommand(newAuthProvidersDescribeCommand())
	return cmd
}

func newAuthProvidersListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List known OAuth providers",
		RunE: func(_ *cobra.Command, _ []string) error {
			items := providers.ListProviders()
			sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })

			out := make([]map[string]any, 0, len(items))
			for _, item := range items {
				out = append(out, map[string]any{
					"id":              item.ID,
					"display_name":    item.DisplayName,
					"supported_flows": item.SupportedFlows,
					"configured":      providerIsConfigured(item),
				})
			}

			return printOutput(map[string]any{
				"status":    "ok",
				"operation": "auth.providers.list",
				"count":     len(out),
				"providers": out,
			})
		},
	}
	return cmd
}

func newAuthProvidersDescribeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe <provider>",
		Short: "Describe OAuth provider metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			providerID := strings.TrimSpace(args[0])
			if providerID == "" {
				return fmt.Errorf("provider is required")
			}
			provider, err := providers.GetProvider(providerID)
			if err != nil {
				return printOutput(map[string]any{
					"status":    "not_found",
					"operation": "auth.providers.describe",
					"provider":  providerID,
					"message":   err.Error(),
				})
			}

			return printOutput(map[string]any{
				"status":                 "ok",
				"operation":              "auth.providers.describe",
				"id":                     provider.ID,
				"display_name":           provider.DisplayName,
				"supported_flows":        provider.SupportedFlows,
				"default_scopes":         provider.DefaultScopes,
				"scope_presets":          provider.ScopePresets,
				"auth_endpoint":          provider.AuthEndpoint,
				"token_endpoint":         provider.TokenEndpoint,
				"device_endpoint":        provider.DeviceEndpoint,
				"revocation_endpoint":    provider.RevocationEndpoint,
				"userinfo_endpoint":      provider.UserInfoEndpoint,
				"connection_scope_model": provider.ConnectionScopeModel,
				"pkce_required":          provider.PKCERequired,
				"notes":                  provider.Notes,
			})
		},
	}
	return cmd
}
