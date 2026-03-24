package services

import (
	"encoding/json"
	"testing"

	"github.com/dunialabs/kimbap-core/internal/database"
	"github.com/dunialabs/kimbap-core/internal/types"
	"gorm.io/datatypes"
)

func TestMapPolicyDecisionDefaultsToDenyForUnknown(t *testing.T) {
	if got := mapPolicyDecision("UNKNOWN_VALUE"); got != types.PolicyDecisionDeny {
		t.Fatalf("expected unknown decision to map to DENY, got %q", got)
	}
}

func TestMapPolicyDecisionNormalizesCase(t *testing.T) {
	if got := mapPolicyDecision("allow"); got != types.PolicyDecisionAllow {
		t.Fatalf("expected allow to normalize to %q, got %q", types.PolicyDecisionAllow, got)
	}
	if got := mapPolicyDecision(" require_approval "); got != types.PolicyDecisionRequireApproval {
		t.Fatalf("expected require_approval to normalize to %q, got %q", types.PolicyDecisionRequireApproval, got)
	}
	if got := mapPolicyDecision("deny"); got != types.PolicyDecisionDeny {
		t.Fatalf("expected deny to normalize to %q, got %q", types.PolicyDecisionDeny, got)
	}
}

func TestEvaluatePolicySetRejectsInvalidDSL(t *testing.T) {
	engine := &PolicyEngine{evaluator: PolicyDslEvaluatorInstance()}
	rawDsl, err := json.Marshal(map[string]any{
		"rules": []map[string]any{
			{
				"id":    "bad-op",
				"match": map[string]any{"tool": "svc.tool"},
				"when": []map[string]any{
					{"left": "$args.x", "op": "contains", "right": 1},
				},
				"effect": map[string]any{"decision": "ALLOW"},
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal dsl: %v", err)
	}

	_, matched, evalErr := engine.evaluatePolicySet(database.ToolPolicySet{
		ID:      "policy-1",
		Version: 7,
		Dsl:     datatypes.JSON(rawDsl),
	}, PolicyEvaluateParams{ToolName: "svc.tool", Args: map[string]any{"x": 1}})
	if matched {
		t.Fatal("expected invalid policy to not match")
	}
	if evalErr == nil {
		t.Fatal("expected invalid policy DSL to return error")
	}
}

func TestFallbackToDangerLevelDefaultDeny(t *testing.T) {
	engine := &PolicyEngine{evaluator: PolicyDslEvaluatorInstance()}

	if got := engine.fallbackToDangerLevel(types.DangerLevelSilent); got.Decision != types.PolicyDecisionAllow {
		t.Fatalf("expected DangerLevelSilent to fallback to ALLOW, got %q", got.Decision)
	}
	if got := engine.fallbackToDangerLevel(types.DangerLevelNotification); got.Decision != types.PolicyDecisionAllow {
		t.Fatalf("expected DangerLevelNotification to fallback to ALLOW, got %q", got.Decision)
	}
	if got := engine.fallbackToDangerLevel(types.DangerLevelApproval); got.Decision != types.PolicyDecisionRequireApproval {
		t.Fatalf("expected DangerLevelApproval to fallback to REQUIRE_APPROVAL, got %q", got.Decision)
	}
	if got := engine.fallbackToDangerLevel(999); got.Decision != types.PolicyDecisionDeny {
		t.Fatalf("expected unknown danger level to fallback to DENY, got %q", got.Decision)
	}
}
