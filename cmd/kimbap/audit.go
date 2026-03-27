package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
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

			events, err := readAuditEvents(cfg.Audit.Path)
			if err != nil {
				return err
			}

			selected := make([]audit.AuditEvent, 0, limit)
			for i := len(events) - 1; i >= 0 && len(selected) < limit; i-- {
				e := events[i]
				if !auditEventMatches(e, agent, service) {
					continue
				}
				selected = append(selected, e)
			}

			for i, j := 0, len(selected)-1; i < j; i, j = i+1, j-1 {
				selected[i], selected[j] = selected[j], selected[i]
			}

			return printOutput(map[string]any{
				"path":    cfg.Audit.Path,
				"count":   len(selected),
				"filters": map[string]any{"agent": strings.TrimSpace(agent), "service": strings.TrimSpace(service), "limit": limit},
				"events":  selected,
			})
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
			events, err := readAuditEvents(cfg.Audit.Path)
			if err != nil {
				return err
			}

			filtered := make([]audit.AuditEvent, 0, len(events))
			for _, e := range events {
				if e.Timestamp.Before(fromTime) || e.Timestamp.After(toTime) {
					continue
				}
				if !auditEventMatches(e, agent, service) {
					continue
				}
				filtered = append(filtered, e)
			}

			switch strings.ToLower(strings.TrimSpace(format)) {
			case "", "jsonl", "json":
				enc := json.NewEncoder(os.Stdout)
				for i := range filtered {
					if err := enc.Encode(filtered[i]); err != nil {
						return err
					}
				}
				return nil
			case "csv":
				writer := csv.NewWriter(os.Stdout)
				headers := []string{"id", "timestamp", "request_id", "trace_id", "tenant_id", "principal_id", "agent_name", "service", "action", "mode", "status", "policy_decision", "duration_ms", "error_code", "error_message"}
				if err := writer.Write(headers); err != nil {
					return err
				}
				for i := range filtered {
					e := filtered[i]
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
					if err := writer.Write(row); err != nil {
						return err
					}
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

func readAuditEvents(path string) ([]audit.AuditEvent, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("audit path is required")
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []audit.AuditEvent{}, nil
		}
		return nil, err
	}
	defer file.Close()

	out := make([]audit.AuditEvent, 0)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4<<20), 4<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event audit.AuditEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, fmt.Errorf("parse audit line: %w", err)
		}
		out = append(out, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return out, nil
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
