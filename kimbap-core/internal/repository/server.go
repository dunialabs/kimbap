package repository

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/dunialabs/kimbap-core/internal/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ServerCapabilitiesCacheInput struct {
	Tools             any
	Resources         any
	ResourceTemplates any
	Prompts           any
}

type ServerRepository struct {
	db *gorm.DB
}

func NewServerRepository(db *gorm.DB) *ServerRepository {
	if db == nil {
		db = database.DB
	}
	return &ServerRepository{db: db}
}

func (r *ServerRepository) FindAll() ([]database.Server, error) {
	var servers []database.Server
	err := r.db.Find(&servers).Error
	return servers, err
}

func (r *ServerRepository) Upsert(serverID string, createData *database.Server, updateData map[string]any) (*database.Server, error) {
	if createData == nil {
		return nil, errors.New("createData is required")
	}
	createData.ServerID = serverID
	if updateData != nil {
		delete(updateData, "server_id")
	}
	if len(updateData) == 0 {
		err := r.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "server_id"}},
			DoNothing: true,
		}).Create(createData).Error
		if err != nil {
			return nil, err
		}
		return r.FindByServerID(serverID)
	}

	err := r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "server_id"}},
		DoUpdates: clause.Assignments(updateData),
	}).Create(createData).Error
	if err != nil {
		return nil, err
	}

	return r.FindByServerID(serverID)
}

func (r *ServerRepository) FindAllEnabled() ([]database.Server, error) {
	var servers []database.Server
	err := r.db.Where("enabled = ?", true).Find(&servers).Error
	if err != nil {
		return nil, err
	}
	return servers, nil
}

func (r *ServerRepository) FindByServerID(serverID string) (*database.Server, error) {
	var server database.Server
	err := r.db.Where("server_id = ?", serverID).First(&server).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &server, nil
}

func (r *ServerRepository) Update(serverID string, updates map[string]any) (*database.Server, error) {
	updates["updated_at"] = int(time.Now().Unix())
	result := r.db.Model(&database.Server{}).Where("server_id = ?", serverID).Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return r.FindByServerID(serverID)
}

func (r *ServerRepository) Exists(serverID string) (bool, error) {
	var count int64
	err := r.db.Model(&database.Server{}).Where("server_id = ?", serverID).Count(&count).Error
	return count > 0, err
}

func (r *ServerRepository) UpdateLaunchConfig(serverID string, launchConfig string) (*database.Server, error) {
	return r.Update(serverID, map[string]any{"launch_config": launchConfig})
}

func (r *ServerRepository) UpdateCapabilities(serverID string, capabilities string) (*database.Server, error) {
	return r.Update(serverID, map[string]any{"capabilities": capabilities})
}

func (r *ServerRepository) UpdateCapabilitiesCache(serverID string, data ServerCapabilitiesCacheInput) error {
	updates := map[string]any{}

	if data.Tools != nil {
		b, err := json.Marshal(data.Tools)
		if err != nil {
			return err
		}
		updates["cached_tools"] = string(b)
	}
	if data.Resources != nil {
		b, err := json.Marshal(data.Resources)
		if err != nil {
			return err
		}
		updates["cached_resources"] = string(b)
	}
	if data.ResourceTemplates != nil {
		b, err := json.Marshal(data.ResourceTemplates)
		if err != nil {
			return err
		}
		updates["cached_resource_templates"] = string(b)
	}
	if data.Prompts != nil {
		b, err := json.Marshal(data.Prompts)
		if err != nil {
			return err
		}
		updates["cached_prompts"] = string(b)
	}

	if len(updates) == 0 {
		return nil
	}

	return r.db.Model(&database.Server{}).Where("server_id = ?", serverID).Updates(updates).Error
}
