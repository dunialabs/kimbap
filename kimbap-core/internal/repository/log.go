package repository

import (
	"github.com/dunialabs/kimbap-core/internal/database"
	"gorm.io/gorm"
)

type LogRepository struct {
	db *gorm.DB
}

const maxFindLogsLimit = 1000

func NewLogRepository(db *gorm.DB) *LogRepository {
	if db == nil {
		db = database.DB
	}
	return &LogRepository{db: db}
}

func (r *LogRepository) FindLogsFromID(startID, limit int) ([]database.Log, error) {
	if limit <= 0 {
		return []database.Log{}, nil
	}
	if limit > maxFindLogsLimit {
		limit = maxFindLogsLimit
	}
	if startID <= 0 {
		startID = 1
	}
	var logs []database.Log
	err := r.db.Where("id >= ?", startID).Order("id asc").Limit(limit).Find(&logs).Error
	return logs, err
}

func (r *LogRepository) Save(log database.Log) (*database.Log, error) {
	if err := r.db.Create(&log).Error; err != nil {
		return nil, err
	}
	return &log, nil
}
