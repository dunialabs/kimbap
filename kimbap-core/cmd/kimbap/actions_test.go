package main

import (
	"testing"

	"github.com/dunialabs/kimbap-core/internal/actions"
)

func TestBriefOutputFormat(t *testing.T) {
	defs := []actions.ActionDefinition{
		{Name: "github.issues.create", Description: "Create an issue", Risk: actions.RiskWrite},
		{Name: "brave-search.web-search", Description: "Search the web", Risk: actions.RiskRead},
	}

	for _, def := range defs {
		brief := map[string]string{
			"name":        def.Name,
			"description": def.Description,
			"risk":        string(def.Risk),
		}
		if brief["name"] == "" {
			t.Fatal("brief output missing name")
		}
		if brief["description"] == "" {
			t.Fatal("brief output missing description")
		}
		if brief["risk"] == "" {
			t.Fatal("brief output missing risk")
		}
	}
}
