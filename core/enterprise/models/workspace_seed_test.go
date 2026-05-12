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
	require.NoError(t, db.AutoMigrate(
		&Workspace{},
		&OrgUnit{},
		&EnterpriseUser{},
		&UserOrgUnit{},
		&FeishuDepartment{},
		&model.Group{},
		&FeishuUser{},
	))

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

func TestBackfillWorkspaceGovernanceMirrorsExistingFeishuData(t *testing.T) {
	db, err := model.OpenSQLite(t.TempDir() + "/workspace_mirror_backfill.db")
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&Workspace{},
		&OrgUnit{},
		&EnterpriseUser{},
		&UserOrgUnit{},
		&FeishuDepartment{},
		&FeishuUser{},
		&model.Group{},
	))

	require.NoError(t, db.Create(&FeishuDepartment{
		DepartmentID:     "dept-root",
		OpenDepartmentID: "od-root",
		Name:             "Root",
		Status:           EntityStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&FeishuDepartment{
		DepartmentID:     "dept-child",
		OpenDepartmentID: "od-child",
		ParentID:         "dept-root",
		Name:             "Child",
		Status:           EntityStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&model.Group{ID: "feishu_ou_backfill", Name: "Backfill User"}).Error)
	require.NoError(t, db.Create(&FeishuUser{
		OpenID:        "ou_backfill",
		UnionID:       "on_backfill",
		UserID:        "user_backfill",
		TenantID:      "tenant-a",
		Name:          "Backfill User",
		Email:         "backfill@example.com",
		DepartmentID:  "dept-child",
		DepartmentIDs: `["dept-child","dept-root","missing"]`,
		GroupID:       "feishu_ou_backfill",
		Status:        EntityStatusEnabled,
	}).Error)

	require.NoError(t, EnsureDefaultWorkspace(db))
	require.NoError(t, BackfillWorkspaceGovernance(db))

	userID := "eu:default:feishu:ou_backfill"
	childID := "ou:default:feishu:dept-child"

	var unit OrgUnit
	require.NoError(t, db.First(&unit, "id = ?", childID).Error)
	require.Equal(t, "dept-child", unit.ExternalID)
	require.Equal(t, "ou:default:feishu:dept-root", unit.ParentID)
	require.Equal(t, "/ou:default:feishu:dept-root/ou:default:feishu:dept-child", unit.Path)
	require.Equal(t, 1, unit.Depth)

	var enterpriseUser EnterpriseUser
	require.NoError(t, db.First(&enterpriseUser, "id = ?", userID).Error)
	require.Equal(t, "feishu_ou_backfill", enterpriseUser.DefaultGroupID)
	require.Equal(t, childID, enterpriseUser.PrimaryOrgUnitID)

	var memberships []UserOrgUnit
	require.NoError(t, db.Where("user_id = ?", userID).Find(&memberships).Error)
	require.Len(t, memberships, 2)

	var feishuUser FeishuUser
	require.NoError(t, db.First(&feishuUser, "open_id = ?", "ou_backfill").Error)
	require.Equal(t, userID, feishuUser.EnterpriseUserID)

	var group model.Group
	require.NoError(t, db.First(&group, "id = ?", "feishu_ou_backfill").Error)
	require.Equal(t, userID, group.OwnerUserID)
	require.Equal(t, childID, group.OrgUnitID)
}
