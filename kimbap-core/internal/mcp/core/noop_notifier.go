package core

import (
	"context"
	"time"
)

type NoopSocketNotifier struct{}

func NewNoopSocketNotifier() SocketNotifier {
	return &NoopSocketNotifier{}
}

func (n *NoopSocketNotifier) AskUserConfirm(_ context.Context, _ string, _ string, _ string, _ string, _ string, _ string, _ time.Duration) (bool, error) {
	return false, nil
}

func (n *NoopSocketNotifier) NotifyUserPermissionChanged(_ string) {}

func (n *NoopSocketNotifier) NotifyUserPermissionChangedByServer(_ string) {}

func (n *NoopSocketNotifier) NotifyUserDisabled(_ string, _ string) bool { return false }

func (n *NoopSocketNotifier) NotifyUserExpired(_ string) bool { return false }

func (n *NoopSocketNotifier) NotifyOnlineSessions(_ string) bool { return false }

func (n *NoopSocketNotifier) NotifyApprovalCreated(_ string, _ string, _ string, _ *string, _ any, _ time.Time, _ time.Time, _ string, _ *string, _ int, _ *string, _ *string) {
}

func (n *NoopSocketNotifier) NotifyApprovalDecided(_ string, _ string, _ string, _ string, _ *string) {
}

func (n *NoopSocketNotifier) NotifyApprovalExpired(_ string, _ string, _ string) {}

func (n *NoopSocketNotifier) NotifyApprovalExecuted(_ string, _ string, _ string, _ bool, _ *string) {
}

func (n *NoopSocketNotifier) NotifyApprovalFailed(_ string, _ string, _ string, _ string, _ bool, _ *string) {
}

func (n *NoopSocketNotifier) NotifyServerStatusChanged(_ string, _ string, _ int, _ int) {}

func (n *NoopSocketNotifier) UpdateServerInfo() {}
