package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/dunialabs/kimbap-core/internal/database"
	"github.com/dunialabs/kimbap-core/internal/security"
	"gorm.io/gorm"
)

type OAuthTokenRepository struct {
	db *gorm.DB
}

func NewOAuthTokenRepository(db *gorm.DB) *OAuthTokenRepository {
	if db == nil {
		db = database.DB
	}
	return &OAuthTokenRepository{db: db}
}

func (r *OAuthTokenRepository) FindByAccessToken(ctx context.Context, accessToken string) (*security.OAuthTokenRecord, error) {
	var token database.OAuthToken
	err := r.db.WithContext(ctx).Where("access_token = ?", accessToken).First(&token).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	scopes := []string{}
	if err := json.Unmarshal(token.Scopes, &scopes); err != nil {
		scopes = nil
	}

	return &security.OAuthTokenRecord{
		AccessToken: token.AccessToken,
		UserID:      token.UserID,
		ClientID:    token.ClientID,
		Scopes:      scopes,
		ExpiresAt:   token.AccessTokenExpiresAt,
		Revoked:     token.Revoked,
	}, nil
}
