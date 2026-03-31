package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"sync"
)

const (
	defaultKeyID      = "default"
	envelopeAlgorithm = "AES-256-GCM"
	envelopeVersion   = 1
	nonceSize         = 12
	dekSize           = 32
)

var (
	ErrInvalidMasterKey = errors.New("master key must be 32 bytes")
	ErrEnvelopeNil      = errors.New("encrypted envelope is required")
	ErrKEKNotFound      = errors.New("key encryption key not found")
)

// EnvelopeService provides encryption/decryption using AES-256-GCM.
//
// Envelope flow:
//   - Plaintext is encrypted with a per-record DEK (AES-256-GCM)
//   - The DEK is encrypted (wrapped) with a KEK identified by KeyID
type EnvelopeService struct {
	mu   sync.RWMutex
	keys map[string][]byte
}

// EncryptedEnvelope holds an encrypted value with metadata.
type EncryptedEnvelope struct {
	Ciphertext []byte // AES-256-GCM encrypted data
	Nonce      []byte // unique per encryption, 12 bytes
	Salt       []byte // for KDF if using password-based
	KeyID      string // identifies which KEK was used
	Algorithm  string // "AES-256-GCM"
	Version    int    // envelope format version

	WrappedDEK []byte // DEK encrypted with KEK
	DEKNonce   []byte // nonce used for WrappedDEK encryption
}

func NewEnvelopeService(masterKey []byte) (*EnvelopeService, error) {
	if len(masterKey) != dekSize {
		return nil, ErrInvalidMasterKey
	}

	copyKey := make([]byte, len(masterKey))
	copy(copyKey, masterKey)

	return &EnvelopeService{
		keys: map[string][]byte{defaultKeyID: copyKey},
	}, nil
}

func (e *EnvelopeService) Encrypt(plaintext []byte, keyID string) (*EncryptedEnvelope, error) {
	if e == nil {
		return nil, errors.New("envelope service is nil")
	}
	if keyID == "" {
		keyID = defaultKeyID
	}

	kek, err := e.getKey(keyID)
	if err != nil {
		return nil, err
	}

	dek, err := GenerateRandomKey(dekSize)
	if err != nil {
		return nil, fmt.Errorf("generate DEK: %w", err)
	}

	dataNonce, err := GenerateRandomKey(nonceSize)
	if err != nil {
		return nil, fmt.Errorf("generate data nonce: %w", err)
	}

	dataCiphertext, err := sealGCM(dek, dataNonce, plaintext)
	if err != nil {
		return nil, fmt.Errorf("encrypt plaintext: %w", err)
	}

	dekNonce, err := GenerateRandomKey(nonceSize)
	if err != nil {
		return nil, fmt.Errorf("generate DEK nonce: %w", err)
	}

	wrappedDEK, err := sealGCM(kek, dekNonce, dek)
	if err != nil {
		return nil, fmt.Errorf("encrypt DEK: %w", err)
	}

	return &EncryptedEnvelope{
		Ciphertext: dataCiphertext,
		Nonce:      dataNonce,
		KeyID:      keyID,
		Algorithm:  envelopeAlgorithm,
		Version:    envelopeVersion,
		WrappedDEK: wrappedDEK,
		DEKNonce:   dekNonce,
	}, nil
}

func (e *EnvelopeService) Decrypt(envelope *EncryptedEnvelope) ([]byte, error) {
	if e == nil {
		return nil, errors.New("envelope service is nil")
	}
	if envelope == nil {
		return nil, ErrEnvelopeNil
	}
	if envelope.Algorithm != "" && envelope.Algorithm != envelopeAlgorithm {
		return nil, fmt.Errorf("unsupported algorithm: %s", envelope.Algorithm)
	}
	if len(envelope.Nonce) != nonceSize {
		return nil, errors.New("invalid data nonce size")
	}
	if len(envelope.DEKNonce) != nonceSize {
		return nil, errors.New("invalid DEK nonce size")
	}

	keyID := envelope.KeyID
	if keyID == "" {
		keyID = defaultKeyID
	}

	kek, err := e.getKey(keyID)
	if err != nil {
		if errors.Is(err, ErrKEKNotFound) && keyID != defaultKeyID {
			if ensureErr := e.EnsureKey(keyID); ensureErr != nil {
				return nil, ensureErr
			}
			kek, err = e.getKey(keyID)
		}
		if err != nil {
			return nil, err
		}
	}

	dek, err := openGCM(kek, envelope.DEKNonce, envelope.WrappedDEK)
	if err != nil {
		return nil, fmt.Errorf("decrypt DEK: %w", err)
	}

	plaintext, err := openGCM(dek, envelope.Nonce, envelope.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decrypt payload: %w", err)
	}

	return plaintext, nil
}

func (e *EnvelopeService) EnsureKey(keyID string) error {
	if e == nil {
		return errors.New("envelope service is nil")
	}
	if keyID == "" {
		return errors.New("key ID is required")
	}

	e.mu.RLock()
	_, exists := e.keys[keyID]
	e.mu.RUnlock()
	if exists {
		return nil
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	if _, exists := e.keys[keyID]; exists {
		return nil
	}

	masterKey, ok := e.keys[defaultKeyID]
	if !ok {
		return ErrKEKNotFound
	}

	derived, err := DeriveKey(masterKey, []byte(keyID), dekSize)
	if err != nil {
		return fmt.Errorf("derive tenant key: %w", err)
	}

	e.keys[keyID] = derived
	return nil
}

func (e *EnvelopeService) RotateKey(oldKeyID, newKeyID string, newKey []byte) error {
	if e == nil {
		return errors.New("envelope service is nil")
	}
	if oldKeyID == "" {
		oldKeyID = defaultKeyID
	}
	if newKeyID == "" {
		return errors.New("new key ID is required")
	}
	if len(newKey) != dekSize {
		return ErrInvalidMasterKey
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if _, ok := e.keys[oldKeyID]; !ok {
		return ErrKEKNotFound
	}

	copyKey := make([]byte, len(newKey))
	copy(copyKey, newKey)
	e.keys[newKeyID] = copyKey

	return nil
}

func (e *EnvelopeService) getKey(keyID string) ([]byte, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	kek, ok := e.keys[keyID]
	if !ok {
		return nil, ErrKEKNotFound
	}

	copyKey := make([]byte, len(kek))
	copy(copyKey, kek)
	return copyKey, nil
}

func sealGCM(key []byte, nonce []byte, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Seal(nil, nonce, plaintext, nil), nil
}

func openGCM(key []byte, nonce []byte, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, ciphertext, nil)
}
