package services

import (
	"testing"
)

func TestParseManifest_WithFilterSpec(t *testing.T) {
	yaml := `name: test-http
version: 1.0.0
adapter: http
base_url: https://api.example.com
auth:
  type: none
actions:
  list:
    risk:
      level: low
    method: GET
    path: /items
    response:
      extract: "items"
      type: array
      filter:
        select:
          id: "id"
          name: "name"
        exclude: ["password"]
        max_items: 50
        drop_nulls: true
`

	m, err := ParseManifest([]byte(yaml))
	if err != nil {
		t.Fatalf("parse manifest failed: %v", err)
	}
	a, ok := m.Actions["list"]
	if !ok {
		t.Fatalf("expected action 'list' to exist")
	}
	if a.Response.Filter == nil {
		t.Fatalf("expected response.filter to be parsed")
	}
	if a.Response.Filter.MaxItems != 50 {
		t.Fatalf("expected filter.max_items=50, got %d", a.Response.Filter.MaxItems)
	}
	if a.Response.Filter.DropNulls != true {
		t.Fatalf("expected filter.drop_nulls=true, got %v", a.Response.Filter.DropNulls)
	}
	if v := a.Response.Filter.Select["id"]; v != "id" {
		t.Fatalf("unexpected select.id value: %q", v)
	}
	if excl := len(a.Response.Filter.Exclude); excl != 1 {
		t.Fatalf("expected 1 exclude, got %d", excl)
	}
}
