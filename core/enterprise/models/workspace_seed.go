//go:build enterprise

package models

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func EnsureDefaultWorkspace(db *gorm.DB) error {
	ws := Workspace{
		ID:              WorkspaceDefaultID,
		Name:            "Default",
		Slug:            "default",
		Kind:            WorkspaceKindPrimary,
		Status:          WorkspaceStatusEnabled,
		DefaultProvider: ProviderFeishu,
	}

	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoNothing: true,
	}).Create(&ws).Error
}

func BackfillWorkspaceGovernance(db *gorm.DB) error {
	if err := db.Table("groups").
		Where("workspace_id = '' OR workspace_id IS NULL").
		Update("workspace_id", WorkspaceDefaultID).Error; err != nil {
		return err
	}

	var users []FeishuUser
	if err := db.Find(&users).Error; err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := backfillFeishuOrgUnits(tx); err != nil {
			return err
		}
		departmentIDs, err := loadFeishuDepartmentIDs(tx)
		if err != nil {
			return err
		}

		for _, user := range users {
			enterpriseUserID := ""
			if user.OpenID != "" {
				enterpriseUserID = enterpriseUserIDForBackfill(WorkspaceDefaultID, ProviderFeishu, user.OpenID)
			}

			updates := map[string]any{}
			if user.WorkspaceID == "" {
				updates["workspace_id"] = WorkspaceDefaultID
			}
			if user.EnterpriseUserID == "" && enterpriseUserID != "" {
				updates["enterprise_user_id"] = enterpriseUserID
			}
			if user.ExternalTenantID == "" && user.TenantID != "" {
				updates["external_tenant_id"] = user.TenantID
			}
			if len(updates) > 0 {
				if err := tx.Model(&FeishuUser{}).Where("id = ?", user.ID).Updates(updates).Error; err != nil {
					return err
				}
			}

			if user.GroupID == "" || !strings.HasPrefix(user.GroupID, "feishu_") {
				continue
			}

			if enterpriseUserID != "" {
				if err := backfillFeishuEnterpriseUser(tx, user, enterpriseUserID, departmentIDs); err != nil {
					return err
				}
				if err := backfillFeishuUserOrgUnits(tx, user, enterpriseUserID, departmentIDs); err != nil {
					return err
				}
			}

			groupUpdates := map[string]any{
				"type":          "personal",
				"owner_open_id": user.OpenID,
			}
			if enterpriseUserID != "" {
				groupUpdates["owner_user_id"] = enterpriseUserID
			}
			if orgUnitID := orgUnitIDForBackfill(WorkspaceDefaultID, ProviderFeishu, user.DepartmentID, departmentIDs); orgUnitID != "" {
				groupUpdates["org_unit_id"] = orgUnitID
			}

			if err := tx.Table("groups").
				Where("id = ?", user.GroupID).
				Where("workspace_id = '' OR workspace_id IS NULL").
				Update("workspace_id", WorkspaceDefaultID).Error; err != nil {
				return err
			}

			if err := tx.Table("groups").
				Where("id = ?", user.GroupID).
				Updates(groupUpdates).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func loadFeishuDepartmentIDs(tx *gorm.DB) (map[string]struct{}, error) {
	var departments []FeishuDepartment
	if err := tx.Where("status = ?", EntityStatusEnabled).Find(&departments).Error; err != nil {
		return nil, err
	}

	departmentIDs := make(map[string]struct{}, len(departments))
	for _, dept := range departments {
		if dept.DepartmentID != "" {
			departmentIDs[dept.DepartmentID] = struct{}{}
		}
	}

	return departmentIDs, nil
}

func backfillFeishuOrgUnits(tx *gorm.DB) error {
	var departments []FeishuDepartment
	if err := tx.Where("status = ?", EntityStatusEnabled).Find(&departments).Error; err != nil {
		return err
	}

	departmentIDs := make(map[string]struct{}, len(departments))
	for _, dept := range departments {
		if dept.DepartmentID != "" {
			departmentIDs[dept.DepartmentID] = struct{}{}
		}
	}
	departmentsByID := departmentsByExternalID(departments)

	for _, dept := range departments {
		if dept.DepartmentID == "" {
			continue
		}

		parentID := ""
		if dept.ParentID != "" && dept.ParentID != "0" {
			if _, ok := departmentIDs[dept.ParentID]; ok {
				parentID = orgUnitIDForBackfill(WorkspaceDefaultID, ProviderFeishu, dept.ParentID, departmentIDs)
			}
		}

		unit := OrgUnit{
			ID:             orgUnitIDForBackfill(WorkspaceDefaultID, ProviderFeishu, dept.DepartmentID, departmentIDs),
			WorkspaceID:    WorkspaceDefaultID,
			Provider:       ProviderFeishu,
			ExternalID:     dept.DepartmentID,
			ExternalOpenID: dept.OpenDepartmentID,
			ParentID:       parentID,
			Path:           buildOrgUnitPathForBackfill(WorkspaceDefaultID, ProviderFeishu, dept.DepartmentID, departmentsByID),
			Depth:          orgUnitDepthForBackfill(dept.DepartmentID, departmentsByID),
			Name:           dept.Name,
			Order:          dept.Order,
			MemberCount:    dept.MemberCount,
			Status:         EntityStatusEnabled,
		}
		if unit.Name == "" {
			unit.Name = dept.DepartmentID
		}

		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"workspace_id",
				"provider",
				"external_id",
				"external_open_id",
				"parent_id",
				"path",
				"depth",
				"name",
				"order",
				"member_count",
				"status",
				"updated_at",
			}),
		}).Create(&unit).Error; err != nil {
			return err
		}
	}

	return nil
}

func departmentsByExternalID(departments []FeishuDepartment) map[string]FeishuDepartment {
	records := make(map[string]FeishuDepartment, len(departments))
	for _, dept := range departments {
		if dept.DepartmentID != "" {
			records[dept.DepartmentID] = dept
		}
	}

	return records
}

func buildOrgUnitPathForBackfill(
	workspaceID string,
	provider string,
	externalID string,
	departments map[string]FeishuDepartment,
) string {
	segments := buildOrgUnitPathSegmentsForBackfill(workspaceID, provider, externalID, departments, map[string]struct{}{})
	return "/" + strings.Join(segments, "/")
}

func buildOrgUnitPathSegmentsForBackfill(
	workspaceID string,
	provider string,
	externalID string,
	departments map[string]FeishuDepartment,
	visiting map[string]struct{},
) []string {
	if _, ok := visiting[externalID]; ok {
		return []string{boundedIDForBackfill("ou", workspaceID, provider, externalID)}
	}
	visiting[externalID] = struct{}{}

	dept, ok := departments[externalID]
	if !ok || dept.ParentID == "" || dept.ParentID == "0" {
		return []string{boundedIDForBackfill("ou", workspaceID, provider, externalID)}
	}
	if _, ok := departments[dept.ParentID]; !ok {
		return []string{boundedIDForBackfill("ou", workspaceID, provider, externalID)}
	}

	parent := buildOrgUnitPathSegmentsForBackfill(workspaceID, provider, dept.ParentID, departments, visiting)
	return append(parent, boundedIDForBackfill("ou", workspaceID, provider, externalID))
}

func orgUnitDepthForBackfill(externalID string, departments map[string]FeishuDepartment) int {
	return orgUnitDepthForBackfillWithVisited(externalID, departments, map[string]struct{}{})
}

func orgUnitDepthForBackfillWithVisited(
	externalID string,
	departments map[string]FeishuDepartment,
	visiting map[string]struct{},
) int {
	if _, ok := visiting[externalID]; ok {
		return 0
	}
	visiting[externalID] = struct{}{}

	dept, ok := departments[externalID]
	if !ok || dept.ParentID == "" || dept.ParentID == "0" {
		return 0
	}
	if _, ok := departments[dept.ParentID]; !ok {
		return 0
	}

	return orgUnitDepthForBackfillWithVisited(dept.ParentID, departments, visiting) + 1
}

func backfillFeishuEnterpriseUser(
	tx *gorm.DB,
	user FeishuUser,
	enterpriseUserID string,
	departmentIDs map[string]struct{},
) error {
	primaryOrgUnitID := orgUnitIDForBackfill(WorkspaceDefaultID, ProviderFeishu, user.DepartmentID, departmentIDs)
	enterpriseUser := EnterpriseUser{
		ID:               enterpriseUserID,
		WorkspaceID:      WorkspaceDefaultID,
		Provider:         ProviderFeishu,
		ExternalUserID:   user.UserID,
		ExternalOpenID:   user.OpenID,
		ExternalUnionID:  user.UnionID,
		Name:             user.Name,
		Email:            user.Email,
		Avatar:           user.Avatar,
		Status:           EntityStatusEnabled,
		DefaultGroupID:   user.GroupID,
		PrimaryOrgUnitID: primaryOrgUnitID,
	}

	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"workspace_id",
			"provider",
			"external_user_id",
			"external_open_id",
			"external_union_id",
			"name",
			"email",
			"avatar",
			"status",
			"default_group_id",
			"primary_org_unit_id",
			"updated_at",
		}),
	}).Create(&enterpriseUser).Error
}

func backfillFeishuUserOrgUnits(
	tx *gorm.DB,
	user FeishuUser,
	enterpriseUserID string,
	knownDepartmentIDs map[string]struct{},
) error {
	if err := tx.
		Where("workspace_id = ? AND user_id = ?", WorkspaceDefaultID, enterpriseUserID).
		Delete(&UserOrgUnit{}).Error; err != nil {
		return err
	}

	rawDepartmentIDs := feishuDepartmentIDsForBackfill(user)
	seen := make(map[string]struct{}, len(rawDepartmentIDs))
	for _, departmentID := range rawDepartmentIDs {
		orgUnitID := orgUnitIDForBackfill(WorkspaceDefaultID, ProviderFeishu, departmentID, knownDepartmentIDs)
		if orgUnitID == "" {
			continue
		}
		if _, ok := seen[orgUnitID]; ok {
			continue
		}
		seen[orgUnitID] = struct{}{}

		if err := tx.Create(&UserOrgUnit{
			WorkspaceID: WorkspaceDefaultID,
			UserID:      enterpriseUserID,
			OrgUnitID:   orgUnitID,
			IsPrimary:   departmentID == user.DepartmentID,
		}).Error; err != nil {
			return err
		}
	}

	return nil
}

func feishuDepartmentIDsForBackfill(user FeishuUser) []string {
	var ids []string
	if user.DepartmentID != "" && user.DepartmentID != "0" {
		ids = append(ids, user.DepartmentID)
	}

	var extra []string
	if user.DepartmentIDs != "" && json.Unmarshal([]byte(user.DepartmentIDs), &extra) == nil {
		ids = append(ids, extra...)
	}

	return ids
}

func orgUnitIDForBackfill(workspaceID, provider, externalID string, knownExternalIDs map[string]struct{}) string {
	if externalID == "" || externalID == "0" {
		return ""
	}
	if _, ok := knownExternalIDs[externalID]; !ok {
		return ""
	}

	return boundedIDForBackfill("ou", workspaceID, provider, externalID)
}

func enterpriseUserIDForBackfill(workspaceID, provider, externalOpenID string) string {
	return boundedIDForBackfill("eu", workspaceID, provider, externalOpenID)
}

func boundedIDForBackfill(parts ...string) string {
	raw := strings.Join(parts, ":")
	if len(raw) <= 64 {
		return raw
	}

	sum := sha256.Sum256([]byte(raw))
	prefix := strings.Join(parts[:len(parts)-1], ":")
	if len(prefix) > 24 {
		prefix = prefix[:24]
	}

	return fmt.Sprintf("%s:%s", prefix, hex.EncodeToString(sum[:])[:32])
}
