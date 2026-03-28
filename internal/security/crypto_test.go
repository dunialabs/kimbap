package security

import (
	"strings"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
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
