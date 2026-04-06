package security

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestDecryptLegacyFormat(t *testing.T) {
	plaintext := "legacy-sensitive-token"
	key := "legacy-encryption-key"
	salt := []byte("1234567890abcdef")
	iv := []byte("123456789012")

	derivedKey := deriveKey(key, salt, pbkdf2IterationsV0)
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		t.Fatalf("aes.NewCipher() error = %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("cipher.NewGCM() error = %v", err)
	}

	encrypted := gcm.Seal(nil, iv, []byte(plaintext), nil)
	ciphertext := encrypted[:len(encrypted)-gcmTagLength]
	tag := encrypted[len(encrypted)-gcmTagLength:]

	payload := encryptedData{
		Data: base64.StdEncoding.EncodeToString(ciphertext),
		IV:   base64.StdEncoding.EncodeToString(iv),
		Salt: base64.StdEncoding.EncodeToString(salt),
		Tag:  base64.StdEncoding.EncodeToString(tag),
	}

	buf, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	decrypted, err := DecryptDataFromString(string(buf), key)
	if err != nil {
		t.Fatalf("DecryptDataFromString() error = %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("decrypted value = %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptUsesNewFormat(t *testing.T) {
	encrypted, err := EncryptData("sensitive-token", "encryption-key")
	if err != nil {
		t.Fatalf("EncryptData() error = %v", err)
	}

	var payload encryptedData
	if err := json.Unmarshal([]byte(encrypted), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.Version != 1 {
		t.Fatalf("payload version = %d, want 1", payload.Version)
	}
}

func TestRoundTripNewFormat(t *testing.T) {
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
