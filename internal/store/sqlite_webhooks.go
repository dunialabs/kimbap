package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (s *SQLStore) UpsertWebhookSubscription(ctx context.Context, sub *WebhookSubscriptionRecord) error {
	if sub == nil {
		return errors.New("webhook subscription is required")
	}
	sub.ID = strings.TrimSpace(sub.ID)
	sub.TenantID = strings.TrimSpace(sub.TenantID)
	if sub.ID == "" {
		return errors.New("webhook subscription id is required")
	}
	if sub.TenantID == "" {
		return ErrInvalidTenantID
	}
	if strings.TrimSpace(sub.EventsJSON) == "" {
		sub.EventsJSON = "[]"
	}
	now := time.Now().UTC()
	if sub.CreatedAt.IsZero() {
		sub.CreatedAt = now
	}
	sub.UpdatedAt = now
	_, err := s.db.ExecContext(ctx, s.bind(`
		INSERT INTO webhook_subscriptions (
			id, tenant_id, url, secret, events_json, active, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (id, tenant_id)
		DO UPDATE SET
			url = EXCLUDED.url,
			secret = EXCLUDED.secret,
			events_json = EXCLUDED.events_json,
			active = EXCLUDED.active,
			updated_at = EXCLUDED.updated_at
	`),
		sub.ID,
		sub.TenantID,
		sub.URL,
		sub.Secret,
		sub.EventsJSON,
		sub.Active,
		sub.CreatedAt,
		sub.UpdatedAt,
	)
	return err
}

func (s *SQLStore) DeleteWebhookSubscription(ctx context.Context, id string, tenantID string) error {
	id = strings.TrimSpace(id)
	tenantID = strings.TrimSpace(tenantID)
	if id == "" {
		return errors.New("webhook subscription id is required")
	}
	if tenantID == "" {
		return ErrInvalidTenantID
	}
	_, err := s.db.ExecContext(ctx, s.bind(`DELETE FROM webhook_subscriptions WHERE id = ? AND tenant_id = ?`), id, tenantID)
	return err
}

func (s *SQLStore) ListWebhookSubscriptions(ctx context.Context, tenantID string) ([]WebhookSubscriptionRecord, error) {
	tenantID = strings.TrimSpace(tenantID)
	query := `
		SELECT id, tenant_id, url, secret, events_json, active, created_at, updated_at
		FROM webhook_subscriptions
		WHERE active = ?`
	args := []any{true}
	if tenantID != "" {
		query += ` AND tenant_id = ?`
		args = append(args, tenantID)
	}
	query += ` ORDER BY updated_at DESC, id DESC`

	rows, err := s.db.QueryContext(ctx, s.bind(query), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]WebhookSubscriptionRecord, 0)
	for rows.Next() {
		var rec WebhookSubscriptionRecord
		if err := rows.Scan(&rec.ID, &rec.TenantID, &rec.URL, &rec.Secret, &rec.EventsJSON, &rec.Active, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (s *SQLStore) WriteWebhookEvent(ctx context.Context, event *WebhookEventRecord) error {
	if event == nil {
		return errors.New("webhook event is required")
	}
	event.ID = strings.TrimSpace(event.ID)
	event.TenantID = strings.TrimSpace(event.TenantID)
	event.Type = strings.TrimSpace(event.Type)
	if event.ID == "" {
		event.ID = "evt_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	}
	if event.TenantID == "" {
		return ErrInvalidTenantID
	}
	if event.Type == "" {
		return errors.New("webhook event type is required")
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if strings.TrimSpace(event.DataJSON) == "" {
		event.DataJSON = "{}"
	}
	_, err := s.db.ExecContext(ctx, s.bind(`
		INSERT INTO webhook_events (id, tenant_id, type, timestamp, data_json)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET
			tenant_id = EXCLUDED.tenant_id,
			type = EXCLUDED.type,
			timestamp = EXCLUDED.timestamp,
			data_json = EXCLUDED.data_json
	`), event.ID, event.TenantID, event.Type, event.Timestamp, event.DataJSON)
	return err
}

func (s *SQLStore) ListWebhookEvents(ctx context.Context, tenantID string, limit int) ([]WebhookEventRecord, error) {
	tenantID = strings.TrimSpace(tenantID)
	if limit <= 0 || limit > 1000 {
		limit = 1000
	}
	query := `
		SELECT id, tenant_id, type, timestamp, data_json
		FROM webhook_events
		WHERE 1 = 1`
	args := make([]any, 0, 2)
	if tenantID != "" {
		query += ` AND tenant_id = ?`
		args = append(args, tenantID)
	}
	query += ` ORDER BY timestamp DESC, id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, s.bind(query), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tmp := make([]WebhookEventRecord, 0)
	for rows.Next() {
		var rec WebhookEventRecord
		if err := rows.Scan(&rec.ID, &rec.TenantID, &rec.Type, &rec.Timestamp, &rec.DataJSON); err != nil {
			return nil, err
		}
		tmp = append(tmp, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i, j := 0, len(tmp)-1; i < j; i, j = i+1, j-1 {
		tmp[i], tmp[j] = tmp[j], tmp[i]
	}
	return tmp, nil
}
