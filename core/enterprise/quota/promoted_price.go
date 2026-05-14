//go:build enterprise

package quota

import (
	"context"
	"errors"
	"time"

	entmodels "github.com/labring/aiproxy/core/enterprise/models"
	"github.com/labring/aiproxy/core/model"
	"gorm.io/gorm"
)

func ResolvePromotedModelPrice(group model.GroupCache, requestModel string, fallback model.Price) (model.Price, error) {
	policy, err := effectiveQuotaPolicyForGroup(context.Background(), group.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fallback, nil
		}
		return model.Price{}, err
	}
	if policy == nil {
		return fallback, nil
	}

	var entry entmodels.PromotedModelPolicy
	if err := model.DB.
		Where("quota_policy_id = ? AND model = ? AND enabled = ?", policy.ID, requestModel, true).
		Order("sort_order ASC, id DESC").
		First(&entry).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fallback, nil
		}
		return model.Price{}, err
	}

	if !entry.ActiveAt(time.Now()) {
		return fallback, nil
	}

	return modelPriceFromCommercialPrice(entry.OverridePrice)
}

func effectiveQuotaPolicyForGroup(ctx context.Context, groupID string) (*entmodels.QuotaPolicy, error) {
	var feishuUser entmodels.FeishuUser
	err := model.DB.Where("group_id = ?", groupID).First(&feishuUser).Error
	if err == nil {
		policy, policyErr := GetPolicyForUser(ctx, feishuUser.OpenID)
		if policyErr != nil && errors.Is(policyErr, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return policy, policyErr
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return GetGroupQuotaPolicy(ctx, groupID)
}
