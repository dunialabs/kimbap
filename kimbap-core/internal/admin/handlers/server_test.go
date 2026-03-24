package handlers

import "testing"

func TestParseCapabilitiesRejectsInvalidJSON(t *testing.T) {
	_, err := parseCapabilities("{")
	if err == nil {
		t.Fatal("expected parseCapabilities to fail on invalid JSON")
	}
}

func TestMergeCapabilitiesRejectsInvalidIncomingJSON(t *testing.T) {
	_, _, err := mergeCapabilities("{}", "{")
	if err == nil {
		t.Fatal("expected mergeCapabilities to fail on invalid incoming JSON")
	}
}

func TestMergeCapabilitiesAllowsRepairWhenExistingInvalid(t *testing.T) {
	merged, changed, err := mergeCapabilities("{", `{"tools":{"x":{"enabled":false}}}`)
	if err != nil {
		t.Fatalf("expected mergeCapabilities to allow repair, got error: %v", err)
	}
	if !changed {
		t.Fatal("expected mergeCapabilities to report changed for repair")
	}
	if merged == "{" {
		t.Fatal("expected merged output to differ from invalid existing payload")
	}
}

func TestMergeCapabilitiesRejectsInvalidIncomingEvenWhenEqualToExisting(t *testing.T) {
	_, _, err := mergeCapabilities("{", "{")
	if err == nil {
		t.Fatal("expected invalid incoming JSON to error even when equal to existing")
	}
}

func TestParseBoolLikeRejectsInvalidString(t *testing.T) {
	_, err := parseBoolLike("truthy", false)
	if err == nil {
		t.Fatal("expected parseBoolLike to reject malformed boolean string")
	}
}

func TestParseBoolLikeRejectsNonBinaryNumber(t *testing.T) {
	_, err := parseBoolLike(float64(2), false)
	if err == nil {
		t.Fatal("expected parseBoolLike to reject non-binary numeric values")
	}
}
