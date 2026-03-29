package main

import (
	"context"

	"github.com/dunialabs/kimbap/internal/approvals"
	"github.com/dunialabs/kimbap/internal/store"
)

func expirePendingApprovalsWithSideEffects(ctx context.Context, st *store.SQLStore, tenantID string, onExpired func(store.ApprovalRecord)) (int, error) {
	if st == nil {
		return 0, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return approvals.ExpirePendingWithSideEffects(ctx, st, tenantID, func(item store.ApprovalRecord) {
		_ = st.RemoveExecution(ctx, item.ID)
		if onExpired != nil {
			onExpired(item)
		}
	})
}
