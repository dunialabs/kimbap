package repository

import (
	"errors"

	"github.com/dunialabs/kimbap-core/internal/database"
	"gorm.io/gorm"
)

type ProxyRepository struct {
	db *gorm.DB
}

func NewProxyRepository(db *gorm.DB) *ProxyRepository {
	if db == nil {
		db = database.DB
	}
	return &ProxyRepository{db: db}
}

func (r *ProxyRepository) FindFirst() (*database.Proxy, error) {
	var proxy database.Proxy
	err := r.db.First(&proxy).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &proxy, nil
}

func (r *ProxyRepository) Update(id int, updates map[string]any) (*database.Proxy, error) {
	result := r.db.Model(&database.Proxy{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	var proxy database.Proxy
	err := r.db.Where("id = ?", id).First(&proxy).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &proxy, nil
}
