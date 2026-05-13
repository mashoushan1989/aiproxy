//go:build enterprise

package analyticsx

import (
	"context"
	"sort"

	enterprisemodels "github.com/labring/aiproxy/core/enterprise/models"
	"github.com/labring/aiproxy/core/model"
	"gorm.io/gorm"
)

type OrgDirectory interface {
	DescendantOrgUnitIDs(ctx context.Context, workspaceID string, orgUnitIDs []string) ([]string, error)
	GroupIDsForOrgUnits(ctx context.Context, workspaceID string, orgUnitIDs []string) ([]string, error)
	GroupIDsForUsers(ctx context.Context, workspaceID string, userIDs []string) ([]string, error)
}

type GORMOrgDirectory struct {
	db *gorm.DB
}

func NewGORMOrgDirectory(db *gorm.DB) *GORMOrgDirectory {
	return &GORMOrgDirectory{db: db}
}

func (d *GORMOrgDirectory) DescendantOrgUnitIDs(
	ctx context.Context,
	workspaceID string,
	orgUnitIDs []string,
) ([]string, error) {
	workspaceID = defaultWorkspaceID(workspaceID)
	orgUnitIDs = compactStrings(orgUnitIDs)
	if len(orgUnitIDs) == 0 {
		return []string{}, nil
	}

	roots, err := d.resolveOrgUnits(ctx, workspaceID, orgUnitIDs)
	if err != nil {
		return nil, err
	}
	if len(roots) == 0 {
		return []string{}, nil
	}

	seen := map[string]struct{}{}
	for _, root := range roots {
		var ids []string
		tx := d.db.WithContext(ctx).
			Model(&enterprisemodels.OrgUnit{}).
			Where("workspace_id = ? AND status = ?", workspaceID, enterprisemodels.EntityStatusEnabled)
		if root.Path != "" {
			tx = tx.Where("id = ? OR path LIKE ?", root.ID, root.Path+"/%")
		} else {
			tx = tx.Where("id = ?", root.ID)
		}
		if err := tx.Pluck("id", &ids).Error; err != nil {
			return nil, err
		}
		for _, id := range ids {
			seen[id] = struct{}{}
		}
	}

	return sortedKeys(seen), nil
}

func (d *GORMOrgDirectory) GroupIDsForOrgUnits(
	ctx context.Context,
	workspaceID string,
	orgUnitIDs []string,
) ([]string, error) {
	workspaceID = defaultWorkspaceID(workspaceID)
	orgUnitIDs = compactStrings(orgUnitIDs)
	if len(orgUnitIDs) == 0 {
		return []string{}, nil
	}

	descendantIDs, err := d.DescendantOrgUnitIDs(ctx, workspaceID, orgUnitIDs)
	if err != nil {
		return nil, err
	}
	if len(descendantIDs) == 0 {
		return []string{}, nil
	}

	var groupIDs []string
	err = d.db.WithContext(ctx).
		Model(&model.Group{}).
		Where("workspace_id = ? AND status = ?", workspaceID, model.GroupStatusEnabled).
		Where("org_unit_id IN ?", descendantIDs).
		Pluck("id", &groupIDs).Error
	sort.Strings(groupIDs)
	return groupIDs, err
}

func (d *GORMOrgDirectory) GroupIDsForUsers(
	ctx context.Context,
	workspaceID string,
	userIDs []string,
) ([]string, error) {
	workspaceID = defaultWorkspaceID(workspaceID)
	userIDs = compactStrings(userIDs)
	if len(userIDs) == 0 {
		return []string{}, nil
	}

	var users []enterprisemodels.EnterpriseUser
	if err := d.db.WithContext(ctx).
		Where("workspace_id = ? AND status = ?", workspaceID, enterprisemodels.EntityStatusEnabled).
		Where("id IN ? OR external_user_id IN ? OR external_open_id IN ?", userIDs, userIDs, userIDs).
		Find(&users).Error; err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return []string{}, nil
	}

	ownerUserIDs := make([]string, 0, len(users))
	defaultGroupIDs := make([]string, 0, len(users))
	canonicalUserIDs := make([]string, 0, len(users))
	for _, user := range users {
		if user.ID != "" {
			ownerUserIDs = append(ownerUserIDs, user.ID)
			canonicalUserIDs = append(canonicalUserIDs, user.ID)
		}
		if user.DefaultGroupID != "" {
			defaultGroupIDs = append(defaultGroupIDs, user.DefaultGroupID)
		}
	}

	orgUnitIDs, err := d.orgUnitIDsForUsers(ctx, workspaceID, canonicalUserIDs)
	if err != nil {
		return nil, err
	}

	var groupIDs []string
	err = d.db.WithContext(ctx).
		Model(&model.Group{}).
		Where("workspace_id = ? AND status = ?", workspaceID, model.GroupStatusEnabled).
		Where(
			"owner_user_id IN ? OR id IN ? OR org_unit_id IN ?",
			nonEmptyOrNoMatch(ownerUserIDs),
			nonEmptyOrNoMatch(defaultGroupIDs),
			nonEmptyOrNoMatch(orgUnitIDs),
		).
		Pluck("id", &groupIDs).Error
	sort.Strings(groupIDs)
	return groupIDs, err
}

func (d *GORMOrgDirectory) AllGroupIDsForWorkspace(ctx context.Context, workspaceID string) ([]string, error) {
	workspaceID = defaultWorkspaceID(workspaceID)

	var groupIDs []string
	err := d.db.WithContext(ctx).
		Model(&model.Group{}).
		Where("workspace_id = ? AND status = ?", workspaceID, model.GroupStatusEnabled).
		Pluck("id", &groupIDs).Error
	sort.Strings(groupIDs)
	return groupIDs, err
}

func (d *GORMOrgDirectory) resolveOrgUnits(
	ctx context.Context,
	workspaceID string,
	orgUnitIDs []string,
) ([]enterprisemodels.OrgUnit, error) {
	var roots []enterprisemodels.OrgUnit
	err := d.db.WithContext(ctx).
		Where("workspace_id = ? AND status = ?", workspaceID, enterprisemodels.EntityStatusEnabled).
		Where("id IN ? OR external_id IN ? OR external_open_id IN ?", orgUnitIDs, orgUnitIDs, orgUnitIDs).
		Find(&roots).Error
	return roots, err
}

func defaultWorkspaceID(workspaceID string) string {
	if workspaceID == "" {
		return enterprisemodels.WorkspaceDefaultID
	}
	return workspaceID
}

func (d *GORMOrgDirectory) orgUnitIDsForUsers(
	ctx context.Context,
	workspaceID string,
	userIDs []string,
) ([]string, error) {
	userIDs = compactStrings(userIDs)
	if len(userIDs) == 0 {
		return []string{}, nil
	}

	var orgUnitIDs []string
	err := d.db.WithContext(ctx).
		Model(&enterprisemodels.UserOrgUnit{}).
		Where("workspace_id = ? AND user_id IN ?", workspaceID, userIDs).
		Distinct("org_unit_id").
		Pluck("org_unit_id", &orgUnitIDs).Error
	sort.Strings(orgUnitIDs)
	return orgUnitIDs, err
}

func nonEmptyOrNoMatch(values []string) []string {
	values = compactStrings(values)
	if len(values) == 0 {
		return []string{"__analyticsx_no_match__"}
	}
	return values
}

func compactStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func sortedKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
