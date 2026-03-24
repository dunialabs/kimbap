package audit

import "time"

type AuditEvent struct {
	ID             string         `json:"id"`
	Timestamp      time.Time      `json:"timestamp"`
	RequestID      string         `json:"request_id"`
	TraceID        string         `json:"trace_id"`
	TenantID       string         `json:"tenant_id"`
	PrincipalID    string         `json:"principal_id"`
	AgentName      string         `json:"agent_name"`
	Service        string         `json:"service"`
	Action         string         `json:"action"`
	Mode           string         `json:"mode"`
	Status         AuditStatus    `json:"status"`
	PolicyDecision string         `json:"policy_decision"`
	DurationMS     int64          `json:"duration_ms"`
	Error          *AuditError    `json:"error,omitempty"`
	Input          map[string]any `json:"input,omitempty"`
	Meta           map[string]any `json:"meta,omitempty"`
}

type AuditError struct {
	Code    string         `json:"code,omitempty"`
	Message string         `json:"message,omitempty"`
	Meta    map[string]any `json:"meta,omitempty"`
}

type AuditStatus string

const (
	AuditStatusSuccess          AuditStatus = "success"
	AuditStatusError            AuditStatus = "error"
	AuditStatusDenied           AuditStatus = "denied"
	AuditStatusApprovalRequired AuditStatus = "approval_required"
	AuditStatusValidationFailed AuditStatus = "validation_failed"
	AuditStatusTimeout          AuditStatus = "timeout"
)
