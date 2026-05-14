//go:build enterprise

package quota

import (
	"context"
	"errors"

	"github.com/labring/aiproxy/core/enterprise/models"
	"github.com/labring/aiproxy/core/model"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// CheckQuotaTier evaluates the group's period usage against the quota policy
// with multi-level priority: user policy > department policy > group policy.
// Enterprise builds use group_summaries (Group-level) for usage calculation,
// matching the display layer in access_info.go.
// Returns: effectiveModel, rpmMultiplier, tpmMultiplier, blocked
func CheckQuotaTier(
	group model.GroupCache,
	token model.TokenCache,
	requestModel string,
) (string, float64, float64, bool) {
	ctx := context.Background()

	// Check if this group is associated with a Feishu user
	// If so, use multi-level policy (user > department > group)
	var feishuUser models.FeishuUser
	err := model.DB.Where("group_id = ?", group.ID).First(&feishuUser).Error
	if err == nil {
		// This is a Feishu user group, check multi-level policies
		policy, err := GetPolicyForUser(ctx, feishuUser.OpenID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Errorf("failed to get multi-level quota policy for user %s: %v", feishuUser.OpenID, err)
		}
		if policy != nil {
			usageRatio := computeGroupUsageRatio(group.ID, policy)
			price := effectivePriceForQuotaBlocking(group, requestModel)
			effModel, rpmMul, tpmMul, blocked := applyPolicyTiersWithPrice(policy, usageRatio, requestModel, price)

			policyPeriodType := PolicyPeriodTypeToTokenPeriodType(policy.PeriodType)
			tier := ComputeTier(policy, usageRatio, blocked)
			if tier >= 2 {
				go MaybeNotifyUser(
					feishuUser.OpenID,
					feishuUser.Name,
					policyPeriodType,
					tier,
					usageRatio,
					policy.PeriodQuota,
					tierThreshold(policy, tier),
				)
			}
			// Admin webhook alert: independent threshold, checked on every request
			go MaybeNotifyAdmin(
				feishuUser.OpenID,
				feishuUser.Name,
				policyPeriodType,
				usageRatio,
				policy.PeriodQuota,
			)

			return effModel, rpmMul, tpmMul, blocked
		}
		// No policy found for this Feishu user, fall through to default
		return requestModel, 1.0, 1.0, false
	}

	// Not a Feishu user or user not found, use traditional group-level policy
	policy, err := GetGroupQuotaPolicy(ctx, group.ID)
	if err != nil {
		log.Errorf("failed to get quota policy for group %s: %v", group.ID, err)
		return requestModel, 1.0, 1.0, false
	}

	if policy == nil {
		return requestModel, 1.0, 1.0, false
	}

	usageRatio := computeGroupUsageRatio(group.ID, policy)
	price := effectivePriceForQuotaBlocking(group, requestModel)
	return applyPolicyTiersWithPrice(policy, usageRatio, requestModel, price)
}

// computeGroupUsageRatio returns the fraction of the period quota consumed at Group level.
func computeGroupUsageRatio(groupID string, policy *models.QuotaPolicy) float64 {
	if policy.PeriodQuota <= 0 {
		return 0
	}

	periodUsed := getCachedGroupPeriodUsage(groupID, policy.PeriodType)
	return periodUsed / policy.PeriodQuota
}

// ComputeTier returns the effective tier (1–4) for the given usage state.
// tier 1 = normal, 2 = tier2 throttle, 3 = tier3 throttle, 4 = exhausted/blocked.
func ComputeTier(policy *models.QuotaPolicy, usageRatio float64, blocked bool) int {
	switch {
	case blocked || usageRatio >= 1.0:
		if blocked && policy.BlockAtTier3 {
			return 4 // exhausted
		}

		return 3
	case usageRatio >= policy.Tier2Ratio:
		return 3
	case usageRatio >= policy.Tier1Ratio:
		return 2
	default:
		return 1
	}
}

// tierThreshold returns the ratio threshold that triggered the given tier notification.
func tierThreshold(policy *models.QuotaPolicy, tier int) float64 {
	switch tier {
	case 2:
		return policy.Tier1Ratio
	case 3:
		return policy.Tier2Ratio
	default: // 4 (exhaust)
		return 1.0
	}
}

// applyPolicyTiers applies the tiered policy logic based on a pre-computed usage ratio.
func applyPolicyTiers(policy *models.QuotaPolicy, usageRatio float64, requestModel string) (string, float64, float64, bool) {
	return applyPolicyTiersWithPrice(policy, usageRatio, requestModel, fallbackModelPrice(requestModel))
}

func fallbackModelPrice(requestModel string) model.Price {
	if mc := model.LoadModelCaches(); mc != nil && mc.ModelConfig != nil {
		if cfg, ok := mc.ModelConfig.GetModelConfig(requestModel); ok {
			return cfg.Price
		}
	}
	return model.Price{}
}

func effectivePriceForQuotaBlocking(group model.GroupCache, requestModel string) model.Price {
	fallback := fallbackModelPrice(requestModel)
	if groupModelConfig, ok := group.ModelConfigs[requestModel]; ok && groupModelConfig.OverridePrice {
		return groupModelConfig.Price
	}
	price, err := ResolvePromotedModelPrice(group, requestModel, fallback)
	if err != nil {
		log.Errorf("failed to resolve promoted model price for quota blocking, group=%s model=%s: %v", group.ID, requestModel, err)
		return fallback
	}
	return price
}

func applyPolicyTiersWithPrice(policy *models.QuotaPolicy, usageRatio float64, requestModel string, price model.Price) (string, float64, float64, bool) {
	// Guard against zero or no-limit policy
	if policy.PeriodQuota <= 0 {
		return requestModel, 1.0, 1.0, false
	}

	// Resolve model pricing once for price-based blocking.
	// Normalize to ¥/M tokens so thresholds match the "my access" model price display.
	inputPrice := float64(price.InputPrice) / float64(price.GetInputPriceUnit()) * 1e6
	outputPrice := float64(price.OutputPrice) / float64(price.GetOutputPriceUnit()) * 1e6

	switch {
	case usageRatio >= policy.Tier2Ratio:
		// Tier 3: usage >= Tier2Ratio
		if policy.BlockAtTier3 ||
			policy.IsModelBlockedAtTier(3, requestModel) ||
			policy.IsModelBlockedByPrice(3, inputPrice, outputPrice) {
			return requestModel, 0, 0, true
		}

		return requestModel, policy.Tier3RPMMultiplier, policy.Tier3TPMMultiplier, false
	case usageRatio >= policy.Tier1Ratio:
		// Tier 2: Tier1Ratio <= usage < Tier2Ratio
		if policy.IsModelBlockedAtTier(2, requestModel) ||
			policy.IsModelBlockedByPrice(2, inputPrice, outputPrice) {
			return requestModel, 0, 0, true
		}

		return requestModel, policy.Tier2RPMMultiplier, policy.Tier2TPMMultiplier, false
	default:
		// Tier 1: usage < Tier1Ratio
		return requestModel, policy.Tier1RPMMultiplier, policy.Tier1TPMMultiplier, false
	}
}
