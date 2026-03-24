package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/connectors"
	realFlows "github.com/dunialabs/kimbap-core/internal/connectors/flows"
	"github.com/spf13/cobra"
)

var detectFlowEnvironment = realFlows.DetectEnvironment

func newAuthDoctorCommand() *cobra.Command {
	var tenant string
	var profile string
	var extras []string
	cmd := &cobra.Command{
		Use:   "doctor [provider]",
		Short: "Run OAuth-specific diagnostics",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			activeTenant := connectorTenant(tenant)
			extraValues := parseExtras(extras)
			providerID := ""
			if len(args) == 1 {
				providerID = strings.TrimSpace(args[0])
				if p, pErr := providers.GetProvider(providerID); pErr == nil {
					providerID = p.ID
				}
			}

			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

			checks := make([]doctorCheck, 0, 8)
			hasFailure := false

			keyCheck := checkConnectorEncryptionKeyPresent()
			checks = append(checks, keyCheck)
			hasFailure = hasFailure || keyCheck.Status == "fail"

			providerCheck := checkAuthProviderConfig(providerID)
			checks = append(checks, providerCheck)
			hasFailure = hasFailure || providerCheck.Status == "fail"

			expiryCheck, refreshCheck := checkAuthTokenState(cfg, activeTenant, providerID, profile)
			checks = append(checks, expiryCheck, refreshCheck)
			hasFailure = hasFailure || expiryCheck.Status == "fail" || refreshCheck.Status == "fail"

			reachabilityChecks := checkProviderEndpointsReachable(providerID, extraValues)
			for _, c := range reachabilityChecks {
				checks = append(checks, c)
				hasFailure = hasFailure || c.Status == "fail"
			}

			loopbackCheck := checkLoopbackPortBindable(0)
			checks = append(checks, loopbackCheck)
			hasFailure = hasFailure || loopbackCheck.Status == "fail"

			browserCheck := checkBrowserLaunchFeasible(providerID)
			checks = append(checks, browserCheck)
			hasFailure = hasFailure || browserCheck.Status == "fail"

			scopeCheck := checkGrantedScopes(cfg, activeTenant, providerID, profile)
			checks = append(checks, scopeCheck)
			hasFailure = hasFailure || scopeCheck.Status == "fail"

			if !outputAsJSON() {
				for _, c := range checks {
					icon := "✓"
					switch c.Status {
					case "fail":
						icon = "✗"
					case "warn":
						icon = "!"
					case "skip":
						icon = "-"
					}
					_, _ = fmt.Fprintf(os.Stdout, "  %s %-35s %s\n", icon, c.Name, c.Detail)
				}
				if hasFailure {
					_, _ = fmt.Fprintln(os.Stdout, "\nSome checks failed. Run with --format json for details.")
					return fmt.Errorf("auth doctor found failing checks")
				}
				_, _ = fmt.Fprintln(os.Stdout, "\nAll checks passed.")
				return nil
			}

			if err := printOutput(map[string]any{
				"status":    ternary(hasFailure, "fail", "ok"),
				"operation": "auth.doctor",
				"tenant_id": activeTenant,
				"provider":  stringOrNil(providerID),
				"checks":    checks,
			}); err != nil {
				return err
			}

			if hasFailure {
				return fmt.Errorf("auth doctor found failing checks")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&tenant, "tenant", "", "tenant id")
	cmd.Flags().StringVar(&profile, "profile", "default", "connection profile name for multiple accounts per provider")
	cmd.Flags().StringArrayVar(&extras, "extra", nil, "provider-specific key=value pairs for placeholder endpoints")
	return cmd
}

func checkConnectorEncryptionKeyPresent() doctorCheck {
	if strings.TrimSpace(os.Getenv("KIMBAP_CONNECTOR_ENCRYPTION_KEY")) == "" {
		return doctorCheck{Name: "connector encryption key set", Status: "fail", Detail: "KIMBAP_CONNECTOR_ENCRYPTION_KEY is not set"}
	}
	return doctorCheck{Name: "connector encryption key set", Status: "ok", Detail: "KIMBAP_CONNECTOR_ENCRYPTION_KEY is set"}
}

func checkAuthProviderConfig(providerID string) doctorCheck {
	if strings.TrimSpace(providerID) == "" {
		return doctorCheck{Name: "provider config exists", Status: "skip", Detail: "no provider specified"}
	}
	_, err := providers.GetProvider(providerID)
	if err != nil {
		return doctorCheck{Name: "provider config exists", Status: "fail", Detail: err.Error()}
	}
	return doctorCheck{Name: "provider config exists", Status: "ok", Detail: providerID}
}

func checkAuthTokenState(cfg *config.KimbapConfig, tenantID, providerID, profile string) (doctorCheck, doctorCheck) {
	if strings.TrimSpace(providerID) == "" {
		return doctorCheck{Name: "token expiry status", Status: "skip", Detail: "no provider specified"}, doctorCheck{Name: "refresh token availability", Status: "skip", Detail: "no provider specified"}
	}

	store, storeErr := openConnectorStore(cfg)
	if storeErr != nil {
		return doctorCheck{Name: "token expiry status", Status: "fail", Detail: storeErr.Error()}, doctorCheck{Name: "refresh token availability", Status: "fail", Detail: storeErr.Error()}
	}
	mgr := connectors.NewManager(store)
	state, statusErr := mgr.Status(contextBackground(), tenantID, connectorStoreName(providerID, profile))
	if statusErr != nil {
		if errors.Is(statusErr, connectors.ErrConnectorNotFound) {
			return doctorCheck{Name: "token expiry status", Status: "skip", Detail: "no connection found"}, doctorCheck{Name: "refresh token availability", Status: "skip", Detail: "no connection found"}
		}
		return doctorCheck{Name: "token expiry status", Status: "fail", Detail: statusErr.Error()}, doctorCheck{Name: "refresh token availability", Status: "fail", Detail: statusErr.Error()}
	}

	cs := statusFromSanitizedState(state)

	expiry := doctorCheck{Name: "token expiry status", Status: "ok", Detail: "token expiry not recorded"}
	if state.ExpiresAt != nil {
		if time.Now().After(*state.ExpiresAt) {
			expiry = doctorCheck{Name: "token expiry status", Status: "fail", Detail: fmt.Sprintf("token expired at %s", state.ExpiresAt.Format(time.RFC3339))}
		} else if time.Until(*state.ExpiresAt) <= 10*time.Minute {
			expiry = doctorCheck{Name: "token expiry status", Status: "warn", Detail: fmt.Sprintf("token expires soon at %s", state.ExpiresAt.Format(time.RFC3339))}
		} else {
			expiry = doctorCheck{Name: "token expiry status", Status: "ok", Detail: fmt.Sprintf("token valid until %s", state.ExpiresAt.Format(time.RFC3339))}
		}
	}

	refreshAvailability := doctorCheck{Name: "refresh health", Status: "ok", Detail: "no issues detected"}
	if strings.TrimSpace(state.LastRefreshError) != "" {
		refreshAvailability = doctorCheck{Name: "refresh health", Status: "fail", Detail: fmt.Sprintf("last refresh failed: %s", state.LastRefreshError)}
	} else if state.LastRefresh != nil {
		refreshAvailability = doctorCheck{Name: "refresh health", Status: "ok", Detail: fmt.Sprintf("last refresh at %s", state.LastRefresh.Format(time.RFC3339))}
	}
	if cs == connectors.StatusReconnectRequired || cs == connectors.StatusRevoked || cs == connectors.StatusRefreshFailed {
		refreshAvailability = doctorCheck{Name: "refresh health", Status: "fail", Detail: "connection requires reauthentication"}
	}
	return expiry, refreshAvailability
}

func checkProviderEndpointsReachable(providerID string, extras map[string]string) []doctorCheck {
	if strings.TrimSpace(providerID) == "" {
		return []doctorCheck{{Name: "provider endpoints reachable", Status: "skip", Detail: "no provider specified"}}
	}

	provider, err := providers.GetProvider(providerID)
	if err != nil {
		return []doctorCheck{{Name: "provider endpoints reachable", Status: "fail", Detail: err.Error()}}
	}
	if valErr := validateProviderExtraValues(provider, extras); valErr != nil {
		return []doctorCheck{{Name: "provider endpoints reachable", Status: "fail", Detail: valErr.Error()}}
	}

	provider = substituteProviderEndpoints(provider, extras)
	if hasUnresolvedPlaceholders(provider) {
		missing := listUnresolvedPlaceholders(provider)
		if len(missing) == 0 {
			return []doctorCheck{{Name: "provider endpoints reachable", Status: "fail", Detail: "provider endpoints contain unresolved placeholders; use --extra key=value"}}
		}
		return []doctorCheck{{Name: "provider endpoints reachable", Status: "fail", Detail: fmt.Sprintf("unresolved placeholders: %s (use --extra key=value)", strings.Join(missing, ", "))}}
	}
	if vErr := validateProviderEndpoints(provider); vErr != nil {
		return []doctorCheck{{Name: "provider endpoints reachable", Status: "fail", Detail: vErr.Error()}}
	}

	client := &http.Client{Timeout: 8 * time.Second}
	checks := make([]doctorCheck, 0, 4)
	endpoints := []struct {
		name string
		url  string
	}{
		{name: "auth endpoint reachable", url: provider.AuthEndpoint},
		{name: "token endpoint reachable", url: provider.TokenEndpoint},
		{name: "device endpoint reachable", url: provider.DeviceEndpoint},
		{name: "revocation endpoint reachable", url: provider.RevocationEndpoint},
	}
	for _, ep := range endpoints {
		if strings.TrimSpace(ep.url) == "" {
			checks = append(checks, doctorCheck{Name: ep.name, Status: "skip", Detail: "endpoint not configured"})
			continue
		}
		req, reqErr := http.NewRequestWithContext(contextBackground(), http.MethodHead, ep.url, nil)
		if reqErr != nil {
			checks = append(checks, doctorCheck{Name: ep.name, Status: "fail", Detail: reqErr.Error()})
			continue
		}
		res, callErr := client.Do(req)
		if callErr != nil {
			checks = append(checks, doctorCheck{Name: ep.name, Status: "fail", Detail: callErr.Error()})
			continue
		}
		_ = res.Body.Close()
		if res.StatusCode >= 200 && res.StatusCode < 500 {
			checks = append(checks, doctorCheck{Name: ep.name, Status: "ok", Detail: fmt.Sprintf("HTTP %d", res.StatusCode)})
		} else {
			checks = append(checks, doctorCheck{Name: ep.name, Status: "fail", Detail: fmt.Sprintf("HTTP %d", res.StatusCode)})
		}
	}
	return checks
}

func checkLoopbackPortBindable(port int) doctorCheck {
	addr := "127.0.0.1:0"
	if port > 0 {
		addr = fmt.Sprintf("127.0.0.1:%d", port)
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return doctorCheck{Name: "loopback port bindable", Status: "fail", Detail: err.Error()}
	}
	bound := ln.Addr().String()
	_ = ln.Close()
	return doctorCheck{Name: "loopback port bindable", Status: "ok", Detail: bound}
}

func checkBrowserLaunchFeasible(providerID string) doctorCheck {
	if strings.TrimSpace(providerID) == "" {
		return doctorCheck{Name: "browser launch feasible", Status: "skip", Detail: "no provider specified"}
	}
	provider, err := providers.GetProvider(providerID)
	if err != nil {
		return doctorCheck{Name: "browser launch feasible", Status: "skip", Detail: "provider unknown"}
	}
	browserRequired := provider.SupportsBrowserFlow() && !provider.SupportsDeviceFlow() && !provider.SupportsClientCredentials()
	env := detectFlowEnvironment()
	if env.IsSSH {
		if browserRequired {
			return doctorCheck{Name: "browser launch feasible", Status: "fail", Detail: "SSH session detected; provider requires browser flow"}
		}
		return doctorCheck{Name: "browser launch feasible", Status: "warn", Detail: "SSH session detected; browser flow may not work"}
	}
	if !env.HasTTY {
		if browserRequired {
			return doctorCheck{Name: "browser launch feasible", Status: "fail", Detail: "no TTY detected; provider requires browser flow"}
		}
		return doctorCheck{Name: "browser launch feasible", Status: "warn", Detail: "no TTY detected; browser flow may not work"}
	}
	if !env.HasDisplay {
		if browserRequired {
			return doctorCheck{Name: "browser launch feasible", Status: "fail", Detail: "no display detected; provider requires browser flow"}
		}
		return doctorCheck{Name: "browser launch feasible", Status: "warn", Detail: "no display detected; browser flow may not work"}
	}
	if !env.CanOpenBrowser {
		if !browserRequired {
			return doctorCheck{Name: "browser launch feasible", Status: "warn", Detail: "environment cannot open browser; non-browser OAuth flows are available"}
		}
		return doctorCheck{Name: "browser launch feasible", Status: "fail", Detail: "environment cannot open a browser"}
	}
	return doctorCheck{Name: "browser launch feasible", Status: "ok", Detail: "browser launch should work"}
}

func checkGrantedScopes(cfg *config.KimbapConfig, tenantID, providerID, profile string) doctorCheck {
	if strings.TrimSpace(providerID) == "" {
		return doctorCheck{Name: "scope coverage", Status: "skip", Detail: "no provider specified"}
	}

	provider, err := providers.GetProvider(providerID)
	if err != nil {
		return doctorCheck{Name: "scope coverage", Status: "skip", Detail: "provider not found"}
	}

	store, storeErr := openConnectorStore(cfg)
	if storeErr != nil {
		return doctorCheck{Name: "scope coverage", Status: "skip", Detail: "store unavailable"}
	}
	mgr := connectors.NewManager(store)
	state, statusErr := mgr.Status(contextBackground(), tenantID, connectorStoreName(providerID, profile))
	if statusErr != nil {
		if errors.Is(statusErr, connectors.ErrConnectorNotFound) {
			return doctorCheck{Name: "scope coverage", Status: "skip", Detail: "no connection found"}
		}
		return doctorCheck{Name: "scope coverage", Status: "fail", Detail: statusErr.Error()}
	}
	if state == nil {
		return doctorCheck{Name: "scope coverage", Status: "skip", Detail: "no connection found"}
	}

	if len(provider.DefaultScopes) == 0 {
		return doctorCheck{Name: "scope coverage", Status: "ok", Detail: "provider has no default scopes to check"}
	}

	grantedSet := map[string]struct{}{}
	for _, s := range state.Scopes {
		grantedSet[s] = struct{}{}
	}

	missing := make([]string, 0)
	for _, required := range provider.DefaultScopes {
		if _, ok := grantedSet[required]; !ok {
			missing = append(missing, required)
		}
	}

	if len(missing) > 0 {
		return doctorCheck{Name: "scope coverage", Status: "warn", Detail: fmt.Sprintf("missing default scopes: %s", strings.Join(missing, ", "))}
	}
	return doctorCheck{Name: "scope coverage", Status: "ok", Detail: fmt.Sprintf("all %d default scopes granted", len(provider.DefaultScopes))}
}
