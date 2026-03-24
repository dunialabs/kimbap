package main

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/connectors"
	realFlows "github.com/dunialabs/kimbap-core/internal/connectors/flows"
	realProviders "github.com/dunialabs/kimbap-core/internal/connectors/providers"
	"github.com/dunialabs/kimbap-core/internal/security"
	"github.com/spf13/cobra"
)

func newAuthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage OAuth connections to external services",
	}

	cmd.AddCommand(newAuthConnectCommand())
	cmd.AddCommand(newAuthListCommand())
	cmd.AddCommand(newAuthStatusCommand())
	cmd.AddCommand(newAuthReconnectCommand())
	cmd.AddCommand(newAuthRevokeCommand())
	cmd.AddCommand(newAuthProvidersCommand())
	cmd.AddCommand(newAuthDoctorCommand())

	return cmd
}

func parseConnectionScope(raw string) connectors.ConnectionScope {
	switch connectors.ConnectionScope(strings.ToLower(strings.TrimSpace(raw))) {
	case connectors.ScopeWorkspace:
		return connectors.ScopeWorkspace
	case connectors.ScopeService:
		return connectors.ScopeService
	default:
		return connectors.ScopeUser
	}
}

func scopeValues(raw string, fallback []string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return append([]string(nil), fallback...)
	}
	parts := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\n' || r == '\t'
	})
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func statusFromSanitizedState(state *connectors.ConnectorState) connectors.ConnectionStatus {
	if state.RevokedAt != nil {
		return connectors.StatusRevoked
	}
	if strings.TrimSpace(state.LastRefreshError) != "" && state.LastRefresh != nil {
		return connectors.StatusRefreshFailed
	}
	return connectors.MapLegacyStatus(state.Status)
}

func refreshHealthFromConnectionStatus(status connectors.ConnectionStatus) string {
	switch status {
	case connectors.StatusConnected:
		return "healthy"
	case connectors.StatusDegraded, connectors.StatusRefreshFailed:
		return "degraded"
	case connectors.StatusExpired, connectors.StatusReconnectRequired:
		return "requires_reauth"
	default:
		return "unknown"
	}
}

func refreshStateFromConnectionStatus(status connectors.ConnectionStatus) string {
	switch status {
	case connectors.StatusConnected:
		return "ok"
	case connectors.StatusDegraded:
		return "stale"
	case connectors.StatusRefreshFailed:
		return "failed"
	case connectors.StatusReconnectRequired, connectors.StatusExpired:
		return "reauth_required"
	default:
		return "unknown"
	}
}

func remediationFromStatus(status connectors.ConnectionStatus) any {
	if !status.NeedsAttention() {
		return nil
	}
	switch status {
	case connectors.StatusExpired:
		return "Connection expired. Run: kimbap auth reconnect <provider>"
	case connectors.StatusReconnectRequired, connectors.StatusRevoked:
		return "Reauthorization required. Run: kimbap auth connect <provider>"
	case connectors.StatusRefreshFailed:
		return "Refresh failed. Check provider token endpoint and retry reconnect"
	case connectors.StatusDegraded:
		return "Connection near expiry. Refresh or reconnect soon"
	default:
		return "Inspect provider configuration and run: kimbap auth doctor <provider>"
	}
}

func scopeDelta(prev, next []string) map[string][]string {
	prevSet := map[string]struct{}{}
	nextSet := map[string]struct{}{}
	for _, v := range prev {
		prevSet[v] = struct{}{}
	}
	for _, v := range next {
		nextSet[v] = struct{}{}
	}
	added := make([]string, 0)
	removed := make([]string, 0)
	for v := range nextSet {
		if _, ok := prevSet[v]; !ok {
			added = append(added, v)
		}
	}
	for v := range prevSet {
		if _, ok := nextSet[v]; !ok {
			removed = append(removed, v)
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	return map[string][]string{"added": added, "removed": removed}
}

func stringOrNil(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func ternary[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}

func resolveClientID(cfg *config.KimbapConfig, providerID string) string {
	if cfg != nil {
		if v := os.Getenv("KIMBAP_" + strings.ToUpper(providerID) + "_CLIENT_ID"); v != "" {
			return v
		}
	}
	return os.Getenv("KIMBAP_OAUTH_CLIENT_ID")
}

func resolveClientSecret(cfg *config.KimbapConfig, providerID string) string {
	if cfg != nil {
		if v := os.Getenv("KIMBAP_" + strings.ToUpper(providerID) + "_CLIENT_SECRET"); v != "" {
			return v
		}
	}
	return os.Getenv("KIMBAP_OAUTH_CLIENT_SECRET")
}

func decryptStoredToken(encrypted string) string {
	if strings.TrimSpace(encrypted) == "" {
		return ""
	}
	key := strings.TrimSpace(os.Getenv("KIMBAP_CONNECTOR_ENCRYPTION_KEY"))
	if key == "" {
		return ""
	}
	plaintext, err := security.DecryptDataFromString(encrypted, key)
	if err != nil {
		return ""
	}
	return plaintext
}

func confirmRevocation(providerID string) (bool, error) {
	_, _ = fmt.Fprintf(os.Stdout, "This will disconnect %s. Proceed? [y/N] ", providerID)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, os.ErrClosed) {
		if errors.Is(err, os.ErrDeadlineExceeded) {
			return false, nil
		}
		if len(line) == 0 {
			return false, err
		}
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}

func deleteConnectorState(cfg *config.KimbapConfig, tenantID, providerID string) error {
	db, dialect, err := openConnectorDB(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := migrateConnectorTable(contextBackground(), db, dialect); err != nil {
		return err
	}

	q := `DELETE FROM connector_states WHERE tenant_id = ? AND name = ?`
	res, err := db.ExecContext(contextBackground(), bindQuery(q, dialect), tenantID, providerID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err == nil && affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func providerIsConfigured(p connectors.ProviderDefinition) bool {
	return strings.TrimSpace(p.AuthEndpoint) != "" && strings.TrimSpace(p.TokenEndpoint) != ""
}

func callRevocationEndpoint(endpoint, clientID, clientSecret, token string) error {
	form := url.Values{}
	form.Set("client_id", clientID)
	if clientSecret != "" {
		form.Set("client_secret", clientSecret)
	}
	if token != "" {
		form.Set("token", token)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	_ = res.Body.Close()
	if res.StatusCode >= 400 {
		return fmt.Errorf("revocation endpoint returned HTTP %d", res.StatusCode)
	}
	return nil
}

var providers = struct {
	GetProvider   func(string) (connectors.ProviderDefinition, error)
	ListProviders func() []connectors.ProviderDefinition
}{
	GetProvider:   realProviders.GetProvider,
	ListProviders: realProviders.ListProviders,
}

var flows = struct {
	SelectFlow func(string, connectors.ProviderDefinition, []string) (connectors.FlowType, error)
}{
	SelectFlow: func(raw string, provider connectors.ProviderDefinition, _ []string) (connectors.FlowType, error) {
		normalized := strings.ToLower(strings.TrimSpace(raw))
		requested := connectors.FlowType(normalized)
		if normalized == "" || normalized == "auto" {
			requested = ""
		}
		return (&realFlows.FlowSelector{}).SelectFlow(requested, provider)
	},
}
