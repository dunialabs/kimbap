package policy

import (
	"context"
	"testing"
	"time"
)

func TestEvaluatorAllowRuleMatches(t *testing.T) {
	e := NewEvaluator(&PolicyDocument{
		Version: "1.0.0",
		Rules: []PolicyRule{{
			ID:       "allow-github-read",
			Priority: 10,
			Match: PolicyMatch{
				Agents:   []string{"assistant"},
				Services: []string{"github"},
				Actions:  []string{"github.list-pull-requests"},
				Risk:     []string{"low"},
			},
			Decision: DecisionAllow,
		}},
	})

	res, err := e.Evaluate(context.Background(), EvalRequest{
		TenantID:  "tenant-1",
		AgentName: "assistant",
		Service:   "github",
		Action:    "github.list-pull-requests",
		Risk:      "low",
	})
	if err != nil {
		t.Fatalf("evaluate failed: %v", err)
	}
	if res.Decision != DecisionAllow {
		t.Fatalf("expected allow, got %s", res.Decision)
	}
}

func TestEvaluatorDenyBeforeAllowWithHigherPriority(t *testing.T) {
	e := NewEvaluator(&PolicyDocument{
		Version: "1.0.0",
		Rules: []PolicyRule{
			{
				ID:       "deny-all-github",
				Priority: 1,
				Match: PolicyMatch{
					Services: []string{"github"},
					Actions:  []string{"github.*"},
				},
				Decision: DecisionDeny,
			},
			{
				ID:       "allow-github-read",
				Priority: 100,
				Match: PolicyMatch{
					Services: []string{"github"},
					Actions:  []string{"github.list-pull-requests"},
				},
				Decision: DecisionAllow,
			},
		},
	})

	res, err := e.Evaluate(context.Background(), EvalRequest{
		Service: "github",
		Action:  "github.list-pull-requests",
	})
	if err != nil {
		t.Fatalf("evaluate failed: %v", err)
	}
	if res.Decision != DecisionDeny {
		t.Fatalf("expected deny, got %s", res.Decision)
	}
	if res.MatchedRule == nil || res.MatchedRule.ID != "deny-all-github" {
		t.Fatalf("unexpected matched rule: %+v", res.MatchedRule)
	}
}

func TestEvaluatorPriorityOrdering(t *testing.T) {
	e := NewEvaluator(&PolicyDocument{
		Version: "1.0.0",
		Rules: []PolicyRule{
			{
				ID:       "allow-first",
				Priority: 5,
				Match:    PolicyMatch{Actions: []string{"github.*"}},
				Decision: DecisionAllow,
			},
			{
				ID:       "deny-second",
				Priority: 10,
				Match:    PolicyMatch{Actions: []string{"github.*"}},
				Decision: DecisionDeny,
			},
		},
	})

	res, err := e.Evaluate(context.Background(), EvalRequest{Action: "github.list-pull-requests"})
	if err != nil {
		t.Fatalf("evaluate failed: %v", err)
	}
	if res.Decision != DecisionAllow {
		t.Fatalf("expected allow from higher priority rule, got %s", res.Decision)
	}
}

func TestEvaluatorWildcardMatching(t *testing.T) {
	e := NewEvaluator(&PolicyDocument{
		Version: "1.0.0",
		Rules: []PolicyRule{{
			ID:       "allow-github-wildcard",
			Priority: 10,
			Match: PolicyMatch{
				Actions: []string{"github.*"},
			},
			Decision: DecisionAllow,
		}},
	})

	res, err := e.Evaluate(context.Background(), EvalRequest{Action: "github.list-pull-requests"})
	if err != nil {
		t.Fatalf("evaluate failed: %v", err)
	}
	if res.Decision != DecisionAllow {
		t.Fatalf("expected wildcard allow, got %s", res.Decision)
	}
}

func TestEvaluatorConditionAmountThresholdRequireApproval(t *testing.T) {
	e := NewEvaluator(&PolicyDocument{
		Version: "1.0.0",
		Rules: []PolicyRule{{
			ID:       "approve-high-amount",
			Priority: 10,
			Match: PolicyMatch{
				Actions: []string{"payments.create"},
			},
			Decision: DecisionRequireApproval,
			Conditions: []PolicyCondition{{
				Field:    "args.amount",
				Operator: "gt",
				Value:    1000,
			}},
		}},
	})

	res, err := e.Evaluate(context.Background(), EvalRequest{
		Action: "payments.create",
		Args:   map[string]any{"amount": 1500},
	})
	if err != nil {
		t.Fatalf("evaluate failed: %v", err)
	}
	if res.Decision != DecisionRequireApproval {
		t.Fatalf("expected require_approval, got %s", res.Decision)
	}
}

func TestEvaluatorDefaultDenyWhenNoRuleMatches(t *testing.T) {
	e := NewEvaluator(&PolicyDocument{
		Version: "1.0.0",
		Rules: []PolicyRule{{
			ID:       "allow-github",
			Priority: 10,
			Match:    PolicyMatch{Actions: []string{"github.*"}},
			Decision: DecisionAllow,
		}},
	})

	res, err := e.Evaluate(context.Background(), EvalRequest{Action: "slack.send-message"})
	if err != nil {
		t.Fatalf("evaluate failed: %v", err)
	}
	if res.Decision != DecisionDeny {
		t.Fatalf("expected default deny, got %s", res.Decision)
	}
}

func TestEvaluatorMutatingConditionCheck(t *testing.T) {
	e := NewEvaluator(&PolicyDocument{
		Version: "1.0.0",
		Rules: []PolicyRule{{
			ID:       "require-approval-if-mutating",
			Priority: 10,
			Match:    PolicyMatch{Actions: []string{"repo.delete"}},
			Decision: DecisionRequireApproval,
			Conditions: []PolicyCondition{{
				Field:    "risk.mutating",
				Operator: "eq",
				Value:    true,
			}},
		}},
	})

	res, err := e.Evaluate(context.Background(), EvalRequest{
		Action:   "repo.delete",
		Mutating: true,
	})
	if err != nil {
		t.Fatalf("evaluate failed: %v", err)
	}
	if res.Decision != DecisionRequireApproval {
		t.Fatalf("expected require approval for mutating action, got %s", res.Decision)
	}
}

func TestEvaluatorRateLimitScopeTenantIsolated(t *testing.T) {
	e := NewEvaluator(&PolicyDocument{
		Version: "1.0.0",
		Rules: []PolicyRule{{
			ID:       "tenant-limit",
			Priority: 1,
			Match:    PolicyMatch{Actions: []string{"payments.create"}},
			Decision: DecisionAllow,
			RateLimit: &RateLimitRule{
				MaxRequests: 1,
				WindowSec:   60,
				Scope:       "tenant",
			},
		}},
	})

	resA1, err := e.Evaluate(context.Background(), EvalRequest{
		TenantID: "tenant-a",
		Action:   "payments.create",
	})
	if err != nil {
		t.Fatalf("evaluate tenant-a first request failed: %v", err)
	}
	if resA1.Decision != DecisionAllow {
		t.Fatalf("expected tenant-a first request allow, got %s", resA1.Decision)
	}

	resA2, err := e.Evaluate(context.Background(), EvalRequest{
		TenantID: "tenant-a",
		Action:   "payments.create",
	})
	if err != nil {
		t.Fatalf("evaluate tenant-a second request failed: %v", err)
	}
	if resA2.Decision != DecisionDeny {
		t.Fatalf("expected tenant-a second request deny, got %s", resA2.Decision)
	}

	resB1, err := e.Evaluate(context.Background(), EvalRequest{
		TenantID: "tenant-b",
		Action:   "payments.create",
	})
	if err != nil {
		t.Fatalf("evaluate tenant-b first request failed: %v", err)
	}
	if resB1.Decision != DecisionAllow {
		t.Fatalf("expected tenant-b first request allow, got %s", resB1.Decision)
	}
}

func TestRateLimiterSweepsExpiredKeys(t *testing.T) {
	now := time.Now().UTC()
	rl := rateLimiter{
		windows: map[string][]time.Time{
			"stale": {now.Add(-2 * time.Minute)},
		},
		expires: map[string]time.Time{
			"stale": now.Add(-1 * time.Minute),
		},
		lastSweep:     now.Add(-2 * time.Minute),
		sweepInterval: time.Second,
	}

	status := rl.check("active", 2, 60)
	if !status.Allowed {
		t.Fatalf("expected active key to be allowed")
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()
	if _, ok := rl.windows["stale"]; ok {
		t.Fatalf("expected stale key to be swept from rate limiter")
	}
	if _, ok := rl.expires["stale"]; ok {
		t.Fatalf("expected stale expiry metadata to be swept from rate limiter")
	}
}
