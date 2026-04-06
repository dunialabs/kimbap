package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/dunialabs/kimbap/internal/actions"
)

func TestAppleScriptAdapterType(t *testing.T) {
	a := NewAppleScriptAdapter(NewMockRunner(nil, nil, nil))
	if got := a.Type(); got != "applescript" {
		t.Fatalf("Type() = %q, want %q", got, "applescript")
	}
}

func TestAppleScriptAdapterValidate_Valid(t *testing.T) {
	a := NewAppleScriptAdapter(NewMockRunner(nil, nil, nil))
	err := a.Validate(actions.ActionDefinition{
		Adapter: actions.AdapterConfig{
			Command:   "list-notes",
			TargetApp: "Notes",
		},
	})
	if err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestAppleScriptAdapterValidate_MissingCommand(t *testing.T) {
	a := NewAppleScriptAdapter(NewMockRunner(nil, nil, nil))
	err := a.Validate(actions.ActionDefinition{Adapter: actions.AdapterConfig{TargetApp: "Notes"}})
	if err == nil {
		t.Fatal("Validate() error = nil, want error")
	}
}

func TestAppleScriptAdapterValidate_UnknownCommand(t *testing.T) {
	a := NewAppleScriptAdapter(NewMockRunner(nil, nil, nil))
	err := a.Validate(actions.ActionDefinition{
		Adapter: actions.AdapterConfig{Command: "not-a-real-command", TargetApp: "Notes"},
	})
	if err == nil {
		t.Fatal("Validate() error = nil, want error")
	}
}

func TestAppleScriptAdapterValidate_InlineScriptWithoutCommand(t *testing.T) {
	a := NewAppleScriptAdapter(NewMockRunner(nil, nil, nil))
	err := a.Validate(actions.ActionDefinition{
		Adapter: actions.AdapterConfig{
			TargetApp:      "Notes",
			ScriptSource:   `ObjC.import('stdlib'); JSON.stringify({ok:true});`,
			ScriptLanguage: "jxa",
			ApprovalRef:    "approval.default",
			AuditRef:       "audit.default",
		},
	})
	if err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestAppleScriptAdapterValidate_RejectsUnknownInlineScriptLanguage(t *testing.T) {
	a := NewAppleScriptAdapter(NewMockRunner(nil, nil, nil))
	err := a.Validate(actions.ActionDefinition{
		Adapter: actions.AdapterConfig{
			TargetApp:      "Notes",
			ScriptSource:   `ObjC.import('stdlib'); JSON.stringify({ok:true});`,
			ScriptLanguage: "jython",
			ApprovalRef:    "approval.default",
			AuditRef:       "audit.default",
		},
	})
	if err == nil {
		t.Fatal("Validate() error = nil, want unsupported language error")
	}
	if !strings.Contains(err.Error(), "script language") {
		t.Fatalf("error = %q, want script language guidance", err.Error())
	}
}

func TestAppleScriptAdapterValidate_InlineScriptRequiresApprovalAndAuditRefs(t *testing.T) {
	a := NewAppleScriptAdapter(NewMockRunner(nil, nil, nil))
	err := a.Validate(actions.ActionDefinition{
		Adapter: actions.AdapterConfig{
			TargetApp:    "Notes",
			ScriptSource: `ObjC.import('stdlib'); JSON.stringify({ok:true});`,
		},
	})
	if err == nil {
		t.Fatal("Validate() error = nil, want missing approval/audit refs error")
	}
	if !strings.Contains(err.Error(), "approval_ref") {
		t.Fatalf("error = %q, want approval_ref requirement", err.Error())
	}
}

func TestAppleScriptAdapterValidate_InlineScriptRequiresAuditRef(t *testing.T) {
	a := NewAppleScriptAdapter(NewMockRunner(nil, nil, nil))
	err := a.Validate(actions.ActionDefinition{
		Adapter: actions.AdapterConfig{
			TargetApp:    "Notes",
			ScriptSource: `ObjC.import('stdlib'); JSON.stringify({ok:true});`,
			ApprovalRef:  "approval.default",
		},
	})
	if err == nil {
		t.Fatal("Validate() error = nil, want missing audit_ref error")
	}
	if !strings.Contains(err.Error(), "audit_ref") {
		t.Fatalf("error = %q, want audit_ref requirement", err.Error())
	}
}

func TestAppleScriptAdapterLoadsAllRegistries(t *testing.T) {
	a := NewAppleScriptAdapter(NewMockRunner(nil, nil, nil))
	tests := []actions.AdapterConfig{
		{Command: "list-notes", TargetApp: "Notes"},
		{Command: "list-events", TargetApp: "Calendar"},
		{Command: "list-reminders", TargetApp: "Reminders"},
		{Command: "list-mailboxes", TargetApp: "Mail"},
		{Command: "word-create-document", TargetApp: "Microsoft Word"},
		{Command: "excel-create-workbook", TargetApp: "Microsoft Excel"},
		{Command: "ppt-create-presentation", TargetApp: "Microsoft PowerPoint"},
		{Command: "keynote-create-presentation", TargetApp: "Keynote"},
		{Command: "numbers-create-spreadsheet", TargetApp: "Numbers"},
		{Command: "pages-create-document", TargetApp: "Pages"},
	}

	for _, tc := range tests {
		err := a.Validate(actions.ActionDefinition{Adapter: tc})
		if err != nil {
			t.Fatalf("Validate(%s) error = %v, want nil", tc.Command, err)
		}
	}
}

func TestAppleScriptAdapterExecute_Success(t *testing.T) {
	mock := NewMockRunner([]byte(`{"ok":true,"count":1}`), nil, nil)
	a := NewAppleScriptAdapter(mock)

	res, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{Adapter: actions.AdapterConfig{Command: "list-notes", TargetApp: "Notes"}},
		Input:  map[string]any{"folder": "Work"},
	})

	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if res.HTTPStatus != 200 {
		t.Fatalf("HTTPStatus = %d, want 200", res.HTTPStatus)
	}
	if ok, _ := res.Output["ok"].(bool); !ok {
		t.Fatalf("Output[ok] = %v, want true", res.Output["ok"])
	}
	if got := len(mock.Calls); got != 1 {
		t.Fatalf("mock calls = %d, want 1", got)
	}
	if mock.Calls[0].Name != "/usr/bin/osascript" {
		t.Fatalf("command name = %q, want /usr/bin/osascript", mock.Calls[0].Name)
	}
}

func TestAppleScriptAdapterExecute_ArrayOutput(t *testing.T) {
	mock := NewMockRunner([]byte(`[{"name":"A"}]`), nil, nil)
	a := NewAppleScriptAdapter(mock)

	res, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{Adapter: actions.AdapterConfig{Command: "list-notes", TargetApp: "Notes"}},
		Input:  map[string]any{},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	data, ok := res.Output["data"].([]any)
	if !ok {
		t.Fatalf("Output[data] type = %T, want []any", res.Output["data"])
	}
	if len(data) != 1 {
		t.Fatalf("len(Output[data]) = %d, want 1", len(data))
	}
}

func TestAppleScriptAdapterExecute_UsesInlineScriptSource(t *testing.T) {
	mock := NewMockRunner([]byte(`{"ok":true}`), nil, nil)
	a := NewAppleScriptAdapter(mock)
	inlineScript := `ObjC.import('stdlib'); JSON.stringify({ok:true, source:"inline"});`

	_, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{Adapter: actions.AdapterConfig{Command: "notes.inline", TargetApp: "Notes", ScriptSource: inlineScript, ApprovalRef: "approval.default", AuditRef: "audit.default"}},
		Input:  map[string]any{},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if len(mock.Calls) != 1 {
		t.Fatalf("mock calls = %d, want 1", len(mock.Calls))
	}
	args := mock.Calls[0].Args
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "JavaScript") {
		t.Fatalf("osascript args = %v, want JavaScript mode", args)
	}
	if !strings.Contains(joined, "source:\"inline\"") {
		t.Fatalf("osascript args should include inline script content, got %v", args)
	}
}

func TestAppleScriptAdapterExecute_InlineAppleScriptLanguageUsesNativeMode(t *testing.T) {
	mock := NewMockRunner([]byte(`{"ok":true}`), nil, nil)
	a := NewAppleScriptAdapter(mock)
	inlineScript := `return "ok"`

	_, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{Adapter: actions.AdapterConfig{TargetApp: "Notes", ScriptSource: inlineScript, ScriptLanguage: "applescript", ApprovalRef: "approval.default", AuditRef: "audit.default"}},
		Input:  map[string]any{},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if len(mock.Calls) != 1 {
		t.Fatalf("mock calls = %d, want 1", len(mock.Calls))
	}
	args := mock.Calls[0].Args
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "JavaScript") {
		t.Fatalf("osascript args = %v, did not expect JavaScript language flag for applescript mode", args)
	}
}

func TestAppleScriptAdapterExecute_ManifestModeFailsClosedWithoutInlineScript(t *testing.T) {
	a := NewAppleScriptAdapter(NewMockRunner([]byte(`{"ok":true}`), nil, nil))
	res, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{Adapter: actions.AdapterConfig{Command: "list-notes", TargetApp: "Notes", RegistryMode: "manifest"}},
	})
	if err == nil {
		t.Fatal("Execute() error = nil, want validation failure")
	}
	if res == nil || res.HTTPStatus != 400 {
		t.Fatalf("HTTPStatus = %v, want 400", res)
	}
	if actions.AsExecutionError(err).Code != actions.ErrValidationFailed {
		t.Fatalf("error code = %q, want %q", actions.AsExecutionError(err).Code, actions.ErrValidationFailed)
	}
}

func TestAppleScriptAdapterExecute_InlineScriptFailsWithoutApprovalAuditRefs(t *testing.T) {
	a := NewAppleScriptAdapter(NewMockRunner([]byte(`{"ok":true}`), nil, nil))
	res, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{Adapter: actions.AdapterConfig{TargetApp: "Notes", ScriptSource: `ObjC.import('stdlib'); JSON.stringify({ok:true});`}},
	})
	if err == nil {
		t.Fatal("Execute() error = nil, want validation failure")
	}
	if res == nil || res.HTTPStatus != 400 {
		t.Fatalf("HTTPStatus = %v, want 400", res)
	}
}

func TestAppleScriptAdapterExecute_UnknownCommandReturnsResult(t *testing.T) {
	a := NewAppleScriptAdapter(NewMockRunner(nil, nil, nil))

	res, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{Adapter: actions.AdapterConfig{Command: "no-such-command", TargetApp: "Notes"}},
	})
	if err == nil {
		t.Fatal("Execute() error = nil, want error")
	}
	if res == nil {
		t.Fatal("Execute() result = nil, want non-nil result even on error")
	}
	if res.HTTPStatus != 400 {
		t.Fatalf("HTTPStatus = %d, want 400", res.HTTPStatus)
	}
}

func TestAppleScriptAdapterExecute_Timeout(t *testing.T) {
	mock := NewMockRunner(nil, []byte("execution error -1712"), errors.New("exit status 1"))
	a := NewAppleScriptAdapter(mock)

	res, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{Adapter: actions.AdapterConfig{Command: "list-notes", TargetApp: "Notes"}},
	})
	if err == nil {
		t.Fatal("Execute() error = nil, want error")
	}
	if res == nil {
		t.Fatal("Execute() result = nil, want result")
	}
	if res.HTTPStatus != 504 {
		t.Fatalf("HTTPStatus = %d, want 504", res.HTTPStatus)
	}
	execErr := actions.AsExecutionError(err)
	if execErr.HTTPStatus != 504 {
		t.Fatalf("ExecutionError.HTTPStatus = %d, want 504", execErr.HTTPStatus)
	}
}

func TestAppleScriptAdapterExecute_TCC(t *testing.T) {
	mock := NewMockRunner(nil, []byte("Not authorized to send Apple events to Notes. (-1743)"), errors.New("exit status 1"))
	a := NewAppleScriptAdapter(mock)

	res, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{Adapter: actions.AdapterConfig{Command: "list-notes", TargetApp: "Notes"}},
	})
	if err == nil {
		t.Fatal("Execute() error = nil, want error")
	}
	if res.HTTPStatus != 403 {
		t.Fatalf("HTTPStatus = %d, want 403", res.HTTPStatus)
	}
	if !strings.Contains(actions.AsExecutionError(err).Message, "-1743") {
		t.Fatalf("error message = %q, want contains -1743", actions.AsExecutionError(err).Message)
	}
}

func TestAppleScriptAdapterExecute_StdinJSON(t *testing.T) {
	mock := NewMockRunner([]byte(`{"ok":true}`), nil, nil)
	a := NewAppleScriptAdapter(mock)

	input := map[string]any{
		"title": "hello",
		"count": 3,
		"tags":  []string{"a", "b"},
	}

	_, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{Adapter: actions.AdapterConfig{Command: "create-note", TargetApp: "Notes"}},
		Input:  input,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if len(mock.Calls) != 1 {
		t.Fatalf("mock calls = %d, want 1", len(mock.Calls))
	}

	var got map[string]any
	if jsonErr := json.Unmarshal(mock.Calls[0].Stdin, &got); jsonErr != nil {
		t.Fatalf("stdin JSON unmarshal failed: %v", jsonErr)
	}
	if got["title"] != "hello" {
		t.Fatalf("stdin title = %v, want hello", got["title"])
	}
	if int(got["count"].(float64)) != 3 {
		t.Fatalf("stdin count = %v, want 3", got["count"])
	}
}

func TestTCCPreflightSuccess(t *testing.T) {
	mock := NewMockRunner([]byte("Notes"), nil, nil)
	a := NewAppleScriptAdapter(mock)
	if err := a.Preflight(context.Background(), "Notes"); err != nil {
		t.Fatalf("Preflight() error = %v, want nil", err)
	}
	if len(mock.Calls) != 1 {
		t.Fatalf("mock calls = %d, want 1", len(mock.Calls))
	}
	call := mock.Calls[0]
	if !strings.Contains(strings.Join(call.Args, " "), `Application("Notes").name()`) {
		t.Fatalf("script = %v, want Application(\"Notes\").name()", call.Args)
	}
}

func TestTCCPreflightDenied(t *testing.T) {
	mock := NewMockRunner(nil, []byte("Not authorized to send Apple events to Notes. (-1743)"), errors.New("exit status 1"))
	a := NewAppleScriptAdapter(mock)
	err := a.Preflight(context.Background(), "Notes")
	if err == nil {
		t.Fatal("Preflight() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "System Settings") {
		t.Fatalf("error = %q, want contains 'System Settings'", err.Error())
	}
	if !strings.Contains(err.Error(), "Automation") {
		t.Fatalf("error = %q, want contains 'Automation'", err.Error())
	}
}

func TestTCCPreflightAppNotFound(t *testing.T) {
	mock := NewMockRunner(nil, []byte("execution error: Can't get application \"FakeApp\". (-600)"), errors.New("exit status 1"))
	a := NewAppleScriptAdapter(mock)
	err := a.Preflight(context.Background(), "FakeApp")
	if err == nil {
		t.Fatal("Preflight() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "preflight check failed") {
		t.Fatalf("error = %q, want contains 'preflight check failed'", err.Error())
	}
	if !strings.Contains(err.Error(), "-600") {
		t.Fatalf("error = %q, want contains '-600'", err.Error())
	}
}

func TestErrorMappingTable(t *testing.T) {
	cases := []struct {
		name       string
		stderr     string
		wantStatus int
	}{
		{"TCC denied", "execution error: (-1743)", 403},
		{"Not supported sentinel", "execution error: [NOT_SUPPORTED] unsupported operation", 400},
		{"App not running", "execution error: (-600)", 503},
		{"App missing", "execution error: Application can't be found. (-2700)", 404},
		{"Message not understood", "execution error: Message not understood. (-1708)", 400},
		{"User cancelled", "execution error: (-128)", 499},
		{"Timeout", "execution error: (-1712)", 504},
		{"Object not found", "execution error: (-1728)", 404},
		{"NOT_FOUND sentinel", "execution error: [NOT_FOUND] note not found (-2700)", 404},
		{"TIMEOUT sentinel", "execution error: [TIMEOUT] get-event timed out scanning 19 calendars", 504},
		{"Script error", "execution error: (-2700)", 500},
		{"Unknown error", "some random error", 500},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mapAppleScriptError([]byte(tc.stderr), errors.New("exit status 1"))
			if got.HTTPStatus != tc.wantStatus {
				t.Fatalf("HTTPStatus = %d, want %d", got.HTTPStatus, tc.wantStatus)
			}
		})
	}
}

func TestUnknownErrorFallback(t *testing.T) {
	stderr := "completely unexpected osascript failure text"
	got := mapAppleScriptError([]byte(stderr), errors.New("exit status 1"))
	if got.HTTPStatus != 500 {
		t.Fatalf("HTTPStatus = %d, want 500", got.HTTPStatus)
	}
	if !strings.Contains(got.Message, stderr) {
		t.Fatalf("Message = %q, want stderr preserved", got.Message)
	}
	if details, ok := got.Details["stderr"].(string); !ok || details != stderr {
		t.Fatalf("Details[stderr] = %v, want %q", got.Details["stderr"], stderr)
	}
}

func TestMessageNotUnderstoodIncludesGuidance(t *testing.T) {
	err := mapAppleScriptError([]byte("execution error: Message not understood. (-1708)"), errors.New("exit status 1"))
	if err.HTTPStatus != 400 {
		t.Fatalf("HTTPStatus = %d, want 400", err.HTTPStatus)
	}
	if err.Code != actions.ErrValidationFailed {
		t.Fatalf("Code = %q, want %q", err.Code, actions.ErrValidationFailed)
	}
	if !strings.Contains(err.Message, "not supported by the target app") {
		t.Fatalf("Message = %q, want guidance for unsupported command", err.Message)
	}
}

func TestInputSanitization(t *testing.T) {
	mock := NewMockRunner([]byte(`{"ok":true}`), nil, nil)
	a := NewAppleScriptAdapter(mock)

	suspicious := `$(rm -rf /) \` + "`" + `echo pwned\` + "`" + ` \"quoted\"`

	_, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{Adapter: actions.AdapterConfig{Command: "search-notes", TargetApp: "Notes"}},
		Input: map[string]any{
			"query": suspicious,
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	var got map[string]any
	if jsonErr := json.Unmarshal(mock.Calls[0].Stdin, &got); jsonErr != nil {
		t.Fatalf("stdin JSON unmarshal failed: %v", jsonErr)
	}
	if got["query"] != suspicious {
		t.Fatalf("stdin query = %q, want %q", got["query"], suspicious)
	}
}

func TestAppleScriptAdapterExecute_NilInput(t *testing.T) {
	mock := NewMockRunner([]byte(`{"ok":true}`), nil, nil)
	a := NewAppleScriptAdapter(mock)

	res, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{Adapter: actions.AdapterConfig{Command: "list-notes", TargetApp: "Notes"}},
		Input:  nil,
	})

	if err != nil {
		t.Fatalf("Execute() with nil input error = %v, want nil", err)
	}
	if res.HTTPStatus != 200 {
		t.Fatalf("HTTPStatus = %d, want 200", res.HTTPStatus)
	}

	var got map[string]any
	if jsonErr := json.Unmarshal(mock.Calls[0].Stdin, &got); jsonErr != nil {
		t.Fatalf("stdin JSON unmarshal failed: %v (nil input should produce {})", jsonErr)
	}
	if len(got) != 0 {
		t.Fatalf("stdin = %v, want empty map {}", got)
	}
}

type ctxCapturingRunner struct {
	inner       CommandRunner
	capturedCtx context.Context
}

type truncatingRunner struct{}

func (r *truncatingRunner) Run(ctx context.Context, name string, args []string, stdin io.Reader) ([]byte, []byte, bool, bool, error) {
	_ = ctx
	_ = name
	_ = args
	_ = stdin
	return []byte(`{"partial":`), nil, true, false, nil
}

func (r *ctxCapturingRunner) Run(ctx context.Context, name string, args []string, stdin io.Reader) ([]byte, []byte, bool, bool, error) {
	r.capturedCtx = ctx
	return r.inner.Run(ctx, name, args, stdin)
}

func TestAppleScriptAdapterExecute_DefaultTimeout(t *testing.T) {
	capturing := &ctxCapturingRunner{
		inner: NewMockRunner([]byte(`{"ok":true}`), nil, nil),
	}
	a := NewAppleScriptAdapter(capturing)

	_, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{
			Adapter: actions.AdapterConfig{
				Command:   "list-notes",
				TargetApp: "Notes",
			},
		},
		Input: map[string]any{},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if capturing.capturedCtx == nil {
		t.Fatal("context was never passed to runner")
	}
	deadline, hasDeadline := capturing.capturedCtx.Deadline()
	if !hasDeadline {
		t.Fatal("Execute() without explicit timeout should still set a default deadline")
	}
	remaining := time.Until(deadline)
	if remaining > 31*time.Second || remaining < 1*time.Second {
		t.Fatalf("default deadline remaining = %v, want ~30s", remaining)
	}
}

func TestAppleScriptAdapterExecute_ConfigurableTimeout(t *testing.T) {
	capturing := &ctxCapturingRunner{
		inner: NewMockRunner([]byte(`{"ok":true}`), nil, nil),
	}
	a := NewAppleScriptAdapter(capturing)

	_, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{
			Adapter: actions.AdapterConfig{
				Command:   "list-notes",
				TargetApp: "Notes",
				Timeout:   5 * time.Second,
			},
		},
		Input: map[string]any{},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if capturing.capturedCtx == nil {
		t.Fatal("context was never passed to runner")
	}
	_, hasDeadline := capturing.capturedCtx.Deadline()
	if !hasDeadline {
		t.Fatal("Execute() with Adapter.Timeout > 0 should set a deadline on the context passed to runner")
	}
}

func TestAppleScriptAdapterExecute_RequestTimeoutTakesPriority(t *testing.T) {
	mock := NewMockRunner([]byte(`{"ok":true}`), nil, nil)
	a := NewAppleScriptAdapter(mock)

	res, err := a.Execute(context.Background(), AdapterRequest{
		Timeout: 3 * time.Second,
		Action: actions.ActionDefinition{
			Adapter: actions.AdapterConfig{
				Command:   "list-notes",
				TargetApp: "Notes",
			},
		},
		Input: map[string]any{},
	})

	if err != nil {
		t.Fatalf("Execute() with req.Timeout error = %v, want nil", err)
	}
	if res.HTTPStatus != 200 {
		t.Fatalf("HTTPStatus = %d, want 200", res.HTTPStatus)
	}
	if len(mock.Calls) != 1 {
		t.Fatalf("mock calls = %d, want 1", len(mock.Calls))
	}
}

func TestAppleScriptAdapterExecute_FailsOnTruncatedStdout(t *testing.T) {
	a := NewAppleScriptAdapter(&truncatingRunner{})

	res, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{
			Adapter: actions.AdapterConfig{
				Command:   "list-notes",
				TargetApp: "Notes",
			},
		},
		Input: map[string]any{},
	})

	if err == nil {
		t.Fatal("expected truncation error, got nil")
	}
	if res == nil {
		t.Fatal("expected adapter result, got nil")
	}
	if res.HTTPStatus != 502 {
		t.Fatalf("HTTPStatus = %d, want 502", res.HTTPStatus)
	}
	if _, ok := res.Output["error"]; !ok {
		t.Fatal("expected error payload for truncated stdout")
	}
}
