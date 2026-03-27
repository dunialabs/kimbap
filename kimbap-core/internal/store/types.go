package store

import "time"

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
