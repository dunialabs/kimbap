package auth

import (
	"fmt"

	coretypes "github.com/dunialabs/kimbap-core/internal/types"
	"github.com/rs/zerolog/log"
)

type AuthStrategyFactory struct{}

func (f *AuthStrategyFactory) Create(authType int, oauthConfig map[string]any) (AuthStrategy, error) {
	switch authType {
	case coretypes.ServerAuthTypeGoogleAuth, coretypes.ServerAuthTypeGoogleCalendarAuth:
		return NewGoogleAuthStrategy(map[string]any{
			"clientId":     oauthConfig["clientId"],
			"clientSecret": oauthConfig["clientSecret"],
			"refreshToken": oauthConfig["refreshToken"],
		})
	case coretypes.ServerAuthTypeNotionAuth:
		return NewNotionAuthStrategy(oauthConfig)
	case coretypes.ServerAuthTypeFigmaAuth:
		return NewFigmaAuthStrategy(oauthConfig)
	case coretypes.ServerAuthTypeGithubAuth:
		return NewGithubAuthStrategy(oauthConfig)
	case coretypes.ServerAuthTypeCanvaAuth:
		return NewCanvaAuthStrategy(oauthConfig)
	case coretypes.ServerAuthTypeCanvasAuth:
		return NewCanvasAuthStrategy(oauthConfig)
	case coretypes.ServerAuthTypeZendeskAuth:
		return NewZendeskAuthStrategy(oauthConfig)
	case coretypes.ServerAuthTypeApiKey:
		return nil, nil
	default:
		log.Warn().Int("authType", authType).Msg("unsupported auth type")
		return nil, fmt.Errorf("unsupported auth type: %d", authType)
	}
}

func Create(authType int, oauthConfig map[string]any) (AuthStrategy, error) {
	return (&AuthStrategyFactory{}).Create(authType, oauthConfig)
}
