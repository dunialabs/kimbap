package services

import (
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/dunialabs/kimbap-core/internal/types"
)

type PolicyRule struct {
	ID       string                     `json:"id"`
	Priority *int                       `json:"priority,omitempty"`
	Match    PolicyMatch                `json:"match"`
	Extract  map[string]PolicyExtractor `json:"extract,omitempty"`
	When     []PolicyCondition          `json:"when,omitempty"`
	Effect   PolicyEffect               `json:"effect"`
}

type PolicyMatch struct {
	Tool     *string `json:"tool,omitempty"`
	ServerID *string `json:"serverId,omitempty"`
}

type PolicyExtractor struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

type PolicyCondition struct {
	Left  any    `json:"left"`
	Op    string `json:"op"`
	Right any    `json:"right"`
}

type PolicyEffect struct {
	Decision string  `json:"decision"`
	Reason   *string `json:"reason,omitempty"`
}

type PolicyDsl struct {
	Rules []PolicyRule `json:"rules"`
}

type PolicyEvaluationContext struct {
	ServerID *string
	ToolName string
	Args     map[string]any
}

type PolicyEvaluationResult struct {
	Decision      string
	MatchedRuleID *string
	Reason        *string
}

type PolicyDslEvaluator struct{}

func NewPolicyDslEvaluator() *PolicyDslEvaluator { return &PolicyDslEvaluator{} }

var policyDslEvaluator = NewPolicyDslEvaluator()

func PolicyDslEvaluatorInstance() *PolicyDslEvaluator { return policyDslEvaluator }

func NormalizePolicyDecision(decision string) string {
	switch strings.ToUpper(strings.TrimSpace(decision)) {
	case types.PolicyDecisionAllow:
		return types.PolicyDecisionAllow
	case types.PolicyDecisionRequireApproval:
		return types.PolicyDecisionRequireApproval
	case types.PolicyDecisionDeny:
		return types.PolicyDecisionDeny
	default:
		return ""
	}
}

func IsValidPolicyDecision(decision string) bool {
	return NormalizePolicyDecision(decision) != ""
}

func ValidatePolicyDSL(dsl PolicyDsl) error {
	if len(dsl.Rules) == 0 {
		return fmt.Errorf("policy must contain at least one rule")
	}
	for i, rule := range dsl.Rules {
		if !IsValidPolicyDecision(rule.Effect.Decision) {
			return fmt.Errorf("invalid policy decision %q at rule index %d", rule.Effect.Decision, i)
		}
		for name, extractor := range rule.Extract {
			if !isSupportedExtractorType(extractor.Type) {
				return fmt.Errorf("unsupported extractor type %q for %q at rule index %d", extractor.Type, name, i)
			}
		}
		for j, condition := range rule.When {
			if !isSupportedPolicyOperator(condition.Op) {
				return fmt.Errorf("unsupported policy operator %q at rule index %d condition index %d", condition.Op, i, j)
			}
			if strings.EqualFold(strings.TrimSpace(condition.Op), "matches") {
				pattern, ok := condition.Right.(string)
				if !ok {
					return fmt.Errorf("matches operator requires string pattern at rule index %d condition index %d", i, j)
				}
				if strings.HasPrefix(pattern, "$") {
					continue
				}
				if len(pattern) > 512 {
					return fmt.Errorf("regex pattern too long at rule index %d condition index %d", i, j)
				}
				if _, err := regexp.Compile(pattern); err != nil {
					return fmt.Errorf("invalid regex pattern at rule index %d condition index %d: %w", i, j, err)
				}
			}
		}
	}
	return nil
}

func isSupportedExtractorType(t string) bool {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "string", "number", "boolean", "url.host", "bytes.length":
		return true
	default:
		return false
	}
}

func isSupportedPolicyOperator(op string) bool {
	switch strings.ToLower(strings.TrimSpace(op)) {
	case "eq", "neq", "gt", "gte", "lt", "lte", "in", "not_in", "matches":
		return true
	default:
		return false
	}
}

func (e *PolicyDslEvaluator) Evaluate(dsl PolicyDsl, context PolicyEvaluationContext) PolicyEvaluationResult {
	rules := append([]PolicyRule(nil), dsl.Rules...)
	sort.SliceStable(rules, func(i, j int) bool {
		left := 1000
		right := 1000
		if rules[i].Priority != nil {
			left = *rules[i].Priority
		}
		if rules[j].Priority != nil {
			right = *rules[j].Priority
		}
		return left < right
	})

	for _, rule := range rules {
		if !e.matchesRule(rule, context) {
			continue
		}
		extractedVars, extractOK := e.extractVariables(rule.Extract, context.Args)
		if !extractOK && len(rule.Extract) > 0 {
			extractFailReason := "extraction failed for required variables"
			return PolicyEvaluationResult{Decision: types.PolicyDecisionDeny, MatchedRuleID: &rule.ID, Reason: &extractFailReason}
		}
		scope := map[string]any{}
		for k, v := range context.Args {
			scope[k] = v
		}
		for k, v := range extractedVars {
			scope[k] = v
		}
		scope["args"] = context.Args

		allPass := true
		for _, condition := range rule.When {
			left := e.resolveOperand(condition.Left, scope)
			right := e.resolveOperand(condition.Right, scope)
			if !e.evaluateCondition(left, condition.Op, right) {
				allPass = false
				break
			}
		}
		if !allPass {
			continue
		}
		normalizedDecision := NormalizePolicyDecision(rule.Effect.Decision)
		if normalizedDecision == "" {
			normalizedDecision = types.PolicyDecisionDeny
		}
		return PolicyEvaluationResult{Decision: normalizedDecision, MatchedRuleID: &rule.ID, Reason: rule.Effect.Reason}
	}

	return PolicyEvaluationResult{Decision: types.PolicyDecisionAllow}
}

func (e *PolicyDslEvaluator) matchesRule(rule PolicyRule, context PolicyEvaluationContext) bool {
	if rule.Match.ServerID != nil {
		if context.ServerID == nil || !e.matchGlob(*rule.Match.ServerID, *context.ServerID) {
			return false
		}
	}
	if rule.Match.Tool != nil && !e.matchGlob(*rule.Match.Tool, context.ToolName) {
		return false
	}
	return true
}

func (e *PolicyDslEvaluator) extractVariables(extractors map[string]PolicyExtractor, args map[string]any) (map[string]any, bool) {
	out := map[string]any{}
	for name, extractor := range extractors {
		raw := e.extractByPath(args, extractor.Path)
		if raw == nil {
			return nil, false
		}
		coerced := e.coerceType(raw, extractor.Type)
		if coerced == nil {
			return nil, false
		}
		out[name] = coerced
	}
	return out, true
}

func (e *PolicyDslEvaluator) coerceType(value any, kind string) any {
	switch kind {
	case "string":
		return fmt.Sprintf("%v", value)
	case "number":
		v, ok := toFloat64(value)
		if !ok {
			return nil
		}
		return v
	case "boolean":
		switch typed := value.(type) {
		case bool:
			return typed
		case string:
			v := strings.ToLower(strings.TrimSpace(typed))
			if v == "true" {
				return true
			}
			if v == "false" {
				return false
			}
		default:
			if n, ok := toFloat64(typed); ok {
				return n != 0
			}
		}
		return value != nil
	case "url.host":
		parsed, err := url.Parse(fmt.Sprintf("%v", value))
		if err != nil {
			return nil
		}
		return parsed.Hostname()
	case "bytes.length":
		return len([]byte(fmt.Sprintf("%v", value)))
	default:
		return nil
	}
}

func (e *PolicyDslEvaluator) resolveOperand(operand any, scope map[string]any) any {
	text, ok := operand.(string)
	if !ok {
		return operand
	}
	// $-prefixed strings are explicit variable references into the scope
	if strings.HasPrefix(text, "$") {
		return e.extractByPath(scope, strings.TrimPrefix(text, "$"))
	}
	// Plain strings are always treated as literal values — no path fallback
	return text
}

func (e *PolicyDslEvaluator) evaluateCondition(left any, op string, right any) bool {
	switch op {
	case "eq":
		return reflect.DeepEqual(left, right)
	case "neq":
		return !reflect.DeepEqual(left, right)
	case "gt":
		return compareNumbers(left, right, func(a, b float64) bool { return a > b })
	case "gte":
		return compareNumbers(left, right, func(a, b float64) bool { return a >= b })
	case "lt":
		return compareNumbers(left, right, func(a, b float64) bool { return a < b })
	case "lte":
		return compareNumbers(left, right, func(a, b float64) bool { return a <= b })
	case "in":
		return contains(right, left)
	case "not_in":
		return !contains(right, left)
	case "matches":
		pattern := fmt.Sprintf("%v", right)
		if len(pattern) > 512 {
			return false
		}
		subject := fmt.Sprintf("%v", left)
		if len(subject) > 4096 {
			return false
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return false
		}
		return re.MatchString(subject)
	default:
		return false
	}
}

func (e *PolicyDslEvaluator) matchGlob(pattern, value string) bool {
	if len(pattern) > 512 {
		return false
	}
	var b strings.Builder
	b.WriteString("^")
	for _, ch := range pattern {
		switch ch {
		case '*':
			b.WriteString(".*")
		case '?':
			b.WriteString(".")
		case '\\', '^', '$', '+', '.', '(', ')', '|', '{', '}', '[', ']':
			b.WriteString("\\")
			b.WriteRune(ch)
		default:
			b.WriteRune(ch)
		}
	}
	b.WriteString("$")
	re, err := regexp.Compile(b.String())
	if err != nil {
		return false
	}
	return re.MatchString(value)
}

func (e *PolicyDslEvaluator) extractByPath(obj any, path string) any {
	if strings.TrimSpace(path) == "" {
		return obj
	}
	current := obj
	for _, segment := range strings.Split(path, ".") {
		asMap, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		next, exists := asMap[segment]
		if !exists {
			return nil
		}
		current = next
	}
	return current
}

func compareNumbers(left any, right any, cmp func(float64, float64) bool) bool {
	a, ok := toFloat64(left)
	if !ok {
		return false
	}
	b, ok := toFloat64(right)
	if !ok {
		return false
	}
	return cmp(a, b)
}

func contains(container any, needle any) bool {
	v := reflect.ValueOf(container)
	if !v.IsValid() || (v.Kind() != reflect.Slice && v.Kind() != reflect.Array) {
		return false
	}
	for i := 0; i < v.Len(); i++ {
		if reflect.DeepEqual(v.Index(i).Interface(), needle) {
			return true
		}
	}
	return false
}

func toFloat64(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int8:
		return float64(typed), true
	case int16:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case uint:
		return float64(typed), true
	case uint8:
		return float64(typed), true
	case uint16:
		return float64(typed), true
	case uint32:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}
