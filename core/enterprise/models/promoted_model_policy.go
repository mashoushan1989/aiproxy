//go:build enterprise

package models

import (
	"time"

	"gorm.io/gorm"
)

const (
	PromotedModelAuditActionCreate              = "create"
	PromotedModelAuditActionUpdate              = "update"
	PromotedModelAuditActionEnable              = "enable"
	PromotedModelAuditActionDisable             = "disable"
	PromotedModelAuditActionPriceLock           = "price_lock"
	PromotedModelAuditActionPriceUnlock         = "price_unlock"
	PromotedModelAuditActionPriceChange         = "price_change"
	PromotedModelAuditActionForceLockedOverride = "force_locked_override"
	PromotedModelAuditActionDelete              = "delete"
	PromotedModelAuditActionRollback            = "rollback"
)

const (
	PromotedModelPricingModeManual   = "manual"
	PromotedModelPricingModeDiscount = "discount"
)

type PromotedModelPolicy struct {
	ID             int             `json:"id"              gorm:"primaryKey"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	DeletedAt      gorm.DeletedAt  `json:"-"               gorm:"index"`
	QuotaPolicyID  int             `json:"quota_policy_id" gorm:"index;not null"`
	QuotaPolicy    *QuotaPolicy    `json:"quota_policy"    gorm:"foreignKey:QuotaPolicyID"`
	Model          string          `json:"model"           gorm:"size:191;index;not null"`
	ChannelID      int             `json:"channel_id"      gorm:"default:0"`
	DisplayName    string          `json:"display_name"    gorm:"size:191"`
	RecommendBadge string          `json:"recommend_badge" gorm:"size:64"`
	SortOrder      int             `json:"sort_order"      gorm:"default:0"`
	Enabled        bool            `json:"enabled"         gorm:"default:true"`
	BasePrice      CommercialPrice `json:"base_price"      gorm:"embedded;embeddedPrefix:base_"`
	OverridePrice  CommercialPrice `json:"override_price"  gorm:"embedded;embeddedPrefix:override_"`
	PricingMode    string          `json:"pricing_mode"    gorm:"size:32;default:manual"`
	DiscountRate   float64         `json:"discount_rate"   gorm:"default:0"`
	PriceLocked    bool            `json:"price_locked"    gorm:"default:false"`
	EffectiveAt    *time.Time      `json:"effective_at"    gorm:"index"`
	ExpiresAt      *time.Time      `json:"expires_at"      gorm:"index"`
	Version        int             `json:"version"         gorm:"default:1"`
	CreatedBy      string          `json:"created_by"      gorm:"size:191"`
	UpdatedBy      string          `json:"updated_by"      gorm:"size:191"`
}

type CommercialPrice struct {
	PerRequestPrice float64 `json:"per_request_price,omitempty"`

	InputPrice     float64 `json:"input_price,omitempty"`
	InputPriceUnit int64   `json:"input_price_unit,omitempty"`

	ImageInputPrice     float64 `json:"image_input_price,omitempty"`
	ImageInputPriceUnit int64   `json:"image_input_price_unit,omitempty"`

	AudioInputPrice     float64 `json:"audio_input_price,omitempty"`
	AudioInputPriceUnit int64   `json:"audio_input_price_unit,omitempty"`

	OutputPrice     float64 `json:"output_price,omitempty"`
	OutputPriceUnit int64   `json:"output_price_unit,omitempty"`

	ImageOutputPrice     float64 `json:"image_output_price,omitempty"`
	ImageOutputPriceUnit int64   `json:"image_output_price_unit,omitempty"`

	ThinkingModeOutputPrice     float64 `json:"thinking_mode_output_price,omitempty"`
	ThinkingModeOutputPriceUnit int64   `json:"thinking_mode_output_price_unit,omitempty"`

	CachedPrice     float64 `json:"cached_price,omitempty"`
	CachedPriceUnit int64   `json:"cached_price_unit,omitempty"`

	CacheCreationPrice     float64 `json:"cache_creation_price,omitempty"`
	CacheCreationPriceUnit int64   `json:"cache_creation_price_unit,omitempty"`

	WebSearchPrice     float64 `json:"web_search_price,omitempty"`
	WebSearchPriceUnit int64   `json:"web_search_price_unit,omitempty"`

	ConditionalPrices string `json:"conditional_prices,omitempty" gorm:"type:text"`
}

func (PromotedModelPolicy) TableName() string {
	return "enterprise_promoted_model_policies"
}

func (p PromotedModelPolicy) ActiveAt(now time.Time) bool {
	if !p.Enabled {
		return false
	}
	if p.EffectiveAt != nil && now.Before(*p.EffectiveAt) {
		return false
	}
	if p.ExpiresAt != nil && !now.Before(*p.ExpiresAt) {
		return false
	}
	return true
}

type PromotedModelPolicyAudit struct {
	ID                    int            `json:"id"                       gorm:"primaryKey"`
	CreatedAt             time.Time      `json:"created_at"`
	DeletedAt             gorm.DeletedAt `json:"-"                        gorm:"index"`
	PromotedModelPolicyID int            `json:"promoted_model_policy_id" gorm:"index;not null"`
	QuotaPolicyID         int            `json:"quota_policy_id"          gorm:"index;not null"`
	Action                string         `json:"action"                   gorm:"size:64;not null"`
	Before                string         `json:"before"                   gorm:"type:text"`
	After                 string         `json:"after"                    gorm:"type:text"`
	Summary               string         `json:"summary"                  gorm:"size:512"`
	OperatorID            string         `json:"operator_id"              gorm:"size:191"`
	OperatorName          string         `json:"operator_name"            gorm:"size:191"`
}

func (PromotedModelPolicyAudit) TableName() string {
	return "enterprise_promoted_model_policy_audits"
}
