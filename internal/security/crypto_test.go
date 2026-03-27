package security

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCalculateUserIDDeterministic(t *testing.T) {
	a := CalculateUserID("token-a")
	b := CalculateUserID("token-a")
	c := CalculateUserID("token-b")

	if a != b {
		t.Fatalf("same token should produce same user id: %q vs %q", a, b)
	}
	if a == c {
		t.Fatalf("different token should produce different user id: %q", a)
	}
}

func TestEncryptDecryptRoundTripAndVerifyToken(t *testing.T) {
	plaintext := "sensitive-token"
	key := "encryption-key"

	encrypted, err := EncryptData(plaintext, key)
	if err != nil {
		t.Fatalf("EncryptData() error = %v", err)
	}
	decrypted, err := DecryptDataFromString(encrypted, key)
	if err != nil {
		t.Fatalf("DecryptDataFromString() error = %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("decrypted value = %q, want %q", decrypted, plaintext)
	}

	encryptedForVerify, err := EncryptData(plaintext, plaintext)
	if err != nil {
		t.Fatalf("EncryptData() for VerifyTokenAgainstEncrypted error = %v", err)
	}

	if !VerifyTokenAgainstEncrypted(plaintext, encryptedForVerify) {
		t.Fatal("VerifyTokenAgainstEncrypted should succeed with matching token")
	}
	if VerifyTokenAgainstEncrypted("wrong-token", encryptedForVerify) {
		t.Fatal("VerifyTokenAgainstEncrypted should fail with non-matching token")
	}
}

func TestEncryptDataToObjectAndEncryptedAnyToString(t *testing.T) {
	obj, err := EncryptDataToObject("secret", "key")
	if err != nil {
		t.Fatalf("EncryptDataToObject() error = %v", err)
	}
	if obj["data"] == "" || obj["iv"] == "" || obj["salt"] == "" || obj["tag"] == "" {
		t.Fatalf("encrypted object missing required fields: %+v", obj)
	}

	str, err := EncryptedAnyToString(obj)
	if err != nil {
		t.Fatalf("EncryptedAnyToString(map) error = %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(str), &parsed); err != nil {
		t.Fatalf("marshal/unmarshal encrypted object string failed: %v", err)
	}

	passthrough, err := EncryptedAnyToString(str)
	if err != nil {
		t.Fatalf("EncryptedAnyToString(string) error = %v", err)
	}
	if passthrough != str {
		t.Fatalf("passthrough string mismatch")
	}

	if _, err := EncryptedAnyToString(123); err == nil {
		t.Fatal("unsupported encrypted data type should fail")
	}
}

func TestDecryptDataValidationErrors(t *testing.T) {
	if _, err := DecryptDataFromString("", "key"); err == nil {
		t.Fatal("empty encrypted string should fail")
	}
	if _, err := DecryptDataFromString("{}", ""); err == nil {
		t.Fatal("empty key should fail")
	}

	if _, err := DecryptDataFromString("not-json", "key"); err == nil || !strings.Contains(err.Error(), "invalid JSON format") {
		t.Fatalf("expected invalid JSON format error, got %v", err)
	}

	if _, err := DecryptDataFromString(`{"data":"x","iv":"x","salt":"x"}`, "key"); err == nil || !strings.Contains(err.Error(), "missing required fields") {
		t.Fatalf("expected missing required fields error, got %v", err)
	}
}

func TestDecryptFailsWithWrongKeyAndTamperedPayload(t *testing.T) {
	encrypted, err := EncryptData("secret", "correct-key")
	if err != nil {
		t.Fatalf("EncryptData() error = %v", err)
	}

	if _, err := DecryptDataFromString(encrypted, "wrong-key"); err == nil {
		t.Fatal("decrypt with wrong key should fail")
	}

	tampered := strings.Replace(encrypted, "\"data\":\"", "\"data\":\"A", 1)
	if _, err := DecryptDataFromString(tampered, "correct-key"); err == nil {
		t.Fatal("decrypt with tampered payload should fail")
	}
}

func TestEncryptDataUsesRandomSaltAndIV(t *testing.T) {
	first, err := EncryptData("same-plaintext", "same-key")
	if err != nil {
		t.Fatalf("EncryptData(first) error = %v", err)
	}
	second, err := EncryptData("same-plaintext", "same-key")
	if err != nil {
		t.Fatalf("EncryptData(second) error = %v", err)
	}
	if first == second {
		t.Fatal("expected randomized encryption output to differ across calls")
	}
}
