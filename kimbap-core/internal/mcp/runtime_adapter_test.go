package mcp

import "testing"

func TestNormalizeToolName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "valid dotted", input: "service.action", want: "service.action"},
		{name: "missing action", input: "service.", want: ""},
		{name: "missing service", input: ".action", want: ""},
		{name: "underscore fallback", input: "github_list_repos", want: "github.list_repos"},
		{name: "hyphen not supported", input: "github-list-repos", want: ""},
		{name: "empty", input: "", want: ""},
		{name: "no delimiter", input: "nodelimiter", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeToolName(tt.input)
			if got != tt.want {
				t.Fatalf("normalizeToolName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
