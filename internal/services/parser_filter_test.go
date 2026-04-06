package services

import (
	"strings"
	"testing"
)

func TestManifestValidation_FilterMaxItems(t *testing.T) {
	// Negative max_items should fail validation
	manifest := validHTTPManifest()
	action := manifest.Actions["get_item"]
	action.Response.Filter = &FilterSpec{MaxItems: -1}
	manifest.Actions["get_item"] = action

	errs := ValidateManifest(manifest)
	if len(errs) == 0 {
		t.Error("expected validation error for negative max_items, got none")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e.Field, "max_items") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected max_items validation error, got: %v", errs)
	}
}

func TestManifestValidation_CompactItemRequired(t *testing.T) {
	// Empty compact.item should fail validation
	manifest := validHTTPManifest()
	action := manifest.Actions["get_item"]
	action.Response.Compact = &CompactSpec{Header: "Header:", Item: ""}
	manifest.Actions["get_item"] = action

	errs := ValidateManifest(manifest)
	if len(errs) == 0 {
		t.Error("expected validation error for empty compact.item, got none")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e.Field, "compact.item") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected compact.item validation error, got: %v", errs)
	}
}

func TestManifestValidation_FilterEmptySelectKey(t *testing.T) {
	// Empty output key in select should fail validation
	manifest := validHTTPManifest()
	action := manifest.Actions["get_item"]
	action.Response.Filter = &FilterSpec{
		Select: map[string]string{"": "id"}, // empty output key
	}
	manifest.Actions["get_item"] = action

	errs := ValidateManifest(manifest)
	if len(errs) == 0 {
		t.Error("expected validation error for empty select key, got none")
	}
}
