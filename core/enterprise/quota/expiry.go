//go:build enterprise

package quota

import (
	"context"
	"time"

	"github.com/labring/aiproxy/core/enterprise/models"
	"github.com/labring/aiproxy/core/model"
	log "github.com/sirupsen/logrus"
)

const bindingExpirySweepInterval = time.Minute

// StartBindingExpiryScheduler periodically removes expired user/department policy
// overrides and re-syncs affected token quota projections to their fallback policy.
func StartBindingExpiryScheduler(ctx context.Context) {
	go func() {
		expirePolicyBindingsOnce(time.Now())

		ticker := time.NewTicker(bindingExpirySweepInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Info("quota policy binding expiry scheduler stopped")
				return
			case <-ticker.C:
				expirePolicyBindingsOnce(time.Now())
			}
		}
	}()

	log.Infof("quota policy binding expiry scheduler started (interval: %s)", bindingExpirySweepInterval)
}

func expirePolicyBindingsOnce(now time.Time) {
	if model.DB == nil {
		return
	}

	var expiredUsers []models.UserQuotaPolicy
	if err := model.DB.
		Where("expires_at IS NOT NULL AND expires_at <= ?", now).
		Find(&expiredUsers).Error; err != nil {
		log.Errorf("quota policy binding expiry: list user bindings: %v", err)
		return
	}

	var expiredDepartments []models.DepartmentQuotaPolicy
	if err := model.DB.
		Where("expires_at IS NOT NULL AND expires_at <= ?", now).
		Find(&expiredDepartments).Error; err != nil {
		log.Errorf("quota policy binding expiry: list department bindings: %v", err)
		return
	}

	if len(expiredUsers) == 0 && len(expiredDepartments) == 0 {
		return
	}

	userIDs := make([]int, 0, len(expiredUsers))
	affectedOpenIDs := make(map[string]struct{}, len(expiredUsers))
	for _, binding := range expiredUsers {
		userIDs = append(userIDs, binding.ID)
		affectedOpenIDs[binding.OpenID] = struct{}{}
	}

	departmentIDs := make([]int, 0, len(expiredDepartments))
	affectedDepartmentIDs := make([]string, 0, len(expiredDepartments))
	for _, binding := range expiredDepartments {
		departmentIDs = append(departmentIDs, binding.ID)
		affectedDepartmentIDs = append(affectedDepartmentIDs, binding.DepartmentID)
	}

	if len(userIDs) > 0 {
		if err := model.DB.Where("id IN ?", userIDs).Delete(&models.UserQuotaPolicy{}).Error; err != nil {
			log.Errorf("quota policy binding expiry: delete user bindings: %v", err)
			return
		}
	}

	if len(departmentIDs) > 0 {
		if err := model.DB.
			Where("id IN ?", departmentIDs).
			Delete(&models.DepartmentQuotaPolicy{}).Error; err != nil {
			log.Errorf("quota policy binding expiry: delete department bindings: %v", err)
			return
		}
	}

	for _, departmentID := range affectedDepartmentIDs {
		for _, openID := range getDepartmentUserIDsWithoutOverride(departmentID) {
			affectedOpenIDs[openID] = struct{}{}
		}
	}

	openIDs := make([]string, 0, len(affectedOpenIDs))
	for openID := range affectedOpenIDs {
		openIDs = append(openIDs, openID)
	}

	if len(openIDs) > 0 {
		syncEffectivePolicyToTokenBatch(openIDs)
	}

	log.Infof(
		"quota policy binding expiry: expired %d user bindings, %d department bindings, synced %d users",
		len(expiredUsers),
		len(expiredDepartments),
		len(openIDs),
	)
}
