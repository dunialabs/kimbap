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
	ID             string
	Timestamp      time.Time
	RequestID      string
	TraceID        string
	TenantID       string
	PrincipalID    string
	AgentName      string
	Service        string
	Action         string
	Mode           string
	Status         string
	PolicyDecision string
	DurationMS     int64
	ErrorCode      string
	ErrorMessage   string
	InputJSON      string
	MetaJSON       string
}

type ApprovalRecord struct {
	ID         string
	TenantID   string
	RequestID  string
	AgentName  string
	Service    string
	Action     string
	Status     string
	InputJSON  string
	CreatedAt  time.Time
	ExpiresAt  time.Time
	ResolvedAt *time.Time
	ResolvedBy string
	Reason     string
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
