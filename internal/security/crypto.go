package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	pbkdf2Iterations = 100000
	keyLengthBytes   = 32
	ivLengthBytes    = 12
	saltLengthBytes  = 16
	gcmTagLength     = 16
)

type encryptedData struct {
	Data string `json:"data"`
	IV   string `json:"iv"`
	Salt string `json:"salt"`
	Tag  string `json:"tag"`
}

func EncryptData(plaintext string, key string) (string, error) {
	if key == "" {
		return "", errors.New("invalid encryption key: must be a non-empty string")
	}

	salt := make([]byte, saltLengthBytes)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	iv := make([]byte, ivLengthBytes)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", fmt.Errorf("failed to generate iv: %w", err)
	}

	derivedKey := deriveKey(key, salt)
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM cipher: %w", err)
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
		return "", fmt.Errorf("failed to marshal encrypted payload: %w", err)
	}

	return string(buf), nil
}

func DecryptDataFromString(encryptedStr string, key string) (string, error) {
	if encryptedStr == "" {
		return "", errors.New("invalid encrypted data string: must be a non-empty string")
	}
	if key == "" {
		return "", errors.New("invalid decryption key: must be a non-empty string")
	}

	var payload encryptedData
	if err := json.Unmarshal([]byte(encryptedStr), &payload); err != nil {
		return "", errors.New("invalid JSON format in encrypted data string")
	}
	if payload.Data == "" || payload.IV == "" || payload.Salt == "" || payload.Tag == "" {
		return "", errors.New("missing required fields in encrypted data object")
	}

	ciphertext, err := base64.StdEncoding.DecodeString(payload.Data)
	if err != nil {
		return "", errors.New("invalid base64 encoding in encrypted data")
	}
	iv, err := base64.StdEncoding.DecodeString(payload.IV)
	if err != nil {
		return "", errors.New("invalid base64 encoding in encrypted data")
	}
	salt, err := base64.StdEncoding.DecodeString(payload.Salt)
	if err != nil {
		return "", errors.New("invalid base64 encoding in encrypted data")
	}
	tag, err := base64.StdEncoding.DecodeString(payload.Tag)
	if err != nil {
		return "", errors.New("invalid base64 encoding in encrypted data")
	}

	if len(iv) != ivLengthBytes {
		return "", errors.New("decryption failed: invalid key or corrupted data")
	}

	combined := make([]byte, 0, len(ciphertext)+len(tag))
	combined = append(combined, ciphertext...)
	combined = append(combined, tag...)

	derivedKey := deriveKey(key, salt)
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return "", errors.New("decryption failed: invalid key or corrupted data")
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", errors.New("decryption failed: invalid key or corrupted data")
	}

	plaintxt, err := gcm.Open(nil, iv, combined, nil)
	if err != nil {
		return "", errors.New("decryption failed: invalid key or corrupted data")
	}

	return string(plaintxt), nil
}

func deriveKey(password string, salt []byte) []byte {
	return pbkdf2.Key([]byte(password), salt, pbkdf2Iterations, keyLengthBytes, sha256.New)
}
