package store

import (
	"context"
	"io"
)

type Store interface {
	AuditStore
	ApprovalStore
	PolicyStore
	Close() error
	Migrate(ctx context.Context) error
}

type AuditStore interface {
	WriteAuditEvent(ctx context.Context, event *AuditRecord) error
	QueryAuditEvents(ctx context.Context, filter AuditFilter) ([]AuditRecord, error)
	ExportAuditEvents(ctx context.Context, filter AuditFilter, format string, w io.Writer) error
}

type ApprovalStore interface {
	CreateApproval(ctx context.Context, req *ApprovalRecord) error
	GetApproval(ctx context.Context, id string) (*ApprovalRecord, error)
	UpdateApprovalStatus(ctx context.Context, id string, status string, resolvedBy string, reason string) error
	ListApprovals(ctx context.Context, tenantID string, status string) ([]ApprovalRecord, error)
}

type PolicyStore interface {
	SetPolicy(ctx context.Context, tenantID string, document []byte) error
	GetPolicy(ctx context.Context, tenantID string) ([]byte, error)
}
