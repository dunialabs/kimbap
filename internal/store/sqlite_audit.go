package store

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (s *SQLStore) WriteAuditEvent(ctx context.Context, event *AuditRecord) error {
	if event == nil {
		return errors.New("audit event is required")
	}
	if strings.TrimSpace(event.ID) == "" {
		event.ID = uuid.NewString()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, s.bind(`
		INSERT INTO audit_events (
			id, timestamp, request_id, trace_id, tenant_id, principal_id,
			agent_name, service, action, mode, status, policy_decision,
			duration_ms, error_code, error_message, input_json, meta_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`),
		event.ID,
		event.Timestamp,
		event.RequestID,
		event.TraceID,
		event.TenantID,
		event.PrincipalID,
		event.AgentName,
		event.Service,
		event.Action,
		event.Mode,
		event.Status,
		event.PolicyDecision,
		event.DurationMS,
		event.ErrorCode,
		event.ErrorMessage,
		event.InputJSON,
		event.MetaJSON,
	)
	return err
}

func buildAuditQueryArgs(filter AuditFilter, unlimited bool) (string, []any) {
	args := make([]any, 0, 12)
	query := `
		SELECT id, timestamp, request_id, trace_id, tenant_id, principal_id,
			agent_name, service, action, mode, status, policy_decision,
			duration_ms, error_code, error_message, input_json, meta_json
		FROM audit_events
		WHERE 1 = 1`

	if filter.TenantID != "" {
		query += " AND tenant_id = ?"
		args = append(args, filter.TenantID)
	}
	if filter.AgentName != "" {
		query += " AND agent_name = ?"
		args = append(args, filter.AgentName)
	}
	if filter.Service != "" {
		query += " AND service = ?"
		args = append(args, filter.Service)
	}
	if filter.Action != "" {
		query += " AND action = ?"
		args = append(args, filter.Action)
	}
	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, filter.Status)
	}
	if filter.From != nil {
		query += " AND timestamp >= ?"
		args = append(args, *filter.From)
	}
	if filter.To != nil {
		query += " AND timestamp <= ?"
		args = append(args, *filter.To)
	}

	query += " ORDER BY timestamp DESC, id DESC"
	if !unlimited {
		effectiveLimit := filter.Limit
		if effectiveLimit <= 0 || effectiveLimit > maxAuditQueryLimit {
			effectiveLimit = maxAuditQueryLimit
		}
		query += " LIMIT ?"
		args = append(args, effectiveLimit)
		if filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	}
	return query, args
}

func (s *SQLStore) QueryAuditEvents(ctx context.Context, filter AuditFilter) ([]AuditRecord, error) {
	query, args := buildAuditQueryArgs(filter, false)
	rows, err := s.db.QueryContext(ctx, s.bind(query), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]AuditRecord, 0)
	for rows.Next() {
		rec, scanErr := scanAudit(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, *rec)
	}
	return out, rows.Err()
}

func (s *SQLStore) ExportAuditEvents(ctx context.Context, filter AuditFilter, format string, w io.Writer) error {
	if w == nil {
		return errors.New("writer is required")
	}
	query, args := buildAuditQueryArgs(filter, true)
	rows, err := s.db.QueryContext(ctx, s.bind(query), args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "json", "jsonl":
		enc := json.NewEncoder(w)
		for rows.Next() {
			rec, scanErr := scanAudit(rows)
			if scanErr != nil {
				return scanErr
			}
			if err := enc.Encode(*rec); err != nil {
				return err
			}
		}
		return rows.Err()
	case "csv":
		cw := csv.NewWriter(w)
		headers := []string{"id", "timestamp", "request_id", "trace_id", "tenant_id", "principal_id", "agent_name", "service", "action", "mode", "status", "policy_decision", "duration_ms", "error_code", "error_message", "input_json", "meta_json"}
		if err := cw.Write(headers); err != nil {
			return err
		}
		for rows.Next() {
			r, scanErr := scanAudit(rows)
			if scanErr != nil {
				return scanErr
			}
			row := []string{r.ID, r.Timestamp.Format(time.RFC3339Nano), r.RequestID, r.TraceID, r.TenantID, r.PrincipalID, r.AgentName, r.Service, r.Action, r.Mode, r.Status, r.PolicyDecision, fmt.Sprintf("%d", r.DurationMS), r.ErrorCode, r.ErrorMessage, r.InputJSON, r.MetaJSON}
			if err := cw.Write(row); err != nil {
				return err
			}
		}
		if err := rows.Err(); err != nil {
			return err
		}
		cw.Flush()
		return cw.Error()
	default:
		return fmt.Errorf("unsupported export format %q", format)
	}
}
