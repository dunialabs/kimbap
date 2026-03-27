package actions

import (
	"strings"
	"testing"
)

func TestValidateInputRequiredFieldPresent(t *testing.T) {
	schema := &Schema{
		Type:     "object",
		Required: []string{"name"},
		Properties: map[string]*Schema{
			"name": {Type: "string"},
		},
		AdditionalProperties: false,
	}

	err := ValidateInput(schema, map[string]any{"name": "kim"})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestValidateInputRequiredFieldMissing(t *testing.T) {
	schema := &Schema{
		Type:     "object",
		Required: []string{"name"},
		Properties: map[string]*Schema{
			"name": {Type: "string"},
		},
		AdditionalProperties: false,
	}

	err := ValidateInput(schema, map[string]any{})
	if err == nil {
		t.Fatalf("expected error for missing required field")
	}
	if err.Code != ErrValidationFailed {
		t.Fatalf("expected %q, got %q", ErrValidationFailed, err.Code)
	}
	if err.Details["field"] != "name" {
		t.Fatalf("expected missing field detail to be name, got %v", err.Details["field"])
	}
}

func TestValidateInputTypeString(t *testing.T) {
	schema := &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"name": {Type: "string"},
		},
		AdditionalProperties: true,
	}

	err := ValidateInput(schema, map[string]any{"name": 42})
	if err == nil {
		t.Fatalf("expected type validation error")
	}
	if !strings.Contains(err.Message, "must be string") {
		t.Fatalf("expected string type error message, got %q", err.Message)
	}
}

func TestValidateInputTypeInteger(t *testing.T) {
	schema := &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"age": {Type: "integer"},
		},
		AdditionalProperties: true,
	}

	err := ValidateInput(schema, map[string]any{"age": "30"})
	if err == nil {
		t.Fatalf("expected type validation error")
	}
	if !strings.Contains(err.Message, "must be integer") {
		t.Fatalf("expected integer type error message, got %q", err.Message)
	}
}

func TestValidateInputTypeBoolean(t *testing.T) {
	schema := &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"enabled": {Type: "boolean"},
		},
		AdditionalProperties: true,
	}

	err := ValidateInput(schema, map[string]any{"enabled": "true"})
	if err == nil {
		t.Fatalf("expected type validation error")
	}
	if !strings.Contains(err.Message, "must be boolean") {
		t.Fatalf("expected boolean type error message, got %q", err.Message)
	}
}

func TestValidateInputEnumValid(t *testing.T) {
	schema := &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"status": {Type: "string", Enum: []any{"pending", "approved", "denied"}},
		},
		AdditionalProperties: true,
	}

	err := ValidateInput(schema, map[string]any{"status": "approved"})
	if err != nil {
		t.Fatalf("expected nil error for valid enum, got %v", err)
	}
}

func TestValidateInputEnumInvalid(t *testing.T) {
	schema := &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"status": {Type: "string", Enum: []any{"pending", "approved", "denied"}},
		},
		AdditionalProperties: true,
	}

	err := ValidateInput(schema, map[string]any{"status": "blocked"})
	if err == nil {
		t.Fatalf("expected enum validation error")
	}
	if !strings.Contains(err.Message, "invalid enum value") {
		t.Fatalf("expected enum error message, got %q", err.Message)
	}
}

func TestValidateInputEnumNumericCrossTypeMatch(t *testing.T) {
	schema := &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"tier": {Type: "integer", Enum: []any{1, 2, 3}},
		},
		AdditionalProperties: true,
	}

	if err := ValidateInput(schema, map[string]any{"tier": float64(2)}); err != nil {
		t.Fatalf("expected numeric enum to match across int/float decode types, got %v", err)
	}

	if err := ValidateInput(schema, map[string]any{"tier": uint8(3)}); err != nil {
		t.Fatalf("expected numeric enum to match uint/int variants, got %v", err)
	}
}

func TestValidateInputExtraFieldIgnored(t *testing.T) {
	schema := &Schema{
		Type:                 "object",
		Properties:           map[string]*Schema{},
		AdditionalProperties: true,
	}

	err := ValidateInput(schema, map[string]any{"unexpected": "ok"})
	if err != nil {
		t.Fatalf("expected nil error for additional properties, got %v", err)
	}
}

func TestValidateInputEmptySchemaPassesAll(t *testing.T) {
	err := ValidateInput(&Schema{}, map[string]any{"anything": 123, "nested": map[string]any{"x": true}})
	if err != nil {
		t.Fatalf("expected nil error for empty schema, got %v", err)
	}
}

func TestValidateInputTypeNumberAcceptsFloatAndInteger(t *testing.T) {
	schema := &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"price": {Type: "number"},
		},
		AdditionalProperties: false,
	}

	if err := ValidateInput(schema, map[string]any{"price": 12.5}); err != nil {
		t.Fatalf("expected float number to pass, got %v", err)
	}

	if err := ValidateInput(schema, map[string]any{"price": 12}); err != nil {
		t.Fatalf("expected integer number to pass, got %v", err)
	}

	err := ValidateInput(schema, map[string]any{"price": "12.5"})
	if err == nil {
		t.Fatalf("expected number type validation error")
	}
	if !strings.Contains(err.Message, "must be number") {
		t.Fatalf("expected number type error message, got %q", err.Message)
	}
}

func TestValidateInputRejectsUnknownNestedFieldWhenStrictSchema(t *testing.T) {
	schema := &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"config": {
				Type: "object",
				Properties: map[string]*Schema{
					"name": {Type: "string"},
				},
				AdditionalProperties: false,
			},
		},
		AdditionalProperties: true,
	}

	valid := map[string]any{"config": map[string]any{"name": "test"}}
	if err := ValidateInput(schema, valid); err != nil {
		t.Fatalf("expected valid nested input to pass, got %v", err)
	}

	invalid := map[string]any{"config": map[string]any{"name": "test", "extra": "bad"}}
	err := ValidateInput(schema, invalid)
	if err == nil {
		t.Fatal("expected unknown nested field to be rejected when AdditionalProperties is false")
	}
	if !strings.Contains(err.Message, "unknown nested field") {
		t.Fatalf("expected unknown nested field error, got %q", err.Message)
	}
}
