package approvals

import (
	"context"

	"github.com/dunialabs/kimbap/internal/store"
)

type ExpiryStore interface {
	ListExpiredPendingApprovals(ctx context.Context, tenantID string) ([]store.ApprovalRecord, error)
	ExpireApproval(ctx context.Context, id string) (bool, error)
}

func ExpirePendingWithSideEffects(ctx context.Context, st ExpiryStore, tenantID string, onExpired func(store.ApprovalRecord)) (int, error) {
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
		count++
		if onExpired != nil {
			onExpired(item)
		}
	}
	return count, nil
}
