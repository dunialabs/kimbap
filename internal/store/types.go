package store

import "time"

type TokenRecord struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	AgentName   string     `json:"agent_name"`
	TokenHash   string     `json:"-"`
	DisplayHint string     `json:"display_hint"`
	Scopes      string     `json:"scopes"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   time.Time  `json:"expires_at"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	CreatedBy   string     `json:"created_by"`
}

type AuditRecord struct {
	ID             string    `json:"id"`
	Timestamp      time.Time `json:"timestamp"`
	RequestID      string    `json:"request_id"`
	TraceID        string    `json:"trace_id"`
	TenantID       string    `json:"tenant_id"`
	PrincipalID    string    `json:"principal_id"`
	AgentName      string    `json:"agent_name"`
	Service        string    `json:"service"`
	Action         string    `json:"action"`
	Mode           string    `json:"mode"`
	Status         string    `json:"status"`
	PolicyDecision string    `json:"policy_decision"`
	DurationMS     int64     `json:"duration_ms"`
	ErrorCode      string    `json:"error_code,omitempty"`
	ErrorMessage   string    `json:"error_message,omitempty"`
	InputJSON      string    `json:"input_json,omitempty"`
	MetaJSON       string    `json:"meta_json,omitempty"`
}

type ApprovalRecord struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	RequestID  string     `json:"request_id"`
	AgentName  string     `json:"agent_name"`
	Service    string     `json:"service"`
	Action     string     `json:"action"`
	Status     string     `json:"status"`
	InputJSON  string     `json:"input_json,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
	ResolvedBy string     `json:"resolved_by,omitempty"`
	Reason     string     `json:"reason,omitempty"`
}

type WebhookSubscriptionRecord struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	URL        string    `json:"url"`
	Secret     string    `json:"secret,omitempty"`
	EventsJSON string    `json:"events_json"`
	Active     bool      `json:"active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type WebhookEventRecord struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	DataJSON  string    `json:"data_json"`
}

type AuditFilter struct {
	TenantID  string
	AgentName string
	Service   string
	Action    string
	Status    string
	From      *time.Time
	To        *time.Time
	Limit     int
	Offset    int
}
