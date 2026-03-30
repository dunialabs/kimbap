package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	adaptercommands "github.com/dunialabs/kimbap/internal/adapters/commands"
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

func TestParseManifest_AcceptsLegacyRiskMutatingField(t *testing.T) {
	manifestYAML := `name: legacy
version: 1.0.0
adapter: http
base_url: https://api.example.com
auth:
  type: none
actions:
  list:
    method: GET
    path: /items
    request: {}
    response:
      extract: ""
      type: array
    risk:
      level: low
      mutating: false
`

	m, err := ParseManifest([]byte(manifestYAML))
	if err != nil {
		t.Fatalf("expected parse success for legacy risk.mutating, got %v", err)
	}

	action := m.Actions["list"]
	if action.Risk.Mutating == nil {
		t.Fatal("expected risk.mutating to be preserved")
	}
	if *action.Risk.Mutating {
		t.Fatal("expected risk.mutating=false")
	}
}

func TestParseManifest_RejectsUnknownRiskFieldWithStrictDecoding(t *testing.T) {
	manifestYAML := `name: legacy
version: 1.0.0
adapter: http
base_url: https://api.example.com
auth:
  type: none
actions:
  list:
    method: GET
    path: /items
    request: {}
    response:
      extract: ""
      type: array
    risk:
      level: low
      deprecated_flag: false
`

	_, err := ParseManifest([]byte(manifestYAML))
	if err == nil {
		t.Fatal("expected parse error for unknown risk field")
	}
	if !strings.Contains(err.Error(), "deprecated_flag") {
		t.Fatalf("expected unknown field error to mention deprecated_flag, got %v", err)
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

func TestValidateAppleScriptManifest_RejectsServiceAuth(t *testing.T) {
	m := validAppleScriptManifest()
	m.Auth = ServiceAuth{Type: "bearer", CredentialRef: "notes.token"}

	errList := ValidateManifest(m)
	if !hasValidationError(errList, "auth.type", "must be none for applescript adapter") {
		t.Fatalf("expected auth.type applescript rejection, got %v", errList)
	}
}

func TestValidateAppleScriptManifest_RejectsActionAuth(t *testing.T) {
	m := validAppleScriptManifest()
	a := m.Actions["list_notes"]
	a.Auth = &ServiceAuth{Type: "query", QueryParam: "api_key", CredentialRef: "notes.key"}
	m.Actions["list_notes"] = a

	errList := ValidateManifest(m)
	if !hasValidationError(errList, "actions.list_notes.auth.type", "must be none for applescript adapter") {
		t.Fatalf("expected action auth applescript rejection, got %v", errList)
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
	if !hasValidationError(errList, "adapter", "must be one of http, applescript, command") {
		t.Fatalf("expected adapter validation error, got %v", errList)
	}
}

func TestValidateCommandManifest_Valid(t *testing.T) {
	m := validCommandManifest()
	errList := ValidateManifest(m)
	if len(errList) != 0 {
		t.Fatalf("expected 0 validation errors, got %v", errList)
	}
}

func TestValidateCommandManifest_RejectsHTTPAppleScriptFields(t *testing.T) {
	m := validCommandManifest()
	m.BaseURL = "https://api.example.com"
	m.TargetApp = "Notes"

	a := m.Actions["create_diagram"]
	a.Method = "POST"
	a.Path = "/diagrams"
	a.Request.Query = map[string]string{"q": "{q}"}
	a.Request.Headers = map[string]string{"x": "y"}
	a.Request.Body = map[string]any{"name": "{name}"}
	a.Request.PathParams = map[string]string{"id": "{id}"}
	a.Retry = &RetrySpec{MaxAttempts: 2}
	a.Pagination = &PageSpec{Type: "cursor"}
	a.ErrorMapping = map[int]string{500: "oops"}
	m.Actions["create_diagram"] = a

	errList := ValidateManifest(m)
	if !hasValidationError(errList, "base_url", "must not be set for command adapter") {
		t.Fatalf("expected base_url rejection, got %v", errList)
	}
	if !hasValidationError(errList, "target_app", "must not be set for command adapter") {
		t.Fatalf("expected target_app rejection, got %v", errList)
	}
	if !hasValidationError(errList, "actions.create_diagram.method", "must not be set for command adapter") {
		t.Fatalf("expected method rejection, got %v", errList)
	}
	if !hasValidationError(errList, "actions.create_diagram.path", "must not be set for command adapter") {
		t.Fatalf("expected path rejection, got %v", errList)
	}
	if !hasValidationError(errList, "actions.create_diagram.request.query", "must not be set for command adapter") {
		t.Fatalf("expected request.query rejection, got %v", errList)
	}
	if !hasValidationError(errList, "actions.create_diagram.request.headers", "must not be set for command adapter") {
		t.Fatalf("expected request.headers rejection, got %v", errList)
	}
	if !hasValidationError(errList, "actions.create_diagram.request.body", "must not be set for command adapter") {
		t.Fatalf("expected request.body rejection, got %v", errList)
	}
	if !hasValidationError(errList, "actions.create_diagram.request.path_params", "must not be set for command adapter") {
		t.Fatalf("expected request.path_params rejection, got %v", errList)
	}
	if !hasValidationError(errList, "actions.create_diagram.pagination", "must not be set for command adapter") {
		t.Fatalf("expected pagination rejection, got %v", errList)
	}
	if !hasValidationError(errList, "actions.create_diagram.retry", "must not be set for command adapter") {
		t.Fatalf("expected retry rejection, got %v", errList)
	}
	if !hasValidationError(errList, "actions.create_diagram.error_mapping", "must not be set for command adapter") {
		t.Fatalf("expected error_mapping rejection, got %v", errList)
	}
}

func TestValidateCommandManifest_AuthConstraints(t *testing.T) {
	m := validCommandManifest()
	m.Auth = ServiceAuth{Type: "basic", CredentialRef: "tool.token"}
	a := m.Actions["create_diagram"]
	a.Auth = &ServiceAuth{Type: "query", QueryParam: "token", CredentialRef: "tool.token"}
	m.Actions["create_diagram"] = a

	errList := ValidateManifest(m)
	if !hasValidationError(errList, "auth.type", "must be none or bearer for command adapter") {
		t.Fatalf("expected service auth.type rejection, got %v", errList)
	}
	if !hasValidationError(errList, "actions.create_diagram.auth.type", "must be none or bearer for command adapter") {
		t.Fatalf("expected action auth.type rejection, got %v", errList)
	}
}

func TestValidateManifest_AliasesAcceptValidEntries(t *testing.T) {
	m := validHTTPManifest()
	m.Name = "open-meteo-geocoding"
	m.Aliases = []string{"geo", "weather-geo"}
	a := m.Actions["get_item"]
	a.Aliases = []string{"geosearch"}
	m.Actions["get_item"] = a

	errList := ValidateManifest(m)
	if len(errList) != 0 {
		t.Fatalf("expected no validation errors, got %v", errList)
	}
}

func TestValidateManifest_AliasesRejectInvalidEntries(t *testing.T) {
	m := validHTTPManifest()
	m.Name = "open-meteo-geocoding"
	m.Aliases = []string{"", "open-meteo-geocoding", "bad.alias", "geo", "geo"}

	errList := ValidateManifest(m)
	if !hasValidationError(errList, "aliases[0]", "must be non-empty") {
		t.Fatalf("expected non-empty aliases[0] validation error, got %v", errList)
	}
	if !hasValidationError(errList, "aliases[1]", "must differ from service name") {
		t.Fatalf("expected aliases[1] differs-from-name validation error, got %v", errList)
	}
	if !hasValidationError(errList, "aliases[2]", "must not contain dots") {
		t.Fatalf("expected aliases[2] no-dot validation error, got %v", errList)
	}
	if !hasValidationError(errList, "aliases[4]", "duplicates aliases[3]") {
		t.Fatalf("expected aliases[4] duplicate validation error, got %v", errList)
	}
}

func TestValidateManifest_ActionAliasesRejectInvalidEntries(t *testing.T) {
	m := validHTTPManifest()
	m.Name = "open-meteo-geocoding"
	m.Aliases = []string{"geo"}
	a := m.Actions["get_item"]
	a.Aliases = []string{"", "bad.alias", "geo", "geosearch", "geosearch", "open-meteo-geocoding"}
	m.Actions["get_item"] = a

	errList := ValidateManifest(m)
	if !hasValidationError(errList, "actions.get_item.aliases[0]", "must be non-empty") {
		t.Fatalf("expected action alias non-empty error, got %v", errList)
	}
	if !hasValidationError(errList, "actions.get_item.aliases[1]", "must not contain dots") {
		t.Fatalf("expected action alias no-dot error, got %v", errList)
	}
	if !hasValidationError(errList, "actions.get_item.aliases[2]", "must not duplicate service-level aliases") {
		t.Fatalf("expected action alias service-level duplicate error, got %v", errList)
	}
	if !hasValidationError(errList, "actions.get_item.aliases[4]", "duplicates actions.get_item.aliases[3]") {
		t.Fatalf("expected action alias duplicate error, got %v", errList)
	}
	if !hasValidationError(errList, "actions.get_item.aliases[5]", "must differ from service name") {
		t.Fatalf("expected action alias differs-from-service-name error, got %v", errList)
	}
}

func TestValidateCommandManifest_CommandSpecExecutableWhenPresent(t *testing.T) {
	m := validCommandManifest()
	m.CommandSpec = &CommandSpec{}
	errList := ValidateManifest(m)
	if !hasValidationError(errList, "command_spec.executable", "must be non-empty") {
		t.Fatalf("expected command_spec.executable validation error, got %v", errList)
	}
}

func TestValidateCommandManifest_CommandSpecRequired(t *testing.T) {
	m := validCommandManifest()
	m.CommandSpec = nil
	errList := ValidateManifest(m)
	if !hasValidationError(errList, "command_spec", "must be set for command adapter") {
		t.Fatalf("expected command_spec required error, got %v", errList)
	}
}

func TestValidateCommandManifest_InvalidTimeoutFormat(t *testing.T) {
	m := validCommandManifest()
	m.CommandSpec.Timeout = "30seconds"
	errList := ValidateManifest(m)
	if !hasValidationError(errList, "command_spec.timeout", "must be a valid Go duration") {
		t.Fatalf("expected timeout format validation error, got %v", errList)
	}
}

func TestValidateCommandManifest_CommandRequired(t *testing.T) {
	m := validCommandManifest()
	a := m.Actions["create_diagram"]
	a.Command = ""
	m.Actions["create_diagram"] = a
	errList := ValidateManifest(m)
	if !hasValidationError(errList, "actions.create_diagram.command", "is required") {
		t.Fatalf("expected command required error, got %v", errList)
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

func TestValidateHTTPManifest_RejectsUnusedPathParams(t *testing.T) {
	m := validHTTPManifest()
	a := m.Actions["get_item"]
	a.Request.PathParams = map[string]string{
		"item_id": "{item_id}",
		"unused":  "{unused}",
	}
	m.Actions["get_item"] = a

	errList := ValidateManifest(m)
	if !hasValidationError(errList, "actions.get_item.request.path_params", `declares unused path param "unused"`) {
		t.Fatalf("expected unused path param validation error, got %v", errList)
	}
}

func TestValidateManifest_RejectsDuplicateArgNames(t *testing.T) {
	m := validHTTPManifest()
	a := m.Actions["get_item"]
	a.Args = []ActionArg{
		{Name: "item_id", Type: "string", Required: true},
		{Name: "item_id", Type: "string", Required: false},
	}
	m.Actions["get_item"] = a

	errList := ValidateManifest(m)
	if !hasValidationError(errList, "actions.get_item.args[1].name", `duplicates args[0].name "item_id"`) {
		t.Fatalf("expected duplicate arg name validation error, got %v", errList)
	}
}

func TestAppleScriptCommandAllowlistMatchesRegisteredCommands(t *testing.T) {
	registered := make(map[string]struct{})
	registries := []map[string]adaptercommands.Command{
		adaptercommands.NotesCommands(),
		adaptercommands.CalendarCommands(),
		adaptercommands.RemindersCommands(),
		adaptercommands.MailCommands(),
		adaptercommands.FinderCommands(),
		adaptercommands.SafariCommands(),
		adaptercommands.MessagesCommands(),
		adaptercommands.ContactsCommands(),
		adaptercommands.MSOfficeCommands(),
		adaptercommands.IWorkCommands(),
		adaptercommands.SpotifyCommands(),
		adaptercommands.ShortcutsCommands(),
	}
	for _, registry := range registries {
		for name := range registry {
			registered[name] = struct{}{}
		}
	}

	for name := range registered {
		if _, ok := validAppleScriptCommands[name]; !ok {
			t.Errorf("registered command %q missing from parser allowlist", name)
		}
	}
	for name := range validAppleScriptCommands {
		if _, ok := registered[name]; !ok {
			t.Errorf("parser allowlist command %q missing implementation registry", name)
		}
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

func validCommandManifest() *ServiceManifest {
	return &ServiceManifest{
		Name:    "diagram-cli",
		Version: "1.0.0",
		Adapter: "command",
		Auth:    ServiceAuth{Type: "bearer", CredentialRef: "diagram.token"},
		CommandSpec: &CommandSpec{
			Executable: "mermaid",
			JSONFlag:   "--json",
			Timeout:    "30s",
			EnvInject:  map[string]string{"MERMAID_ENV": "dev"},
		},
		Actions: map[string]ServiceAction{
			"create_diagram": {
				Command: "diagram create",
				Risk:    RiskSpec{Level: "low"},
				Args: []ActionArg{
					{Name: "title", Type: "string", Required: true},
					{Name: "source", Type: "string", Required: false},
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
