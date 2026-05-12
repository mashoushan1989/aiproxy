//go:build enterprise

package models_test

import (
	"testing"

	. "github.com/labring/aiproxy/core/enterprise/models"
	"github.com/labring/aiproxy/core/model"
	"github.com/stretchr/testify/require"
)

func TestEnsureDefaultWorkspaceCreatesDefault(t *testing.T) {
	db, err := model.OpenSQLite(t.TempDir() + "/workspace_seed.db")
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Workspace{}))

	require.NoError(t, EnsureDefaultWorkspace(db))
	require.NoError(t, EnsureDefaultWorkspace(db))

	var count int64
	require.NoError(t, db.Model(&Workspace{}).Where("id = ?", WorkspaceDefaultID).Count(&count).Error)
	require.Equal(t, int64(1), count)

	var ws Workspace
	require.NoError(t, db.First(&ws, "id = ?", WorkspaceDefaultID).Error)
	require.Equal(t, "Default", ws.Name)
	require.Equal(t, WorkspaceKindPrimary, ws.Kind)
	require.Equal(t, WorkspaceStatusEnabled, ws.Status)
}

func TestBackfillWorkspaceGovernanceSetsPersonalFeishuGroups(t *testing.T) {
	db, err := model.OpenSQLite(t.TempDir() + "/workspace_backfill.db")
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Workspace{}, &model.Group{}, &FeishuUser{}))

	require.NoError(t, db.Create(&model.Group{ID: "feishu_ou_1", Name: "User One"}).Error)
	require.NoError(t, db.Create(&model.Group{ID: "manual-team", Name: "Manual Team"}).Error)
	require.NoError(t, db.Create(&model.Group{ID: "feishu_ou_2", Name: "User Two", WorkspaceID: "sandbox"}).Error)
	require.NoError(t, db.Create(&FeishuUser{
		OpenID:   "ou_1",
		TenantID: "tenant_a",
		GroupID:  "feishu_ou_1",
		Status:   1,
	}).Error)
	require.NoError(t, db.Create(&FeishuUser{
		OpenID:           "ou_2",
		TenantID:         "tenant_b",
		WorkspaceID:      "sandbox",
		ExternalTenantID: "external_b",
		GroupID:          "feishu_ou_2",
		Status:           1,
	}).Error)

	require.NoError(t, EnsureDefaultWorkspace(db))
	require.NoError(t, BackfillWorkspaceGovernance(db))
	require.NoError(t, BackfillWorkspaceGovernance(db))

	var personal model.Group
	require.NoError(t, db.First(&personal, "id = ?", "feishu_ou_1").Error)
	require.Equal(t, WorkspaceDefaultID, personal.WorkspaceID)
	require.Equal(t, model.GroupTypePersonal, personal.Type)
	require.Equal(t, "ou_1", personal.OwnerOpenID)

	var manual model.Group
	require.NoError(t, db.First(&manual, "id = ?", "manual-team").Error)
	require.Equal(t, WorkspaceDefaultID, manual.WorkspaceID)
	require.Empty(t, manual.Type)

	var user FeishuUser
	require.NoError(t, db.First(&user, "open_id = ?", "ou_1").Error)
	require.Equal(t, WorkspaceDefaultID, user.WorkspaceID)
	require.Equal(t, "tenant_a", user.ExternalTenantID)

	var existingGroup model.Group
	require.NoError(t, db.First(&existingGroup, "id = ?", "feishu_ou_2").Error)
	require.Equal(t, "sandbox", existingGroup.WorkspaceID)
	require.Equal(t, model.GroupTypePersonal, existingGroup.Type)
	require.Equal(t, "ou_2", existingGroup.OwnerOpenID)

	var existingUser FeishuUser
	require.NoError(t, db.First(&existingUser, "open_id = ?", "ou_2").Error)
	require.Equal(t, "sandbox", existingUser.WorkspaceID)
	require.Equal(t, "external_b", existingUser.ExternalTenantID)
}
