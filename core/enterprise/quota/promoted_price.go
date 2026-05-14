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

	entry, ok, err := ActivePromotedModelEntry(policy.ID, requestModel, time.Now())
	if err != nil {
		return model.Price{}, err
	}
	if !ok {
		return fallback, nil
	}

	return modelPriceFromCommercialPrice(entry.OverridePrice)
}

func ActivePromotedModelEntry(policyID int, modelName string, now time.Time) (entmodels.PromotedModelPolicy, bool, error) {
	entries, err := ActivePromotedModelEntries(policyID, modelName, now)
	if err != nil {
		return entmodels.PromotedModelPolicy{}, false, err
	}
	if len(entries) == 0 {
		return entmodels.PromotedModelPolicy{}, false, nil
	}
	return entries[0], true, nil
}

func ActivePromotedModelEntries(policyID int, modelName string, now time.Time) ([]entmodels.PromotedModelPolicy, error) {
	if policyID <= 0 {
		return nil, nil
	}

	query := model.DB.
		Where("quota_policy_id = ? AND enabled = ?", policyID, true).
		Order("sort_order ASC, id DESC")
	if modelName != "" {
		query = query.Where("model = ?", modelName)
	}

	var entries []entmodels.PromotedModelPolicy
	if err := query.Find(&entries).Error; err != nil {
		return nil, err
	}

	active := make([]entmodels.PromotedModelPolicy, 0, len(entries))
	for _, entry := range entries {
		if entry.ActiveAt(now) {
			active = append(active, entry)
		}
	}

	return active, nil
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
