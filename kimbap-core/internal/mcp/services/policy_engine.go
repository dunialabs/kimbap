package services

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/dunialabs/kimbap-core/internal/database"
	"github.com/dunialabs/kimbap-core/internal/types"
	"gorm.io/gorm"
)

type PolicyEvaluateParams struct {
	UserID      string
	ServerID    *string
	ToolName    string
	Args        map[string]any
	DangerLevel int
}

type PolicyEvaluateResult struct {
	Decision      string
	PolicyVersion int
	MatchedRuleID *string
	Reason        *string
}

type policyCacheEntry struct {
	policies  []database.ToolPolicySet
	fetchedAt time.Time
}

type PolicyEngine struct {
	db        *gorm.DB
	evaluator *PolicyDslEvaluator
	cacheMu   sync.RWMutex
	cache     map[string]*policyCacheEntry
}

var (
	policyEngineOnce sync.Once
	policyEngineInst *PolicyEngine
)

func PolicyEngineInstance() *PolicyEngine {
	policyEngineOnce.Do(func() {
		policyEngineInst = &PolicyEngine{
			db:        database.DB,
			evaluator: PolicyDslEvaluatorInstance(),
			cache:     make(map[string]*policyCacheEntry),
		}
	})
	return policyEngineInst
}

func (e *PolicyEngine) Evaluate(params PolicyEvaluateParams) (PolicyEvaluateResult, error) {
	policies, err := e.getCachedEffectivePolicy(params.ServerID)
	if err != nil {
		return PolicyEvaluateResult{}, fmt.Errorf("failed to load policies: %w", err)
	}

	for _, policySet := range policies {
		result, matched, evalErr := e.evaluatePolicySet(policySet, params)
		if evalErr != nil {
			return PolicyEvaluateResult{Decision: types.PolicyDecisionDeny, PolicyVersion: policySet.Version}, fmt.Errorf("invalid policy set %s: %w", policySet.ID, evalErr)
		}
		if matched {
			return result, nil
		}
	}

	return e.fallbackToDangerLevel(params.DangerLevel), nil
}

func (e *PolicyEngine) evaluatePolicySet(policySet database.ToolPolicySet, params PolicyEvaluateParams) (PolicyEvaluateResult, bool, error) {
	var dsl PolicyDsl
	if err := json.Unmarshal(policySet.Dsl, &dsl); err != nil {
		return PolicyEvaluateResult{}, false, err
	}
	if err := ValidatePolicyDSL(dsl); err != nil {
		return PolicyEvaluateResult{}, false, err
	}

	result := e.evaluator.Evaluate(dsl, PolicyEvaluationContext{
		ServerID: params.ServerID,
		ToolName: params.ToolName,
		Args:     params.Args,
	})
	if result.MatchedRuleID == nil {
		return PolicyEvaluateResult{}, false, nil
	}
	return PolicyEvaluateResult{
		Decision:      mapPolicyDecision(result.Decision),
		PolicyVersion: policySet.Version,
		MatchedRuleID: result.MatchedRuleID,
		Reason:        result.Reason,
	}, true, nil
}

func (e *PolicyEngine) getEffectivePolicy(serverID *string) ([]database.ToolPolicySet, error) {
	db := e.db
	if db == nil {
		db = database.DB
	}

	globals := make([]database.ToolPolicySet, 0)
	if err := db.Where("status = ? AND server_id IS NULL", "active").Order("version DESC").Find(&globals).Error; err != nil {
		return nil, err
	}
	if serverID == nil {
		return globals, nil
	}

	serverPolicies := make([]database.ToolPolicySet, 0)
	if err := db.Where("status = ? AND server_id = ?", "active", *serverID).Order("version DESC").Find(&serverPolicies).Error; err != nil {
		return nil, err
	}
	return append(serverPolicies, globals...), nil
}

const policyCacheTTL = 30 * time.Second

func (e *PolicyEngine) getCachedEffectivePolicy(serverID *string) ([]database.ToolPolicySet, error) {
	cacheKey := "__global__"
	if serverID != nil {
		cacheKey = *serverID
	}

	e.cacheMu.RLock()
	if entry, ok := e.cache[cacheKey]; ok && time.Since(entry.fetchedAt) < policyCacheTTL {
		e.cacheMu.RUnlock()
		return entry.policies, nil
	}
	e.cacheMu.RUnlock()

	policies, err := e.getEffectivePolicy(serverID)
	if err != nil {
		return nil, err
	}

	e.cacheMu.Lock()
	e.cache[cacheKey] = &policyCacheEntry{policies: policies, fetchedAt: time.Now()}
	e.cacheMu.Unlock()

	return policies, nil
}

func (e *PolicyEngine) ClearCache(serverID *string) {
	e.cacheMu.Lock()
	defer e.cacheMu.Unlock()
	if serverID != nil {
		delete(e.cache, *serverID)
		delete(e.cache, "__global__")
	} else {
		e.cache = make(map[string]*policyCacheEntry)
	}
}

func (e *PolicyEngine) fallbackToDangerLevel(dangerLevel int) PolicyEvaluateResult {
	switch dangerLevel {
	case types.DangerLevelSilent:
		return PolicyEvaluateResult{Decision: types.PolicyDecisionAllow, PolicyVersion: 0}
	case types.DangerLevelNotification:
		return PolicyEvaluateResult{Decision: types.PolicyDecisionAllow, PolicyVersion: 0}
	case types.DangerLevelApproval:
		return PolicyEvaluateResult{Decision: types.PolicyDecisionRequireApproval, PolicyVersion: 0}
	default:
		return PolicyEvaluateResult{Decision: types.PolicyDecisionDeny, PolicyVersion: 0}
	}
}

func mapPolicyDecision(decision string) string {
	normalized := NormalizePolicyDecision(decision)
	if normalized == "" {
		return types.PolicyDecisionDeny
	}
	return normalized
}
