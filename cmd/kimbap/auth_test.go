package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/dunialabs/kimbap/internal/audit"
	"github.com/dunialabs/kimbap/internal/connectors"
	realFlows "github.com/dunialabs/kimbap/internal/connectors/flows"
	"github.com/dunialabs/kimbap/internal/connectors/flows/browser"
	"github.com/dunialabs/kimbap/internal/security"
)

type testAuditWriter struct {
	events []audit.AuditEvent
}

func (w *testAuditWriter) Write(_ context.Context, event audit.AuditEvent) error {
	w.events = append(w.events, event)
	return nil
}

func (w *testAuditWriter) Close() error { return nil }

func TestNormalizeFlowInput(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "client-credentials", want: string(connectors.FlowClientCredentials)},
		{in: "client_credentials", want: string(connectors.FlowClientCredentials)},
		{in: "browser", want: "browser"},
		{in: "auto", want: "auto"},
	}

	for _, tc := range tests {
		if got := normalizeFlowInput(tc.in); got != tc.want {
			t.Fatalf("normalizeFlowInput(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestSelectFlowBrowserNonePrefersDevice(t *testing.T) {
	provider := connectors.ProviderDefinition{
		ID:             "test-provider",
		SupportedFlows: []connectors.FlowType{connectors.FlowBrowser, connectors.FlowDevice},
	}

	selected, err := flows.SelectFlow("auto", provider, []string{"browser=none"})
	if err != nil {
		t.Fatalf("SelectFlow returned error: %v", err)
	}
	if selected != connectors.FlowDevice {
		t.Fatalf("SelectFlow returned %q, want %q", selected, connectors.FlowDevice)
	}
}

func TestSelectFlowSupportsClientCredentialsAlias(t *testing.T) {
	provider := connectors.ProviderDefinition{
		ID:             "machine-provider",
		SupportedFlows: []connectors.FlowType{connectors.FlowClientCredentials},
	}

	selected, err := flows.SelectFlow("client-credentials", provider, nil)
	if err != nil {
		t.Fatalf("SelectFlow returned error: %v", err)
	}
	if selected != connectors.FlowClientCredentials {
		t.Fatalf("SelectFlow returned %q, want %q", selected, connectors.FlowClientCredentials)
	}
}

func TestAuthListAndStatusItemContractConsistency(t *testing.T) {
	now := time.Now().UTC()
	state := connectors.ConnectorState{
		Name:               "github",
		Provider:           "github",
		ConnectionScope:    connectors.ScopeWorkspace,
		ConnectedPrincipal: "jane@example.com",
		Scopes:             []string{"repo", "read:org"},
		Status:             connectors.StatusHealthy,
		LastRefresh:        &now,
		LastUsedAt:         &now,
	}

	listItem := authListItem(state)
	statusItem := authStatusItem(state)

	sharedKeys := []string{
		"provider", "connection_id", "connection_scope", "scope_level",
		"connected_principal", "status_detail", "status",
		"refresh_state", "refresh_health", "last_refresh_result",
		"revocation_state", "last_used_at", "expires_at",
	}
	for _, key := range sharedKeys {
		if _, ok := listItem[key]; !ok {
			t.Fatalf("list item missing key %q", key)
		}
		if _, ok := statusItem[key]; !ok {
			t.Fatalf("status item missing key %q", key)
		}
	}
}

func TestAuthItemHelpersRefreshAndRevocationStates(t *testing.T) {
	now := time.Now().UTC()
	revokedAt := now.Add(-time.Minute)

	failed := connectors.ConnectorState{LastRefreshError: "token invalid", LastRefresh: &now}
	if got := authLastRefreshResult(failed); got != "failed" {
		t.Fatalf("authLastRefreshResult failed state = %q, want failed", got)
	}

	success := connectors.ConnectorState{LastRefresh: &now}
	if got := authLastRefreshResult(success); got != "success" {
		t.Fatalf("authLastRefreshResult success state = %q, want success", got)
	}

	unknown := connectors.ConnectorState{}
	if got := authLastRefreshResult(unknown); got != "unknown" {
		t.Fatalf("authLastRefreshResult unknown state = %q, want unknown", got)
	}

	revoked := connectors.ConnectorState{RevokedAt: &revokedAt}
	if got := authRevocationState(revoked); got != "revoked" {
		t.Fatalf("authRevocationState revoked state = %q, want revoked", got)
	}

	active := connectors.ConnectorState{}
	if got := authRevocationState(active); got != "active" {
		t.Fatalf("authRevocationState active state = %q, want active", got)
	}
}

func TestAuthSingleStatusPayloadContract(t *testing.T) {
	now := time.Now().UTC()
	state := connectors.ConnectorState{
		Name:               "github",
		Provider:           "github",
		ConnectionScope:    connectors.ScopeUser,
		ConnectedPrincipal: "jane@example.com",
		Scopes:             []string{"repo"},
		Status:             connectors.StatusHealthy,
		LastRefresh:        &now,
		LastUsedAt:         &now,
	}

	payload := authSingleStatusPayload("tenant-a", state)
	if payload["status"] != "ok" {
		t.Fatalf("payload status = %v, want ok", payload["status"])
	}
	for _, key := range []string{"operation", "tenant_id", "provider", "connection_id", "connection_scope", "scope_level", "refresh_state", "refresh_health", "status_detail", "connection_status", "expires_at"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("single status payload missing key %q", key)
		}
	}
}

func TestSelectFlowBrowserNoneErrorsForBrowserOnlyProvider(t *testing.T) {
	provider := connectors.ProviderDefinition{
		ID:             "browser-only",
		SupportedFlows: []connectors.FlowType{connectors.FlowBrowser},
	}

	_, err := flows.SelectFlow("auto", provider, []string{"browser=none"})
	if err == nil {
		t.Fatal("expected error for browser=none with browser-only provider, got nil")
	}
}

func TestProviderIsConfiguredRejectsPlaceholderEndpoints(t *testing.T) {
	provider := connectors.ProviderDefinition{
		AuthEndpoint:  "https://{subdomain}.example.com/oauth/authorize",
		TokenEndpoint: "https://{subdomain}.example.com/oauth/token",
	}
	if providerIsConfigured(provider) {
		t.Fatal("expected provider with unresolved placeholders to be unconfigured")
	}
}

func TestHasUnresolvedPlaceholdersIncludesUserInfoAndRevocation(t *testing.T) {
	provider := connectors.ProviderDefinition{
		AuthEndpoint:       "https://auth.example.com/oauth/authorize",
		TokenEndpoint:      "https://auth.example.com/oauth/token",
		RevocationEndpoint: "https://{subdomain}.example.com/oauth/revoke",
	}
	if !hasUnresolvedPlaceholders(provider) {
		t.Fatal("expected unresolved revocation placeholder to be detected")
	}

	provider.RevocationEndpoint = "https://api.example.com/oauth/revoke"
	provider.UserInfoEndpoint = "https://{subdomain}.example.com/me"
	if !hasUnresolvedPlaceholders(provider) {
		t.Fatal("expected unresolved userinfo placeholder to be detected")
	}
}

func TestIsBrowserEnvFailureUsesSentinelError(t *testing.T) {
	err := fmt.Errorf("wrapped: %w", browser.ErrLoopbackListener)
	if !isBrowserEnvFailure(err) {
		t.Fatal("expected loopback sentinel error to be treated as environment failure")
	}
}

func TestResolveConnectionScopeRejectsInvalidValue(t *testing.T) {
	provider := connectors.ProviderDefinition{ID: "p1"}
	_, err := resolveConnectionScope("invalid-scope", provider)
	if err == nil {
		t.Fatal("expected invalid connection scope to return error")
	}
}

func TestResolveConnectionScopeRejectsUnsupportedProviderScope(t *testing.T) {
	provider := connectors.ProviderDefinition{
		ID:                   "p2",
		ConnectionScopeModel: []connectors.ConnectionScope{connectors.ScopeUser},
	}
	_, err := resolveConnectionScope("workspace", provider)
	if err == nil {
		t.Fatal("expected unsupported provider scope to return error")
	}
}

func TestResolveConnectionScopeAutoSelectsFromSingleScopeModel(t *testing.T) {
	provider := connectors.ProviderDefinition{
		ID:                   "slack",
		ConnectionScopeModel: []connectors.ConnectionScope{connectors.ScopeWorkspace},
	}
	scope, err := resolveConnectionScope(string(connectors.ScopeUser), provider)
	if err != nil {
		t.Fatalf("expected auto-select to succeed, got error: %v", err)
	}
	if scope != connectors.ScopeWorkspace {
		t.Fatalf("expected auto-selected scope %q, got %q", connectors.ScopeWorkspace, scope)
	}
}

func TestValidateProviderExtraValuesRejectsHostInjection(t *testing.T) {
	provider := connectors.ProviderDefinition{
		ID:            "canvas",
		AuthEndpoint:  "https://{domain}/login/oauth2/auth",
		TokenEndpoint: "https://{domain}/login/oauth2/token",
	}
	err := validateProviderExtraValues(provider, map[string]string{"domain": "school.example.com/evil"})
	if err == nil {
		t.Fatal("expected invalid domain extra to fail")
	}
}

func TestValidateProviderExtraValuesAcceptsValidHostAndLabel(t *testing.T) {
	canvas := connectors.ProviderDefinition{
		ID:            "canvas",
		AuthEndpoint:  "https://{domain}/login/oauth2/auth",
		TokenEndpoint: "https://{domain}/login/oauth2/token",
	}
	if err := validateProviderExtraValues(canvas, map[string]string{"domain": "school.example.com:8443"}); err != nil {
		t.Fatalf("expected valid domain extra, got error: %v", err)
	}

	zendesk := connectors.ProviderDefinition{
		ID:            "zendesk",
		AuthEndpoint:  "https://{subdomain}.zendesk.com/oauth/authorizations/new",
		TokenEndpoint: "https://{subdomain}.zendesk.com/oauth/tokens",
	}
	if err := validateProviderExtraValues(zendesk, map[string]string{"subdomain": "acme-ops"}); err != nil {
		t.Fatalf("expected valid subdomain extra, got error: %v", err)
	}
}

func TestCheckBrowserLaunchFeasibleProviderAware(t *testing.T) {
	origProviders := providers
	origDetect := detectFlowEnvironment
	t.Cleanup(func() {
		providers = origProviders
		detectFlowEnvironment = origDetect
	})

	providers.GetProvider = func(id string) (connectors.ProviderDefinition, error) {
		switch id {
		case "browser-only":
			return connectors.ProviderDefinition{ID: id, SupportedFlows: []connectors.FlowType{connectors.FlowBrowser}}, nil
		case "browser-device":
			return connectors.ProviderDefinition{ID: id, SupportedFlows: []connectors.FlowType{connectors.FlowBrowser, connectors.FlowDevice}}, nil
		default:
			return connectors.ProviderDefinition{}, fmt.Errorf("unknown provider: %s", id)
		}
	}
	detectFlowEnvironment = func() realFlows.EnvironmentInfo {
		return realFlows.EnvironmentInfo{IsSSH: false, HasTTY: true, HasDisplay: true, CanOpenBrowser: false}
	}

	browserOnly := checkBrowserLaunchFeasible("browser-only")
	if browserOnly.Status != "fail" {
		t.Fatalf("expected browser-only provider to fail when browser unavailable, got %s", browserOnly.Status)
	}

	browserDevice := checkBrowserLaunchFeasible("browser-device")
	if browserDevice.Status != "warn" {
		t.Fatalf("expected browser+device provider to warn when browser unavailable, got %s", browserDevice.Status)
	}

	detectFlowEnvironment = func() realFlows.EnvironmentInfo {
		return realFlows.EnvironmentInfo{IsSSH: false, HasTTY: true, HasDisplay: false, CanOpenBrowser: false}
	}
	browserOnlyNoDisplay := checkBrowserLaunchFeasible("browser-only")
	if browserOnlyNoDisplay.Status != "fail" {
		t.Fatalf("expected browser-only provider to fail with no display, got %s", browserOnlyNoDisplay.Status)
	}
}

func TestCheckBrowserLaunchFeasibleGenericWarnsWithoutProvider(t *testing.T) {
	origDetect := detectFlowEnvironment
	t.Cleanup(func() {
		detectFlowEnvironment = origDetect
	})

	detectFlowEnvironment = func() realFlows.EnvironmentInfo {
		return realFlows.EnvironmentInfo{IsSSH: true, HasTTY: true, HasDisplay: true, CanOpenBrowser: true}
	}

	check := checkBrowserLaunchFeasible("")
	if check.Status != "warn" {
		t.Fatalf("expected generic browser check to warn under SSH, got %s", check.Status)
	}
}

func TestPrepareRevocationProviderRejectsInvalidExtra(t *testing.T) {
	origProviders := providers
	t.Cleanup(func() {
		providers = origProviders
	})

	providers.GetProvider = func(id string) (connectors.ProviderDefinition, error) {
		if id != "canvas" {
			return connectors.ProviderDefinition{}, fmt.Errorf("unknown provider: %s", id)
		}
		return connectors.ProviderDefinition{
			ID:            "canvas",
			AuthEndpoint:  "https://{domain}/login/oauth2/auth",
			TokenEndpoint: "https://{domain}/login/oauth2/token",
		}, nil
	}

	_, known, _, err := prepareRevocationProvider("canvas", map[string]string{"domain": "school.example.com/evil"})
	if err == nil {
		t.Fatal("expected prepareRevocationProvider to fail for invalid domain extra")
	}
	if !known {
		t.Fatal("expected known provider result")
	}
}

func TestParseExtrasStrictRejectsMalformedEntries(t *testing.T) {
	if _, err := parseExtrasStrict([]string{"domain"}); err == nil {
		t.Fatal("expected parseExtrasStrict to reject entries without equals")
	}
	if _, err := parseExtrasStrict([]string{"=value"}); err == nil {
		t.Fatal("expected parseExtrasStrict to reject empty key")
	}
	if _, err := parseExtrasStrict([]string{"domain="}); err == nil {
		t.Fatal("expected parseExtrasStrict to reject empty value")
	}
}

func TestPrepareRevocationProviderDetectsUnresolvedRevocationPlaceholder(t *testing.T) {
	origProviders := providers
	t.Cleanup(func() {
		providers = origProviders
	})

	providers.GetProvider = func(id string) (connectors.ProviderDefinition, error) {
		if id != "p" {
			return connectors.ProviderDefinition{}, fmt.Errorf("unknown provider: %s", id)
		}
		return connectors.ProviderDefinition{
			ID:                 "p",
			AuthEndpoint:       "https://example.com/oauth/authorize",
			TokenEndpoint:      "https://example.com/oauth/token",
			RevocationEndpoint: "https://{domain}/oauth/revoke",
		}, nil
	}

	_, known, result, err := prepareRevocationProvider("p", nil)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if !known {
		t.Fatal("expected provider metadata known")
	}
	if result != "not_supported: unresolved endpoint placeholders" {
		t.Fatalf("unexpected revocation preparation result: %s", result)
	}
}

func TestEmitRevokePrepareErrorAuditWritesEvent(t *testing.T) {
	writer := &testAuditWriter{}
	emitter := &connectors.AuditEmitter{Writer: writer}
	err := fmt.Errorf("invalid --extra values")

	returned := emitRevokePrepareErrorAudit(emitter, "github", "tenant-a", err)
	if returned == nil || returned.Error() != err.Error() {
		t.Fatalf("expected same error to be returned, got %v", returned)
	}
	if len(writer.events) != 1 {
		t.Fatalf("expected one audit event, got %d", len(writer.events))
	}
	if writer.events[0].Action != "auth.revoke.completed" {
		t.Fatalf("expected revoke completed action, got %s", writer.events[0].Action)
	}
}

func TestDecryptStoredTokenReturnsErrorWhenKeyMissing(t *testing.T) {
	t.Setenv("KIMBAP_CONNECTOR_ENCRYPTION_KEY", "")

	_, err := decryptStoredToken(`{"data":"x"}`)
	if err == nil {
		t.Fatal("expected missing encryption key to return error")
	}
	if !strings.Contains(err.Error(), "connector encryption key is not configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecryptStoredTokenReturnsErrorOnDecryptFailure(t *testing.T) {
	t.Setenv("KIMBAP_CONNECTOR_ENCRYPTION_KEY", "correct-key")
	encrypted, err := security.EncryptData("refresh-token", "different-key")
	if err != nil {
		t.Fatalf("encrypt token: %v", err)
	}

	_, err = decryptStoredToken(encrypted)
	if err == nil {
		t.Fatal("expected decrypt failure to return error")
	}
	if !strings.Contains(err.Error(), "decrypt stored token") {
		t.Fatalf("unexpected error: %v", err)
	}
}
