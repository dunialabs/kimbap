package classifier

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/dunialabs/kimbap/internal/services"
)

var pathParamRe = regexp.MustCompile(`\{[^/}]+\}`)

type Rule struct {
	ID          string
	Service     string
	Action      string
	HostPattern string
	PathPattern string
	Method      string
	Priority    int
}

type ClassificationResult struct {
	Matched    bool
	Service    string
	Action     string
	RuleID     string
	Confidence string
}

var ErrActionCollision = fmt.Errorf("action name collision detected")

type Classifier struct {
	rules     []Rule
	actionIDs map[string]string
}

func NewClassifier() *Classifier {
	return &Classifier{rules: make([]Rule, 0), actionIDs: make(map[string]string)}
}

func (c *Classifier) AddRule(rule Rule) error {
	rule.Method = normalizeMethod(rule.Method)
	rule.HostPattern = normalizeHostPattern(rule.HostPattern)
	rule.PathPattern = normalizePathPattern(rule.PathPattern)

	canonicalID := rule.Service + "." + rule.Action
	if existingSource, ok := c.actionIDs[canonicalID]; ok && existingSource != rule.ID {
		return fmt.Errorf("%w: %q registered by both %q and %q", ErrActionCollision, canonicalID, existingSource, rule.ID)
	}
	c.actionIDs[canonicalID] = rule.ID
	c.rules = append(c.rules, rule)
	c.sortRules()
	return nil
}

func (c *Classifier) AddRulesFromService(svc *services.ServiceManifest) error {
	if svc == nil {
		return nil
	}

	// Skip non-HTTP adapters (e.g., applescript) — they have no HTTP classification rules
	adapterType := strings.ToLower(strings.TrimSpace(svc.Adapter))
	if adapterType != "" && adapterType != "http" {
		return nil
	}

	hostPattern := "*"
	basePath := ""
	if strings.TrimSpace(svc.BaseURL) != "" {
		u, err := url.Parse(svc.BaseURL)
		if err != nil || !u.IsAbs() || strings.TrimSpace(u.Host) == "" {
			return fmt.Errorf("invalid base URL for service %q: %q", strings.TrimSpace(svc.Name), svc.BaseURL)
		}
		hostPattern = normalizeHostPattern(u.Host)
		basePath = strings.TrimSuffix(u.Path, "/")
	}

	actionNames := make([]string, 0, len(svc.Actions))
	for actionName := range svc.Actions {
		actionNames = append(actionNames, actionName)
	}
	sort.Strings(actionNames)

	for _, actionName := range actionNames {
		action := svc.Actions[actionName]
		actionPath := normalizePathPattern(joinURLPath(basePath, action.Path))
		id := fmt.Sprintf("service:%s:%s", strings.TrimSpace(svc.Name), actionName)
		if err := c.AddRule(Rule{
			ID:          id,
			Service:     strings.TrimSpace(svc.Name),
			Action:      actionName,
			HostPattern: hostPattern,
			PathPattern: actionPath,
			Method:      normalizeMethod(action.Method),
			Priority:    100,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (c *Classifier) Classify(method, host, reqPath string) *ClassificationResult {
	nMethod := normalizeMethod(method)
	nHost := normalizeHostPattern(host)
	nPath := normalizePathPattern(reqPath)

	for _, rule := range c.rules {
		if !matchMethod(rule.Method, nMethod) {
			continue
		}
		if !matchHostPattern(rule.HostPattern, nHost) {
			continue
		}
		if !matchGlob(rule.PathPattern, nPath) {
			continue
		}

		return &ClassificationResult{
			Matched:    true,
			Service:    rule.Service,
			Action:     rule.Action,
			RuleID:     rule.ID,
			Confidence: confidenceFor(rule),
		}
	}

	return &ClassificationResult{Matched: false, Confidence: "none"}
}

func (c *Classifier) Explain(method, host, reqPath string) string {
	res := c.Classify(method, host, reqPath)
	if res == nil || !res.Matched {
		return fmt.Sprintf("no matching rule for %s %s%s", normalizeMethod(method), normalizeHostPattern(host), normalizePathPattern(reqPath))
	}

	return fmt.Sprintf(
		"matched rule %q for %s %s%s -> service=%q action=%q confidence=%s",
		res.RuleID,
		normalizeMethod(method),
		normalizeHostPattern(host),
		normalizePathPattern(reqPath),
		res.Service,
		res.Action,
		res.Confidence,
	)
}

func (c *Classifier) sortRules() {
	sort.SliceStable(c.rules, func(i, j int) bool {
		if c.rules[i].Priority == c.rules[j].Priority {
			return c.rules[i].ID < c.rules[j].ID
		}
		return c.rules[i].Priority > c.rules[j].Priority
	})
}

func matchMethod(ruleMethod, method string) bool {
	n := normalizeMethod(ruleMethod)
	if n == "*" {
		return true
	}
	return n == normalizeMethod(method)
}

func matchGlob(pattern, value string) bool {
	if pattern == "" || pattern == "*" {
		return true
	}
	matched, err := path.Match(pattern, value)
	if err != nil {
		return false
	}
	return matched
}

func matchHostPattern(pattern, host string) bool {
	if matchGlob(pattern, host) {
		return true
	}
	if strings.Contains(pattern, ":") {
		return false
	}
	if parsed, err := url.Parse("http://" + host); err == nil && parsed.Hostname() != "" {
		return matchGlob(pattern, strings.ToLower(parsed.Hostname()))
	}
	return false
}

func confidenceFor(rule Rule) string {
	if !hasGlob(rule.HostPattern) && !hasGlob(rule.PathPattern) && normalizeMethod(rule.Method) != "*" {
		return "exact"
	}
	return "pattern"
}

func hasGlob(s string) bool {
	return strings.ContainsAny(s, "*?[")
}

func normalizeMethod(method string) string {
	m := strings.TrimSpace(strings.ToUpper(method))
	if m == "" {
		return "*"
	}
	return m
}

func normalizeHostPattern(host string) string {
	h := strings.TrimSpace(strings.ToLower(host))
	if h == "" {
		return "*"
	}
	if hasGlob(h) {
		return h
	}
	if strings.Contains(h, "://") {
		if parsed, err := url.Parse(h); err == nil && parsed.Host != "" {
			h = strings.TrimSpace(strings.ToLower(parsed.Host))
		}
	}
	if parsed, err := url.Parse("http://" + h); err == nil && parsed.Host != "" {
		h = strings.ToLower(parsed.Host)
	}
	return h
}

func normalizePathPattern(reqPath string) string {
	p := strings.TrimSpace(reqPath)
	if p == "" {
		return "/"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if u, err := url.Parse(p); err == nil {
		if u.Path != "" {
			p = u.Path
		}
	}
	p = pathParamRe.ReplaceAllString(p, "*")
	return p
}

func joinURLPath(basePath, actionPath string) string {
	b := strings.TrimSuffix(strings.TrimSpace(basePath), "/")
	a := strings.TrimSpace(actionPath)
	if a == "" {
		if b == "" {
			return "/"
		}
		return b
	}
	if !strings.HasPrefix(a, "/") {
		a = "/" + a
	}
	if b == "" {
		return a
	}
	return b + a
}
