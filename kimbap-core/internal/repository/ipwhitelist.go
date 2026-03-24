package repository

import (
	"time"

	"github.com/dunialabs/kimbap-core/internal/database"
	"gorm.io/gorm"
)

type IPWhitelistRepository struct {
	db *gorm.DB
}

func NewIPWhitelistRepository(db *gorm.DB) *IPWhitelistRepository {
	if db == nil {
		db = database.DB
	}
	return &IPWhitelistRepository{db: db}
}

func (r *IPWhitelistRepository) FindAll() ([]database.IPWhitelist, error) {
	var rows []database.IPWhitelist
	err := r.db.Order("id asc").Find(&rows).Error
	return rows, err
}

func (r *IPWhitelistRepository) Create(ip string) (*database.IPWhitelist, error) {
	row := &database.IPWhitelist{IP: ip, Addtime: int(time.Now().Unix())}
	if err := r.db.Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

func (r *IPWhitelistRepository) DeleteByIP(ip string) (bool, error) {
	result := r.db.Where("ip = ?", ip).Delete(&database.IPWhitelist{})
	return result.RowsAffected > 0, result.Error
}

func (r *IPWhitelistRepository) ReplaceAll(ips []string) (int64, error) {
	var inserted int64
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&database.IPWhitelist{}).Error; err != nil {
			return err
		}
		if len(ips) == 0 {
			inserted = 0
			return nil
		}
		now := int(time.Now().Unix())
		rows := make([]database.IPWhitelist, 0, len(ips))
		for _, ip := range ips {
			rows = append(rows, database.IPWhitelist{IP: ip, Addtime: now})
		}
		if err := tx.Create(&rows).Error; err != nil {
			return err
		}
		inserted = int64(len(rows))
		return nil
	})
	return inserted, err
}

func (r *IPWhitelistRepository) Exists(ip string) (bool, error) {
	var count int64
	err := r.db.Model(&database.IPWhitelist{}).Where("ip = ?", ip).Count(&count).Error
	return count > 0, err
}
