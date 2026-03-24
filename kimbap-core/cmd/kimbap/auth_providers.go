package main

import (
	"fmt"
	"os"
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
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			items := providers.ListProviders()
			sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })

			out := make([]map[string]any, 0, len(items))
			for _, item := range items {
				configured := providerIsConfigured(item) && strings.TrimSpace(resolveClientID(cfg, item.ID)) != ""
				out = append(out, map[string]any{
					"id":                     item.ID,
					"display_name":           item.DisplayName,
					"supported_flows":        item.SupportedFlows,
					"configured":             configured,
					"supports_browser_flow":  item.SupportsBrowserFlow(),
					"supports_device_flow":   item.SupportsDeviceFlow(),
					"supports_client_creds":  item.SupportsClientCredentials(),
					"connection_scope_model": item.ConnectionScopeModel,
				})
			}

			if !outputAsJSON() {
				if len(items) == 0 {
					_, _ = fmt.Fprintln(os.Stdout, "No providers configured.")
					return nil
				}
				_, _ = fmt.Fprintf(os.Stdout, "%-15s  %-20s  %-30s  %-12s  %s\n", "ID", "NAME", "FLOWS", "CONFIGURED", "SCOPES")
				for _, item := range items {
					configured := providerIsConfigured(item) && strings.TrimSpace(resolveClientID(cfg, item.ID)) != ""
					flowStrs := make([]string, 0, len(item.SupportedFlows))
					for _, f := range item.SupportedFlows {
						flowStrs = append(flowStrs, string(f))
					}
					scopeStrs := make([]string, 0, len(item.ConnectionScopeModel))
					for _, s := range item.ConnectionScopeModel {
						scopeStrs = append(scopeStrs, string(s))
					}
					_, _ = fmt.Fprintf(os.Stdout, "%-15s  %-20s  %-30s  %-12v  %s\n",
						item.ID, item.DisplayName,
						strings.Join(flowStrs, ", "),
						configured,
						strings.Join(scopeStrs, ", "),
					)
				}
				return nil
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
				if outputAsJSON() {
					_ = printOutput(map[string]any{
						"status":    "not_found",
						"operation": "auth.providers.describe",
						"provider":  providerID,
						"message":   err.Error(),
					})
				}
				return fmt.Errorf("provider %q not found: %w", providerID, err)
			}

			if !outputAsJSON() {
				_, _ = fmt.Fprintf(os.Stdout, "Provider:             %s\n", provider.ID)
				_, _ = fmt.Fprintf(os.Stdout, "Display Name:         %s\n", provider.DisplayName)
				flowStrs := make([]string, 0, len(provider.SupportedFlows))
				for _, f := range provider.SupportedFlows {
					flowStrs = append(flowStrs, string(f))
				}
				_, _ = fmt.Fprintf(os.Stdout, "Supported Flows:      %s\n", strings.Join(flowStrs, ", "))
				_, _ = fmt.Fprintf(os.Stdout, "Browser Flow:         %v\n", provider.SupportsBrowserFlow())
				_, _ = fmt.Fprintf(os.Stdout, "Device Flow:          %v\n", provider.SupportsDeviceFlow())
				_, _ = fmt.Fprintf(os.Stdout, "Client Credentials:   %v\n", provider.SupportsClientCredentials())
				if len(provider.DefaultScopes) > 0 {
					_, _ = fmt.Fprintf(os.Stdout, "Default Scopes:       %s\n", strings.Join(provider.DefaultScopes, ", "))
				}
				if len(provider.ScopePresets) > 0 {
					presetNames := make([]string, 0, len(provider.ScopePresets))
					for name := range provider.ScopePresets {
						presetNames = append(presetNames, name)
					}
					sort.Strings(presetNames)
					_, _ = fmt.Fprintf(os.Stdout, "Scope Presets:        %s\n", strings.Join(presetNames, ", "))
				}
				scopeStrs := make([]string, 0, len(provider.ConnectionScopeModel))
				for _, s := range provider.ConnectionScopeModel {
					scopeStrs = append(scopeStrs, string(s))
				}
				_, _ = fmt.Fprintf(os.Stdout, "Connection Scopes:    %s\n", strings.Join(scopeStrs, ", "))
				_, _ = fmt.Fprintf(os.Stdout, "PKCE Required:        %v\n", provider.PKCERequired)
				if provider.Notes != "" {
					_, _ = fmt.Fprintf(os.Stdout, "Notes:                %s\n", provider.Notes)
				}
				return nil
			}

			return printOutput(map[string]any{
				"status":                 "ok",
				"operation":              "auth.providers.describe",
				"id":                     provider.ID,
				"display_name":           provider.DisplayName,
				"supported_flows":        provider.SupportedFlows,
				"supports_browser_flow":  provider.SupportsBrowserFlow(),
				"supports_device_flow":   provider.SupportsDeviceFlow(),
				"supports_client_creds":  provider.SupportsClientCredentials(),
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
