package app

import (
	"context"
	"strings"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/audit"
	runtimepkg "github.com/dunialabs/kimbap/internal/runtime"
)

type auditWriterAdapter struct {
	writer audit.Writer
}

func NewAuditWriterAdapter(writer audit.Writer) runtimepkg.AuditWriter {
	if writer == nil {
		return nil
	}
	return &auditWriterAdapter{writer: writer}
}

func (a *auditWriterAdapter) Write(ctx context.Context, event runtimepkg.AuditEvent) error {
	if a == nil || a.writer == nil {
		return nil
	}
	service := ""
	action := event.ActionName
	if left, right, ok := strings.Cut(event.ActionName, "."); ok {
		service = left
		action = right
	}

	out := audit.AuditEvent{
		Timestamp:      event.Timestamp,
		RequestID:      event.RequestID,
		TraceID:        event.TraceID,
		TenantID:       event.TenantID,
		PrincipalID:    event.PrincipalID,
		AgentName:      event.AgentName,
		Service:        service,
		Action:         action,
		Input:          event.Input,
		Mode:           string(event.Mode),
		Status:         mapAuditStatus(event),
		PolicyDecision: event.PolicyDecision,
		DurationMS:     event.DurationMS,
		Meta:           event.Meta,
	}
	if event.ErrorCode != "" || event.ErrorMessage != "" {
		out.Error = &audit.AuditError{Code: event.ErrorCode, Message: event.ErrorMessage}
	}
	return a.writer.Write(ctx, out)
}

func mapAuditStatus(event runtimepkg.AuditEvent) audit.AuditStatus {
	if strings.EqualFold(event.PolicyDecision, "deny") {
		return audit.AuditStatusDenied
	}
	switch event.Status {
	case actions.StatusSuccess:
		return audit.AuditStatusSuccess
	case actions.StatusApprovalRequired:
		return audit.AuditStatusApprovalRequired
	case actions.StatusTimeout:
		return audit.AuditStatusTimeout
	case actions.StatusCancelled:
		return audit.AuditStatusCancelled
	default:
		if event.ErrorCode == actions.ErrValidationFailed {
			return audit.AuditStatusValidationFailed
		}
		return audit.AuditStatusError
	}
}
