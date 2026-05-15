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
	PricingMode    string      `json:"pricing_mode"`
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
	PricingMode    string      `json:"pricing_mode"`
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

func normalizedPromotedModelPricingMode(pricingMode string) (string, error) {
	if pricingMode == "" {
		return entmodels.PromotedModelPricingModeManual, nil
	}

	switch pricingMode {
	case entmodels.PromotedModelPricingModeManual, entmodels.PromotedModelPricingModeDiscount:
		return pricingMode, nil
	default:
		return "", errors.New("pricing_mode must be manual or discount")
	}
}

func discountedPromotedModelPrice(basePrice model.Price, discountRate float64) model.Price {
	price := basePrice
	apply := func(value model.ZeroNullFloat64) model.ZeroNullFloat64 {
		return model.ZeroNullFloat64(float64(value) * discountRate)
	}

	price.PerRequestPrice = apply(price.PerRequestPrice)
	price.InputPrice = apply(price.InputPrice)
	price.ImageInputPrice = apply(price.ImageInputPrice)
	price.AudioInputPrice = apply(price.AudioInputPrice)
	price.OutputPrice = apply(price.OutputPrice)
	price.ImageOutputPrice = apply(price.ImageOutputPrice)
	price.ThinkingModeOutputPrice = apply(price.ThinkingModeOutputPrice)
	price.CachedPrice = apply(price.CachedPrice)
	price.CacheCreationPrice = apply(price.CacheCreationPrice)
	price.WebSearchPrice = apply(price.WebSearchPrice)

	for i := range price.ConditionalPrices {
		price.ConditionalPrices[i].Price = discountedPromotedModelPrice(
			price.ConditionalPrices[i].Price,
			discountRate,
		)
	}

	return price
}

func promotedModelOverridePrice(
	basePrice model.Price,
	overridePrice model.Price,
	pricingMode string,
	discountRate float64,
) (model.Price, string, error) {
	normalizedMode, err := normalizedPromotedModelPricingMode(pricingMode)
	if err != nil {
		return model.Price{}, "", err
	}
	if normalizedMode == entmodels.PromotedModelPricingModeDiscount {
		if discountRate <= 0 || discountRate > 1 {
			return model.Price{}, "", errors.New("discount_rate must be greater than 0 and less than or equal to 1")
		}
		return discountedPromotedModelPrice(basePrice, discountRate), normalizedMode, nil
	}
	return overridePrice, normalizedMode, nil
}

func validatePromotedModelEntry(
	db *gorm.DB,
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
	if expiresAt != nil && !expiresAt.After(time.Now()) {
		return model.Price{}, errors.New("expires_at must be in the future")
	}
	if err := overridePrice.ValidateConditionalPrices(); err != nil {
		return model.Price{}, err
	}

	var policy entmodels.QuotaPolicy
	if err := db.First(&policy, policyID).Error; err != nil {
		return model.Price{}, fmt.Errorf("quota policy not found: %w", err)
	}

	var mc model.ModelConfig
	if err := db.Where("model = ?", modelName).First(&mc).Error; err != nil {
		return model.Price{}, fmt.Errorf("model config not found: %w", err)
	}

	if channelID > 0 {
		var channel model.Channel
		if err := db.First(&channel, channelID).Error; err != nil {
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
	var entry entmodels.PromotedModelPolicy

	err := model.DB.Transaction(func(tx *gorm.DB) error {
		basePrice, err := validatePromotedModelEntry(
			tx,
			req.QuotaPolicyID,
			req.Model,
			req.ChannelID,
			req.OverridePrice,
			req.EffectiveAt,
			req.ExpiresAt,
		)
		if err != nil {
			return err
		}

		baseCommercialPrice, err := commercialPriceFromModelPrice(basePrice)
		if err != nil {
			return err
		}

		overridePrice, pricingMode, err := promotedModelOverridePrice(
			basePrice,
			req.OverridePrice,
			req.PricingMode,
			req.DiscountRate,
		)
		if err != nil {
			return err
		}

		discountRate := req.DiscountRate
		if pricingMode == entmodels.PromotedModelPricingModeManual {
			discountRate = 0
		}

		overrideCommercialPrice, err := commercialPriceFromModelPrice(overridePrice)
		if err != nil {
			return err
		}

		entry = entmodels.PromotedModelPolicy{
			QuotaPolicyID:  req.QuotaPolicyID,
			Model:          req.Model,
			ChannelID:      req.ChannelID,
			DisplayName:    req.DisplayName,
			RecommendBadge: req.RecommendBadge,
			SortOrder:      req.SortOrder,
			Enabled:        req.Enabled,
			BasePrice:      baseCommercialPrice,
			OverridePrice:  overrideCommercialPrice,
			PricingMode:    pricingMode,
			DiscountRate:   discountRate,
			PriceLocked:    req.PriceLocked,
			EffectiveAt:    req.EffectiveAt,
			ExpiresAt:      req.ExpiresAt,
			Version:        1,
			CreatedBy:      op.ID,
			UpdatedBy:      op.ID,
		}

		if err := tx.Create(&entry).Error; err != nil {
			return err
		}

		return writePromotedModelAudit(
			tx,
			&entry,
			entmodels.PromotedModelAuditActionCreate,
			nil,
			&entry,
			op,
			"created promoted model",
		)
	})
	if err != nil {
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
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&existing, id).Error; err != nil {
			return err
		}

		existingOverridePrice, err := modelPriceFromCommercialPrice(existing.OverridePrice)
		if err != nil {
			return err
		}

		basePrice, err := modelPriceFromCommercialPrice(existing.BasePrice)
		if err != nil {
			return err
		}

		requestOverridePrice, pricingMode, err := promotedModelOverridePrice(
			basePrice,
			req.OverridePrice,
			req.PricingMode,
			req.DiscountRate,
		)
		if err != nil {
			return err
		}

		if existing.PriceLocked && !overrideLocked && !samePrice(existingOverridePrice, requestOverridePrice) {
			return ErrPromotedModelPriceLocked
		}

		if _, err := validatePromotedModelEntry(
			tx,
			existing.QuotaPolicyID,
			existing.Model,
			existing.ChannelID,
			req.OverridePrice,
			req.EffectiveAt,
			req.ExpiresAt,
		); err != nil {
			return err
		}

		discountRate := req.DiscountRate
		if pricingMode == entmodels.PromotedModelPricingModeManual {
			discountRate = 0
		}

		overrideCommercialPrice, err := commercialPriceFromModelPrice(requestOverridePrice)
		if err != nil {
			return err
		}

		before := existing
		existing.DisplayName = req.DisplayName
		existing.RecommendBadge = req.RecommendBadge
		existing.SortOrder = req.SortOrder
		existing.Enabled = req.Enabled
		existing.OverridePrice = overrideCommercialPrice
		existing.PricingMode = pricingMode
		existing.DiscountRate = discountRate
		existing.PriceLocked = req.PriceLocked
		existing.EffectiveAt = req.EffectiveAt
		existing.ExpiresAt = req.ExpiresAt
		existing.Version++
		existing.UpdatedBy = op.ID

		if err := tx.Save(&existing).Error; err != nil {
			return err
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

		return writePromotedModelAudit(tx, &existing, action, &before, &existing, op, "updated promoted model")
	})
	if err != nil {
		return nil, err
	}

	return &existing, nil
}

func DeletePromotedModelEntry(id int, op AuditOperator) error {
	return model.DB.Transaction(func(tx *gorm.DB) error {
		var existing entmodels.PromotedModelPolicy
		if err := tx.First(&existing, id).Error; err != nil {
			return err
		}

		if err := tx.Delete(&existing).Error; err != nil {
			return err
		}

		return writePromotedModelAudit(
			tx,
			&existing,
			entmodels.PromotedModelAuditActionDelete,
			&existing,
			nil,
			op,
			"deleted promoted model",
		)
	})
}

func RollbackPromotedModelEntry(
	id int,
	version int,
	op AuditOperator,
) (*entmodels.PromotedModelPolicy, error) {
	var current entmodels.PromotedModelPolicy
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&current, id).Error; err != nil {
			return err
		}

		var audits []entmodels.PromotedModelPolicyAudit
		if err := tx.
			Where("promoted_model_policy_id = ?", id).
			Order("id ASC").
			Find(&audits).Error; err != nil {
			return err
		}

		var target *entmodels.PromotedModelPolicy
		for i := range audits {
			if audits[i].After == "" {
				continue
			}

			var snapshot entmodels.PromotedModelPolicy
			if err := json.Unmarshal([]byte(audits[i].After), &snapshot); err != nil {
				return err
			}

			if snapshot.Version == version {
				target = &snapshot
				break
			}
		}
		if target == nil {
			return gorm.ErrRecordNotFound
		}

		before := current
		current.DisplayName = target.DisplayName
		current.RecommendBadge = target.RecommendBadge
		current.SortOrder = target.SortOrder
		current.Enabled = target.Enabled
		current.OverridePrice = target.OverridePrice
		current.PricingMode = target.PricingMode
		current.DiscountRate = target.DiscountRate
		current.PriceLocked = target.PriceLocked
		current.EffectiveAt = target.EffectiveAt
		current.ExpiresAt = target.ExpiresAt
		current.Version++
		current.UpdatedBy = op.ID

		if err := tx.Save(&current).Error; err != nil {
			return err
		}

		return writePromotedModelAudit(
			tx,
			&current,
			entmodels.PromotedModelAuditActionRollback,
			&before,
			&current,
			op,
			"rolled back promoted model",
		)
	})
	if err != nil {
		return nil, err
	}

	return &current, nil
}

func writePromotedModelAudit(
	db *gorm.DB,
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

	return db.Create(&entmodels.PromotedModelPolicyAudit{
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
