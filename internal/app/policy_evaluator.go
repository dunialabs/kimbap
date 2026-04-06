package app

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dunialabs/kimbap/internal/policy"
	runtimepkg "github.com/dunialabs/kimbap/internal/runtime"
	"github.com/dunialabs/kimbap/internal/store"
)

type policyEvaluatorAdapter struct {
	evaluator *policy.Evaluator
}

func (a *policyEvaluatorAdapter) Evaluate(ctx context.Context, req runtimepkg.PolicyRequest) (*runtimepkg.PolicyDecision, error) {
	if a == nil || a.evaluator == nil {
		return nil, fmt.Errorf("policy evaluator is not initialized")
	}
	service, actionName := resolveServiceAction(req)

	res, err := a.evaluator.Evaluate(ctx, policy.EvalRequest{
		TenantID:  req.TenantID,
		AgentName: req.Principal.AgentName,
		Service:   service,
		Action:    actionName,
		Risk:      req.Action.Risk.DocVocab(),
		Mutating:  !req.Action.Idempotent,
		Args:      req.Input,
	})
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}

	decision := &runtimepkg.PolicyDecision{
		Decision: string(res.Decision),
		Reason:   res.Reason,
		Meta:     map[string]any{},
	}
	if res.MatchedRule != nil {
		decision.RuleID = res.MatchedRule.ID
	}
	if res.RateStatus != nil {
		decision.Meta["rate_allowed"] = res.RateStatus.Allowed
		decision.Meta["rate_remaining"] = res.RateStatus.Remaining
		decision.Meta["rate_reset_at"] = res.RateStatus.ResetAt
	}
	return decision, nil
}

type cachedPolicyEntry struct {
	eval        *policyEvaluatorAdapter
	fingerprint [32]byte
	missing     bool
	checkedAt   time.Time
}

type storePolicyEvaluator struct {
	policyStore store.PolicyStore
	fallback    runtimepkg.PolicyEvaluator

	mu    sync.Mutex
	cache map[string]cachedPolicyEntry
}

func (e *storePolicyEvaluator) InvalidateTenantPolicyCache(tenantID string) {
	if e == nil {
		return
	}
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cache == nil {
		return
	}
	delete(e.cache, tenantID)
}

const policyStoreProbeInterval = 500 * time.Millisecond

func (e *storePolicyEvaluator) Evaluate(ctx context.Context, req runtimepkg.PolicyRequest) (*runtimepkg.PolicyDecision, error) {
	now := time.Now()
	e.mu.Lock()
	if entry, ok := e.cache[req.TenantID]; ok && now.Sub(entry.checkedAt) < policyStoreProbeInterval {
		if entry.missing {
			e.mu.Unlock()
			if e.fallback != nil {
				return e.fallback.Evaluate(ctx, req)
			}
			return nil, nil
		}
		eval := entry.eval
		e.mu.Unlock()
		if eval != nil {
			return eval.Evaluate(ctx, req)
		}
		return nil, nil
	}
	e.mu.Unlock()

	data, err := e.policyStore.GetPolicy(ctx, req.TenantID)
	if err == nil && len(data) > 0 {
		fp := sha256.Sum256(data)

		e.mu.Lock()
		if e.cache == nil {
			e.cache = make(map[string]cachedPolicyEntry)
		}
		if entry, ok := e.cache[req.TenantID]; ok && entry.fingerprint == fp {
			entry.checkedAt = now
			e.cache[req.TenantID] = entry
			eval := entry.eval
			e.mu.Unlock()
			return eval.Evaluate(ctx, req)
		}
		doc, parseErr := policy.ParseDocument(data)
		if parseErr != nil {
			e.mu.Unlock()
			return nil, fmt.Errorf("parse tenant policy: %w", parseErr)
		}
		newEntry := cachedPolicyEntry{
			eval:        &policyEvaluatorAdapter{evaluator: policy.NewEvaluator(doc)},
			fingerprint: fp,
			checkedAt:   now,
		}
		e.cache[req.TenantID] = newEntry
		eval := newEntry.eval
		e.mu.Unlock()
		return eval.Evaluate(ctx, req)
	}
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return nil, fmt.Errorf("load tenant policy: %w", err)
	}
	if errors.Is(err, store.ErrNotFound) {
		e.mu.Lock()
		if e.cache == nil {
			e.cache = make(map[string]cachedPolicyEntry)
		}
		e.cache[req.TenantID] = cachedPolicyEntry{missing: true, checkedAt: now}
		e.mu.Unlock()
	}
	if e.fallback != nil {
		return e.fallback.Evaluate(ctx, req)
	}
	return nil, nil
}

func resolveServiceAction(req runtimepkg.PolicyRequest) (string, string) {
	if req.Classification != nil {
		service := strings.TrimSpace(req.Classification.Service)
		action := strings.TrimSpace(req.Classification.ActionName)
		if service != "" && strings.HasPrefix(action, service+".") {
			action = strings.TrimPrefix(action, service+".")
		}
		if service != "" || action != "" {
			return service, action
		}
	}
	name := strings.TrimSpace(req.Action.Name)
	if left, right, ok := strings.Cut(name, "."); ok {
		return strings.TrimSpace(left), strings.TrimSpace(right)
	}
	return strings.TrimSpace(req.Action.Namespace), name
}
