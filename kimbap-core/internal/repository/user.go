package repository

import (
	"bytes"
	"encoding/json"
	"errors"
	"time"

	"github.com/dunialabs/kimbap-core/internal/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	if db == nil {
		db = database.DB
	}
	return &UserRepository{db: db}
}

func (r *UserRepository) Upsert(userID string, createData *database.User, updateData map[string]any) (*database.User, error) {
	if createData == nil {
		return nil, errors.New("createData is required")
	}
	createData.UserID = userID
	if updateData != nil {
		delete(updateData, "user_id")
	}
	if len(updateData) == 0 {
		err := r.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoNothing: true,
		}).Create(createData).Error
		if err != nil {
			return nil, err
		}
		return r.FindByUserID(userID)
	}

	err := r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(updateData),
	}).Create(createData).Error
	if err != nil {
		return nil, err
	}

	return r.FindByUserID(userID)
}

func (r *UserRepository) FindAll() ([]database.User, error) {
	var users []database.User
	err := r.db.Find(&users).Error
	return users, err
}

func (r *UserRepository) FindByUserID(userID string) (*database.User, error) {
	var user database.User
	err := r.db.Where("user_id = ?", userID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Update(userID string, updates map[string]any) (*database.User, error) {
	if updates == nil {
		updates = map[string]any{}
	}
	delete(updates, "user_id")
	updates["updated_at"] = int(time.Now().Unix())
	result := r.db.Model(&database.User{}).Where("user_id = ?", userID).Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return r.FindByUserID(userID)
}

func (r *UserRepository) UpdatePermissions(userID string, permissions any) (*database.User, error) {
	b, err := json.Marshal(permissions)
	if err != nil {
		return nil, err
	}
	b = normalizeJSONObjectBytes(b)
	return r.Update(userID, map[string]any{"permissions": string(b)})
}

func (r *UserRepository) UpdateUserPreferences(userID string, userPreferences any) (*database.User, error) {
	b, err := json.Marshal(userPreferences)
	if err != nil {
		return nil, err
	}
	b = normalizeJSONObjectBytes(b)
	return r.Update(userID, map[string]any{"user_preferences": string(b)})
}

func (r *UserRepository) UpdateLaunchConfigs(userID string, launchConfigs any) (*database.User, error) {
	b, err := json.Marshal(launchConfigs)
	if err != nil {
		return nil, err
	}
	b = normalizeJSONObjectBytes(b)
	return r.Update(userID, map[string]any{"launch_configs": string(b)})
}

func normalizeJSONObjectBytes(b []byte) []byte {
	if bytes.Equal(bytes.TrimSpace(b), []byte("null")) {
		return []byte("{}")
	}
	return b
}

func (r *UserRepository) Exists(userID string) (bool, error) {
	var count int64
	err := r.db.Model(&database.User{}).Where("user_id = ?", userID).Count(&count).Error
	return count > 0, err
}

func (r *UserRepository) RemoveServerFromAllUsers(serverID string) error {
	now := int(time.Now().Unix())
	return r.db.Model(&database.User{}).
		Where("CASE WHEN launch_configs LIKE '{%' THEN jsonb_exists(launch_configs::jsonb, ?) ELSE false END", serverID).
		Or("CASE WHEN user_preferences LIKE '{%' THEN jsonb_exists(user_preferences::jsonb, ?) ELSE false END", serverID).
		Updates(map[string]any{
			"launch_configs":   gorm.Expr("CASE WHEN launch_configs LIKE '{%' THEN (launch_configs::jsonb - ?)::text ELSE launch_configs END", serverID),
			"user_preferences": gorm.Expr("CASE WHEN user_preferences LIKE '{%' THEN (user_preferences::jsonb - ?)::text ELSE user_preferences END", serverID),
			"updated_at":       now,
		}).Error
}
