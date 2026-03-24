package approvals

import "time"

type ApprovalStatus string

const (
	StatusPending  ApprovalStatus = "pending"
	StatusApproved ApprovalStatus = "approved"
	StatusDenied   ApprovalStatus = "denied"
	StatusExpired  ApprovalStatus = "expired"
)

type ApprovalVote struct {
	ApproverID string
	Decision   ApprovalStatus
	Reason     string
	VotedAt    time.Time
}

type ApprovalRequest struct {
	ID                string
	TenantID          string
	RequestID         string
	AgentName         string
	Service           string
	Action            string
	Risk              string
	Input             map[string]any
	PolicyRuleID      string
	CreatedAt         time.Time
	ExpiresAt         time.Time
	Status            ApprovalStatus
	ResolvedAt        *time.Time
	ResolvedBy        string
	DenyReason        string
	RequiredApprovals int
	Votes             []ApprovalVote
}
