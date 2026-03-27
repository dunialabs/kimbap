package runtime

import (
	"context"
	"maps"
	"testing"

	"github.com/dunialabs/kimbap/internal/actions"
	"github.com/dunialabs/kimbap/internal/adapters"
)

type inputCaptureAdapter struct {
	seenInput map[string]any
}

func (a *inputCaptureAdapter) Type() string {
	return "http"
}

func (a *inputCaptureAdapter) Validate(def actions.ActionDefinition) error {
	return nil
}

func (a *inputCaptureAdapter) Execute(ctx context.Context, req adapters.AdapterRequest) (*adapters.AdapterResult, error) {
	a.seenInput = map[string]any{}
	maps.Copy(a.seenInput, req.Input)
	return &adapters.AdapterResult{Output: map[string]any{"ok": true}, HTTPStatus: 200}, nil
}

func TestPipeline_DefaultsAppliedBeforeValidation(t *testing.T) {
	adapter := &inputCaptureAdapter{}
	rt := Runtime{
		Adapters: map[string]adapters.Adapter{"http": adapter},
	}

	req := baseRequest(actions.ActionDefinition{
		Name:        "service.list",
		Defaults:    map[string]any{"limit": 100},
		InputSchema: &actions.Schema{Type: "object", Required: []string{"limit"}, Properties: map[string]*actions.Schema{"limit": {Type: "integer"}}},
		Adapter:     actions.AdapterConfig{Type: "http", URLTemplate: "https://example.com"},
	})
	req.Input = map[string]any{}

	res := rt.Execute(context.Background(), req)
	if res.Status != actions.StatusSuccess {
		t.Fatalf("expected success with defaults applied, got status=%s error=%+v", res.Status, res.Error)
	}
	if got := adapter.seenInput["limit"]; got != 100 {
		t.Fatalf("expected adapter to receive default limit=100, got %v", got)
	}
}

func TestPipeline_DefaultsDoNotOverrideProvidedInput(t *testing.T) {
	adapter := &inputCaptureAdapter{}
	rt := Runtime{
		Adapters: map[string]adapters.Adapter{"http": adapter},
	}

	req := baseRequest(actions.ActionDefinition{
		Name:        "service.list",
		Defaults:    map[string]any{"limit": 100},
		InputSchema: &actions.Schema{Type: "object", Required: []string{"limit"}, Properties: map[string]*actions.Schema{"limit": {Type: "integer"}}},
		Adapter:     actions.AdapterConfig{Type: "http", URLTemplate: "https://example.com"},
	})
	req.Input = map[string]any{"limit": 50}

	res := rt.Execute(context.Background(), req)
	if res.Status != actions.StatusSuccess {
		t.Fatalf("expected success, got status=%s error=%+v", res.Status, res.Error)
	}
	if got := adapter.seenInput["limit"]; got != 50 {
		t.Fatalf("expected provided limit=50 to be preserved, got %v", got)
	}
}

func TestPipeline_NilDefaultsNoop(t *testing.T) {
	adapter := &inputCaptureAdapter{}
	rt := Runtime{
		Adapters: map[string]adapters.Adapter{"http": adapter},
	}

	req := baseRequest(actions.ActionDefinition{
		Name:     "service.list",
		Defaults: nil,
		Adapter:  actions.AdapterConfig{Type: "http", URLTemplate: "https://example.com"},
	})
	req.Input = map[string]any{"page": 2}
	original := maps.Clone(req.Input)

	res := rt.Execute(context.Background(), req)
	if res.Status != actions.StatusSuccess {
		t.Fatalf("expected success with nil defaults, got status=%s error=%+v", res.Status, res.Error)
	}
	if got := adapter.seenInput["page"]; got != 2 {
		t.Fatalf("expected adapter to receive unchanged input page=2, got %v", got)
	}
	if len(adapter.seenInput) != len(original) {
		t.Fatalf("expected unchanged input size=%d, got %d", len(original), len(adapter.seenInput))
	}
	for k, v := range original {
		if got, ok := adapter.seenInput[k]; !ok || got != v {
			t.Fatalf("expected unchanged input key %q=%v, got %v (exists=%v)", k, v, got, ok)
		}
	}
}
