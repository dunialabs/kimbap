package main

import (
	"strings"
	"testing"

	"github.com/dunialabs/kimbap/internal/actions"
)

func TestRenderTypeScriptSnippets_IncludesIndexSignatureForFreeformInput(t *testing.T) {
	grouped := map[string][]actions.ActionDefinition{
		"demo": {
			{
				Name:      "demo.send",
				Namespace: "demo",
				InputSchema: &actions.Schema{
					Type:                 "object",
					AdditionalProperties: true,
				},
			},
		},
	}

	out := renderTypeScriptSnippets(grouped)
	if !strings.Contains(out, "[key: string]: unknown;") {
		t.Fatalf("expected generated TypeScript to include freeform index signature, got:\n%s", out)
	}
}

func TestRenderPythonSnippets_UsesDictAliasForFreeformInput(t *testing.T) {
	grouped := map[string][]actions.ActionDefinition{
		"demo": {
			{
				Name:      "demo.send",
				Namespace: "demo",
				InputSchema: &actions.Schema{
					Type:                 "object",
					AdditionalProperties: true,
				},
			},
		},
	}

	out := renderPythonSnippets(grouped)
	if !strings.Contains(out, "DemoSendInput = dict[str, Any]") {
		t.Fatalf("expected generated Python to include dict alias for freeform input, got:\n%s", out)
	}
}

func TestRenderPythonSnippets_UsesDictAliasForMixedFreeformInput(t *testing.T) {
	grouped := map[string][]actions.ActionDefinition{
		"demo": {
			{
				Name:      "demo.send",
				Namespace: "demo",
				InputSchema: &actions.Schema{
					Type: "object",
					Properties: map[string]*actions.Schema{
						"title": {Type: "string"},
					},
					AdditionalProperties: true,
				},
			},
		},
	}

	out := renderPythonSnippets(grouped)
	if !strings.Contains(out, "DemoSendInput = dict[str, Any]") {
		t.Fatalf("expected generated Python to use dict alias for mixed freeform input, got:\n%s", out)
	}
}
