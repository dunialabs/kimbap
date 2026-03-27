package flows

import (
	"strings"
	"testing"

	"github.com/dunialabs/kimbap/internal/connectors"
)

func TestFlowSelectorSelectFlow_RequestedUnsupportedReturnsError(t *testing.T) {
	selector := &FlowSelector{}
	provider := connectors.ProviderDefinition{
		ID:             "demo",
		SupportedFlows: []connectors.FlowType{connectors.FlowDevice},
	}

	_, err := selector.SelectFlow(connectors.FlowBrowser, provider)
	if err == nil {
		t.Fatal("expected unsupported requested flow to fail")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFlowSelectorSelectFlow_RequestedSupportedWins(t *testing.T) {
	selector := &FlowSelector{}
	provider := connectors.ProviderDefinition{
		ID:             "demo",
		SupportedFlows: []connectors.FlowType{connectors.FlowBrowser, connectors.FlowDevice},
	}

	flow, err := selector.SelectFlow(connectors.FlowDevice, provider)
	if err != nil {
		t.Fatalf("SelectFlow() error = %v", err)
	}
	if flow != connectors.FlowDevice {
		t.Fatalf("flow = %q, want %q", flow, connectors.FlowDevice)
	}
}

func TestFlowSelectorSelectFlow_AutoFallsBackToDevice(t *testing.T) {
	selector := &FlowSelector{}
	provider := connectors.ProviderDefinition{
		ID:             "demo",
		SupportedFlows: []connectors.FlowType{connectors.FlowDevice},
	}

	flow, err := selector.SelectFlow("", provider)
	if err != nil {
		t.Fatalf("SelectFlow(auto) error = %v", err)
	}
	if flow != connectors.FlowDevice {
		t.Fatalf("flow = %q, want %q", flow, connectors.FlowDevice)
	}
}

func TestFlowSelectorSelectFlow_AutoFallsBackToClientCredentials(t *testing.T) {
	selector := &FlowSelector{}
	provider := connectors.ProviderDefinition{
		ID:             "demo",
		SupportedFlows: []connectors.FlowType{connectors.FlowClientCredentials},
	}

	flow, err := selector.SelectFlow(connectors.FlowType("auto"), provider)
	if err != nil {
		t.Fatalf("SelectFlow(auto) error = %v", err)
	}
	if flow != connectors.FlowClientCredentials {
		t.Fatalf("flow = %q, want %q", flow, connectors.FlowClientCredentials)
	}
}

func TestFlowSelectorSelectFlow_NoViableFlowReturnsError(t *testing.T) {
	selector := &FlowSelector{}
	provider := connectors.ProviderDefinition{ID: "demo"}

	_, err := selector.SelectFlow("", provider)
	if err == nil {
		t.Fatal("expected no viable flow error")
	}
	if !strings.Contains(err.Error(), "no viable oauth flow") {
		t.Fatalf("unexpected error: %v", err)
	}
}
