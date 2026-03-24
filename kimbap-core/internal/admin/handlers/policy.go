package handlers

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/dunialabs/kimbap-core/internal/database"
	services "github.com/dunialabs/kimbap-core/internal/mcp/services"
	"github.com/dunialabs/kimbap-core/internal/types"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type PolicyHandler struct {
	db *gorm.DB
}

func NewPolicyHandler(db *gorm.DB) *PolicyHandler {
	if db == nil {
		db = database.DB
	}
	return &PolicyHandler{db: db}
}

func (h *PolicyHandler) CreatePolicySet(data map[string]any) (any, error) {
	dsl, ok := data["dsl"]
	if !ok || dsl == nil {
		return nil, &types.AdminError{Message: "Missing or invalid field: dsl", Code: types.AdminErrorCodeInvalidRequest}
	}
	dslBytes, err := json.Marshal(dsl)
	if err != nil {
		return nil, &types.AdminError{Message: "Missing or invalid field: dsl", Code: types.AdminErrorCodeInvalidRequest}
	}
	var parsedDsl services.PolicyDsl
	if err := json.Unmarshal(dslBytes, &parsedDsl); err != nil {
		return nil, &types.AdminError{Message: "Missing or invalid field: dsl", Code: types.AdminErrorCodeInvalidRequest}
	}
	if err := services.ValidatePolicyDSL(parsedDsl); err != nil {
		return nil, &types.AdminError{Message: err.Error(), Code: types.AdminErrorCodeInvalidRequest}
	}

	var serverID *string
	if raw, ok := data["serverId"].(string); ok && raw != "" {
		serverID = &raw
	}
	for attempt := 0; attempt < 3; attempt++ {
		var maxVersion int
		query := h.db.Model(&database.ToolPolicySet{})
		if serverID != nil {
			query = query.Where("server_id = ?", *serverID)
		} else {
			query = query.Where("server_id IS NULL")
		}
		if err := query.Select("COALESCE(MAX(version), 0)").Row().Scan(&maxVersion); err != nil {
			return nil, err
		}

		policy := database.ToolPolicySet{ServerID: serverID, Status: "active", Version: maxVersion + 1, Dsl: datatypes.JSON(dslBytes)}
		if err := h.db.Create(&policy).Error; err != nil {
			if isUniqueViolation(err) && attempt < 2 {
				continue
			}
			return nil, err
		}
		services.PolicyEngineInstance().ClearCache(serverID)
		return policy, nil
	}

	return nil, &types.AdminError{Message: "Failed to allocate unique policy version", Code: types.AdminErrorCodeDatabaseOpFailed}
}

func (h *PolicyHandler) GetPolicySets(data map[string]any) (any, error) {
	id := toString(data["id"])
	if id != "" {
		policies := make([]database.ToolPolicySet, 0)
		if err := h.db.Where("id = ?", id).Find(&policies).Error; err != nil {
			return nil, err
		}
		if len(policies) == 0 {
			return nil, &types.AdminError{Message: "Policy set not found", Code: types.AdminErrorCodeInvalidRequest}
		}
		return map[string]any{"policySets": policies}, nil
	}

	if serverID, ok := data["serverId"].(string); ok && strings.TrimSpace(serverID) != "" {
		policies := make([]database.ToolPolicySet, 0)
		if err := h.db.Where("server_id = ?", serverID).Order("version DESC").Find(&policies).Error; err != nil {
			return nil, err
		}
		return map[string]any{"policySets": policies}, nil
	}

	policies := make([]database.ToolPolicySet, 0)
	if err := h.db.Find(&policies).Error; err != nil {
		return nil, err
	}
	return map[string]any{"policySets": policies}, nil
}

func (h *PolicyHandler) UpdatePolicySet(data map[string]any) (any, error) {
	id := toString(data["id"])
	if id == "" {
		return nil, &types.AdminError{Message: "Missing required field: id", Code: types.AdminErrorCodeInvalidRequest}
	}

	var existing database.ToolPolicySet
	if err := h.db.Where("id = ?", id).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &types.AdminError{Message: "Policy set not found", Code: types.AdminErrorCodeInvalidRequest}
		}
		return nil, err
	}

	updates := map[string]any{}
	hasDsl := false
	if dsl, ok := data["dsl"]; ok {
		hasDsl = true
		dslBytes, err := json.Marshal(dsl)
		if err != nil {
			return nil, &types.AdminError{Message: "Missing or invalid field: dsl", Code: types.AdminErrorCodeInvalidRequest}
		}
		var parsedDsl services.PolicyDsl
		if err := json.Unmarshal(dslBytes, &parsedDsl); err != nil {
			return nil, &types.AdminError{Message: "Missing or invalid field: dsl", Code: types.AdminErrorCodeInvalidRequest}
		}
		if err := services.ValidatePolicyDSL(parsedDsl); err != nil {
			return nil, &types.AdminError{Message: err.Error(), Code: types.AdminErrorCodeInvalidRequest}
		}
		updates["dsl"] = datatypes.JSON(dslBytes)
	}
	if status, ok := data["status"].(string); ok && status != "" {
		updates["status"] = status
	}
	if len(updates) > 0 {
		if hasDsl {
			for attempt := 0; attempt < 3; attempt++ {
				var maxVersion int
				vQuery := h.db.Model(&database.ToolPolicySet{})
				if existing.ServerID != nil {
					vQuery = vQuery.Where("server_id = ?", *existing.ServerID)
				} else {
					vQuery = vQuery.Where("server_id IS NULL")
				}
				if err := vQuery.Select("COALESCE(MAX(version), 0)").Row().Scan(&maxVersion); err != nil {
					return nil, err
				}
				updates["version"] = maxVersion + 1

				if err := h.db.Model(&database.ToolPolicySet{}).Where("id = ?", id).Updates(updates).Error; err != nil {
					if isUniqueViolation(err) && attempt < 2 {
						continue
					}
					return nil, err
				}
				break
			}
		} else {
			if err := h.db.Model(&database.ToolPolicySet{}).Where("id = ?", id).Updates(updates).Error; err != nil {
				return nil, err
			}
		}
	}

	if err := h.db.Where("id = ?", id).First(&existing).Error; err != nil {
		return nil, err
	}
	services.PolicyEngineInstance().ClearCache(existing.ServerID)
	return existing, nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "duplicate key") || strings.Contains(lower, "unique constraint") || strings.Contains(lower, "unique violation")
}

func (h *PolicyHandler) DeletePolicySet(data map[string]any) (any, error) {
	id := toString(data["id"])
	if id == "" {
		return nil, &types.AdminError{Message: "Missing required field: id", Code: types.AdminErrorCodeInvalidRequest}
	}
	var existing database.ToolPolicySet
	if err := h.db.Where("id = ?", id).First(&existing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &types.AdminError{Message: "Policy set not found", Code: types.AdminErrorCodeInvalidRequest}
		}
		return nil, err
	}
	if err := h.db.Delete(&database.ToolPolicySet{}, "id = ?", id).Error; err != nil {
		return nil, err
	}
	services.PolicyEngineInstance().ClearCache(existing.ServerID)
	return existing, nil
}

func (h *PolicyHandler) GetEffectivePolicy(data map[string]any) (any, error) {
	var serverID *string
	if raw, ok := data["serverId"].(string); ok && raw != "" {
		serverID = &raw
	}

	globals := make([]database.ToolPolicySet, 0)
	if err := h.db.Where("status = ? AND server_id IS NULL", "active").Order("version DESC").Find(&globals).Error; err != nil {
		return nil, err
	}
	if serverID == nil {
		return map[string]any{"policySets": globals}, nil
	}

	serverPolicies := make([]database.ToolPolicySet, 0)
	if err := h.db.Where("status = ? AND server_id = ?", "active", *serverID).Order("version DESC").Find(&serverPolicies).Error; err != nil {
		return nil, err
	}
	return map[string]any{"policySets": append(serverPolicies, globals...)}, nil
}
