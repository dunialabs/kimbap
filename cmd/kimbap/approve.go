package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"time"

	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/store"
	"github.com/dunialabs/kimbap/internal/webhooks"
	"github.com/spf13/cobra"
)

func newApproveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "approve",
		Short: "Manage approval requests",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return runApproveAccept(args[0])
		},
	}

	cmd.AddCommand(newApproveListCommand())
	cmd.AddCommand(newApproveAcceptCommand())
	cmd.AddCommand(newApproveDenyCommand())

	return cmd
}

func newApproveListCommand() *cobra.Command {
	var (
		tenant string
		status string
	)
	cmd := &cobra.Command{
		Use:   "list [--tenant <id>] [--status pending]",
		Short: "List approval requests",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			err = withRuntimeStore(cfg, func(st *store.SQLStore) error {
				s := approvalStatus(status)
				if s == "" || s == "pending" {
					dispatcher := webhooks.NewDispatcher()
					configureWebhookDispatcherFromStore(contextBackground(), dispatcher, st)
					if _, expErr := expirePendingApprovalsWithSideEffects(contextBackground(), st, approvalTenant(tenant), func(approval store.ApprovalRecord) {
						dispatcher.EmitForTenant(approval.TenantID, webhooks.EventApprovalExpired, map[string]any{
							"approval_id": approval.ID,
							"tenant_id":   approval.TenantID,
							"request_id":  approval.RequestID,
							"agent_name":  approval.AgentName,
							"service":     approval.Service,
							"action":      approval.Action,
							"status":      "expired",
						})
					}); expErr != nil {
						_, _ = fmt.Fprintf(os.Stderr, "warning: approval expiry sweep failed: %v\n", expErr)
					}
				}
				items, err := st.ListApprovals(contextBackground(), approvalTenant(tenant), s)
				if err != nil {
					return err
				}
				if outputAsJSON() {
					return printOutput(items)
				}
				if len(items) == 0 {
					fmt.Println("No approval requests found.")
					return nil
				}
				fmt.Printf("%-36s %-16s %-30s %-12s %-12s %s\n", "ID", "AGENT", "ACTION", "STATUS", "EXPIRES", "CREATED")
				useColor := isColorStdout()
				for _, item := range items {
					statusCol := fmt.Sprintf("%-12s", item.Status)
					if useColor {
						switch item.Status {
						case "approved":
							statusCol = "\x1b[32m" + statusCol + "\x1b[0m"
						case "pending":
							statusCol = "\x1b[33m" + statusCol + "\x1b[0m"
						case "denied":
							statusCol = "\x1b[31m" + statusCol + "\x1b[0m"
						case "expired":
							statusCol = "\x1b[2m" + statusCol + "\x1b[0m"
						}
					}
					expiresStr := fmt.Sprintf("%-12s", approvalTimeRemaining(item.ExpiresAt))
					if useColor {
						remaining := time.Until(item.ExpiresAt)
						if remaining <= 0 {
							expiresStr = "\x1b[31m" + expiresStr + "\x1b[0m"
						} else if remaining < 10*time.Minute {
							expiresStr = "\x1b[33m" + expiresStr + "\x1b[0m"
						}
					}
					fmt.Printf("%-36s %-16s %-30s %s %s %s\n",
						item.ID,
						item.AgentName,
						item.Service+"."+item.Action,
						statusCol,
						expiresStr,
						item.CreatedAt.Format("2006-01-02 15:04"),
					)
				}
				return nil
			})
			if err != nil {
				if isRuntimeStoreUnavailable(err) {
					if outputAsJSON() {
						_ = printOutput(map[string]any{
							"would_execute": true,
							"operation":     "approve.list",
							"tenant_id":     approvalTenant(tenant),
							"status":        approvalStatus(status),
							"note":          unavailableMessage(componentApprovalStore, err),
						})
					}
					return unavailableError(componentApprovalStore, err)
				}
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&tenant, "tenant", "", "tenant id")
	cmd.Flags().StringVar(&status, "status", "pending", "approval status filter")
	return cmd
}

func newApproveAcceptCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "accept <request-id>",
		Aliases: []string{"approve"},
		Short:   "Approve a pending request",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runApproveAccept(args[0])
		},
	}
	return cmd
}

func newApproveDenyCommand() *cobra.Command {
	var reason string
	cmd := &cobra.Command{
		Use:   "deny <request-id> --reason <text>",
		Short: "Deny a pending request with reason",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if strings.TrimSpace(reason) == "" {
				return fmt.Errorf("--reason is required\nRun: kimbap approve deny %s --reason \"<why>\"", args[0])
			}

			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			err = withRuntimeStore(cfg, func(st *store.SQLStore) error {
				if err := st.UpdateApprovalStatus(contextBackground(), args[0], "denied", "cli", reason); err != nil {
					if errors.Is(err, store.ErrApprovalExpired) {
						_, _ = st.ExpireApproval(contextBackground(), args[0])
					}
					return fmt.Errorf("deny failed: %w", err)
				}
				if outputAsJSON() {
					return printOutput(map[string]any{
						"request_id":  args[0],
						"status":      "denied",
						"resolved_by": "cli",
						"reason":      reason,
					})
				}
				return printOutput(fmt.Sprintf(successCheck()+" %s denied", args[0]))
			})
			if err != nil {
				if isRuntimeStoreUnavailable(err) {
					if outputAsJSON() {
						_ = printOutput(map[string]any{
							"would_execute": true,
							"operation":     "approve.deny",
							"request_id":    args[0],
							"denied":        true,
							"resolved_by":   "cli",
							"reason":        reason,
							"note":          unavailableMessage(componentApprovalStore, err),
						})
					}
					return unavailableError(componentApprovalStore, err)
				}
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&reason, "reason", "", "deny reason")
	return cmd
}

func runApproveAccept(requestID string) error {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return fmt.Errorf("request-id is required")
	}

	cfg, err := loadAppConfig()
	if err != nil {
		return err
	}
	err = withRuntimeStore(cfg, func(st *store.SQLStore) error {
		// Pre-read approval record for best-effort retry hint. Non-fatal if it fails.
		rec, lookupErr := st.GetApproval(contextBackground(), requestID)

		if err := st.UpdateApprovalStatus(contextBackground(), requestID, "approved", "cli", ""); err != nil {
			if errors.Is(err, store.ErrApprovalExpired) {
				_, _ = st.ExpireApproval(contextBackground(), requestID)
			}
			return fmt.Errorf("approve failed: %w", err)
		}

		if outputAsJSON() {
			return printOutput(map[string]any{
				"request_id":  requestID,
				"status":      "approved",
				"resolved_by": "cli",
			})
		}
		_, _ = fmt.Fprintf(os.Stdout, successCheck()+" %s approved\n", requestID)
		if lookupErr == nil && rec != nil && rec.Service != "" && rec.Action != "" {
			_, _ = fmt.Fprintf(os.Stdout, "Retry: kimbap call %s.%s\n", rec.Service, rec.Action)
		}
		return nil
	})
	if err != nil {
		if isRuntimeStoreUnavailable(err) {
			if outputAsJSON() {
				_ = printOutput(map[string]any{
					"would_execute": true,
					"operation":     "approve.accept",
					"request_id":    requestID,
					"approved":      true,
					"resolved_by":   "cli",
					"note":          unavailableMessage(componentApprovalStore, err),
				})
			}
			return unavailableError(componentApprovalStore, err)
		}
		return err
	}

	return nil
}

func approvalTenant(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return defaultTenantID()
	}
	return trimmed
}

func approvalTimeRemaining(expires time.Time) string {
	remaining := time.Until(expires)
	if remaining <= 0 {
		return "expired"
	}
	if remaining < time.Hour {
		m := int(remaining.Minutes())
		s := int(remaining.Seconds()) % 60
		if m == 0 {
			return fmt.Sprintf("%ds", s)
		}
		return fmt.Sprintf("%dm%ds", m, s)
	}
	h := int(remaining.Hours())
	m := int(remaining.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", h, m)
}

func approvalStatus(raw string) string {
	return strings.TrimSpace(raw)
}

func openRuntimeStore(cfg *config.KimbapConfig) (*store.SQLStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	driver := strings.ToLower(strings.TrimSpace(cfg.Database.Driver))
	dsn := strings.TrimSpace(cfg.Database.DSN)

	var (
		st  *store.SQLStore
		err error
	)

	switch driver {
	case "", "sqlite", "sqlite3":
		if dsn == "" {
			dsn = filepath.Join(cfg.DataDir, "kimbap.db")
		}
		if dir := sqliteDSNDirectory(dsn); dir != "" {
			if err := os.MkdirAll(dir, 0o700); err != nil {
				return nil, fmt.Errorf("create database directory: %w", err)
			}
		}
		st, err = store.OpenSQLiteStore(dsn)
	case "postgres", "postgresql", "pgx":
		if dsn == "" {
			return nil, fmt.Errorf("database dsn is required for postgres")
		}
		st, err = store.OpenPostgresStore(dsn)
	default:
		return nil, fmt.Errorf("unsupported database driver %q", cfg.Database.Driver)
	}
	if err != nil {
		return nil, err
	}
	if err := st.Migrate(contextBackground()); err != nil {
		_ = st.Close()
		return nil, err
	}
	return st, nil
}

func sqliteDSNDirectory(dsn string) string {
	trimmed := strings.TrimSpace(dsn)
	if trimmed == "" || trimmed == ":memory:" {
		return ""
	}

	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "file:") {
		pathPart := strings.TrimPrefix(trimmed, "file:")
		if idx := strings.IndexByte(pathPart, '?'); idx >= 0 {
			pathPart = pathPart[:idx]
		}
		if pathPart == "" || pathPart == ":memory:" {
			return ""
		}
		if strings.HasPrefix(pathPart, "//") {
			hostPath := strings.TrimPrefix(pathPart, "//")
			slashIdx := strings.IndexByte(hostPath, '/')
			if slashIdx < 0 {
				return ""
			}
			pathPart = hostPath[slashIdx:]
		}
		return filepath.Dir(pathPart)
	}

	return filepath.Dir(trimmed)
}
