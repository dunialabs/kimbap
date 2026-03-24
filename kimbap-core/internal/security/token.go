package security

import (
	"errors"
	"regexp"
	"strings"
)

var traditionalTokenPattern = regexp.MustCompile(`^[a-f0-9]{128}$`)

type TokenValidator struct{}

func NewTokenValidator() *TokenValidator {
	return &TokenValidator{}
}

func (v *TokenValidator) ValidateToken(token string) (string, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", errors.New("authorization token is required")
	}
	if !traditionalTokenPattern.MatchString(token) {
		return "", errors.New("invalid authorization token format")
	}
	return CalculateUserID(token), nil
}
