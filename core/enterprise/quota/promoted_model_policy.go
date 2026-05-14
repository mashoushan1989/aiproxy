//go:build enterprise

package quota

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	entmodels "github.com/labring/aiproxy/core/enterprise/models"
	"github.com/labring/aiproxy/core/model"
	"gorm.io/gorm"
)

var ErrPromotedModelPriceLocked = errors.New("promoted model commercial price is locked")

type AuditOperator struct {
	ID   string
	Name string
}

type CreatePromotedModelEntryRequest struct {
	QuotaPolicyID  int         `json:"quota_policy_id"`
	Model          string      `json:"model"`
	ChannelID      int         `json:"channel_id"`
	DisplayName    string      `json:"display_name"`
	RecommendBadge string      `json:"recommend_badge"`
	SortOrder      int         `json:"sort_order"`
	Enabled        bool        `json:"enabled"`
	OverridePrice  model.Price `json:"override_price"`
	DiscountRate   float64     `json:"discount_rate"`
	PriceLocked    bool        `json:"price_locked"`
	EffectiveAt    *time.Time  `json:"effective_at"`
	ExpiresAt      *time.Time  `json:"expires_at"`
}

type UpdatePromotedModelEntryRequest struct {
	DisplayName    string      `json:"display_name"`
	RecommendBadge string      `json:"recommend_badge"`
	SortOrder      int         `json:"sort_order"`
	Enabled        bool        `json:"enabled"`
	OverridePrice  model.Price `json:"override_price"`
	DiscountRate   float64     `json:"discount_rate"`
	PriceLocked    bool        `json:"price_locked"`
	EffectiveAt    *time.Time  `json:"effective_at"`
	ExpiresAt      *time.Time  `json:"expires_at"`
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func commercialPriceFromModelPrice(price model.Price) (entmodels.CommercialPrice, error) {
	conditionalPrices := ""
	if len(price.ConditionalPrices) > 0 {
		data, err := json.Marshal(price.ConditionalPrices)
		if err != nil {
			return entmodels.CommercialPrice{}, err
		}
		conditionalPrices = string(data)
	}

	return entmodels.CommercialPrice{
		PerRequestPrice:             float64(price.PerRequestPrice),
		InputPrice:                  float64(price.InputPrice),
		InputPriceUnit:              int64(price.InputPriceUnit),
		ImageInputPrice:             float64(price.ImageInputPrice),
		ImageInputPriceUnit:         int64(price.ImageInputPriceUnit),
		AudioInputPrice:             float64(price.AudioInputPrice),
		AudioInputPriceUnit:         int64(price.AudioInputPriceUnit),
		OutputPrice:                 float64(price.OutputPrice),
		OutputPriceUnit:             int64(price.OutputPriceUnit),
		ImageOutputPrice:            float64(price.ImageOutputPrice),
		ImageOutputPriceUnit:        int64(price.ImageOutputPriceUnit),
		ThinkingModeOutputPrice:     float64(price.ThinkingModeOutputPrice),
		ThinkingModeOutputPriceUnit: int64(price.ThinkingModeOutputPriceUnit),
		CachedPrice:                 float64(price.CachedPrice),
		CachedPriceUnit:             int64(price.CachedPriceUnit),
		CacheCreationPrice:          float64(price.CacheCreationPrice),
		CacheCreationPriceUnit:      int64(price.CacheCreationPriceUnit),
		WebSearchPrice:              float64(price.WebSearchPrice),
		WebSearchPriceUnit:          int64(price.WebSearchPriceUnit),
		ConditionalPrices:           conditionalPrices,
	}, nil
}

func modelPriceFromCommercialPrice(price entmodels.CommercialPrice) (model.Price, error) {
	out := model.Price{
		PerRequestPrice:             model.ZeroNullFloat64(price.PerRequestPrice),
		InputPrice:                  model.ZeroNullFloat64(price.InputPrice),
		InputPriceUnit:              model.ZeroNullInt64(price.InputPriceUnit),
		ImageInputPrice:             model.ZeroNullFloat64(price.ImageInputPrice),
		ImageInputPriceUnit:         model.ZeroNullInt64(price.ImageInputPriceUnit),
		AudioInputPrice:             model.ZeroNullFloat64(price.AudioInputPrice),
		AudioInputPriceUnit:         model.ZeroNullInt64(price.AudioInputPriceUnit),
		OutputPrice:                 model.ZeroNullFloat64(price.OutputPrice),
		OutputPriceUnit:             model.ZeroNullInt64(price.OutputPriceUnit),
		ImageOutputPrice:            model.ZeroNullFloat64(price.ImageOutputPrice),
		ImageOutputPriceUnit:        model.ZeroNullInt64(price.ImageOutputPriceUnit),
		ThinkingModeOutputPrice:     model.ZeroNullFloat64(price.ThinkingModeOutputPrice),
		ThinkingModeOutputPriceUnit: model.ZeroNullInt64(price.ThinkingModeOutputPriceUnit),
		CachedPrice:                 model.ZeroNullFloat64(price.CachedPrice),
		CachedPriceUnit:             model.ZeroNullInt64(price.CachedPriceUnit),
		CacheCreationPrice:          model.ZeroNullFloat64(price.CacheCreationPrice),
		CacheCreationPriceUnit:      model.ZeroNullInt64(price.CacheCreationPriceUnit),
		WebSearchPrice:              model.ZeroNullFloat64(price.WebSearchPrice),
		WebSearchPriceUnit:          model.ZeroNullInt64(price.WebSearchPriceUnit),
	}
	if price.ConditionalPrices != "" {
		if err := json.Unmarshal([]byte(price.ConditionalPrices), &out.ConditionalPrices); err != nil {
			return model.Price{}, err
		}
	}
	return out, nil
}

func validatePromotedModelEntry(
	policyID int,
	modelName string,
	channelID int,
	overridePrice model.Price,
	effectiveAt,
	expiresAt *time.Time,
) (model.Price, error) {
	if policyID <= 0 {
		return model.Price{}, errors.New("quota_policy_id is required")
	}
	if modelName == "" {
		return model.Price{}, errors.New("model is required")
	}
	if effectiveAt != nil && expiresAt != nil && !expiresAt.After(*effectiveAt) {
		return model.Price{}, errors.New("expires_at must be after effective_at")
	}
	if err := overridePrice.ValidateConditionalPrices(); err != nil {
		return model.Price{}, err
	}

	var policy entmodels.QuotaPolicy
	if err := model.DB.First(&policy, policyID).Error; err != nil {
		return model.Price{}, fmt.Errorf("quota policy not found: %w", err)
	}

	var mc model.ModelConfig
	if err := model.DB.Where("model = ?", modelName).First(&mc).Error; err != nil {
		return model.Price{}, fmt.Errorf("model config not found: %w", err)
	}

	if channelID > 0 {
		var channel model.Channel
		if err := model.DB.First(&channel, channelID).Error; err != nil {
			return model.Price{}, fmt.Errorf("channel not found: %w", err)
		}

		found := false
		for _, m := range channel.Models {
			if m == modelName {
				found = true
				break
			}
		}
		if !found {
			return model.Price{}, errors.New("channel does not include model")
		}
	}

	return mc.Price, nil
}

func CreatePromotedModelEntry(
	req CreatePromotedModelEntryRequest,
	op AuditOperator,
) (*entmodels.PromotedModelPolicy, error) {
	basePrice, err := validatePromotedModelEntry(
		req.QuotaPolicyID,
		req.Model,
		req.ChannelID,
		req.OverridePrice,
		req.EffectiveAt,
		req.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}

	baseCommercialPrice, err := commercialPriceFromModelPrice(basePrice)
	if err != nil {
		return nil, err
	}

	overrideCommercialPrice, err := commercialPriceFromModelPrice(req.OverridePrice)
	if err != nil {
		return nil, err
	}

	entry := entmodels.PromotedModelPolicy{
		QuotaPolicyID:  req.QuotaPolicyID,
		Model:          req.Model,
		ChannelID:      req.ChannelID,
		DisplayName:    req.DisplayName,
		RecommendBadge: req.RecommendBadge,
		SortOrder:      req.SortOrder,
		Enabled:        req.Enabled,
		BasePrice:      baseCommercialPrice,
		OverridePrice:  overrideCommercialPrice,
		DiscountRate:   req.DiscountRate,
		PriceLocked:    req.PriceLocked,
		EffectiveAt:    req.EffectiveAt,
		ExpiresAt:      req.ExpiresAt,
		Version:        1,
		CreatedBy:      op.ID,
		UpdatedBy:      op.ID,
	}

	if err := model.DB.Create(&entry).Error; err != nil {
		return nil, err
	}

	if err := writePromotedModelAudit(
		&entry,
		entmodels.PromotedModelAuditActionCreate,
		nil,
		&entry,
		op,
		"created promoted model",
	); err != nil {
		return nil, err
	}

	return &entry, nil
}

func UpdatePromotedModelEntry(
	id int,
	req UpdatePromotedModelEntryRequest,
	op AuditOperator,
	overrideLocked bool,
) (*entmodels.PromotedModelPolicy, error) {
	var existing entmodels.PromotedModelPolicy
	if err := model.DB.First(&existing, id).Error; err != nil {
		return nil, err
	}

	existingOverridePrice, err := modelPriceFromCommercialPrice(existing.OverridePrice)
	if err != nil {
		return nil, err
	}

	if existing.PriceLocked && !overrideLocked && !samePrice(existingOverridePrice, req.OverridePrice) {
		return nil, ErrPromotedModelPriceLocked
	}

	if _, err := validatePromotedModelEntry(
		existing.QuotaPolicyID,
		existing.Model,
		existing.ChannelID,
		req.OverridePrice,
		req.EffectiveAt,
		req.ExpiresAt,
	); err != nil {
		return nil, err
	}

	overrideCommercialPrice, err := commercialPriceFromModelPrice(req.OverridePrice)
	if err != nil {
		return nil, err
	}

	before := existing
	existing.DisplayName = req.DisplayName
	existing.RecommendBadge = req.RecommendBadge
	existing.SortOrder = req.SortOrder
	existing.Enabled = req.Enabled
	existing.OverridePrice = overrideCommercialPrice
	existing.DiscountRate = req.DiscountRate
	existing.PriceLocked = req.PriceLocked
	existing.EffectiveAt = req.EffectiveAt
	existing.ExpiresAt = req.ExpiresAt
	existing.Version++
	existing.UpdatedBy = op.ID

	if err := model.DB.Save(&existing).Error; err != nil {
		return nil, err
	}

	action := entmodels.PromotedModelAuditActionUpdate
	if overrideLocked && before.PriceLocked && !sameCommercialPrice(before.OverridePrice, existing.OverridePrice) {
		action = entmodels.PromotedModelAuditActionForceLockedOverride
	} else if before.PriceLocked != existing.PriceLocked {
		if existing.PriceLocked {
			action = entmodels.PromotedModelAuditActionPriceLock
		} else {
			action = entmodels.PromotedModelAuditActionPriceUnlock
		}
	} else if !sameCommercialPrice(before.OverridePrice, existing.OverridePrice) {
		action = entmodels.PromotedModelAuditActionPriceChange
	}

	if err := writePromotedModelAudit(&existing, action, &before, &existing, op, "updated promoted model"); err != nil {
		return nil, err
	}

	return &existing, nil
}

func DeletePromotedModelEntry(id int, op AuditOperator) error {
	var existing entmodels.PromotedModelPolicy
	if err := model.DB.First(&existing, id).Error; err != nil {
		return err
	}

	if err := model.DB.Delete(&existing).Error; err != nil {
		return err
	}

	return writePromotedModelAudit(
		&existing,
		entmodels.PromotedModelAuditActionDelete,
		&existing,
		nil,
		op,
		"deleted promoted model",
	)
}

func RollbackPromotedModelEntry(
	id int,
	version int,
	op AuditOperator,
) (*entmodels.PromotedModelPolicy, error) {
	var current entmodels.PromotedModelPolicy
	if err := model.DB.First(&current, id).Error; err != nil {
		return nil, err
	}

	var audits []entmodels.PromotedModelPolicyAudit
	if err := model.DB.
		Where("promoted_model_policy_id = ?", id).
		Order("id ASC").
		Find(&audits).Error; err != nil {
		return nil, err
	}

	var target *entmodels.PromotedModelPolicy
	for i := range audits {
		if audits[i].After == "" {
			continue
		}

		var snapshot entmodels.PromotedModelPolicy
		if err := json.Unmarshal([]byte(audits[i].After), &snapshot); err != nil {
			return nil, err
		}

		if snapshot.Version == version {
			target = &snapshot
			break
		}
	}
	if target == nil {
		return nil, gorm.ErrRecordNotFound
	}

	before := current
	current.DisplayName = target.DisplayName
	current.RecommendBadge = target.RecommendBadge
	current.SortOrder = target.SortOrder
	current.Enabled = target.Enabled
	current.OverridePrice = target.OverridePrice
	current.DiscountRate = target.DiscountRate
	current.PriceLocked = target.PriceLocked
	current.EffectiveAt = target.EffectiveAt
	current.ExpiresAt = target.ExpiresAt
	current.Version++
	current.UpdatedBy = op.ID

	if err := model.DB.Save(&current).Error; err != nil {
		return nil, err
	}

	if err := writePromotedModelAudit(
		&current,
		entmodels.PromotedModelAuditActionRollback,
		&before,
		&current,
		op,
		"rolled back promoted model",
	); err != nil {
		return nil, err
	}

	return &current, nil
}

func writePromotedModelAudit(
	entry *entmodels.PromotedModelPolicy,
	action string,
	before,
	after *entmodels.PromotedModelPolicy,
	op AuditOperator,
	summary string,
) error {
	beforeJSON := ""
	afterJSON := ""
	if before != nil {
		data, err := json.Marshal(before)
		if err != nil {
			return err
		}
		beforeJSON = string(data)
	}
	if after != nil {
		data, err := json.Marshal(after)
		if err != nil {
			return err
		}
		afterJSON = string(data)
	}

	return model.DB.Create(&entmodels.PromotedModelPolicyAudit{
		PromotedModelPolicyID: entry.ID,
		QuotaPolicyID:         entry.QuotaPolicyID,
		Action:                action,
		Before:                beforeJSON,
		After:                 afterJSON,
		Summary:               summary,
		OperatorID:            op.ID,
		OperatorName:          op.Name,
	}).Error
}

func samePrice(a, b model.Price) bool {
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return string(aj) == string(bj)
}

func sameCommercialPrice(a, b entmodels.CommercialPrice) bool {
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return string(aj) == string(bj)
}
