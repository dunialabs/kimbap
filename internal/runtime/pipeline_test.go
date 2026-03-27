package runtime

import (
	"context"
	"errors"
	"net/http"
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
	result *ApprovalResult
	err    error
}

func (m mockApprovalManager) CreateRequest(ctx context.Context, req ApprovalRequest) (*ApprovalResult, error) {
	return m.result, m.err
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
		PolicyEvaluator: mockPolicyEvaluator{decision: &PolicyDecision{Decision: "require_approval"}},
		ApprovalManager: mockApprovalManager{result: &ApprovalResult{Approved: false, RequestID: "apr-1"}},
		Adapters:        map[string]adapters.Adapter{"http": mockAdapter{kind: "http"}},
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

func TestRuntimeApprovalHoldAndResume(t *testing.T) {
	store := &mockHeldExecutionStore{held: map[string]actions.ExecutionRequest{}}
	rt := Runtime{
		ApprovalManager:    mockApprovalManager{result: &ApprovalResult{Approved: false, RequestID: "apr-42"}},
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
