package runtime

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dunialabs/kimbap-core/internal/actions"
	"github.com/dunialabs/kimbap-core/internal/adapters"
)

type mockPolicyEvaluator struct {
	decision *PolicyDecision
	err      error
}

func (m mockPolicyEvaluator) Evaluate(ctx context.Context, req PolicyRequest) (*PolicyDecision, error) {
	return m.decision, m.err
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
