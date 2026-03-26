package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateAppleScriptManifest_Valid(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "testdata", "apple-notes.yaml")
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	m, err := ParseManifest(data)
	if err != nil {
		t.Fatalf("expected parse success, got %v", err)
	}

	errList := ValidateManifest(m)
	if len(errList) != 0 {
		t.Fatalf("expected 0 validation errors, got %v", errList)
	}
}

func TestValidateAppleScriptManifest_MissingTargetApp(t *testing.T) {
	m := validAppleScriptManifest()
	m.TargetApp = ""

	errList := ValidateManifest(m)
	if !hasValidationError(errList, "target_app", "must be set") {
		t.Fatalf("expected target_app error, got %v", errList)
	}
}

func TestValidateAppleScriptManifest_MissingCommand(t *testing.T) {
	m := validAppleScriptManifest()
	a := m.Actions["list_notes"]
	a.Command = ""
	m.Actions["list_notes"] = a

	errList := ValidateManifest(m)
	if !hasValidationError(errList, "actions.list_notes.command", "is required") {
		t.Fatalf("expected command required error, got %v", errList)
	}
}

func TestValidateAppleScriptManifest_UnknownCommand(t *testing.T) {
	m := validAppleScriptManifest()
	a := m.Actions["list_notes"]
	a.Command = "hack-the-planet"
	m.Actions["list_notes"] = a

	errList := ValidateManifest(m)
	if !hasValidationError(errList, "actions.list_notes.command", "supported applescript command") {
		t.Fatalf("expected command validation error, got %v", errList)
	}
}

func TestValidateAppleScriptManifest_RejectsHTTPFields(t *testing.T) {
	m := validAppleScriptManifest()
	m.BaseURL = "https://example.com"

	errList := ValidateManifest(m)
	if !hasValidationError(errList, "base_url", "must not be set for applescript adapter") {
		t.Fatalf("expected base_url applescript rejection, got %v", errList)
	}
}

func TestValidateAppleScriptManifest_RejectsMethodPath(t *testing.T) {
	m := validAppleScriptManifest()
	a := m.Actions["list_notes"]
	a.Method = "GET"
	a.Path = "/notes"
	m.Actions["list_notes"] = a

	errList := ValidateManifest(m)
	if !hasValidationError(errList, "actions.list_notes.method", "must not be set for applescript adapter") {
		t.Fatalf("expected method rejection error, got %v", errList)
	}
	if !hasValidationError(errList, "actions.list_notes.path", "must not be set for applescript adapter") {
		t.Fatalf("expected path rejection error, got %v", errList)
	}
}

func TestValidateHTTPManifest_RejectsAppleScriptFields(t *testing.T) {
	m := validHTTPManifest()
	m.TargetApp = "Notes"

	errList := ValidateManifest(m)
	if !hasValidationError(errList, "target_app", "must not be set for http adapter") {
		t.Fatalf("expected target_app rejection for http adapter, got %v", errList)
	}
}

func TestValidateManifest_UnknownAdapter(t *testing.T) {
	m := validHTTPManifest()
	m.Adapter = "websocket"

	errList := ValidateManifest(m)
	if !hasValidationError(errList, "adapter", "must be one of http, applescript") {
		t.Fatalf("expected adapter validation error, got %v", errList)
	}
}

func TestValidateHTTPManifest_Unchanged(t *testing.T) {
	m := validHTTPManifest()
	a := m.Actions["get_item"]
	a.Path = "/items/{item_id}"
	a.Args = nil
	a.Request.PathParams = nil
	m.Actions["get_item"] = a

	errList := ValidateManifest(m)
	if !hasValidationError(errList, "actions.get_item.template_ref", `undeclared arg "item_id"`) {
		t.Fatalf("expected legacy template_ref validation error, got %v", errList)
	}
}

func validAppleScriptManifest() *ServiceManifest {
	return &ServiceManifest{
		Name:      "apple-notes",
		Version:   "1.0.0",
		Adapter:   "applescript",
		TargetApp: "Notes",
		Auth:      ServiceAuth{Type: "none"},
		Actions: map[string]ServiceAction{
			"list_notes": {
				Command: "list-notes",
				Risk:    RiskSpec{Level: "low"},
				Response: ResponseSpec{
					Type: "array",
				},
				Args: []ActionArg{{Name: "folder", Type: "string", Required: false}},
			},
		},
	}
}

func validHTTPManifest() *ServiceManifest {
	return &ServiceManifest{
		Name:    "http-service",
		Version: "1.0.0",
		BaseURL: "https://api.example.com",
		Auth:    ServiceAuth{Type: "none"},
		Actions: map[string]ServiceAction{
			"get_item": {
				Method: "GET",
				Path:   "/items/{item_id}",
				Risk:   RiskSpec{Level: "low"},
				Args:   []ActionArg{{Name: "item_id", Type: "string", Required: true}},
				Request: RequestSpec{
					PathParams: map[string]string{"item_id": "{item_id}"},
				},
				Response: ResponseSpec{Type: "object"},
			},
		},
	}
}

func hasValidationError(errs []ValidationError, field, messageContains string) bool {
	for _, err := range errs {
		if err.Field == field && strings.Contains(err.Message, messageContains) {
			return true
		}
	}
	return false
}
