package handlers

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/dunialabs/kimbap-core/internal/database"
	"github.com/dunialabs/kimbap-core/internal/security"
	types "github.com/dunialabs/kimbap-core/internal/types"
	"gorm.io/gorm"
)

type IPWhitelistHandler struct {
	db      *gorm.DB
	service *security.IPWhitelistService
}

func NewIPWhitelistHandler(db *gorm.DB, svc *security.IPWhitelistService) *IPWhitelistHandler {
	if db == nil {
		db = database.DB
	}
	return &IPWhitelistHandler{db: db, service: svc}
}

func (h *IPWhitelistHandler) Update(data map[string]any) (any, error) {
	rawWhitelist, exists := data["whitelist"]
	if !exists {
		return nil, &types.AdminError{Message: "invalid whitelist format", Code: types.AdminErrorCodeInvalidRequest}
	}

	list, isArray := normalizeWhitelist(rawWhitelist)
	if !isArray {
		return nil, &types.AdminError{Message: "invalid whitelist format", Code: types.AdminErrorCodeInvalidRequest}
	}
	validatedList := make([]string, 0, len(list))
	for _, ip := range list {
		ip = strings.TrimSpace(ip)
		if ip == "" || !isValidIPOrCIDR(ip) {
			return nil, &types.AdminError{Message: fmt.Sprintf("Invalid IP/CIDR format: %s", ip), Code: types.AdminErrorCodeInvalidIPFormat}
		}
		validatedList = append(validatedList, ip)
	}
	list = validatedList

	timestamp := int(time.Now().Unix())
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("1 = 1").Delete(&database.IPWhitelist{}).Error; err != nil {
			return err
		}
		for _, ip := range list {
			if err := tx.Create(&database.IPWhitelist{IP: ip, Addtime: timestamp}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := h.reloadWhitelist(); err != nil {
		return nil, err
	}
	loadedWhitelist := list
	if h.service != nil {
		loadedWhitelist = h.service.GetAll()
	}
	return map[string]any{
		"whitelist": loadedWhitelist,
		"message":   fmt.Sprintf("IP whitelist updated successfully. %d IPs loaded.", len(loadedWhitelist)),
	}, nil
}

func normalizeWhitelist(v any) ([]string, bool) {
	switch raw := v.(type) {
	case []string:
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			out = append(out, strings.TrimSpace(item))
		}
		return out, true
	case []any:
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			out = append(out, strings.TrimSpace(fmt.Sprint(item)))
		}
		return out, true
	default:
		return nil, false
	}
}

func (h *IPWhitelistHandler) Get() (any, error) {
	var records []database.IPWhitelist
	if err := h.db.Order("id asc").Find(&records).Error; err != nil {
		return nil, err
	}
	whitelist := make([]string, 0, len(records))
	for _, r := range records {
		whitelist = append(whitelist, r.IP)
	}

	list := []string{"0.0.0.0/0"}
	if len(whitelist) > 0 {
		hasNonWildcard := false
		for _, ip := range whitelist {
			if ip != "0.0.0.0/0" {
				hasNonWildcard = true
				break
			}
		}
		if hasNonWildcard {
			list = whitelist
		}
	}

	return map[string]any{"whitelist": list, "count": len(list)}, nil
}

func (h *IPWhitelistHandler) Delete(data map[string]any) (any, error) {
	ips := toStringSlice(data["ips"])
	if len(ips) == 0 {
		return nil, &types.AdminError{Message: "invalid ips array", Code: types.AdminErrorCodeInvalidRequest}
	}
	res := h.db.Where("ip IN ?", ips).Delete(&database.IPWhitelist{})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return map[string]any{"deletedCount": 0, "message": "No matching IPs found"}, nil
	}
	if err := h.reloadWhitelist(); err != nil {
		return nil, err
	}
	deletedCount := int(res.RowsAffected)
	return map[string]any{"deletedCount": deletedCount, "message": fmt.Sprintf("%d IP(s) deleted from whitelist", deletedCount)}, nil
}

func (h *IPWhitelistHandler) Add(data map[string]any) (any, error) {
	ips := toStringSlice(data["ips"])
	if len(ips) == 0 {
		return nil, &types.AdminError{Message: "invalid ips array", Code: types.AdminErrorCodeInvalidRequest}
	}
	timestamp := int(time.Now().Unix())

	for _, ip := range ips {
		if !isValidIPOrCIDR(ip) {
			return nil, &types.AdminError{Message: fmt.Sprintf("Invalid IP/CIDR format: %s", ip), Code: types.AdminErrorCodeInvalidIPFormat}
		}
	}

	var existingRecords []database.IPWhitelist
	if err := h.db.Where("ip IN ?", ips).Find(&existingRecords).Error; err != nil {
		return nil, err
	}

	existingIPs := make(map[string]struct{}, len(existingRecords))
	for _, record := range existingRecords {
		existingIPs[record.IP] = struct{}{}
	}

	newRecords := make([]database.IPWhitelist, 0, len(ips))
	for _, ip := range ips {
		if _, exists := existingIPs[ip]; exists {
			continue
		}
		newRecords = append(newRecords, database.IPWhitelist{IP: ip, Addtime: timestamp})
	}

	addedIds := make([]int, 0, len(newRecords))
	if len(newRecords) > 0 {
		if err := h.db.Create(&newRecords).Error; err != nil {
			return nil, err
		}
		for _, record := range newRecords {
			addedIds = append(addedIds, record.ID)
		}
	}
	if err := h.reloadWhitelist(); err != nil {
		return nil, err
	}
	return map[string]any{
		"addedIds":     addedIds,
		"addedCount":   len(addedIds),
		"skippedCount": len(ips) - len(addedIds),
		"message":      fmt.Sprintf("%d IP(s) added to whitelist, %d skipped (duplicates)", len(addedIds), len(ips)-len(addedIds)),
	}, nil
}

func (h *IPWhitelistHandler) SpecialOperation(data map[string]any) (any, error) {
	op := toString(data["operation"])
	switch op {
	case "allow-all":
		var count int64
		if err := h.db.Model(&database.IPWhitelist{}).Where("ip = ?", "0.0.0.0/0").Count(&count).Error; err != nil {
			return nil, err
		}
		if count == 0 {
			if err := h.db.Create(&database.IPWhitelist{IP: "0.0.0.0/0", Addtime: int(time.Now().Unix())}).Error; err != nil {
				return nil, err
			}
		}
	case "deny-all":
		var records []database.IPWhitelist
		if err := h.db.Find(&records).Error; err != nil {
			return nil, err
		}
		hasNonWildcard := false
		for _, record := range records {
			if record.IP != "0.0.0.0/0" {
				hasNonWildcard = true
				break
			}
		}
		if hasNonWildcard {
			if err := h.db.Where("ip = ?", "0.0.0.0/0").Delete(&database.IPWhitelist{}).Error; err != nil {
				return nil, err
			}
		}
	default:
		return nil, &types.AdminError{Message: "Invalid operation. Must be \"allow-all\" or \"deny-all\"", Code: types.AdminErrorCodeInvalidRequest}
	}
	if err := h.reloadWhitelist(); err != nil {
		return nil, err
	}
	return nil, nil
}

func (h *IPWhitelistHandler) reloadWhitelist() error {
	if h.service != nil {
		return h.service.LoadFromDB()
	}
	return nil
}

func isValidIPOrCIDR(value string) bool {
	if value == "0.0.0.0/0" {
		return true
	}
	if strings.Contains(value, "/") {
		_, _, err := net.ParseCIDR(value)
		return err == nil
	}
	return net.ParseIP(value) != nil
}

func toStringSlice(v any) []string {
	if v == nil {
		return nil
	}
	switch raw := v.(type) {
	case []string:
		return raw
	case []any:
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
