//go:build enterprise

package models

import (
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
		for _, user := range users {
			updates := map[string]any{}
			if user.WorkspaceID == "" {
				updates["workspace_id"] = WorkspaceDefaultID
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

			groupUpdates := map[string]any{
				"type":          "personal",
				"owner_open_id": user.OpenID,
			}
			if user.EnterpriseUserID != "" {
				groupUpdates["owner_user_id"] = user.EnterpriseUserID
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
