package approvals

import (
	"errors"
	"time"
)

type ApprovalStatus string

const (
	StatusPending  ApprovalStatus = "pending"
	StatusApproved ApprovalStatus = "approved"
	StatusDenied   ApprovalStatus = "denied"
	StatusExpired  ApprovalStatus = "expired"
)

var (
	ErrAlreadyResolved = errors.New("approval already resolved")
	ErrExpired         = errors.New("approval request has expired")
	ErrDuplicateVote   = errors.New("approver has already voted")
	ErrNotFound        = errors.New("approval request not found")
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
