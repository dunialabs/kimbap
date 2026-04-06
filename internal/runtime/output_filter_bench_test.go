package runtime

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/dunialabs/kimbap/internal/actions"
)

func BenchmarkFilterSelect(b *testing.B) {
	data, err := os.ReadFile("testdata/github_list_repos.json")
	if err != nil {
		b.Skipf("fixture not found: %v", err)
	}
	var repos []any
	if err := json.Unmarshal(data, &repos); err != nil {
		b.Fatalf("parse fixture: %v", err)
	}
	output := map[string]any{"result": repos}
	config := &actions.FilterConfig{
		Select: map[string]string{
			"id":          "id",
			"name":        "name",
			"full_name":   "full_name",
			"html_url":    "html_url",
			"description": "description",
			"updated_at":  "updated_at",
		},
		MaxItems: 20,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ApplyFilter(output, config)
	}
}

func BenchmarkFilterDropNulls(b *testing.B) {
	data, err := os.ReadFile("testdata/github_list_issues.json")
	if err != nil {
		b.Skipf("fixture not found: %v", err)
	}
	var issues []any
	if err := json.Unmarshal(data, &issues); err != nil {
		b.Fatalf("parse fixture: %v", err)
	}
	output := map[string]any{"result": issues}
	config := &actions.FilterConfig{
		Exclude:   []string{"body_html", "reactions", "timeline_url", "performed_via_github_app"},
		DropNulls: true,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ApplyFilter(output, config)
	}
}
