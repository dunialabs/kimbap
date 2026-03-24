package services

import "testing"

func TestValidatePolicyDSLRejectsInvalidDecision(t *testing.T) {
	dsl := PolicyDsl{
		Rules: []PolicyRule{
			{ID: "r1", Effect: PolicyEffect{Decision: "BLOCK"}},
		},
	}

	if err := ValidatePolicyDSL(dsl); err == nil {
		t.Fatal("expected invalid decision to be rejected")
	}
}

func TestValidatePolicyDSLAcceptsKnownDecisions(t *testing.T) {
	dsl := PolicyDsl{
		Rules: []PolicyRule{
			{ID: "a", Effect: PolicyEffect{Decision: "ALLOW"}},
			{ID: "b", Effect: PolicyEffect{Decision: "REQUIRE_APPROVAL"}},
			{ID: "c", Effect: PolicyEffect{Decision: "DENY"}},
		},
	}

	if err := ValidatePolicyDSL(dsl); err != nil {
		t.Fatalf("expected known decisions to pass validation, got %v", err)
	}
}

func TestValidatePolicyDSLRejectsEmptyRules(t *testing.T) {
	if err := ValidatePolicyDSL(PolicyDsl{}); err == nil {
		t.Fatal("expected empty policy rules to be rejected")
	}
}

func TestValidatePolicyDSLRejectsUnknownOperator(t *testing.T) {
	dsl := PolicyDsl{
		Rules: []PolicyRule{
			{
				ID: "r1",
				Effect: PolicyEffect{
					Decision: "ALLOW",
				},
				When: []PolicyCondition{
					{Left: "$args.amount", Op: "contains", Right: 10},
				},
			},
		},
	}

	if err := ValidatePolicyDSL(dsl); err == nil {
		t.Fatal("expected unsupported operator to be rejected")
	}
}

func TestValidatePolicyDSLRejectsInvalidRegex(t *testing.T) {
	dsl := PolicyDsl{
		Rules: []PolicyRule{
			{
				ID:     "r1",
				Effect: PolicyEffect{Decision: "ALLOW"},
				When:   []PolicyCondition{{Left: "$args.name", Op: "matches", Right: "("}},
			},
		},
	}

	if err := ValidatePolicyDSL(dsl); err == nil {
		t.Fatal("expected invalid regex to be rejected")
	}
}

func TestEvaluateInvalidRuleDecisionFailsClosed(t *testing.T) {
	evaluator := NewPolicyDslEvaluator()
	ctx := PolicyEvaluationContext{ToolName: "svc.tool", Args: map[string]any{}}
	dsl := PolicyDsl{
		Rules: []PolicyRule{
			{
				ID:    "invalid-decision",
				Match: PolicyMatch{Tool: stringPtr("svc.tool")},
				Effect: PolicyEffect{
					Decision: "BLOCK",
				},
			},
		},
	}

	result := evaluator.Evaluate(dsl, ctx)
	if result.Decision != "DENY" {
		t.Fatalf("expected invalid decision to evaluate as DENY, got %q", result.Decision)
	}
}

func stringPtr(v string) *string { return &v }
