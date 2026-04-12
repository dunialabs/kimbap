package runtime

import (
	"context"
	"time"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/adapters"
)

type PolicyRequest struct {
	TenantID       string
	Principal      actions.Principal
	Action         actions.ActionDefinition
	Input          map[string]any
	Mode           actions.ExecutionMode
	Session        *actions.SessionContext
	Classification *actions.ClassificationInfo
}

type PolicyDecision struct {
	Decision string
	Reason   string
	RuleID   string
	Meta     map[string]any
}

type CredentialResolver interface {
	Resolve(ctx context.Context, tenantID string, req actions.AuthRequirement) (*actions.ResolvedCredentialSet, error)
}

type PolicyEvaluator interface {
	Evaluate(ctx context.Context, req PolicyRequest) (*PolicyDecision, error)
}

type AuditEvent struct {
	RequestID       string
	TraceID         string
	TenantID        string
	PrincipalID     string
	AgentName       string
	ActionName      string
	Input           map[string]any
	Mode            actions.ExecutionMode
	Status          actions.ExecutionStatus
	HTTPStatus      int
	ErrorCode       string
	ErrorMessage    string
	PolicyDecision  string
	ApprovalRequest string
	DurationMS      int64
	Timestamp       time.Time
	Meta            map[string]any
}

type AuditWriter interface {
	Write(ctx context.Context, event AuditEvent) error
}

type ListOptions struct {
	Namespace string
	Resource  string
	Verb      string
	Limit     int
}

type ActionRegistry interface {
	Lookup(ctx context.Context, name string) (*actions.ActionDefinition, error)
	List(ctx context.Context, opts ListOptions) ([]actions.ActionDefinition, error)
}

type ApprovalRequest struct {
	RequestID string
	TraceID   string
	TenantID  string
	Principal actions.Principal
	Action    actions.ActionDefinition
	Input     map[string]any
	Meta      map[string]any
}

type ApprovalResult struct {
	Approved  bool
	RequestID string
	Timeout   bool
	Reason    string
	Meta      map[string]any
}

type ApprovalManager interface {
	CreateRequest(ctx context.Context, req ApprovalRequest) (*ApprovalResult, error)
	CancelRequest(ctx context.Context, approvalRequestID string, reason string) error
}

type PrincipalVerifier interface {
	Verify(ctx context.Context, principal actions.Principal) error
}

type HeldExecutionStore interface {
	Hold(ctx context.Context, approvalRequestID string, req actions.ExecutionRequest) error
	Resume(ctx context.Context, approvalRequestID string) (*actions.ExecutionRequest, error)
	Remove(ctx context.Context, approvalRequestID string) error
}

type Runtime struct {
	PrincipalVerifier  PrincipalVerifier
	PolicyEvaluator    PolicyEvaluator
	CredentialResolver CredentialResolver
	AuditWriter        AuditWriter
	AuditRequired      bool
	ActionRegistry     ActionRegistry
	ApprovalManager    ApprovalManager
	HeldExecutionStore HeldExecutionStore
	Adapters           map[string]adapters.Adapter
	Now                func() time.Time
}

const auditWriteTimeout = 3 * time.Second

type TraceStep struct {
	Step       string `json:"step"`
	Status     string `json:"status"`
	Detail     string `json:"detail,omitempty"`
	DurationMS int64  `json:"duration_ms"`
}

type TraceCollector struct {
	Steps    []TraceStep
	lastStep time.Time
	now      func() time.Time
}

func NewTraceCollector(now func() time.Time) *TraceCollector {
	if now == nil {
		now = time.Now
	}
	t := now()
	return &TraceCollector{lastStep: t, now: now}
}

func (tc *TraceCollector) Record(step, status, detail string) {
	if tc == nil {
		return
	}
	now := tc.now()
	tc.Steps = append(tc.Steps, TraceStep{Step: step, Status: status, Detail: detail, DurationMS: now.Sub(tc.lastStep).Milliseconds()})
	tc.lastStep = now
}
