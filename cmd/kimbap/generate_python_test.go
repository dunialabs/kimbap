package main

import (
	"strings"
	"testing"

	"github.com/dunialabs/kimbap/internal/actions"
)

func TestRenderPythonSnippets_UsesRequiredCompatibilityImport(t *testing.T) {
	grouped := map[string][]actions.ActionDefinition{
		"demo": {
			{
				Name:      "demo.send",
				Namespace: "demo",
				InputSchema: &actions.Schema{
					Type: "object",
					Properties: map[string]*actions.Schema{
						"from": {Type: "string"},
					},
					Required: []string{"from"},
				},
			},
		},
	}

	out := renderPythonSnippets(grouped)

	for _, want := range []string{
		"from typing import Any, TypedDict",
		"try:\n    from typing import Required",
		"except ImportError:\n    from typing_extensions import Required",
		"Required[str],",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected generated output to contain %q, got:\n%s", want, out)
		}
	}
}
