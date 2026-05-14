//go:build enterprise

package quota

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	entmodels "github.com/labring/aiproxy/core/enterprise/models"
	"github.com/labring/aiproxy/core/middleware"
	"github.com/labring/aiproxy/core/model"
	"gorm.io/gorm"
)

func auditOperatorFromContext(c *gin.Context) AuditOperator {
	username := c.GetString("username")
	return AuditOperator{ID: username, Name: username}
}

func policyIDParam(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, "invalid policy id")
		return 0, false
	}

	return id, true
}

func entryIDParam(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("entry_id"))
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, "invalid promoted model id")
		return 0, false
	}

	return id, true
}

func requirePromotedModelEntryInPolicy(c *gin.Context) (int, int, bool) {
	policyID, ok := policyIDParam(c)
	if !ok {
		return 0, 0, false
	}

	entryID, ok := entryIDParam(c)
	if !ok {
		return 0, 0, false
	}

	var count int64
	if err := model.DB.Model(&entmodels.PromotedModelPolicy{}).
		Where("id = ? AND quota_policy_id = ?", entryID, policyID).
		Count(&count).Error; err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return 0, 0, false
	}
	if count == 0 {
		middleware.ErrorResponse(c, http.StatusNotFound, "promoted model not found")
		return 0, 0, false
	}

	return policyID, entryID, true
}

type promotedModelPolicyResponse struct {
	ID             int                    `json:"id"`
	CreatedAt      any                    `json:"created_at"`
	UpdatedAt      any                    `json:"updated_at"`
	QuotaPolicyID  int                    `json:"quota_policy_id"`
	QuotaPolicy    *entmodels.QuotaPolicy `json:"quota_policy"`
	Model          string                 `json:"model"`
	ChannelID      int                    `json:"channel_id"`
	DisplayName    string                 `json:"display_name"`
	RecommendBadge string                 `json:"recommend_badge"`
	SortOrder      int                    `json:"sort_order"`
	Enabled        bool                   `json:"enabled"`
	BasePrice      model.Price            `json:"base_price"`
	OverridePrice  model.Price            `json:"override_price"`
	PricingMode    string                 `json:"pricing_mode"`
	DiscountRate   float64                `json:"discount_rate"`
	PriceLocked    bool                   `json:"price_locked"`
	EffectiveAt    any                    `json:"effective_at"`
	ExpiresAt      any                    `json:"expires_at"`
	Version        int                    `json:"version"`
	CreatedBy      string                 `json:"created_by"`
	UpdatedBy      string                 `json:"updated_by"`
}

type promotedModelCandidateResponse struct {
	Model     string                          `json:"model"`
	Type      int                             `json:"type"`
	TypeName  string                          `json:"type_name"`
	BasePrice model.Price                     `json:"base_price"`
	Channels  []promotedModelCandidateChannel `json:"channels"`
}

type promotedModelCandidateChannel struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type int    `json:"type"`
}

func promotedModelResponse(entry entmodels.PromotedModelPolicy) (promotedModelPolicyResponse, error) {
	basePrice, err := modelPriceFromCommercialPrice(entry.BasePrice)
	if err != nil {
		return promotedModelPolicyResponse{}, err
	}
	overridePrice, err := modelPriceFromCommercialPrice(entry.OverridePrice)
	if err != nil {
		return promotedModelPolicyResponse{}, err
	}

	return promotedModelPolicyResponse{
		ID:             entry.ID,
		CreatedAt:      entry.CreatedAt,
		UpdatedAt:      entry.UpdatedAt,
		QuotaPolicyID:  entry.QuotaPolicyID,
		QuotaPolicy:    entry.QuotaPolicy,
		Model:          entry.Model,
		ChannelID:      entry.ChannelID,
		DisplayName:    entry.DisplayName,
		RecommendBadge: entry.RecommendBadge,
		SortOrder:      entry.SortOrder,
		Enabled:        entry.Enabled,
		BasePrice:      basePrice,
		OverridePrice:  overridePrice,
		PricingMode:    entry.PricingMode,
		DiscountRate:   entry.DiscountRate,
		PriceLocked:    entry.PriceLocked,
		EffectiveAt:    entry.EffectiveAt,
		ExpiresAt:      entry.ExpiresAt,
		Version:        entry.Version,
		CreatedBy:      entry.CreatedBy,
		UpdatedBy:      entry.UpdatedBy,
	}, nil
}

func promotedModelResponses(entries []entmodels.PromotedModelPolicy) ([]promotedModelPolicyResponse, error) {
	responses := make([]promotedModelPolicyResponse, 0, len(entries))
	for _, entry := range entries {
		response, err := promotedModelResponse(entry)
		if err != nil {
			return nil, err
		}
		responses = append(responses, response)
	}
	return responses, nil
}

func ListPromotedModels(c *gin.Context) {
	policyID, ok := policyIDParam(c)
	if !ok {
		return
	}

	var entries []entmodels.PromotedModelPolicy
	if err := model.DB.
		Where("quota_policy_id = ?", policyID).
		Order("sort_order ASC, id DESC").
		Find(&entries).Error; err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	responses, err := promotedModelResponses(entries)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.SuccessResponse(c, gin.H{"entries": responses})
}

func ListPromotedModelCandidates(c *gin.Context) {
	policyID, ok := policyIDParam(c)
	if !ok {
		return
	}
	query := strings.TrimSpace(strings.ToLower(c.DefaultQuery("keyword", c.Query("q"))))
	channelID, _ := strconv.Atoi(c.Query("channel_id"))

	var existing []string
	if err := model.DB.Model(&entmodels.PromotedModelPolicy{}).
		Where("quota_policy_id = ?", policyID).
		Pluck("model", &existing).Error; err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	existingSet := make(map[string]struct{}, len(existing))
	for _, modelName := range existing {
		existingSet[modelName] = struct{}{}
	}

	var channels []model.Channel
	if err := model.DB.
		Where("status = ?", model.ChannelStatusEnabled).
		Order("id ASC").
		Find(&channels).Error; err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	candidateMap := make(map[string]promotedModelCandidateResponse)
	candidateOrder := make([]string, 0)
	enabledConfigs := model.LoadModelCaches().EnabledModelConfigsMap
	for _, channel := range channels {
		if channelID > 0 && channel.ID != channelID {
			continue
		}
		for _, modelName := range channel.Models {
			if _, exists := existingSet[modelName]; exists {
				continue
			}
			if query != "" &&
				!strings.Contains(strings.ToLower(modelName), query) &&
				!strings.Contains(strings.ToLower(channel.Name), query) {
				continue
			}

			config, ok := enabledConfigs[modelName]
			if !ok {
				continue
			}
			candidate, exists := candidateMap[modelName]
			if !exists {
				candidate = promotedModelCandidateResponse{
					Model:     modelName,
					Type:      int(config.Type),
					TypeName:  config.Type.String(),
					BasePrice: config.Price,
					Channels:  []promotedModelCandidateChannel{},
				}
				candidateOrder = append(candidateOrder, modelName)
			}
			candidate.Channels = append(candidate.Channels, promotedModelCandidateChannel{
				ID:   channel.ID,
				Name: channel.Name,
				Type: int(channel.Type),
			})
			candidateMap[modelName] = candidate
		}
	}

	candidates := make([]promotedModelCandidateResponse, 0, len(candidateOrder))
	for _, modelName := range candidateOrder {
		candidates = append(candidates, candidateMap[modelName])
	}

	middleware.SuccessResponse(c, gin.H{"candidates": candidates})
}

func CreatePromotedModel(c *gin.Context) {
	policyID, ok := policyIDParam(c)
	if !ok {
		return
	}

	var req CreatePromotedModelEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	req.QuotaPolicyID = policyID

	entry, err := CreatePromotedModelEntry(req, auditOperatorFromContext(c))
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	response, err := promotedModelResponse(*entry)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.SuccessResponse(c, response)
}

func UpdatePromotedModel(c *gin.Context) {
	_, entryID, ok := requirePromotedModelEntryInPolicy(c)
	if !ok {
		return
	}

	var req struct {
		UpdatePromotedModelEntryRequest
		OverrideLocked bool `json:"override_locked"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	entry, err := UpdatePromotedModelEntry(
		entryID,
		req.UpdatePromotedModelEntryRequest,
		auditOperatorFromContext(c),
		req.OverrideLocked,
	)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, ErrPromotedModelPriceLocked) {
			status = http.StatusConflict
		}
		middleware.ErrorResponse(c, status, err.Error())
		return
	}

	response, err := promotedModelResponse(*entry)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.SuccessResponse(c, response)
}

func DeletePromotedModel(c *gin.Context) {
	_, entryID, ok := requirePromotedModelEntryInPolicy(c)
	if !ok {
		return
	}

	if err := DeletePromotedModelEntry(entryID, auditOperatorFromContext(c)); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	middleware.SuccessResponse(c, gin.H{"deleted": true})
}

func RollbackPromotedModel(c *gin.Context) {
	_, entryID, ok := requirePromotedModelEntryInPolicy(c)
	if !ok {
		return
	}

	var req struct {
		Version int `json:"version" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	entry, err := RollbackPromotedModelEntry(entryID, req.Version, auditOperatorFromContext(c))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			middleware.ErrorResponse(c, http.StatusNotFound, "rollback version not found")
			return
		}
		middleware.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	response, err := promotedModelResponse(*entry)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.SuccessResponse(c, response)
}

func ListPromotedModelAudits(c *gin.Context) {
	policyID, ok := policyIDParam(c)
	if !ok {
		return
	}

	var audits []entmodels.PromotedModelPolicyAudit
	if err := model.DB.
		Where("quota_policy_id = ?", policyID).
		Order("id DESC").
		Find(&audits).Error; err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.SuccessResponse(c, gin.H{"audits": audits})
}
