package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/dunialabs/kimbap/internal/audit"
	"github.com/spf13/cobra"
)

func newAuditCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "View and export audit events",
	}

	cmd.AddCommand(newAuditTailCommand())
	cmd.AddCommand(newAuditExportCommand())

	return cmd
}

func newAuditTailCommand() *cobra.Command {
	var (
		agent   string
		service string
		limit   int
	)
	cmd := &cobra.Command{
		Use:   "tail [--agent <name>] [--service <name>] [--limit 20]",
		Short: "Tail recent audit events",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			if limit <= 0 {
				return fmt.Errorf("--limit must be greater than 0")
			}

			selected := make([]audit.AuditEvent, 0, limit)
			if err := forEachAuditEvent(cfg.Audit.Path, func(e audit.AuditEvent) error {
				if !auditEventMatches(e, agent, service) {
					return nil
				}
				if len(selected) < limit {
					selected = append(selected, e)
					return nil
				}
				copy(selected, selected[1:])
				selected[limit-1] = e
				return nil
			}); err != nil {
				return err
			}

			if outputAsJSON() {
				return printOutput(map[string]any{
					"path":    cfg.Audit.Path,
					"count":   len(selected),
					"filters": map[string]any{"agent": strings.TrimSpace(agent), "service": strings.TrimSpace(service), "limit": limit},
					"events":  selected,
				})
			}
			if len(selected) == 0 {
				fmt.Println("No audit events.")
				return nil
			}
			useColor := isColorStdout()
			fmt.Printf("%-34s %-14s %-20s %7s  %s\n", "ACTION", "AGENT", "STATUS", "DURATION", "TIMESTAMP")
			for _, e := range selected {
				statusStr := fmt.Sprintf("%-18s", string(e.Status))
				if useColor {
					switch e.Status {
					case audit.AuditStatusSuccess:
						statusStr = "[32m" + statusStr + "[0m"
					case audit.AuditStatusApprovalRequired:
						statusStr = "[33m" + statusStr + "[0m"
					default:
						statusStr = "[31m" + statusStr + "[0m"
					}
				}
				fmt.Printf("%-34s %-14s %s %4dms  %s\n",
					e.Service+"."+e.Action,
					e.AgentName,
					statusStr,
					e.DurationMS,
					e.Timestamp.Format("2006-01-02 15:04:05"),
				)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&agent, "agent", "", "filter by agent name")
	cmd.Flags().StringVar(&service, "service", "", "filter by service name")
	cmd.Flags().IntVar(&limit, "limit", 20, "max events to return")
	return cmd
}

func newAuditExportCommand() *cobra.Command {
	var (
		from    string
		to      string
		format  string
		agent   string
		service string
	)
	cmd := &cobra.Command{
		Use:   "export --from <date> --to <date> [--format jsonl|csv]",
		Short: "Export audit events",
		RunE: func(_ *cobra.Command, _ []string) error {
			if strings.TrimSpace(from) == "" {
				return fmt.Errorf("--from is required")
			}
			if strings.TrimSpace(to) == "" {
				return fmt.Errorf("--to is required")
			}

			fromTime, err := parseAuditTime(from)
			if err != nil {
				return fmt.Errorf("parse --from: %w", err)
			}
			toTime, err := parseAuditTimeTo(to)
			if err != nil {
				return fmt.Errorf("parse --to: %w", err)
			}
			if toTime.Before(fromTime) {
				return fmt.Errorf("--to must be after or equal to --from")
			}

			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}
			switch strings.ToLower(strings.TrimSpace(format)) {
			case "", "jsonl", "json":
				enc := json.NewEncoder(os.Stdout)
				if err := forEachAuditEvent(cfg.Audit.Path, func(e audit.AuditEvent) error {
					if e.Timestamp.Before(fromTime) || e.Timestamp.After(toTime) {
						return nil
					}
					if !auditEventMatches(e, agent, service) {
						return nil
					}
					return enc.Encode(e)
				}); err != nil {
					return err
				}
				return nil
			case "csv":
				writer := csv.NewWriter(os.Stdout)
				headers := []string{"id", "timestamp", "request_id", "trace_id", "tenant_id", "principal_id", "agent_name", "service", "action", "mode", "status", "policy_decision", "duration_ms", "error_code", "error_message"}
				if err := writer.Write(headers); err != nil {
					return err
				}
				if err := forEachAuditEvent(cfg.Audit.Path, func(e audit.AuditEvent) error {
					if e.Timestamp.Before(fromTime) || e.Timestamp.After(toTime) {
						return nil
					}
					if !auditEventMatches(e, agent, service) {
						return nil
					}
					errorCode := ""
					errorMessage := ""
					if e.Error != nil {
						errorCode = e.Error.Code
						errorMessage = e.Error.Message
					}
					row := []string{
						e.ID,
						e.Timestamp.Format(time.RFC3339Nano),
						e.RequestID,
						e.TraceID,
						e.TenantID,
						e.PrincipalID,
						e.AgentName,
						e.Service,
						e.Action,
						e.Mode,
						string(e.Status),
						e.PolicyDecision,
						fmt.Sprintf("%d", e.DurationMS),
						errorCode,
						errorMessage,
					}
					return writer.Write(row)
				}); err != nil {
					return err
				}
				writer.Flush()
				return writer.Error()
			default:
				return fmt.Errorf("unsupported format %q", format)
			}
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "from datetime (RFC3339 or YYYY-MM-DD)")
	cmd.Flags().StringVar(&to, "to", "", "to datetime (RFC3339 or YYYY-MM-DD)")
	cmd.Flags().StringVar(&format, "format", "jsonl", "export format: jsonl or csv")
	cmd.Flags().StringVar(&agent, "agent", "", "filter by agent name")
	cmd.Flags().StringVar(&service, "service", "", "filter by service name")
	return cmd
}

func forEachAuditEvent(path string, handle func(audit.AuditEvent) error) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("audit path is required")
	}
	if handle == nil {
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	for {
		var event audit.AuditEvent
		if err := decoder.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("parse audit event: %w", err)
		}
		if err := handle(event); err != nil {
			return err
		}
	}
	return nil
}

func auditEventMatches(event audit.AuditEvent, agent string, service string) bool {
	if a := strings.TrimSpace(agent); a != "" && !strings.EqualFold(event.AgentName, a) {
		return false
	}
	if s := strings.TrimSpace(service); s != "" && !strings.EqualFold(event.Service, s) {
		return false
	}
	return true
}

func parseAuditTime(raw string) (time.Time, error) {
	return parseAuditTimeWithEndOfDay(raw, false)
}

func parseAuditTimeTo(raw string) (time.Time, error) {
	return parseAuditTimeWithEndOfDay(raw, true)
}

func parseAuditTimeWithEndOfDay(raw string, endOfDay bool) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("time value is required")
	}

	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC(), nil
		}
	}

	if t, err := time.Parse("2006-01-02", raw); err == nil {
		if endOfDay {
			return t.UTC().Add(24*time.Hour - time.Nanosecond), nil
		}
		return t.UTC(), nil
	}

	return time.Time{}, fmt.Errorf("unsupported time format %q", raw)
}
