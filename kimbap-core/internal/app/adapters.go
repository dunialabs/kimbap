package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/dunialabs/kimbap-core/internal/database"
	"github.com/dunialabs/kimbap-core/internal/mcp/auth"
	"github.com/dunialabs/kimbap-core/internal/mcp/core"
	"github.com/dunialabs/kimbap-core/internal/middleware"
	"github.com/dunialabs/kimbap-core/internal/repository"
	"github.com/dunialabs/kimbap-core/internal/security"
	coretypes "github.com/dunialabs/kimbap-core/internal/types"
)

type UserRepoAdapter struct{}

type OauthUserRepoAdapter struct{}

type OauthUserValidatorAdapter struct {
	Validator interface {
		ValidateToken(token string) (string, error)
	}
}

type ServerRepoAdapter struct{}

type ManagerUserRepoAdapter struct{}

type AuthFactoryAdapter struct{}

type UserTokenValidatorAdapter struct {
	AuthMW *middleware.AuthMiddleware
}

type CoreAuthStrategyAdapter struct {
	strategy auth.AuthStrategy
}

type EventRepoAdapter struct {
	repo *repository.EventRepository
}

func (UserRepoAdapter) FindByUserID(ctx context.Context, userID string) (*middleware.User, error) {
	_ = ctx
	entity, err := repository.NewUserRepository(nil).FindByUserID(userID)
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, nil
	}
	encToken := ""
	if entity.EncryptedToken != nil {
		encToken = *entity.EncryptedToken
	}
	return &middleware.User{
		UserID:          entity.UserID,
		Role:            entity.Role,
		Status:          entity.Status,
		ExpiresAtUnix:   int64(entity.ExpiresAt),
		RateLimit:       entity.Ratelimit,
		Permissions:     json.RawMessage(entity.Permissions),
		UserPreferences: json.RawMessage(entity.UserPreferences),
		LaunchConfigs:   json.RawMessage(entity.LaunchConfigs),
		EncryptedToken:  encToken,
	}, nil
}

func (OauthUserRepoAdapter) FindByUserID(ctx context.Context, userID string) (*security.UserRecord, error) {
	_ = ctx
	entity, err := repository.NewUserRepository(nil).FindByUserID(userID)
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, nil
	}
	return &security.UserRecord{
		UserID:    entity.UserID,
		Status:    entity.Status,
		ExpiresAt: int64(entity.ExpiresAt),
	}, nil
}

func (a OauthUserValidatorAdapter) ValidateUserToken(token string) (string, error) {
	if a.Validator == nil {
		return "", errors.New("token validator is not configured")
	}

	userID, err := a.Validator.ValidateToken(token)
	if err != nil {
		return "", err
	}

	entity, err := repository.NewUserRepository(nil).FindByUserID(userID)
	if err != nil {
		return "", err
	}
	if entity == nil {
		return "", errors.New("user not found")
	}
	if entity.Status != coretypes.UserStatusEnabled {
		return "", errors.New("user is disabled")
	}
	if entity.ExpiresAt > 0 && time.Now().Unix() > int64(entity.ExpiresAt) {
		return "", errors.New("user authorization has expired")
	}

	return userID, nil
}

func (ServerRepoAdapter) FindAllEnabled(ctx context.Context) ([]database.Server, error) {
	_ = ctx
	return repository.NewServerRepository(nil).FindAllEnabled()
}

func (ServerRepoAdapter) FindByServerID(ctx context.Context, serverID string) (*database.Server, error) {
	_ = ctx
	return repository.NewServerRepository(nil).FindByServerID(serverID)
}

func (ServerRepoAdapter) UpdateCapabilities(ctx context.Context, serverID string, caps string) error {
	_ = ctx
	_, err := repository.NewServerRepository(nil).UpdateCapabilities(serverID, caps)
	return err
}

func (ServerRepoAdapter) UpdateCapabilitiesCache(ctx context.Context, serverID string, data map[string]any) error {
	_ = ctx
	input := repository.ServerCapabilitiesCacheInput{
		Tools:             data["tools"],
		Resources:         data["resources"],
		ResourceTemplates: data["resourceTemplates"],
		Prompts:           data["prompts"],
	}
	return repository.NewServerRepository(nil).UpdateCapabilitiesCache(serverID, input)
}

func (ServerRepoAdapter) UpdateTransportType(ctx context.Context, serverID string, transportType string) error {
	_ = ctx
	_, err := repository.NewServerRepository(nil).Update(serverID, map[string]any{"transport_type": transportType})
	return err
}

func (ServerRepoAdapter) UpdateServerName(ctx context.Context, serverID string, serverName string) error {
	_ = ctx
	_, err := repository.NewServerRepository(nil).Update(serverID, map[string]any{"server_name": serverName})
	return err
}

func (ManagerUserRepoAdapter) FindByUserID(ctx context.Context, userID string) (*database.User, error) {
	_ = ctx
	return repository.NewUserRepository(nil).FindByUserID(userID)
}

func (ManagerUserRepoAdapter) UpdateLaunchConfigs(ctx context.Context, userID string, launchConfigs string) error {
	_ = ctx
	_, err := repository.NewUserRepository(nil).Update(userID, map[string]any{"launch_configs": launchConfigs})
	return err
}

func (ManagerUserRepoAdapter) UpdateUserPreferences(ctx context.Context, userID string, userPreferences string) error {
	_ = ctx
	_, err := repository.NewUserRepository(nil).Update(userID, map[string]any{"user_preferences": userPreferences})
	return err
}

func (a AuthFactoryAdapter) Build(ctx context.Context, server database.Server, launchConfig map[string]any, userToken string) (core.AuthStrategy, error) {
	_ = ctx
	if server.AuthType == coretypes.ServerAuthTypeApiKey {
		return nil, nil
	}

	oauthConfig, ok := launchConfig["oauth"].(map[string]any)
	if !ok || oauthConfig == nil {
		return nil, errors.New("missing OAuth configuration")
	}

	strategyConfig := make(map[string]interface{}, len(oauthConfig)+2)
	for key, value := range oauthConfig {
		strategyConfig[key] = value
	}
	if server.UseKimbapOauthConfig {
		strategyConfig["userToken"] = userToken
		strategyConfig["server"] = server
		strategy, err := auth.NewKimbapAuthStrategy(strategyConfig)
		if err != nil {
			return nil, err
		}
		return CoreAuthStrategyAdapter{strategy: strategy}, nil
	}

	strategy, err := auth.Create(server.AuthType, strategyConfig)
	if err != nil {
		return nil, err
	}
	if strategy == nil {
		return nil, nil
	}

	return CoreAuthStrategyAdapter{strategy: strategy}, nil
}

func (a UserTokenValidatorAdapter) ValidateToken(token string) (*coretypes.AuthContext, error) {
	if a.AuthMW == nil {
		return nil, errors.New("auth middleware is not configured")
	}
	req, err := http.NewRequest(http.MethodPost, "/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	return a.AuthMW.AuthenticateRequest(req)
}

func (a CoreAuthStrategyAdapter) GetInitialToken(ctx context.Context) (string, int64, error) {
	_ = ctx
	if a.strategy == nil {
		return "", 0, errors.New("auth strategy is not configured")
	}
	tokenInfo, err := a.strategy.GetInitialToken()
	if err != nil {
		return "", 0, err
	}
	return tokenInfo.AccessToken, tokenInfo.ExpiresAt, nil
}

func (a CoreAuthStrategyAdapter) RefreshToken(ctx context.Context) (string, int64, error) {
	_ = ctx
	if a.strategy == nil {
		return "", 0, errors.New("auth strategy is not configured")
	}
	tokenInfo, err := a.strategy.RefreshToken()
	if err != nil {
		return "", 0, err
	}
	return tokenInfo.AccessToken, tokenInfo.ExpiresAt, nil
}

func (a CoreAuthStrategyAdapter) GetCurrentOAuthConfig() map[string]interface{} {
	if a.strategy == nil {
		return nil
	}
	return a.strategy.GetCurrentOAuthConfig()
}

func (a CoreAuthStrategyAdapter) MarkConfigAsPersisted() {
	if a.strategy == nil {
		return
	}
	a.strategy.MarkConfigAsPersisted()
}

func NewEventRepoAdapter() *EventRepoAdapter {
	return &EventRepoAdapter{repo: repository.NewEventRepository(nil)}
}

func (a *EventRepoAdapter) Create(ctx context.Context, event *database.Event) error {
	if a == nil || a.repo == nil {
		return errors.New("event repository is not configured")
	}
	_, err := a.repo.Create(ctx, event)
	return err
}

func (a *EventRepoAdapter) FindByStreamIDAfter(ctx context.Context, streamID string, afterEventID string) ([]database.Event, error) {
	if a == nil || a.repo == nil {
		return nil, errors.New("event repository is not configured")
	}
	return a.repo.FindAfterEventID(ctx, streamID, afterEventID)
}

func (a *EventRepoAdapter) DeleteExpired(ctx context.Context) (int64, error) {
	if a == nil || a.repo == nil {
		return 0, errors.New("event repository is not configured")
	}
	return a.repo.DeleteExpired(ctx)
}

func (a *EventRepoAdapter) DeleteByStreamID(ctx context.Context, streamID string) (int64, error) {
	if a == nil || a.repo == nil {
		return 0, errors.New("event repository is not configured")
	}
	return a.repo.DeleteByStreamID(ctx, streamID)
}
