package main

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dunialabs/kimbap/internal/audit"
	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/connectors"
	realFlows "github.com/dunialabs/kimbap/internal/connectors/flows"
	realProviders "github.com/dunialabs/kimbap/internal/connectors/providers"
	"github.com/dunialabs/kimbap/internal/security"
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

func parseConnectionScope(raw string) (connectors.ConnectionScope, bool) {
	normalized := connectors.ConnectionScope(strings.ToLower(strings.TrimSpace(raw)))
	switch normalized {
	case "", connectors.ScopeUser:
		return connectors.ScopeUser, true
	case connectors.ScopeWorkspace:
		return connectors.ScopeWorkspace, true
	case connectors.ScopeService:
		return connectors.ScopeService, true
	default:
		return "", false
	}
}

func resolveConnectionScope(raw string, provider connectors.ProviderDefinition) (connectors.ConnectionScope, error) {
	requested, ok := parseConnectionScope(raw)
	if !ok {
		return "", fmt.Errorf("invalid connection scope %q", strings.TrimSpace(raw))
	}
	if len(provider.ConnectionScopeModel) == 0 {
		return requested, nil
	}
	for _, supported := range provider.ConnectionScopeModel {
		if supported == requested {
			return requested, nil
		}
	}
	if requested == connectors.ScopeUser && len(provider.ConnectionScopeModel) == 1 {
		return connectors.ConnectionScope(provider.ConnectionScopeModel[0]), nil
	}
	allowed := make([]string, 0, len(provider.ConnectionScopeModel))
	for _, s := range provider.ConnectionScopeModel {
		allowed = append(allowed, string(s))
	}
	return "", fmt.Errorf("connection scope %q not supported by provider %q (supported: %s)", requested, provider.ID, strings.Join(allowed, ", "))
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

func parseExtras(raw []string) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]string, len(raw))
	for _, item := range raw {
		key, value, ok := strings.Cut(item, "=")
		if !ok || strings.TrimSpace(key) == "" {
			continue
		}
		out[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return out
}

func parseExtrasStrict(raw []string) (map[string]string, error) {
	for _, item := range raw {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			return nil, fmt.Errorf("invalid --extra %q: expected key=value", item)
		}
		if strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("invalid --extra %q: key must not be empty", item)
		}
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("invalid --extra %q: value must not be empty", item)
		}
	}
	return parseExtras(raw), nil
}

type placeholderKind int

const (
	placeholderHostLabel placeholderKind = iota
	placeholderHostValue
)

func validateProviderExtraValues(provider connectors.ProviderDefinition, extras map[string]string) error {
	if len(extras) == 0 {
		return nil
	}
	kinds := placeholderKindsForProvider(provider)
	for key, kind := range kinds {
		value, ok := extras[key]
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if value == "" {
			return fmt.Errorf("invalid --extra %s: value must not be empty", key)
		}
		switch kind {
		case placeholderHostLabel:
			if !isValidDNSLabel(value) {
				return fmt.Errorf("invalid --extra %s=%q: expected a DNS label (letters, digits, hyphen)", key, value)
			}
		case placeholderHostValue:
			if err := validateHostExtraValue(value); err != nil {
				return fmt.Errorf("invalid --extra %s=%q: %w", key, value, err)
			}
		}
	}
	return nil
}

func placeholderKindsForProvider(provider connectors.ProviderDefinition) map[string]placeholderKind {
	result := map[string]placeholderKind{}
	for _, endpoint := range []string{provider.AuthEndpoint, provider.TokenEndpoint, provider.DeviceEndpoint, provider.RevocationEndpoint, provider.UserInfoEndpoint} {
		host := endpointHostTemplate(endpoint)
		if host == "" {
			continue
		}
		for _, key := range placeholdersInString(host) {
			kind := placeholderHostLabel
			if host == "{"+key+"}" {
				kind = placeholderHostValue
			}
			prev, exists := result[key]
			if !exists || kind < prev {
				result[key] = kind
			}
		}
	}
	return result
}

func endpointHostTemplate(endpoint string) string {
	trimmed := strings.TrimSpace(endpoint)
	if trimmed == "" {
		return ""
	}
	i := strings.Index(trimmed, "://")
	if i < 0 {
		return ""
	}
	rest := trimmed[i+3:]
	if rest == "" {
		return ""
	}
	if j := strings.IndexAny(rest, "/?#"); j >= 0 {
		return rest[:j]
	}
	return rest
}

func placeholdersInString(input string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)
	remaining := input
	for {
		start := strings.IndexByte(remaining, '{')
		if start < 0 {
			break
		}
		endRel := strings.IndexByte(remaining[start:], '}')
		if endRel < 0 {
			break
		}
		key := remaining[start+1 : start+endRel]
		if key != "" {
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				out = append(out, key)
			}
		}
		remaining = remaining[start+endRel+1:]
	}
	return out
}

func hasPlaceholderInEndpoint(endpoint string) bool {
	return strings.Contains(strings.TrimSpace(endpoint), "{")
}

func validateHostExtraValue(value string) error {
	if strings.Contains(value, "://") {
		return errors.New("must be host only (scheme not allowed)")
	}
	u, err := url.Parse("https://" + value)
	if err != nil {
		return fmt.Errorf("invalid host: %w", err)
	}
	if u.User != nil {
		return errors.New("userinfo is not allowed")
	}
	if u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return errors.New("path, query, and fragment are not allowed")
	}
	hostname := strings.TrimSpace(u.Hostname())
	if hostname == "" {
		return errors.New("host is required")
	}
	if !isValidHostNameOrIP(hostname) {
		return errors.New("host must be a valid DNS name or IP address")
	}
	if port := u.Port(); port != "" {
		parsed, pErr := strconv.Atoi(port)
		if pErr != nil || parsed < 1 || parsed > 65535 {
			return errors.New("port must be in range 1-65535")
		}
	}
	return nil
}

func isValidHostNameOrIP(host string) bool {
	if ip := net.ParseIP(host); ip != nil {
		return true
	}
	if len(host) > 253 {
		return false
	}
	if strings.HasPrefix(host, ".") || strings.HasSuffix(host, ".") {
		return false
	}
	parts := strings.Split(host, ".")
	if len(parts) == 0 {
		return false
	}
	for _, part := range parts {
		if !isValidDNSLabel(part) {
			return false
		}
	}
	return true
}

func isValidDNSLabel(label string) bool {
	if len(label) == 0 || len(label) > 63 {
		return false
	}
	if label[0] == '-' || label[len(label)-1] == '-' {
		return false
	}
	for i := 0; i < len(label); i++ {
		ch := label[i]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' {
			continue
		}
		return false
	}
	return true
}

func substituteProviderEndpoints(provider connectors.ProviderDefinition, extras map[string]string) connectors.ProviderDefinition {
	if len(extras) == 0 {
		return provider
	}
	replace := func(s string) string {
		for k, v := range extras {
			s = strings.ReplaceAll(s, "{"+k+"}", v)
		}
		return s
	}
	provider.AuthEndpoint = replace(provider.AuthEndpoint)
	provider.TokenEndpoint = replace(provider.TokenEndpoint)
	provider.DeviceEndpoint = replace(provider.DeviceEndpoint)
	provider.RevocationEndpoint = replace(provider.RevocationEndpoint)
	provider.UserInfoEndpoint = replace(provider.UserInfoEndpoint)
	return provider
}

func hasUnresolvedPlaceholders(provider connectors.ProviderDefinition) bool {
	for _, ep := range []string{provider.AuthEndpoint, provider.TokenEndpoint, provider.DeviceEndpoint, provider.RevocationEndpoint, provider.UserInfoEndpoint} {
		if strings.Contains(ep, "{") {
			return true
		}
	}
	return false
}

func listUnresolvedPlaceholders(provider connectors.ProviderDefinition) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, ep := range []string{provider.AuthEndpoint, provider.TokenEndpoint, provider.DeviceEndpoint, provider.RevocationEndpoint, provider.UserInfoEndpoint} {
		for {
			start := strings.IndexByte(ep, '{')
			if start < 0 {
				break
			}
			end := strings.IndexByte(ep[start:], '}')
			if end < 0 {
				break
			}
			placeholder := ep[start+1 : start+end]
			if _, ok := seen[placeholder]; !ok {
				seen[placeholder] = struct{}{}
				out = append(out, placeholder)
			}
			ep = ep[start+end+1:]
		}
	}
	return out
}

func validateProviderEndpoints(provider connectors.ProviderDefinition) error {
	endpoints := map[string]string{
		"auth":       strings.TrimSpace(provider.AuthEndpoint),
		"token":      strings.TrimSpace(provider.TokenEndpoint),
		"device":     strings.TrimSpace(provider.DeviceEndpoint),
		"revocation": strings.TrimSpace(provider.RevocationEndpoint),
		"userinfo":   strings.TrimSpace(provider.UserInfoEndpoint),
	}
	for name, endpoint := range endpoints {
		if endpoint == "" {
			continue
		}
		u, err := url.Parse(endpoint)
		if err != nil {
			return fmt.Errorf("invalid %s endpoint: %w", name, err)
		}
		if !strings.EqualFold(u.Scheme, "https") {
			return fmt.Errorf("invalid %s endpoint: only https endpoints are allowed", name)
		}
		if strings.TrimSpace(u.Host) == "" {
			return fmt.Errorf("invalid %s endpoint: host is required", name)
		}
	}
	return nil
}

func connectorStoreName(providerID, profile string) string {
	p := strings.TrimSpace(profile)
	if p == "" || p == "default" {
		return providerID
	}
	return providerID + ":" + p
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

func decryptStoredToken(encrypted string) (string, error) {
	if strings.TrimSpace(encrypted) == "" {
		return "", nil
	}
	key := strings.TrimSpace(os.Getenv("KIMBAP_CONNECTOR_ENCRYPTION_KEY"))
	if key == "" {
		return "", errors.New("connector encryption key is not configured")
	}
	plaintext, err := security.DecryptDataFromString(encrypted, key)
	if err != nil {
		return "", fmt.Errorf("decrypt stored token: %w", err)
	}
	return plaintext, nil
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
	auth := strings.TrimSpace(p.AuthEndpoint)
	token := strings.TrimSpace(p.TokenEndpoint)
	return auth != "" && token != "" && !strings.Contains(auth, "{") && !strings.Contains(token, "{")
}

func callRevocationEndpoint(endpoint, clientID, clientSecret, token, authMethod string) error {
	form := url.Values{}
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
	if strings.EqualFold(strings.TrimSpace(authMethod), "basic") {
		req.SetBasicAuth(clientID, clientSecret)
	} else {
		if clientID != "" {
			form.Set("client_id", clientID)
		}
		if clientSecret != "" {
			form.Set("client_secret", clientSecret)
		}
		req.Body = io.NopCloser(strings.NewReader(form.Encode()))
		req.ContentLength = -1
	}
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
	SelectFlow: func(raw string, provider connectors.ProviderDefinition, hints []string) (connectors.FlowType, error) {
		normalized := normalizeFlowInput(raw)
		requested := connectors.FlowType(normalized)
		if normalized == "" || normalized == "auto" {
			requested = ""
		}

		if requested == "" {
			for _, hint := range hints {
				h := strings.ToLower(strings.TrimSpace(hint))
				if h == "browser=none" {
					if provider.SupportsDeviceFlow() {
						return connectors.FlowDevice, nil
					}
					if provider.SupportsClientCredentials() {
						return connectors.FlowClientCredentials, nil
					}
					return "", fmt.Errorf("--browser=none specified but provider %q only supports browser flow", provider.ID)
				}
			}
		}

		return (&realFlows.FlowSelector{}).SelectFlow(requested, provider)
	},
}

func normalizeFlowInput(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch normalized {
	case "client-credentials", "client_credentials":
		return string(connectors.FlowClientCredentials)
	default:
		return normalized
	}
}

func initAuthAuditEmitter(cfg *config.KimbapConfig) *connectors.AuditEmitter {
	auditPath := ""
	if cfg != nil {
		auditPath = strings.TrimSpace(cfg.Audit.Path)
	}
	if auditPath == "" {
		return nil
	}
	writer, err := audit.NewJSONLWriter(auditPath)
	if err != nil {
		return nil
	}
	return &connectors.AuditEmitter{Writer: writer}
}

func closeAuditEmitter(emitter *connectors.AuditEmitter) {
	if emitter == nil || emitter.Writer == nil {
		return
	}
	if closer, ok := emitter.Writer.(interface{ Close() error }); ok {
		_ = closer.Close()
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	hours := int(d.Hours())
	if hours < 24 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dd %dh", hours/24, hours%24)
}
