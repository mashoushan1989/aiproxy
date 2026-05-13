//go:build enterprise

package orgsync

import (
	"context"

	enterprisemodels "github.com/labring/aiproxy/core/enterprise/models"
	"github.com/labring/aiproxy/core/model"
	"gorm.io/gorm"
)

func GetOrgUnitDescendantIDs(ctx context.Context, db *gorm.DB, workspaceID, orgUnitID string) ([]string, error) {
	if workspaceID == "" {
		workspaceID = enterprisemodels.WorkspaceDefaultID
	}

	var root enterprisemodels.OrgUnit
	if err := db.WithContext(ctx).
		Where("workspace_id = ? AND id = ?", workspaceID, orgUnitID).
		First(&root).Error; err != nil {
		return nil, err
	}

	var ids []string
	err := db.WithContext(ctx).
		Model(&enterprisemodels.OrgUnit{}).
		Where("workspace_id = ? AND (id = ? OR path LIKE ?)", workspaceID, orgUnitID, root.Path+"/%").
		Pluck("id", &ids).Error
	return ids, err
}

func GetGroupIDsForOrgUnits(ctx context.Context, db *gorm.DB, workspaceID string, orgUnitIDs []string) ([]string, error) {
	if workspaceID == "" {
		workspaceID = enterprisemodels.WorkspaceDefaultID
	}

	if len(orgUnitIDs) == 0 {
		var groupIDs []string
		err := db.WithContext(ctx).
			Model(&model.Group{}).
			Where("workspace_id = ?", workspaceID).
			Pluck("id", &groupIDs).Error
		return groupIDs, err
	}

	expandedOrgUnitIDs := make([]string, 0, len(orgUnitIDs))
	seen := map[string]struct{}{}
	for _, orgUnitID := range orgUnitIDs {
		if orgUnitID == "" {
			continue
		}

		descendantIDs, err := GetOrgUnitDescendantIDs(ctx, db, workspaceID, orgUnitID)
		if err != nil {
			return nil, err
		}
		for _, descendantID := range descendantIDs {
			if _, ok := seen[descendantID]; ok {
				continue
			}
			seen[descendantID] = struct{}{}
			expandedOrgUnitIDs = append(expandedOrgUnitIDs, descendantID)
		}
	}

	if len(expandedOrgUnitIDs) == 0 {
		return []string{}, nil
	}

	var groupIDs []string
	err := db.WithContext(ctx).
		Model(&model.Group{}).
		Where("workspace_id = ? AND org_unit_id IN ?", workspaceID, expandedOrgUnitIDs).
		Pluck("id", &groupIDs).Error
	return groupIDs, err
}
