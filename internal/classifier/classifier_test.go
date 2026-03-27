package classifier

import (
	"strings"
	"testing"

	"github.com/dunialabs/kimbap/internal/services"
)

func mustAddRule(t *testing.T, c *Classifier, rule Rule) {
	t.Helper()
	if err := c.AddRule(rule); err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}
}

func TestClassifierExactHostPathMatch(t *testing.T) {
	c := NewClassifier()
	mustAddRule(t, c, Rule{
		ID:          "r1",
		Service:     "brave_search",
		Action:      "web_search",
		HostPattern: "api.search.brave.com",
		PathPattern: "/res/v1/web/search",
		Method:      "GET",
		Priority:    10,
	})

	result := c.Classify("GET", "api.search.brave.com", "/res/v1/web/search")
	if result == nil || !result.Matched {
		t.Fatal("expected exact match")
	}
	if result.RuleID != "r1" || result.Service != "brave_search" || result.Action != "web_search" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.Confidence != "exact" {
		t.Fatalf("expected exact confidence, got %q", result.Confidence)
	}
}

func TestClassifierGlobPatternMatching(t *testing.T) {
	c := NewClassifier()
	mustAddRule(t, c, Rule{
		ID:          "glob",
		Service:     "service",
		Action:      "list",
		HostPattern: "*.example.com",
		PathPattern: "/v1/*/items",
		Method:      "*",
		Priority:    1,
	})

	result := c.Classify("POST", "api.example.com", "/v1/foo/items")
	if result == nil || !result.Matched {
		t.Fatal("expected glob match")
	}
	if result.Confidence != "pattern" {
		t.Fatalf("expected pattern confidence, got %q", result.Confidence)
	}
}

func TestClassifierMethodFiltering(t *testing.T) {
	c := NewClassifier()
	mustAddRule(t, c, Rule{
		ID:          "post-only",
		Service:     "svc",
		Action:      "create",
		HostPattern: "api.service.local",
		PathPattern: "/items",
		Method:      "POST",
		Priority:    1,
	})

	if result := c.Classify("GET", "api.service.local", "/items"); result.Matched {
		t.Fatalf("expected method mismatch, got %#v", result)
	}

	if result := c.Classify("POST", "api.service.local", "/items"); !result.Matched {
		t.Fatal("expected POST to match")
	}
}

func TestClassifierPriorityOrdering(t *testing.T) {
	c := NewClassifier()
	mustAddRule(t, c, Rule{
		ID:          "low",
		Service:     "svc",
		Action:      "fallback",
		HostPattern: "api.service.local",
		PathPattern: "/items/*",
		Method:      "*",
		Priority:    1,
	})
	mustAddRule(t, c, Rule{
		ID:          "high",
		Service:     "svc",
		Action:      "specific",
		HostPattern: "api.service.local",
		PathPattern: "/items/42",
		Method:      "GET",
		Priority:    50,
	})

	result := c.Classify("GET", "api.service.local", "/items/42")
	if !result.Matched {
		t.Fatal("expected match")
	}
	if result.RuleID != "high" {
		t.Fatalf("expected high-priority rule, got %q", result.RuleID)
	}
}

func TestClassifierAddRulesFromServiceManifest(t *testing.T) {
	c := NewClassifier()
	if err := c.AddRulesFromService(&services.ServiceManifest{
		Name:    "brave_search",
		BaseURL: "https://api.search.brave.com/res/v1",
		Actions: map[string]services.ServiceAction{
			"web_search": {
				Method: "GET",
				Path:   "/web/search",
			},
		},
	}); err != nil {
		t.Fatalf("AddRulesFromService failed: %v", err)
	}

	result := c.Classify("GET", "api.search.brave.com", "/res/v1/web/search")
	if !result.Matched {
		t.Fatal("expected manifest-generated rule match")
	}
	if result.Service != "brave_search" || result.Action != "web_search" {
		t.Fatalf("unexpected generated classification: %#v", result)
	}
	if !strings.HasPrefix(result.RuleID, "service:brave_search:") {
		t.Fatalf("unexpected generated rule id: %q", result.RuleID)
	}
}

func TestClassifierExplainOutput(t *testing.T) {
	c := NewClassifier()
	mustAddRule(t, c, Rule{
		ID:          "expl",
		Service:     "svc",
		Action:      "act",
		HostPattern: "api.example.com",
		PathPattern: "/path",
		Method:      "GET",
		Priority:    1,
	})

	explainMatched := c.Explain("GET", "api.example.com", "/path")
	if !strings.Contains(explainMatched, "matched rule \"expl\"") {
		t.Fatalf("expected matched explanation, got %q", explainMatched)
	}

	explainNone := c.Explain("POST", "api.example.com", "/path")
	if !strings.Contains(explainNone, "no matching rule") {
		t.Fatalf("expected no-match explanation, got %q", explainNone)
	}
}

func TestClassifierHostWithoutPortMatchesRequestHostWithPort(t *testing.T) {
	c := NewClassifier()
	mustAddRule(t, c, Rule{
		ID:          "no-port",
		Service:     "svc",
		Action:      "act",
		HostPattern: "api.example.com",
		PathPattern: "/path",
		Method:      "GET",
		Priority:    1,
	})

	result := c.Classify("GET", "api.example.com:8443", "/path")
	if result == nil || !result.Matched {
		t.Fatal("expected host without port pattern to match host with port")
	}
}

func TestClassifierHostWithPortDistinguishesPorts(t *testing.T) {
	c := NewClassifier()
	mustAddRule(t, c, Rule{
		ID:          "with-port",
		Service:     "svc",
		Action:      "act",
		HostPattern: "api.example.com:8443",
		PathPattern: "/path",
		Method:      "GET",
		Priority:    1,
	})

	if result := c.Classify("GET", "api.example.com:8443", "/path"); result == nil || !result.Matched {
		t.Fatal("expected exact port match")
	}
	if result := c.Classify("GET", "api.example.com:443", "/path"); result != nil && result.Matched {
		t.Fatalf("expected port mismatch, got %#v", result)
	}
}

func TestClassifierAddRulesFromServiceRejectsInvalidBaseURL(t *testing.T) {
	c := NewClassifier()
	err := c.AddRulesFromService(&services.ServiceManifest{
		Name:    "invalid",
		BaseURL: "/relative/path",
		Actions: map[string]services.ServiceAction{
			"list": {Method: "GET", Path: "/items"},
		},
	})
	if err == nil {
		t.Fatal("expected invalid base URL to be rejected")
	}
}

func TestClassifierSkipsAppleScript(t *testing.T) {
	c := NewClassifier()
	err := c.AddRulesFromService(&services.ServiceManifest{
		Name:    "applescript_service",
		Adapter: "applescript",
		Actions: map[string]services.ServiceAction{
			"run": {Method: "POST", Path: "/run"},
		},
	})
	if err != nil {
		t.Fatalf("AddRulesFromService failed: %v", err)
	}
	if len(c.rules) != 0 {
		t.Fatalf("expected no rules to be added, got %d", len(c.rules))
	}
}

func TestClassifierHTTPUnchanged(t *testing.T) {
	c := NewClassifier()
	err := c.AddRulesFromService(&services.ServiceManifest{
		Name:    "http_service",
		Adapter: "http",
		BaseURL: "https://api.example.com/v1",
		Actions: map[string]services.ServiceAction{
			"list": {Method: "GET", Path: "/items"},
		},
	})
	if err != nil {
		t.Fatalf("AddRulesFromService failed: %v", err)
	}
	if len(c.rules) != 1 {
		t.Fatalf("expected one rule to be added, got %d", len(c.rules))
	}
	result := c.Classify("GET", "api.example.com", "/v1/items")
	if result == nil || !result.Matched {
		t.Fatal("expected HTTP manifest to produce a matching rule")
	}
}
