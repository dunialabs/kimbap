package runtime

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/adapters"
)

type mockPolicyEvaluator struct {
	decision *PolicyDecision
	err      error
}

func (m mockPolicyEvaluator) Evaluate(ctx context.Context, req PolicyRequest) (*PolicyDecision, error) {
	return m.decision, m.err
}

type countingPolicyEvaluator struct {
	called int
}

func (m *countingPolicyEvaluator) Evaluate(ctx context.Context, req PolicyRequest) (*PolicyDecision, error) {
	m.called++
	return &PolicyDecision{Decision: "allow"}, nil
}

type staticDecisionPolicyEvaluator struct {
	decision string
}

func (m staticDecisionPolicyEvaluator) Evaluate(ctx context.Context, req PolicyRequest) (*PolicyDecision, error) {
	return &PolicyDecision{Decision: m.decision}, nil
}

type capturePolicyEvaluator struct {
	lastReq PolicyRequest
	err     error
}

func (m *capturePolicyEvaluator) Evaluate(ctx context.Context, req PolicyRequest) (*PolicyDecision, error) {
	m.lastReq = req
	if m.err != nil {
		return nil, m.err
	}
	return &PolicyDecision{Decision: "allow"}, nil
}

type mockCredentialResolver struct {
	creds *actions.ResolvedCredentialSet
	err   error
}

func (m mockCredentialResolver) Resolve(ctx context.Context, tenantID string, req actions.AuthRequirement) (*actions.ResolvedCredentialSet, error) {
	return m.creds, m.err
}

type mockAuditWriter struct {
	events []AuditEvent
	err    error
}

func (m *mockAuditWriter) Write(ctx context.Context, event AuditEvent) error {
	m.events = append(m.events, event)
	return m.err
}

type mockApprovalManager struct {
	result              *ApprovalResult
	err                 error
	cancelErr           error
	createCalls         int
	cancelCalls         int
	lastRequest         ApprovalRequest
	lastCancelledID     string
	lastCancelledReason string
	lastCancelCtxErr    error
}

func (m *mockApprovalManager) CreateRequest(ctx context.Context, req ApprovalRequest) (*ApprovalResult, error) {
	m.createCalls++
	m.lastRequest = req
	return m.result, m.err
}

func (m *mockApprovalManager) CancelRequest(ctx context.Context, approvalRequestID string, reason string) error {
	m.cancelCalls++
	m.lastCancelledID = approvalRequestID
	m.lastCancelledReason = reason
	m.lastCancelCtxErr = ctx.Err()
	return m.cancelErr
}

type mockPrincipalVerifier struct {
	err    error
	called int
}

func (m *mockPrincipalVerifier) Verify(ctx context.Context, principal actions.Principal) error {
	m.called++
	return m.err
}

type mockHeldExecutionStore struct {
	held        map[string]actions.ExecutionRequest
	holdErr     error
	resumeErr   error
	removeErr   error
	holdCalls   int
	removeCalls int
}

func (m *mockHeldExecutionStore) Hold(ctx context.Context, approvalRequestID string, req actions.ExecutionRequest) error {
	m.holdCalls++
	if m.holdErr != nil {
		return m.holdErr
	}
	if m.held == nil {
		m.held = map[string]actions.ExecutionRequest{}
	}
	m.held[approvalRequestID] = req
	return nil
}

func (m *mockHeldExecutionStore) Resume(ctx context.Context, approvalRequestID string) (*actions.ExecutionRequest, error) {
	if m.resumeErr != nil {
		return nil, m.resumeErr
	}
	req, ok := m.held[approvalRequestID]
	if !ok {
		return nil, nil
	}
	delete(m.held, approvalRequestID)
	copyReq := req
	return &copyReq, nil
}

func (m *mockHeldExecutionStore) Remove(ctx context.Context, approvalRequestID string) error {
	m.removeCalls++
	if m.removeErr != nil {
		return m.removeErr
	}
	delete(m.held, approvalRequestID)
	return nil
}

type mockAdapter struct {
	kind     string
	result   *adapters.AdapterResult
	err      error
	sleep    time.Duration
	validErr error
}

func (m mockAdapter) Type() string {
	return m.kind
}

func (m mockAdapter) Validate(def actions.ActionDefinition) error {
	return m.validErr
}

func (m mockAdapter) Execute(ctx context.Context, req adapters.AdapterRequest) (*adapters.AdapterResult, error) {
	if m.sleep > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(m.sleep):
		}
	}
	return m.result, m.err
}

type captureAdapter struct {
	kind    string
	result  *adapters.AdapterResult
	lastReq adapters.AdapterRequest
	called  int
}

func (c *captureAdapter) Type() string {
	return c.kind
}

func (c *captureAdapter) Validate(def actions.ActionDefinition) error {
	return nil
}

func (c *captureAdapter) Execute(ctx context.Context, req adapters.AdapterRequest) (*adapters.AdapterResult, error) {
	c.called++
	c.lastReq = req
	return c.result, nil
}

func TestRuntimeExecuteSuccess(t *testing.T) {
	audit := &mockAuditWriter{}
	rt := Runtime{
		PolicyEvaluator: mockPolicyEvaluator{decision: &PolicyDecision{Decision: "allow"}},
		AuditWriter:     audit,
		Adapters: map[string]adapters.Adapter{
			"http": mockAdapter{kind: "http", result: &adapters.AdapterResult{Output: map[string]any{"ok": true}, HTTPStatus: 200}},
		},
	}

	res := rt.Execute(context.Background(), baseRequest(actions.ActionDefinition{
		Name:        "github.issues.create",
		InputSchema: &actions.Schema{Type: "object", Required: []string{"owner"}, Properties: map[string]*actions.Schema{"owner": {Type: "string"}}},
		Adapter:     actions.AdapterConfig{Type: "http", URLTemplate: "https://example.com"},
	}))

	if res.Status != actions.StatusSuccess {
		t.Fatalf("expected success, got %s", res.Status)
	}
	if res.HTTPStatus != 200 {
		t.Fatalf("expected status 200, got %d", res.HTTPStatus)
	}
	if len(audit.events) != 1 {
		t.Fatalf("expected audit event, got %d", len(audit.events))
	}
}

func TestRuntimeExecuteAuditIncludesInputPayload(t *testing.T) {
	audit := &mockAuditWriter{}
	rt := Runtime{
		AuditWriter: audit,
		Adapters: map[string]adapters.Adapter{
			"http": mockAdapter{kind: "http", result: &adapters.AdapterResult{Output: map[string]any{"ok": true}, HTTPStatus: 200}},
		},
	}

	req := baseRequest(actions.ActionDefinition{
		Name:        "github.issues.create",
		InputSchema: &actions.Schema{Type: "object", Properties: map[string]*actions.Schema{"title": {Type: "string"}}},
		Adapter:     actions.AdapterConfig{Type: "http", URLTemplate: "https://example.com"},
	})
	req.Input = map[string]any{"title": "hello"}

	res := rt.Execute(context.Background(), req)
	if res.Status != actions.StatusSuccess {
		t.Fatalf("expected success, got %s", res.Status)
	}
	if len(audit.events) != 1 {
		t.Fatalf("expected one audit event, got %d", len(audit.events))
	}
	if audit.events[0].Input == nil || audit.events[0].Input["title"] != "hello" {
		t.Fatalf("expected input payload in audit event, got %+v", audit.events[0].Input)
	}
}

func TestRuntimeExecuteRoutesAppleScriptAdapter(t *testing.T) {
	rt := Runtime{
		Adapters: map[string]adapters.Adapter{
			"applescript": mockAdapter{kind: "applescript", result: &adapters.AdapterResult{Output: map[string]any{"ok": true}, HTTPStatus: 200}},
		},
	}

	res := rt.Execute(context.Background(), baseRequest(actions.ActionDefinition{
		Name:    "notes.create",
		Adapter: actions.AdapterConfig{Type: "applescript", Command: "create_note", TargetApp: "Notes"},
	}))

	if res.Status != actions.StatusSuccess {
		t.Fatalf("expected success, got %s", res.Status)
	}
	if res.Meta["adapter_type"] != "applescript" {
		t.Fatalf("expected applescript adapter routing, got %+v", res.Meta["adapter_type"])
	}
}

func TestRuntimeExecutePolicyDenial(t *testing.T) {
	rt := Runtime{
		PolicyEvaluator: mockPolicyEvaluator{decision: &PolicyDecision{Decision: "deny"}},
		Adapters:        map[string]adapters.Adapter{"http": mockAdapter{kind: "http"}},
	}

	res := rt.Execute(context.Background(), baseRequest(actions.ActionDefinition{
		Name:    "github.issues.create",
		Adapter: actions.AdapterConfig{Type: "http", URLTemplate: "https://example.com"},
	}))

	if res.Status != actions.StatusError {
		t.Fatalf("expected error status, got %s", res.Status)
	}
	if res.Error == nil || res.Error.Code != actions.ErrUnauthorized {
		t.Fatalf("expected unauthorized error, got %+v", res.Error)
	}
}

func TestRuntimeExecuteApprovalRequired(t *testing.T) {
	rt := Runtime{
		PolicyEvaluator:    mockPolicyEvaluator{decision: &PolicyDecision{Decision: "require_approval"}},
		ApprovalManager:    &mockApprovalManager{result: &ApprovalResult{Approved: false, RequestID: "apr-1"}},
		HeldExecutionStore: &mockHeldExecutionStore{held: map[string]actions.ExecutionRequest{}},
		Adapters:           map[string]adapters.Adapter{"http": mockAdapter{kind: "http"}},
	}

	res := rt.Execute(context.Background(), baseRequest(actions.ActionDefinition{
		Name:    "github.issues.create",
		Adapter: actions.AdapterConfig{Type: "http", URLTemplate: "https://example.com"},
	}))

	if res.Status != actions.StatusApprovalRequired {
		t.Fatalf("expected approval_required, got %s", res.Status)
	}
	if res.Error == nil || res.Error.Code != actions.ErrApprovalRequired {
		t.Fatalf("expected approval required error, got %+v", res.Error)
	}
}

func TestRuntimeExecuteInlineAppleScriptApprovalRefForcesApproval(t *testing.T) {
	approvalManager := &mockApprovalManager{result: &ApprovalResult{Approved: false, RequestID: "apr-inline"}}
	rt := Runtime{
		ApprovalManager:    approvalManager,
		HeldExecutionStore: &mockHeldExecutionStore{held: map[string]actions.ExecutionRequest{}},
		Adapters: map[string]adapters.Adapter{
			"applescript": mockAdapter{kind: "applescript", result: &adapters.AdapterResult{Output: map[string]any{"ok": true}, HTTPStatus: 200}},
		},
	}

	res := rt.Execute(context.Background(), baseRequest(actions.ActionDefinition{
		Name: "notes.inline-create",
		Adapter: actions.AdapterConfig{
			Type:         "applescript",
			TargetApp:    "Notes",
			ScriptSource: "return \"ok\"",
			ApprovalRef:  "policy.applescript.inline",
			AuditRef:     "audit.notes.inline-create",
		},
	}))

	if res.Status != actions.StatusApprovalRequired {
		t.Fatalf("expected approval_required for inline applescript approval_ref, got %s", res.Status)
	}
	if approvalManager.createCalls != 1 {
		t.Fatalf("expected one approval request creation, got %d", approvalManager.createCalls)
	}
	if got, _ := approvalManager.lastRequest.Meta["approval_ref"].(string); got != "policy.applescript.inline" {
		t.Fatalf("expected approval_ref metadata propagated, got %q", got)
	}
}

func TestRuntimeExecuteInlineAppleScriptApprovalRefApprovedExecutes(t *testing.T) {
	approvalManager := &mockApprovalManager{result: &ApprovalResult{Approved: true, RequestID: "apr-inline-ok"}}
	rt := Runtime{
		ApprovalManager: approvalManager,
		Adapters: map[string]adapters.Adapter{
			"applescript": mockAdapter{kind: "applescript", result: &adapters.AdapterResult{Output: map[string]any{"ok": true}, HTTPStatus: 200}},
		},
	}

	res := rt.Execute(context.Background(), baseRequest(actions.ActionDefinition{
		Name: "notes.inline-create",
		Adapter: actions.AdapterConfig{
			Type:         "applescript",
			TargetApp:    "Notes",
			ScriptSource: "return \"ok\"",
			ApprovalRef:  "policy.applescript.inline",
			AuditRef:     "audit.notes.inline-create",
		},
	}))

	if res.Status != actions.StatusSuccess {
		t.Fatalf("expected success after immediate approval, got %s", res.Status)
	}
	if approvalManager.createCalls != 1 {
		t.Fatalf("expected one approval request creation, got %d", approvalManager.createCalls)
	}
	if got, _ := res.Meta["approval_request_id"].(string); got != "apr-inline-ok" {
		t.Fatalf("expected approval_request_id in result meta, got %q", got)
	}
}

func TestRuntimeExecuteApprovalRequiredNoHeldStore(t *testing.T) {
	rt := Runtime{
		PolicyEvaluator: mockPolicyEvaluator{decision: &PolicyDecision{Decision: "require_approval"}},
		ApprovalManager: &mockApprovalManager{result: &ApprovalResult{Approved: false, RequestID: "apr-1"}},
		Adapters:        map[string]adapters.Adapter{"http": mockAdapter{kind: "http"}},
	}

	res := rt.Execute(context.Background(), baseRequest(actions.ActionDefinition{
		Name:    "github.issues.create",
		Adapter: actions.AdapterConfig{Type: "http", URLTemplate: "https://example.com"},
	}))

	if res.Status != actions.StatusError {
		t.Fatalf("expected error when held store is nil, got %s", res.Status)
	}
	if res.Error == nil || res.Error.Code != actions.ErrDownstreamUnavailable {
		t.Fatalf("expected downstream unavailable error, got %+v", res.Error)
	}
}

func TestRuntimeExecuteCredentialMissing(t *testing.T) {
	rt := Runtime{
		CredentialResolver: mockCredentialResolver{creds: nil, err: nil},
		Adapters:           map[string]adapters.Adapter{"http": mockAdapter{kind: "http"}},
	}

	res := rt.Execute(context.Background(), baseRequest(actions.ActionDefinition{
		Name: "github.issues.create",
		Auth: actions.AuthRequirement{Type: actions.AuthTypeBearer},
		Adapter: actions.AdapterConfig{
			Type:        "http",
			URLTemplate: "https://example.com",
		},
	}))

	if res.Status != actions.StatusError {
		t.Fatalf("expected error status, got %s", res.Status)
	}
	if res.Error == nil || res.Error.Code != actions.ErrCredentialMissing {
		t.Fatalf("expected credential missing, got %+v", res.Error)
	}
}

func TestRuntimeExecuteInputValidationFailure(t *testing.T) {
	rt := Runtime{
		Adapters: map[string]adapters.Adapter{"http": mockAdapter{kind: "http"}},
	}

	req := baseRequest(actions.ActionDefinition{
		Name:        "github.issues.create",
		InputSchema: &actions.Schema{Type: "object", Required: []string{"owner"}, Properties: map[string]*actions.Schema{"owner": {Type: "string"}}},
		Adapter:     actions.AdapterConfig{Type: "http", URLTemplate: "https://example.com"},
	})
	req.Input = map[string]any{}

	res := rt.Execute(context.Background(), req)
	if res.Error == nil || res.Error.Code != actions.ErrValidationFailed {
		t.Fatalf("expected validation error, got %+v", res.Error)
	}
}

func TestRuntimeExecuteSanitizeFailureBeforePolicyEvaluation(t *testing.T) {
	evaluator := &countingPolicyEvaluator{}
	rt := Runtime{
		PolicyEvaluator: evaluator,
		Adapters:        map[string]adapters.Adapter{"http": mockAdapter{kind: "http"}},
	}

	req := baseRequest(actions.ActionDefinition{
		Name:        "github.issues.create",
		InputSchema: &actions.Schema{Type: "object", Required: []string{"owner"}, Properties: map[string]*actions.Schema{"owner": {Type: "string"}}},
		Adapter:     actions.AdapterConfig{Type: "http", URLTemplate: "https://example.com"},
	})
	req.Input["owner"] = "../../etc/passwd"

	res := rt.Execute(context.Background(), req)
	if res.Status != actions.StatusError {
		t.Fatalf("expected error status, got %s", res.Status)
	}
	if res.Error == nil || res.Error.Code != actions.ErrValidationFailed {
		t.Fatalf("expected validation failed error, got %+v", res.Error)
	}
	if evaluator.called != 0 {
		t.Fatalf("expected policy evaluator not called on sanitize failure, got %d", evaluator.called)
	}
}

func TestRuntimeExecuteAdapterFailure(t *testing.T) {
	rt := Runtime{
		Adapters: map[string]adapters.Adapter{
			"http": mockAdapter{kind: "http", err: actions.NewExecutionError(actions.ErrDownstreamUnavailable, "boom", 502, true, nil)},
		},
	}

	res := rt.Execute(context.Background(), baseRequest(actions.ActionDefinition{
		Name:    "github.issues.create",
		Adapter: actions.AdapterConfig{Type: "http", URLTemplate: "https://example.com"},
	}))

	if res.Status != actions.StatusError {
		t.Fatalf("expected error status, got %s", res.Status)
	}
	if res.Error == nil || res.Error.Code != actions.ErrDownstreamUnavailable {
		t.Fatalf("expected downstream unavailable, got %+v", res.Error)
	}
}

func TestRuntimeExecuteAdapterFailurePreservesOutput(t *testing.T) {
	rt := Runtime{
		Adapters: map[string]adapters.Adapter{
			"command": mockAdapter{
				kind:   "command",
				result: &adapters.AdapterResult{Output: map[string]any{"raw": "compile failed", "_exit_code": float64(1)}, HTTPStatus: 502},
				err:    actions.NewExecutionError(actions.ErrDownstreamUnavailable, "boom", 502, true, nil),
			},
		},
	}

	res := rt.Execute(context.Background(), baseRequest(actions.ActionDefinition{
		Name:    "cargo-cli.test",
		Adapter: actions.AdapterConfig{Type: "command", ExecutablePath: "cargo"},
	}))

	if res.Status != actions.StatusError {
		t.Fatalf("expected error status, got %s", res.Status)
	}
	if res.Output == nil || res.Output["raw"] != "compile failed" {
		t.Fatalf("expected adapter error output preserved, got %#v", res.Output)
	}
	if res.Output["_exit_code"] != float64(1) {
		t.Fatalf("expected _exit_code preserved, got %#v", res.Output)
	}
}

func TestRuntimeExecuteAdapterFailureAppliesTextFilter(t *testing.T) {
	rt := Runtime{
		Adapters: map[string]adapters.Adapter{
			"command": mockAdapter{
				kind:   "command",
				result: &adapters.AdapterResult{Output: map[string]any{"raw": "noise\nERROR boom\nnoise2", "_exit_code": float64(1)}, HTTPStatus: 502},
				err:    actions.NewExecutionError(actions.ErrDownstreamUnavailable, "boom", 502, true, nil),
			},
		},
	}

	res := rt.Execute(context.Background(), baseRequest(actions.ActionDefinition{
		Name:             "pytest-cli.run",
		Adapter:          actions.AdapterConfig{Type: "command", ExecutablePath: "pytest"},
		TextFilterConfig: &actions.TextFilterConfig{KeepLinesMatching: []*regexp.Regexp{regexp.MustCompile("ERROR")}},
	}))

	if res.Status != actions.StatusError {
		t.Fatalf("expected error status, got %s", res.Status)
	}
	if res.Output == nil || res.Output["raw"] != "ERROR boom" {
		t.Fatalf("expected filtered error output, got %#v", res.Output)
	}
	if res.Output["_text_filtered"] != true {
		t.Fatalf("expected _text_filtered marker, got %#v", res.Output)
	}
}

func TestRuntimeExecuteStripsRuntimeKeysBeforeAdapterExecute(t *testing.T) {
	capture := &captureAdapter{
		kind:   "http",
		result: &adapters.AdapterResult{Output: map[string]any{"ok": true}, HTTPStatus: 200},
	}
	rt := Runtime{
		Adapters: map[string]adapters.Adapter{
			"http": capture,
		},
	}

	action := actions.ActionDefinition{
		Name:        "github.issues.list",
		InputSchema: &actions.Schema{Type: "object", AdditionalProperties: true},
		Adapter:     actions.AdapterConfig{Type: "http", URLTemplate: "https://example.com"},
		Pagination: &actions.PaginationConfig{
			MaxPages: 2,
		},
		FilterConfig: &actions.FilterConfig{
			Select: map[string]string{"owner": "owner"},
		},
	}
	req := baseRequest(action)
	req.Input["owner"] = "dunia"
	req.Input["_output_mode"] = "raw"
	req.Input["_budget"] = int64(100)
	req.Input["_max_pages"] = "5"

	res := rt.Execute(context.Background(), req)
	if res.Status != actions.StatusSuccess {
		t.Fatalf("expected success, got %s: %+v", res.Status, res.Error)
	}
	if capture.called != 1 {
		t.Fatalf("expected adapter execute to be called once, got %d", capture.called)
	}
	if got := capture.lastReq.Input["owner"]; got != "dunia" {
		t.Fatalf("expected non-runtime input preserved, got %#v", got)
	}
	if _, ok := capture.lastReq.Input["_output_mode"]; ok {
		t.Fatal("expected _output_mode to be stripped before adapter execute")
	}
	if _, ok := capture.lastReq.Input["_budget"]; ok {
		t.Fatal("expected _budget to be stripped before adapter execute")
	}
	if _, ok := capture.lastReq.Input["_max_pages"]; ok {
		t.Fatal("expected _max_pages to be stripped before adapter execute")
	}
	if capture.lastReq.Action.Pagination == nil {
		t.Fatal("expected pagination config to remain available to adapter")
	}
	if capture.lastReq.Action.Pagination.MaxPages != 2 {
		t.Fatalf("expected _max_pages=5 clamped to manifest cap 2, got %d", capture.lastReq.Action.Pagination.MaxPages)
	}
	if got, _ := req.Input["_output_mode"].(string); got != "raw" {
		t.Fatalf("expected request input to remain unchanged for post-processing, got %#v", req.Input["_output_mode"])
	}
	if got := coerceBudgetInt(req.Input["_budget"]); got != 100 {
		t.Fatalf("expected request input budget to remain unchanged, got %d", got)
	}
	if got := coercePositiveInt(req.Input["_max_pages"]); got != 5 {
		t.Fatalf("expected request input _max_pages to remain unchanged, got %d", got)
	}
}

func TestRuntimeExecuteTimeout(t *testing.T) {
	rt := Runtime{
		Adapters: map[string]adapters.Adapter{
			"http": mockAdapter{kind: "http", sleep: 100 * time.Millisecond},
		},
	}

	req := baseRequest(actions.ActionDefinition{
		Name:    "github.issues.create",
		Adapter: actions.AdapterConfig{Type: "http", URLTemplate: "https://example.com"},
	})
	req.Timeout = 10 * time.Millisecond

	res := rt.Execute(context.Background(), req)
	if res.Status != actions.StatusTimeout {
		t.Fatalf("expected timeout status, got %s", res.Status)
	}
	if res.Error == nil {
		t.Fatal("expected timeout error")
	}
	if !errors.Is(context.DeadlineExceeded, context.DeadlineExceeded) && res.Error.Code == "" {
		t.Fatal("invalid timeout result")
	}
}

func TestRuntimeExecuteRedactsSensitiveHeadersInMeta(t *testing.T) {
	rt := Runtime{
		Adapters: map[string]adapters.Adapter{
			"http": mockAdapter{
				kind: "http",
				result: &adapters.AdapterResult{
					Output:     map[string]any{"ok": true},
					HTTPStatus: 200,
					Headers: map[string]string{
						"Authorization": "Bearer secret",
						"X-Api-Key":     "supersecret",
						"Cookie":        "session=secret",
						"Set-Cookie":    "session=secret",
						"Content-Type":  "application/json",
					},
				},
			},
		},
	}

	res := rt.Execute(context.Background(), baseRequest(actions.ActionDefinition{
		Name:    "github.issues.create",
		Adapter: actions.AdapterConfig{Type: "http", URLTemplate: "https://example.com"},
	}))

	headersAny, ok := res.Meta["http_headers"]
	if !ok {
		t.Fatal("expected http_headers metadata")
	}
	headers, ok := headersAny.(map[string]string)
	if !ok {
		t.Fatalf("unexpected header metadata type: %T", headersAny)
	}
	if _, exists := headers["Authorization"]; exists {
		t.Fatal("expected Authorization header to be redacted")
	}
	if _, exists := headers["X-Api-Key"]; exists {
		t.Fatal("expected X-Api-Key header to be redacted")
	}
	if _, exists := headers["Cookie"]; exists {
		t.Fatal("expected Cookie header to be redacted")
	}
	if _, exists := headers["Set-Cookie"]; exists {
		t.Fatal("expected Set-Cookie header to be redacted")
	}
	if headers["Content-Type"] != "application/json" {
		t.Fatalf("expected non-sensitive header preserved, got %+v", headers)
	}
}

func TestRuntimeExecutePrincipalVerifierRejects(t *testing.T) {
	verifier := &mockPrincipalVerifier{err: errors.New("invalid principal token")}
	rt := Runtime{
		PrincipalVerifier: verifier,
		Adapters: map[string]adapters.Adapter{
			"http": mockAdapter{kind: "http", result: &adapters.AdapterResult{Output: map[string]any{"ok": true}, HTTPStatus: 200}},
		},
	}

	res := rt.Execute(context.Background(), baseRequest(actions.ActionDefinition{
		Name:    "github.issues.create",
		Adapter: actions.AdapterConfig{Type: "http", URLTemplate: "https://example.com"},
	}))

	if res.Status != actions.StatusError {
		t.Fatalf("expected error status, got %s", res.Status)
	}
	if res.Error == nil || res.Error.Code != actions.ErrUnauthenticated {
		t.Fatalf("expected unauthenticated error, got %+v", res.Error)
	}
	if verifier.called != 1 {
		t.Fatalf("expected verifier to be called once, got %d", verifier.called)
	}
}

func TestRuntimeExecuteAuditRequiredFailClosed(t *testing.T) {
	audit := &mockAuditWriter{err: errors.New("audit sink unavailable")}
	rt := Runtime{
		AuditWriter:   audit,
		AuditRequired: true,
		Adapters: map[string]adapters.Adapter{
			"http": mockAdapter{kind: "http", result: &adapters.AdapterResult{Output: map[string]any{"ok": true}, HTTPStatus: 200}},
		},
	}

	res := rt.Execute(context.Background(), baseRequest(actions.ActionDefinition{
		Name:    "github.issues.create",
		Adapter: actions.AdapterConfig{Type: "http", URLTemplate: "https://example.com"},
	}))

	if res.Status != actions.StatusError {
		t.Fatalf("expected error status, got %s", res.Status)
	}
	if res.Error == nil || res.Error.Code != actions.ErrAuditRequired {
		t.Fatalf("expected audit required error, got %+v", res.Error)
	}
	if res.HTTPStatus != 500 {
		t.Fatalf("expected http status 500, got %d", res.HTTPStatus)
	}
	if len(audit.events) != 1 {
		t.Fatalf("expected one attempted audit write, got %d", len(audit.events))
	}
}

func TestRuntimeExecuteUsesAdapterAuditRefWhenProvided(t *testing.T) {
	audit := &mockAuditWriter{}
	rt := Runtime{
		AuditWriter: audit,
		Adapters: map[string]adapters.Adapter{
			"applescript": mockAdapter{kind: "applescript", result: &adapters.AdapterResult{Output: map[string]any{"ok": true}, HTTPStatus: 200}},
		},
	}

	res := rt.Execute(context.Background(), baseRequest(actions.ActionDefinition{
		Name: "notes.inline-create",
		Adapter: actions.AdapterConfig{
			Type:         "applescript",
			TargetApp:    "Notes",
			ScriptSource: "return \"ok\"",
			AuditRef:     "audit.notes.inline-create",
		},
	}))

	if res.Status != actions.StatusSuccess {
		t.Fatalf("expected success, got %s", res.Status)
	}
	if res.AuditRef != "audit.notes.inline-create" {
		t.Fatalf("expected result audit ref from adapter config, got %q", res.AuditRef)
	}
	if len(audit.events) != 1 {
		t.Fatalf("expected one audit event, got %d", len(audit.events))
	}
	if got, _ := audit.events[0].Meta["audit_ref"].(string); got != "audit.notes.inline-create" {
		t.Fatalf("expected audit event metadata audit_ref to match adapter config, got %q", got)
	}
}

func TestRuntimeExecuteAuditRefFallsBackToRequestID(t *testing.T) {
	audit := &mockAuditWriter{}
	rt := Runtime{
		AuditWriter: audit,
		Adapters: map[string]adapters.Adapter{
			"http": mockAdapter{kind: "http", result: &adapters.AdapterResult{Output: map[string]any{"ok": true}, HTTPStatus: 200}},
		},
	}

	res := rt.Execute(context.Background(), baseRequest(actions.ActionDefinition{
		Name:    "github.issues.create",
		Adapter: actions.AdapterConfig{Type: "http", URLTemplate: "https://example.com"},
	}))

	if res.Status != actions.StatusSuccess {
		t.Fatalf("expected success, got %s", res.Status)
	}
	if res.AuditRef != "req-1" {
		t.Fatalf("expected audit ref fallback to request id, got %q", res.AuditRef)
	}
	if got, _ := res.Meta["audit_ref"].(string); got != "req-1" {
		t.Fatalf("expected audit_ref metadata fallback to request id, got %q", got)
	}
	if len(audit.events) != 1 {
		t.Fatalf("expected one audit event, got %d", len(audit.events))
	}
	if got, _ := audit.events[0].Meta["audit_ref"].(string); got != "req-1" {
		t.Fatalf("expected audit event audit_ref fallback to request id, got %q", got)
	}
}

func TestRuntimeExecuteAuditRefFallsBackToTraceIDWhenRequestIDMissing(t *testing.T) {
	audit := &mockAuditWriter{}
	rt := Runtime{
		AuditWriter: audit,
		Adapters: map[string]adapters.Adapter{
			"http": mockAdapter{kind: "http", result: &adapters.AdapterResult{Output: map[string]any{"ok": true}, HTTPStatus: 200}},
		},
	}

	req := baseRequest(actions.ActionDefinition{
		Name:    "github.issues.create",
		Adapter: actions.AdapterConfig{Type: "http", URLTemplate: "https://example.com"},
	})
	req.RequestID = ""
	req.TraceID = "trace-fallback"

	res := rt.Execute(context.Background(), req)

	if res.Status != actions.StatusSuccess {
		t.Fatalf("expected success, got %s", res.Status)
	}
	if res.AuditRef != "trace-fallback" {
		t.Fatalf("expected audit ref fallback to trace id, got %q", res.AuditRef)
	}
	if got, _ := res.Meta["audit_ref"].(string); got != "trace-fallback" {
		t.Fatalf("expected audit_ref metadata fallback to trace id, got %q", got)
	}
	if len(audit.events) != 1 {
		t.Fatalf("expected one audit event, got %d", len(audit.events))
	}
	if got, _ := audit.events[0].Meta["audit_ref"].(string); got != "trace-fallback" {
		t.Fatalf("expected audit event audit_ref fallback to trace id, got %q", got)
	}
}

func TestRuntimeApprovalHoldAndResume(t *testing.T) {
	store := &mockHeldExecutionStore{held: map[string]actions.ExecutionRequest{}}
	rt := Runtime{
		ApprovalManager:    &mockApprovalManager{result: &ApprovalResult{Approved: false, RequestID: "apr-42"}},
		HeldExecutionStore: store,
		Adapters: map[string]adapters.Adapter{
			"http": mockAdapter{kind: "http", result: &adapters.AdapterResult{Output: map[string]any{"resumed": true}, HTTPStatus: 200}},
		},
	}

	req := baseRequest(actions.ActionDefinition{
		Name:         "github.issues.create",
		ApprovalHint: actions.ApprovalRequired,
		Adapter:      actions.AdapterConfig{Type: "http", URLTemplate: "https://example.com"},
	})

	res := rt.Execute(context.Background(), req)
	if res.Status != actions.StatusApprovalRequired {
		t.Fatalf("expected approval_required status, got %s", res.Status)
	}
	if store.holdCalls != 1 {
		t.Fatalf("expected held store Hold once, got %d", store.holdCalls)
	}
	if _, ok := store.held["apr-42"]; !ok {
		t.Fatal("expected held request stored by approval request id")
	}

	resumed := rt.ResumeApproved(context.Background(), "apr-42")
	if resumed.Status != actions.StatusSuccess {
		t.Fatalf("expected resumed execution success, got %s", resumed.Status)
	}
	if resumed.Output["resumed"] != true {
		t.Fatalf("expected resumed output marker, got %+v", resumed.Output)
	}
	if store.removeCalls != 0 {
		t.Fatalf("expected Remove not called (Resume is now consume-once), got %d", store.removeCalls)
	}
}

func TestRuntimeResumeApprovedDeniedAfterPolicyChange(t *testing.T) {
	store := &mockHeldExecutionStore{held: map[string]actions.ExecutionRequest{}}
	store.held["apr-99"] = baseRequest(actions.ActionDefinition{
		Name:    "github.issues.create",
		Adapter: actions.AdapterConfig{Type: "http", URLTemplate: "https://example.com"},
	})

	rt := Runtime{
		PolicyEvaluator:    staticDecisionPolicyEvaluator{decision: "deny"},
		HeldExecutionStore: store,
		Adapters: map[string]adapters.Adapter{
			"http": mockAdapter{kind: "http", result: &adapters.AdapterResult{Output: map[string]any{"resumed": true}, HTTPStatus: 200}},
		},
	}

	resumed := rt.ResumeApproved(context.Background(), "apr-99")
	if resumed.Status != actions.StatusError {
		t.Fatalf("expected resume to fail when policy changes to deny, got %s", resumed.Status)
	}
	if resumed.Error == nil || resumed.Error.Code != actions.ErrUnauthorized {
		t.Fatalf("expected unauthorized error, got %+v", resumed.Error)
	}
	if resumed.HTTPStatus != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", resumed.HTTPStatus)
	}
}

func TestRuntimeResumeApprovedPassesSessionAndClassificationToPolicy(t *testing.T) {
	store := &mockHeldExecutionStore{held: map[string]actions.ExecutionRequest{}}
	session := &actions.SessionContext{SessionID: "sess-123", Mode: actions.ModeCall, Channel: "cli"}
	classInfo := &actions.ClassificationInfo{Service: "github", ActionName: "issues.create", Confidence: 0.91}

	store.held["apr-ctx"] = actions.ExecutionRequest{
		RequestID:      "req-ctx",
		TraceID:        "trace-ctx",
		TenantID:       "tenant-ctx",
		Principal:      actions.Principal{ID: "principal-ctx", TenantID: "tenant-ctx"},
		Action:         actions.ActionDefinition{Name: "github.issues.create", Adapter: actions.AdapterConfig{Type: "http", URLTemplate: "https://example.com"}},
		Input:          map[string]any{"title": "hello"},
		Mode:           actions.ModeCall,
		Session:        session,
		Classification: classInfo,
	}

	evaluator := &capturePolicyEvaluator{}
	rt := Runtime{
		PolicyEvaluator:    evaluator,
		HeldExecutionStore: store,
		Adapters: map[string]adapters.Adapter{
			"http": mockAdapter{kind: "http", result: &adapters.AdapterResult{Output: map[string]any{"ok": true}, HTTPStatus: 200}},
		},
	}

	res := rt.ResumeApproved(context.Background(), "apr-ctx")
	if res.Status != actions.StatusSuccess {
		t.Fatalf("expected success resume, got %s error=%+v", res.Status, res.Error)
	}
	if evaluator.lastReq.Session == nil || evaluator.lastReq.Session.SessionID != "sess-123" {
		t.Fatalf("policy request missing session context: %+v", evaluator.lastReq.Session)
	}
	if evaluator.lastReq.Classification == nil || evaluator.lastReq.Classification.Service != "github" {
		t.Fatalf("policy request missing classification context: %+v", evaluator.lastReq.Classification)
	}
}

func TestRuntimeResumeApprovedRetryableFailureReholdsExecution(t *testing.T) {
	store := &mockHeldExecutionStore{held: map[string]actions.ExecutionRequest{}}
	store.held["apr-retry"] = baseRequest(actions.ActionDefinition{
		Name:    "github.issues.create",
		Adapter: actions.AdapterConfig{Type: "http", URLTemplate: "https://example.com"},
	})

	rt := Runtime{
		HeldExecutionStore: store,
		Adapters: map[string]adapters.Adapter{
			"http": mockAdapter{kind: "http", err: actions.NewExecutionError(actions.ErrDownstreamUnavailable, "temporary downstream error", 502, true, nil)},
		},
	}

	res := rt.ResumeApproved(context.Background(), "apr-retry")
	if res.Status != actions.StatusError {
		t.Fatalf("expected error status, got %s", res.Status)
	}
	if res.Error == nil || !res.Error.Retryable {
		t.Fatalf("expected retryable execution error, got %+v", res.Error)
	}
	if store.holdCalls != 1 {
		t.Fatalf("expected held store Hold once for retryable failure, got %d", store.holdCalls)
	}
	if _, ok := store.held["apr-retry"]; !ok {
		t.Fatal("expected held request to be re-stored for retryable failure")
	}
}

func TestRuntimeResumeApprovedNonRetryableFailureDoesNotReholdExecution(t *testing.T) {
	store := &mockHeldExecutionStore{held: map[string]actions.ExecutionRequest{}}
	store.held["apr-noretry"] = baseRequest(actions.ActionDefinition{
		Name:    "github.issues.create",
		Adapter: actions.AdapterConfig{Type: "http", URLTemplate: "https://example.com"},
	})

	rt := Runtime{
		HeldExecutionStore: store,
		Adapters: map[string]adapters.Adapter{
			"http": mockAdapter{kind: "http", err: actions.NewExecutionError(actions.ErrValidationFailed, "invalid input", 400, false, nil)},
		},
	}

	res := rt.ResumeApproved(context.Background(), "apr-noretry")
	if res.Status != actions.StatusError {
		t.Fatalf("expected error status, got %s", res.Status)
	}
	if res.Error == nil || res.Error.Retryable {
		t.Fatalf("expected non-retryable execution error, got %+v", res.Error)
	}
	if store.holdCalls != 0 {
		t.Fatalf("expected held store Hold not called for non-retryable failure, got %d", store.holdCalls)
	}
	if _, ok := store.held["apr-noretry"]; ok {
		t.Fatal("expected held request to remain consumed after non-retryable failure")
	}
}

func TestRuntimeExecuteApprovalHoldFailureCancelsApprovalRequest(t *testing.T) {
	approvalManager := &mockApprovalManager{result: &ApprovalResult{Approved: false, RequestID: "apr-cancel"}}
	store := &mockHeldExecutionStore{held: map[string]actions.ExecutionRequest{}, holdErr: errors.New("disk full")}
	rt := Runtime{
		PolicyEvaluator:    mockPolicyEvaluator{decision: &PolicyDecision{Decision: "require_approval"}},
		ApprovalManager:    approvalManager,
		HeldExecutionStore: store,
		Adapters:           map[string]adapters.Adapter{"http": mockAdapter{kind: "http"}},
	}

	res := rt.Execute(context.Background(), baseRequest(actions.ActionDefinition{
		Name:    "github.issues.create",
		Adapter: actions.AdapterConfig{Type: "http", URLTemplate: "https://example.com"},
	}))

	if res.Status != actions.StatusError {
		t.Fatalf("expected error status, got %s", res.Status)
	}
	if res.Error == nil || res.Error.Code != actions.ErrDownstreamUnavailable {
		t.Fatalf("expected downstream unavailable error, got %+v", res.Error)
	}
	if approvalManager.cancelCalls != 1 {
		t.Fatalf("expected cancel request to be called once, got %d", approvalManager.cancelCalls)
	}
	if approvalManager.lastCancelledID != "apr-cancel" {
		t.Fatalf("expected canceled approval id apr-cancel, got %q", approvalManager.lastCancelledID)
	}
	if !strings.Contains(approvalManager.lastCancelledReason, "auto-cancel after hold failure") {
		t.Fatalf("expected auto-cancel reason, got %q", approvalManager.lastCancelledReason)
	}
}

func TestRuntimeExecuteApprovalHoldFailureCancelsApprovalWithCanceledContext(t *testing.T) {
	approvalManager := &mockApprovalManager{}
	rt := Runtime{ApprovalManager: approvalManager}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := rt.cancelApprovalOnHoldFailure(ctx, "apr-cancel-canceled-ctx", errors.New("disk full")); err != nil {
		t.Fatalf("cancel approval on hold failure: %v", err)
	}
	if approvalManager.cancelCalls != 1 {
		t.Fatalf("expected cancel request to be called once, got %d", approvalManager.cancelCalls)
	}
	if approvalManager.lastCancelCtxErr != nil {
		t.Fatalf("expected cancel compensation context to ignore parent cancellation, got %v", approvalManager.lastCancelCtxErr)
	}
}

func TestNewTraceCollectorUsesInjectedClock(t *testing.T) {
	t0 := time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC)
	times := []time.Time{t0, t0.Add(10 * time.Millisecond), t0.Add(25 * time.Millisecond)}
	idx := 0

	tc := NewTraceCollector(func() time.Time {
		if idx >= len(times) {
			return times[len(times)-1]
		}
		cur := times[idx]
		idx++
		return cur
	})

	tc.Record("step-1", "ok", "first")
	tc.Record("step-2", "ok", "second")

	if len(tc.Steps) != 2 {
		t.Fatalf("expected 2 trace steps, got %d", len(tc.Steps))
	}
	if tc.Steps[0].DurationMS != 10 {
		t.Fatalf("step-1 duration = %dms, want 10ms", tc.Steps[0].DurationMS)
	}
	if tc.Steps[1].DurationMS != 15 {
		t.Fatalf("step-2 duration = %dms, want 15ms", tc.Steps[1].DurationMS)
	}
}

func TestNewTraceCollectorNilClockFallsBackToTimeNow(t *testing.T) {
	tc := NewTraceCollector(nil)
	tc.Record("step", "ok", "fallback")
	if len(tc.Steps) != 1 {
		t.Fatalf("expected 1 trace step, got %d", len(tc.Steps))
	}
	if tc.Steps[0].DurationMS < 0 {
		t.Fatalf("duration should be non-negative, got %d", tc.Steps[0].DurationMS)
	}
}

func baseRequest(action actions.ActionDefinition) actions.ExecutionRequest {
	return actions.ExecutionRequest{
		RequestID:      "req-1",
		TraceID:        "trace-1",
		TenantID:       "tenant-1",
		IdempotencyKey: "idem-test-1",
		Principal:      actions.Principal{ID: "principal-1", TenantID: "tenant-1"},
		Action:         action,
		Input:          map[string]any{"owner": "dunia"},
		Mode:           actions.ModeCall,
	}
}

// ── Output Filter Pipeline Integration Tests ──────────────────────────────

func TestPipelineWithFilter_AppliedAfterAudit(t *testing.T) {
	audit := &mockAuditWriter{}
	rawOutput := map[string]any{
		"data": []any{
			map[string]any{"id": 1, "name": "x", "bio": "long field that should be stripped"},
		},
	}
	rt := Runtime{
		PolicyEvaluator: mockPolicyEvaluator{decision: &PolicyDecision{Decision: "allow"}},
		AuditWriter:     audit,
		Adapters: map[string]adapters.Adapter{
			"http": mockAdapter{kind: "http", result: &adapters.AdapterResult{Output: rawOutput, HTTPStatus: 200}},
		},
	}

	action := actions.ActionDefinition{
		Name:        "test.filter",
		InputSchema: &actions.Schema{Type: "object", AdditionalProperties: true},
		Adapter:     actions.AdapterConfig{Type: "http", URLTemplate: "https://example.com"},
		FilterConfig: &actions.FilterConfig{
			Select: map[string]string{"id": "id"},
		},
	}
	req := baseRequest(action)
	res := rt.Execute(context.Background(), req)

	if res.Status != actions.StatusSuccess {
		t.Fatalf("expected success, got %s: %v", res.Status, res.Error)
	}

	// Output should be filtered
	arr, ok := res.Output["data"].([]any)
	if !ok || len(arr) != 1 {
		t.Fatalf("expected filtered result array, got %v", res.Output)
	}
	item := arr[0].(map[string]any)
	if _, hasBio := item["bio"]; hasBio {
		t.Error("bio should be filtered from output")
	}
	if _, hasID := item["id"]; !hasID {
		t.Error("id should remain in filtered output")
	}

	// Meta should record filtering
	if res.Meta["filter_applied"] != true {
		t.Errorf("filter_applied = %v, want true", res.Meta["filter_applied"])
	}

	// Audit event should have been written (proves finalizeWithStatus ran)
	if len(audit.events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(audit.events))
	}
	if audit.events[0].Status != actions.StatusSuccess {
		t.Errorf("audit event status = %s, want success", audit.events[0].Status)
	}
}

func TestPipelineNoFilter_Passthrough(t *testing.T) {
	rawOutput := map[string]any{"data": []any{map[string]any{"id": 1, "bio": "preserved"}}}
	rt := Runtime{
		PolicyEvaluator: mockPolicyEvaluator{decision: &PolicyDecision{Decision: "allow"}},
		AuditWriter:     &mockAuditWriter{},
		Adapters: map[string]adapters.Adapter{
			"http": mockAdapter{kind: "http", result: &adapters.AdapterResult{Output: rawOutput, HTTPStatus: 200}},
		},
	}
	action := actions.ActionDefinition{
		Name:        "test.nofilter",
		InputSchema: &actions.Schema{Type: "object", AdditionalProperties: true},
		Adapter:     actions.AdapterConfig{Type: "http", URLTemplate: "https://example.com"},
		// No FilterConfig
	}
	res := rt.Execute(context.Background(), baseRequest(action))
	if res.Status != actions.StatusSuccess {
		t.Fatalf("expected success, got %s", res.Status)
	}
	arr := res.Output["data"].([]any)
	item := arr[0].(map[string]any)
	if item["bio"] != "preserved" {
		t.Error("without FilterConfig, output should pass through unchanged")
	}
	if res.Meta["filter_applied"] == true {
		t.Error("filter_applied should not be set when no filter config")
	}
}

func TestPipelineOutputModeRaw_BypassesFilter(t *testing.T) {
	rawOutput := map[string]any{
		"data": []any{
			map[string]any{"id": 1, "bio": "should remain when _output_mode=raw"},
		},
	}
	rt := Runtime{
		PolicyEvaluator: mockPolicyEvaluator{decision: &PolicyDecision{Decision: "allow"}},
		AuditWriter:     &mockAuditWriter{},
		Adapters: map[string]adapters.Adapter{
			"http": mockAdapter{kind: "http", result: &adapters.AdapterResult{Output: rawOutput, HTTPStatus: 200}},
		},
	}
	action := actions.ActionDefinition{
		Name:        "test.rawmode",
		InputSchema: &actions.Schema{Type: "object", AdditionalProperties: true},
		Adapter:     actions.AdapterConfig{Type: "http", URLTemplate: "https://example.com"},
		FilterConfig: &actions.FilterConfig{
			Select: map[string]string{"id": "id"},
		},
	}
	req := baseRequest(action)
	req.Input["_output_mode"] = "raw"
	res := rt.Execute(context.Background(), req)

	if res.Status != actions.StatusSuccess {
		t.Fatalf("expected success, got %s", res.Status)
	}
	arr := res.Output["data"].([]any)
	item := arr[0].(map[string]any)
	if _, hasBio := item["bio"]; !hasBio {
		t.Error("_output_mode=raw should bypass filter, bio should be present")
	}
}

func TestPipelineFilterErrorImmunity(t *testing.T) {
	// Error responses should not be filtered
	rt := Runtime{
		PolicyEvaluator: mockPolicyEvaluator{decision: &PolicyDecision{Decision: "allow"}},
		AuditWriter:     &mockAuditWriter{},
		Adapters: map[string]adapters.Adapter{
			"http": mockAdapter{kind: "http", err: errors.New("not found")},
		},
	}
	action := actions.ActionDefinition{
		Name:        "test.errimmune",
		InputSchema: &actions.Schema{Type: "object", AdditionalProperties: true},
		Adapter:     actions.AdapterConfig{Type: "http", URLTemplate: "https://example.com"},
		FilterConfig: &actions.FilterConfig{
			Select: map[string]string{"id": "id"},
		},
	}
	res := rt.Execute(context.Background(), baseRequest(action))
	if res.Status == actions.StatusSuccess {
		t.Fatal("expected error status")
	}
	// filter_applied should not be set on error
	if res.Meta["filter_applied"] == true {
		t.Error("filter should not be applied on error response")
	}
}

func TestPipelineBudget_Applied(t *testing.T) {
	// Build output with 20 items — after filter, budget should trim to fit
	items := make([]any, 20)
	for i := range items {
		items[i] = map[string]any{"id": i, "name": "item"}
	}
	rawOutput := map[string]any{"data": items}

	rt := Runtime{
		PolicyEvaluator: mockPolicyEvaluator{decision: &PolicyDecision{Decision: "allow"}},
		AuditWriter:     &mockAuditWriter{},
		Adapters: map[string]adapters.Adapter{
			"http": mockAdapter{kind: "http", result: &adapters.AdapterResult{Output: rawOutput, HTTPStatus: 200}},
		},
	}
	action := actions.ActionDefinition{
		Name:        "test.budget",
		InputSchema: &actions.Schema{Type: "object", AdditionalProperties: true},
		Adapter:     actions.AdapterConfig{Type: "http", URLTemplate: "https://example.com"},
		FilterConfig: &actions.FilterConfig{
			Select: map[string]string{"id": "id"},
		},
	}
	req := baseRequest(action)
	req.Input["_budget"] = int64(100) // CLI path produces int64
	res := rt.Execute(context.Background(), req)

	if res.Status != actions.StatusSuccess {
		t.Fatalf("expected success, got %s: %v", res.Status, res.Error)
	}
	// With budget=100 chars and 20 items, result should be trimmed
	arr, ok := res.Output["data"].([]any)
	if !ok {
		t.Fatal("result should be array")
	}
	if len(arr) == 20 {
		t.Error("budget should have trimmed the array from 20 items")
	}
}
