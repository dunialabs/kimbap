package main

import (
	"context"

	"github.com/dunialabs/kimbap/internal/store"
)

func expirePendingApprovalsWithSideEffects(ctx context.Context, st *store.SQLStore, tenantID string, onExpired func(store.ApprovalRecord)) (int, error) {
	if st == nil {
		return 0, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	candidates, err := st.ListExpiredPendingApprovals(ctx, tenantID)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, item := range candidates {
		expired, expErr := st.ExpireApproval(ctx, item.ID)
		if expErr != nil {
			return count, expErr
		}
		if !expired {
			continue
		}
		_ = st.RemoveExecution(ctx, item.ID)
		count++
		if onExpired != nil {
			onExpired(item)
		}
	}
	return count, nil
}
