package service

import (
	"testing"

	mcptypes "github.com/dunialabs/kimbap-core/internal/mcp/types"
)

func TestIsToolCapabilityListChangedDetectsDisabledEntryAddOrRemove(t *testing.T) {
	oldCaps := map[string]mcptypes.ToolCapabilityConfig{}
	newCaps := map[string]mcptypes.ToolCapabilityConfig{"t1": {Enabled: false}}
	if !isToolCapabilityListChanged(oldCaps, newCaps) {
		t.Fatal("expected add of disabled tool entry to be treated as changed")
	}

	if !isToolCapabilityListChanged(newCaps, oldCaps) {
		t.Fatal("expected remove of disabled tool entry to be treated as changed")
	}
}

func TestIsResourceCapabilityListChangedDetectsDisabledEntryAddOrRemove(t *testing.T) {
	oldCaps := map[string]mcptypes.ResourceCapabilityConfig{}
	newCaps := map[string]mcptypes.ResourceCapabilityConfig{"r1": {Enabled: false}}
	if !isResourceCapabilityListChanged(oldCaps, newCaps) {
		t.Fatal("expected add of disabled resource entry to be treated as changed")
	}

	if !isResourceCapabilityListChanged(newCaps, oldCaps) {
		t.Fatal("expected remove of disabled resource entry to be treated as changed")
	}
}

func TestIsPromptCapabilityListChangedDetectsDisabledEntryAddOrRemove(t *testing.T) {
	oldCaps := map[string]mcptypes.PromptCapabilityConfig{}
	newCaps := map[string]mcptypes.PromptCapabilityConfig{"p1": {Enabled: false}}
	if !isPromptCapabilityListChanged(oldCaps, newCaps) {
		t.Fatal("expected add of disabled prompt entry to be treated as changed")
	}

	if !isPromptCapabilityListChanged(newCaps, oldCaps) {
		t.Fatal("expected remove of disabled prompt entry to be treated as changed")
	}
}
