package utils

import (
	crand "crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/dunialabs/kimbap-core/internal/types"
)

func GenerateSessionID() (string, error) {
	randomBytes := make([]byte, 16)
	if _, err := crand.Read(randomBytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("session-%s", hex.EncodeToString(randomBytes)), nil
}

func OAuthProviderFromAuthType(authType int) string {
	switch authType {
	case types.ServerAuthTypeGoogleAuth, types.ServerAuthTypeGoogleCalendarAuth:
		return "google"
	case types.ServerAuthTypeNotionAuth:
		return "notion"
	case types.ServerAuthTypeFigmaAuth:
		return "figma"
	case types.ServerAuthTypeGithubAuth:
		return "github"
	case types.ServerAuthTypeCanvasAuth:
		return "canvas"
	case types.ServerAuthTypeCanvaAuth:
		return "canva"
	case types.ServerAuthTypeZendeskAuth:
		return "zendesk"
	default:
		return ""
	}
}
