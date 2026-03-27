package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"

	"golang.org/x/crypto/pbkdf2"
)

const pbkdf2Iterations = 600_000

func DeriveKey(password []byte, salt []byte, keyLen int) ([]byte, error) {
	if len(password) == 0 {
		return nil, errors.New("password is required")
	}
	if len(salt) == 0 {
		return nil, errors.New("salt is required")
	}
	if keyLen <= 0 {
		return nil, errors.New("key length must be greater than zero")
	}

	return pbkdf2.Key(password, salt, pbkdf2Iterations, keyLen, sha256.New), nil
}

func GenerateSalt(size int) ([]byte, error) {
	if size <= 0 {
		return nil, errors.New("salt size must be greater than zero")
	}

	salt := make([]byte, size)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	return salt, nil
}

func GenerateRandomKey(size int) ([]byte, error) {
	if size <= 0 {
		return nil, errors.New("key size must be greater than zero")
	}

	key := make([]byte, size)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}

	return key, nil
}
