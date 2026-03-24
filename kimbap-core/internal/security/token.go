package security

import (
	"errors"
	"regexp"
	"strings"
)

var traditionalTokenPattern = regexp.MustCompile(`^[a-f0-9]{128}$`)

// IsTraditionalTokenFormat reports whether token matches the 128-hex-char format.
func IsTraditionalTokenFormat(token string) bool {
	return traditionalTokenPattern.MatchString(token)
}

type TokenValidator struct{}

func NewTokenValidator() *TokenValidator {
	return &TokenValidator{}
}

func (v *TokenValidator) ValidateToken(token string) (string, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", errors.New("authorization token is required")
	}
	if !IsTraditionalTokenFormat(token) {
		return "", errors.New("invalid authorization token format")
	}
	return CalculateUserID(token), nil
}
