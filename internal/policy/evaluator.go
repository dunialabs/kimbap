package policy

import (
	"context"
	"fmt"
	"path"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
)

type EvalRequest struct {
	TenantID  string
	AgentName string
	Service   string
	Action    string
	Risk      string
	Mutating  bool
	Args      map[string]any
}

type EvalResult struct {
	Decision    PolicyDecision
	MatchedRule *PolicyRule
	Reason      string
	RateStatus  *RateStatus
}

type RateStatus struct {
	Allowed   bool
	Remaining int
	ResetAt   time.Time
}

type Evaluator struct {
	document    *PolicyDocument
	sortedRules []PolicyRule
	rl          rateLimiter
}

func NewEvaluator(doc *PolicyDocument) *Evaluator {
	rules := make([]PolicyRule, 0)
	if doc != nil {
		rules = make([]PolicyRule, len(doc.Rules))
		copy(rules, doc.Rules)
		sort.SliceStable(rules, func(i, j int) bool {
			return rules[i].Priority < rules[j].Priority
		})
	}
	return &Evaluator{
		document:    doc,
		sortedRules: rules,
		rl:          rateLimiter{windows: make(map[string][]time.Time)},
	}
}

type rateLimiter struct {
	mu            sync.Mutex
	windows       map[string][]time.Time
	expires       map[string]time.Time
	lastSweep     time.Time
	sweepInterval time.Duration
}

func (rl *rateLimiter) check(key string, maxRequests int, windowSec int) *RateStatus {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if rl.expires == nil {
		rl.expires = make(map[string]time.Time)
	}
	if rl.sweepInterval <= 0 {
		rl.sweepInterval = time.Minute
	}

	now := time.Now().UTC()
	windowDur := time.Duration(windowSec) * time.Second
	cutoff := now.Add(-windowDur)

	timestamps := rl.windows[key]
	pruned := make([]time.Time, 0, len(timestamps))
	for _, ts := range timestamps {
		if ts.After(cutoff) {
			pruned = append(pruned, ts)
		}
	}

	allowed := len(pruned) < maxRequests
	if allowed {
		pruned = append(pruned, now)
	}
	if len(pruned) == 0 {
		delete(rl.windows, key)
		delete(rl.expires, key)
	} else {
		rl.windows[key] = pruned
		rl.expires[key] = pruned[len(pruned)-1].Add(windowDur)
	}

	if now.Sub(rl.lastSweep) >= rl.sweepInterval {
		for k, expiresAt := range rl.expires {
			if !expiresAt.After(now) {
				delete(rl.windows, k)
				delete(rl.expires, k)
			}
		}
		rl.lastSweep = now
	}

	remaining := maxRequests - len(pruned)
	if remaining < 0 {
		remaining = 0
	}
	resetAt := now.Add(windowDur)
	if len(pruned) > 0 {
		resetAt = pruned[0].Add(windowDur)
	}
	return &RateStatus{
		Allowed:   allowed,
		Remaining: remaining,
		ResetAt:   resetAt,
	}
}

func (e *Evaluator) Evaluate(_ context.Context, req EvalRequest) (*EvalResult, error) {
	if e == nil || e.document == nil {
		return nil, fmt.Errorf("policy document is required")
	}

	for i := range e.sortedRules {
		rule := e.sortedRules[i]
		if !matchesRule(rule.Match, req) {
			continue
		}
		ok, err := conditionsPass(rule.Conditions, req)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		if rule.TimeWindow != nil && !timeWindowActive(rule.TimeWindow, time.Now()) {
			continue
		}

		res := &EvalResult{
			Decision:    rule.Decision,
			MatchedRule: &rule,
			Reason:      "matched rule " + rule.ID,
		}
		if rule.RateLimit != nil && rule.RateLimit.MaxRequests > 0 && rule.RateLimit.WindowSec > 0 {
			keyPart := rateLimitKeyPart(rule.RateLimit.Scope, req)
			key := fmt.Sprintf("%s:%s", keyPart, rule.ID)
			status := e.rl.check(key, rule.RateLimit.MaxRequests, rule.RateLimit.WindowSec)
			res.RateStatus = status
			if !status.Allowed {
				res.Decision = DecisionDeny
				attempted := rule.RateLimit.MaxRequests - status.Remaining + 1
				res.Reason = fmt.Sprintf("rate limit exceeded for rule %s (%d/%d)", rule.ID, attempted, rule.RateLimit.MaxRequests)
			}
		}
		return res, nil
	}

	return &EvalResult{
		Decision: DecisionDeny,
		Reason:   "no matching policy rule",
	}, nil
}

func rateLimitKeyPart(scope string, req EvalRequest) string {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "tenant":
		return req.TenantID
	case "agent":
		return fmt.Sprintf("%s:%s", req.TenantID, req.AgentName)
	case "action":
		return fmt.Sprintf("%s:%s:%s:%s", req.TenantID, req.AgentName, req.Service, req.Action)
	default:
		return fmt.Sprintf("%s:%s:%s", req.TenantID, req.AgentName, req.Service)
	}
}

func timeWindowActive(tw *TimeWindow, now time.Time) bool {
	if tw == nil {
		return true
	}

	loc := time.UTC
	if tz := strings.TrimSpace(tw.Timezone); tz != "" {
		if parsed, err := time.LoadLocation(tz); err == nil {
			loc = parsed
		}
	}
	now = now.In(loc)

	if len(tw.Weekdays) > 0 {
		todayName := strings.ToLower(now.Weekday().String())
		todayShort := todayName[:3]
		found := false
		for _, wd := range tw.Weekdays {
			wd = strings.ToLower(strings.TrimSpace(wd))
			if wd == todayName || wd == todayShort {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	nowMinutes := now.Hour()*60 + now.Minute()

	afterStr := strings.TrimSpace(tw.After)
	beforeStr := strings.TrimSpace(tw.Before)
	afterMin, afterOK := parseHHMM(afterStr)
	beforeMin, beforeOK := parseHHMM(beforeStr)

	if afterOK && beforeOK {
		if afterMin > beforeMin {
			return nowMinutes >= afterMin || nowMinutes < beforeMin
		}
		return nowMinutes >= afterMin && nowMinutes < beforeMin
	}
	if afterOK && nowMinutes < afterMin {
		return false
	}
	if beforeOK && nowMinutes >= beforeMin {
		return false
	}

	return true
}

func parseHHMM(s string) (int, bool) {
	t, err := time.Parse("15:04", strings.TrimSpace(s))
	if err != nil {
		return 0, false
	}
	return t.Hour()*60 + t.Minute(), true
}

func matchesRule(match PolicyMatch, req EvalRequest) bool {
	if !matchesAny(match.Agents, req.AgentName) {
		return false
	}
	if !matchesAny(match.Services, req.Service) {
		return false
	}
	if !matchesAny(match.Actions, req.Action) {
		return false
	}
	if !matchesAny(match.Risk, req.Risk) {
		return false
	}
	if !matchesAny(match.Tenants, req.TenantID) {
		return false
	}
	return true
}

func matchesAny(patterns []string, value string) bool {
	if len(patterns) == 0 {
		return true
	}
	for _, pattern := range patterns {
		ok, err := path.Match(pattern, value)
		if err == nil && ok {
			return true
		}
	}
	return false
}

func conditionsPass(conditions []PolicyCondition, req EvalRequest) (bool, error) {
	for _, cond := range conditions {
		fieldValue, ok := resolveFieldValue(req, cond.Field)
		if !ok {
			return false, nil
		}
		passed, err := evaluateCondition(fieldValue, strings.ToLower(strings.TrimSpace(cond.Operator)), cond.Value)
		if err != nil {
			return false, err
		}
		if !passed {
			return false, nil
		}
	}
	return true, nil
}

func resolveFieldValue(req EvalRequest, field string) (any, bool) {
	root := map[string]any{
		"tenant_id":  req.TenantID,
		"agent_name": req.AgentName,
		"service":    req.Service,
		"action":     req.Action,
		"risk": map[string]any{
			"level":    req.Risk,
			"mutating": req.Mutating,
		},
		"args": req.Args,
	}

	parts := strings.Split(field, ".")
	var cur any = root
	for _, part := range parts {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		next, exists := m[part]
		if !exists {
			return nil, false
		}
		cur = next
	}
	return cur, true
}

func evaluateCondition(actual any, operator string, expected any) (bool, error) {
	switch operator {
	case "eq":
		return valuesEqual(actual, expected), nil
	case "ne":
		return !valuesEqual(actual, expected), nil
	case "gt":
		return compareNumber(actual, expected, ">")
	case "lt":
		return compareNumber(actual, expected, "<")
	case "gte":
		return compareNumber(actual, expected, ">=")
	case "lte":
		return compareNumber(actual, expected, "<=")
	case "in":
		values, ok := toSlice(expected)
		if !ok {
			return false, fmt.Errorf("operator in expects array value")
		}
		for _, item := range values {
			if valuesEqual(actual, item) {
				return true, nil
			}
		}
		return false, nil
	case "contains":
		if str, ok := actual.(string); ok {
			part, ok := expected.(string)
			if !ok {
				return false, fmt.Errorf("contains over string requires string expected value")
			}
			return strings.Contains(str, part), nil
		}
		arr, ok := toSlice(actual)
		if !ok {
			return false, fmt.Errorf("contains expects string or array field")
		}
		for _, item := range arr {
			if valuesEqual(item, expected) {
				return true, nil
			}
		}
		return false, nil
	default:
		return false, fmt.Errorf("unsupported operator: %s", operator)
	}
}

func valuesEqual(a any, b any) bool {
	if af, ok := toFloat64(a); ok {
		if bf, ok := toFloat64(b); ok {
			return af == bf
		}
	}
	return reflect.DeepEqual(a, b)
}

func compareNumber(actual any, expected any, op string) (bool, error) {
	left, ok := toFloat64(actual)
	if !ok {
		return false, fmt.Errorf("actual value is not numeric")
	}
	right, ok := toFloat64(expected)
	if !ok {
		return false, fmt.Errorf("expected value is not numeric")
	}

	switch op {
	case ">":
		return left > right, nil
	case "<":
		return left < right, nil
	case ">=":
		return left >= right, nil
	case "<=":
		return left <= right, nil
	default:
		return false, fmt.Errorf("unsupported numeric operator %s", op)
	}
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}

func toSlice(v any) ([]any, bool) {
	switch s := v.(type) {
	case []any:
		return s, true
	case []string:
		out := make([]any, 0, len(s))
		for _, item := range s {
			out = append(out, item)
		}
		return out, true
	case []int:
		out := make([]any, 0, len(s))
		for _, item := range s {
			out = append(out, item)
		}
		return out, true
	default:
		return nil, false
	}
}
