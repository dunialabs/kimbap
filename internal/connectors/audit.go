package connectors

import (
	"context"
	"time"

	"github.com/dunialabs/kimbap/internal/audit"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

const auditServiceName = "auth"

type AuditEmitter struct {
	Writer audit.Writer
}

func (e *AuditEmitter) Emit(ctx context.Context, action string, provider string, tenantID string, status audit.AuditStatus, flowType FlowType, meta map[string]any) {
	if e == nil || e.Writer == nil {
		return
	}
	if meta == nil {
		meta = map[string]any{}
	}
	meta["provider"] = provider
	if flowType != "" {
		meta["flow_type"] = string(flowType)
	}

	event := audit.AuditEvent{
		ID:        generateEventID(),
		Timestamp: time.Now().UTC(),
		TenantID:  tenantID,
		Service:   auditServiceName,
		Action:    action,
		Status:    status,
		Meta:      meta,
	}
	if err := e.Writer.Write(ctx, event); err != nil {
		log.Error().Err(err).Str("action", action).Str("tenantID", tenantID).Msg("failed to write audit event")
	}
}

func (e *AuditEmitter) ConnectStarted(ctx context.Context, provider, tenantID string, flow FlowType) {
	e.Emit(ctx, "auth.connect.started", provider, tenantID, audit.AuditStatusSuccess, flow, nil)
}

func (e *AuditEmitter) ConnectCompleted(ctx context.Context, provider, tenantID string, flow FlowType, scopes []string) {
	e.Emit(ctx, "auth.connect.completed", provider, tenantID, audit.AuditStatusSuccess, flow, map[string]any{
		"granted_scopes": scopes,
	})
}

func (e *AuditEmitter) ConnectFailed(ctx context.Context, provider, tenantID string, flow FlowType, errMsg string) {
	e.Emit(ctx, "auth.connect.failed", provider, tenantID, audit.AuditStatusError, flow, map[string]any{
		"error": errMsg,
	})
}

func (e *AuditEmitter) RefreshSucceeded(ctx context.Context, provider, tenantID string) {
	e.Emit(ctx, "auth.refresh.succeeded", provider, tenantID, audit.AuditStatusSuccess, "", nil)
}

func (e *AuditEmitter) RefreshFailed(ctx context.Context, provider, tenantID string, errMsg string) {
	e.Emit(ctx, "auth.refresh.failed", provider, tenantID, audit.AuditStatusError, "", map[string]any{
		"error": errMsg,
	})
}

func (e *AuditEmitter) ReconnectCompleted(ctx context.Context, provider, tenantID string, flow FlowType) {
	e.Emit(ctx, "auth.reconnect.completed", provider, tenantID, audit.AuditStatusSuccess, flow, nil)
}

func (e *AuditEmitter) RevokeCompleted(ctx context.Context, provider, tenantID string, remoteRevoked bool) {
	e.Emit(ctx, "auth.revoke.completed", provider, tenantID, audit.AuditStatusSuccess, "", map[string]any{
		"remote_revoked": remoteRevoked,
	})
}

func (e *AuditEmitter) DeviceFlowStarted(ctx context.Context, provider, tenantID string) {
	e.Emit(ctx, "auth.device.started", provider, tenantID, audit.AuditStatusSuccess, FlowDevice, nil)
}

func (e *AuditEmitter) DeviceFlowCompleted(ctx context.Context, provider, tenantID string) {
	e.Emit(ctx, "auth.device.completed", provider, tenantID, audit.AuditStatusSuccess, FlowDevice, nil)
}

func generateEventID() string {
	return uuid.NewString()
}
