package crypto

import (
	"bytes"
	"testing"
)

func TestEnvelopeEncryptDecryptRoundTrip(t *testing.T) {
	masterKey, err := GenerateRandomKey(32)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	svc, err := NewEnvelopeService(masterKey)
	if err != nil {
		t.Fatalf("new envelope service: %v", err)
	}

	plaintext := []byte("super-secret-value")
	envelope, err := svc.Encrypt(plaintext, "")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	decrypted, err := svc.Decrypt(envelope)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("decrypted payload mismatch")
	}
}

func TestEnvelopeEncryptUsesDifferentNonces(t *testing.T) {
	masterKey, err := GenerateRandomKey(32)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	svc, err := NewEnvelopeService(masterKey)
	if err != nil {
		t.Fatalf("new envelope service: %v", err)
	}

	plaintext := []byte("same-payload")
	one, err := svc.Encrypt(plaintext, "")
	if err != nil {
		t.Fatalf("encrypt one: %v", err)
	}
	two, err := svc.Encrypt(plaintext, "")
	if err != nil {
		t.Fatalf("encrypt two: %v", err)
	}

	if bytes.Equal(one.Nonce, two.Nonce) {
		t.Fatalf("expected unique data nonce per encryption")
	}
	if bytes.Equal(one.DEKNonce, two.DEKNonce) {
		t.Fatalf("expected unique DEK nonce per encryption")
	}
	if bytes.Equal(one.Ciphertext, two.Ciphertext) {
		t.Fatalf("expected different ciphertext for same plaintext")
	}
}

func TestEnvelopeDetectsTamperedCiphertext(t *testing.T) {
	masterKey, err := GenerateRandomKey(32)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	svc, err := NewEnvelopeService(masterKey)
	if err != nil {
		t.Fatalf("new envelope service: %v", err)
	}

	envelope, err := svc.Encrypt([]byte("tamper-me"), "")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	envelope.Ciphertext[0] ^= 0xFF
	if _, err := svc.Decrypt(envelope); err == nil {
		t.Fatalf("expected tamper detection error")
	}
}

func TestDeriveKeyDeterministic(t *testing.T) {
	salt, err := GenerateSalt(16)
	if err != nil {
		t.Fatalf("generate salt: %v", err)
	}

	one, err := DeriveKey([]byte("hunter2"), salt, 32)
	if err != nil {
		t.Fatalf("derive one: %v", err)
	}
	two, err := DeriveKey([]byte("hunter2"), salt, 32)
	if err != nil {
		t.Fatalf("derive two: %v", err)
	}

	if !bytes.Equal(one, two) {
		t.Fatalf("expected deterministic derivation")
	}
}

func TestDeriveKeyDifferentSaltDifferentOutput(t *testing.T) {
	oneSalt, err := GenerateSalt(16)
	if err != nil {
		t.Fatalf("generate salt one: %v", err)
	}
	twoSalt, err := GenerateSalt(16)
	if err != nil {
		t.Fatalf("generate salt two: %v", err)
	}

	one, err := DeriveKey([]byte("hunter2"), oneSalt, 32)
	if err != nil {
		t.Fatalf("derive one: %v", err)
	}
	two, err := DeriveKey([]byte("hunter2"), twoSalt, 32)
	if err != nil {
		t.Fatalf("derive two: %v", err)
	}

	if bytes.Equal(one, two) {
		t.Fatalf("expected different derived keys for different salts")
	}
}

func TestEnvelopeEmptyPlaintext(t *testing.T) {
	masterKey, err := GenerateRandomKey(32)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	svc, err := NewEnvelopeService(masterKey)
	if err != nil {
		t.Fatalf("new envelope service: %v", err)
	}

	envelope, err := svc.Encrypt(nil, "")
	if err != nil {
		t.Fatalf("encrypt empty: %v", err)
	}

	decrypted, err := svc.Decrypt(envelope)
	if err != nil {
		t.Fatalf("decrypt empty: %v", err)
	}

	if len(decrypted) != 0 {
		t.Fatalf("expected empty payload")
	}
}

func TestEnvelopeLargePayload(t *testing.T) {
	masterKey, err := GenerateRandomKey(32)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	svc, err := NewEnvelopeService(masterKey)
	if err != nil {
		t.Fatalf("new envelope service: %v", err)
	}

	plaintext := bytes.Repeat([]byte("0123456789abcdef"), 256*1024)
	envelope, err := svc.Encrypt(plaintext, "")
	if err != nil {
		t.Fatalf("encrypt large: %v", err)
	}

	decrypted, err := svc.Decrypt(envelope)
	if err != nil {
		t.Fatalf("decrypt large: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("large payload mismatch")
	}
}

func TestEnvelopeUnicodeAndBinary(t *testing.T) {
	masterKey, err := GenerateRandomKey(32)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	svc, err := NewEnvelopeService(masterKey)
	if err != nil {
		t.Fatalf("new envelope service: %v", err)
	}

	plaintext := append([]byte("秘密🍙"), []byte{0x00, 0x01, 0xFE, 0xFF}...)
	envelope, err := svc.Encrypt(plaintext, "")
	if err != nil {
		t.Fatalf("encrypt mixed: %v", err)
	}

	decrypted, err := svc.Decrypt(envelope)
	if err != nil {
		t.Fatalf("decrypt mixed: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("unicode/binary payload mismatch")
	}
}
