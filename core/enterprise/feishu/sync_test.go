//go:build enterprise

package feishu

import (
	"path/filepath"
	"testing"

	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/enterprise/models"
	"github.com/labring/aiproxy/core/model"
)

func setupFeishuSyncTestDB(t *testing.T) {
	t.Helper()

	prevDB := model.DB
	prevUsingSQLite := common.UsingSQLite

	testDB, err := model.OpenSQLite(filepath.Join(t.TempDir(), "enterprise-feishu-sync.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	model.DB = testDB
	common.UsingSQLite = true

	t.Cleanup(func() {
		model.DB = prevDB
		common.UsingSQLite = prevUsingSQLite
	})

	if err := testDB.AutoMigrate(&models.FeishuUser{}, &model.Group{}); err != nil {
		t.Fatalf("migrate test tables: %v", err)
	}
}

func TestUpsertFeishuUserPreservesExistingRole(t *testing.T) {
	setupFeishuSyncTestDB(t)

	if err := model.DB.Create(&models.FeishuUser{
		OpenID:  "ou_existing",
		Name:    "Old Name",
		GroupID: "feishu_ou_existing",
		Role:    models.RoleAdmin,
		Status:  1,
	}).Error; err != nil {
		t.Fatalf("create existing user: %v", err)
	}

	if _, err := upsertFeishuUser(model.DB, feishuUserFields{
		OpenID:       "ou_existing",
		Name:         "New Name",
		Email:        "new@example.com",
		DepartmentID: "dept-a",
		DeptPath:     &DepartmentPath{},
		GroupID:      "feishu_ou_existing",
	}); err != nil {
		t.Fatalf("upsert existing user: %v", err)
	}

	var user models.FeishuUser
	if err := model.DB.Where("open_id = ?", "ou_existing").First(&user).Error; err != nil {
		t.Fatalf("load existing user: %v", err)
	}

	if user.Role != models.RoleAdmin {
		t.Fatalf("role = %q, want %q", user.Role, models.RoleAdmin)
	}
}

func TestUpsertFeishuUserSetsDefaultRoleForNewUser(t *testing.T) {
	setupFeishuSyncTestDB(t)

	if _, err := upsertFeishuUser(model.DB, feishuUserFields{
		OpenID:   "ou_new",
		Name:     "New User",
		DeptPath: &DepartmentPath{},
		GroupID:  "feishu_ou_new",
	}); err != nil {
		t.Fatalf("upsert new user: %v", err)
	}

	var user models.FeishuUser
	if err := model.DB.Where("open_id = ?", "ou_new").First(&user).Error; err != nil {
		t.Fatalf("load new user: %v", err)
	}

	if user.Role != models.RoleViewer {
		t.Fatalf("role = %q, want %q", user.Role, models.RoleViewer)
	}
}

func TestUpsertFeishuUserSetsWorkspaceAndExternalTenant(t *testing.T) {
	setupFeishuSyncTestDB(t)

	if _, err := upsertFeishuUser(model.DB, feishuUserFields{
		OpenID:   "ou_workspace",
		TenantID: "tenant-a",
		Name:     "Workspace User",
		DeptPath: &DepartmentPath{},
		GroupID:  "feishu_ou_workspace",
	}); err != nil {
		t.Fatalf("upsert new user: %v", err)
	}

	var user models.FeishuUser
	if err := model.DB.Where("open_id = ?", "ou_workspace").First(&user).Error; err != nil {
		t.Fatalf("load new user: %v", err)
	}

	if user.WorkspaceID != models.WorkspaceDefaultID {
		t.Fatalf("workspace_id = %q, want %q", user.WorkspaceID, models.WorkspaceDefaultID)
	}

	if user.ExternalTenantID != "tenant-a" {
		t.Fatalf("external_tenant_id = %q, want %q", user.ExternalTenantID, "tenant-a")
	}

	if _, err := upsertFeishuUser(model.DB, feishuUserFields{
		OpenID:   "ou_workspace",
		TenantID: "tenant-b",
		Name:     "Workspace User Updated",
		DeptPath: &DepartmentPath{},
		GroupID:  "feishu_ou_workspace",
	}); err != nil {
		t.Fatalf("upsert existing user: %v", err)
	}

	if err := model.DB.Where("open_id = ?", "ou_workspace").First(&user).Error; err != nil {
		t.Fatalf("reload user: %v", err)
	}

	if user.WorkspaceID != models.WorkspaceDefaultID {
		t.Fatalf("updated workspace_id = %q, want %q", user.WorkspaceID, models.WorkspaceDefaultID)
	}

	if user.ExternalTenantID != "tenant-b" {
		t.Fatalf("updated external_tenant_id = %q, want %q", user.ExternalTenantID, "tenant-b")
	}
}

func TestEnsureFeishuPersonalGroupDoesNotClearOrgUnitOnEmptyDepartment(t *testing.T) {
	setupFeishuSyncTestDB(t)

	if err := ensureFeishuPersonalGroup(model.DB, "feishu_ou_group", "ou_group", "Group User", "dept-a"); err != nil {
		t.Fatalf("ensure group with department: %v", err)
	}
	if err := ensureFeishuPersonalGroup(model.DB, "feishu_ou_group", "ou_group", "Group User", ""); err != nil {
		t.Fatalf("ensure group without department: %v", err)
	}

	var group model.Group
	if err := model.DB.First(&group, "id = ?", "feishu_ou_group").Error; err != nil {
		t.Fatalf("load group: %v", err)
	}

	if group.OrgUnitID == "" {
		t.Fatal("org_unit_id was cleared by empty-department update")
	}
}

func TestUpsertOAuthFeishuUserRejectsInactiveUsers(t *testing.T) {
	setupFeishuSyncTestDB(t)

	if err := model.DB.Create(&models.FeishuUser{
		OpenID:  "ou_disabled",
		GroupID: "feishu_ou_disabled",
		Status:  2,
	}).Error; err != nil {
		t.Fatalf("create disabled user: %v", err)
	}

	_, err := upsertOAuthFeishuUser(model.DB, &UserInfo{
		OpenID:   "ou_disabled",
		TenantID: "tenant-a",
		Name:     "Disabled User",
	}, "feishu_ou_disabled")
	if err != errFeishuUserInactive {
		t.Fatalf("disabled user error = %v, want errFeishuUserInactive", err)
	}

	deleted := models.FeishuUser{
		OpenID:  "ou_deleted",
		GroupID: "feishu_ou_deleted",
		Status:  1,
	}
	if err := model.DB.Create(&deleted).Error; err != nil {
		t.Fatalf("create deleted user: %v", err)
	}
	if err := model.DB.Delete(&deleted).Error; err != nil {
		t.Fatalf("soft delete user: %v", err)
	}

	_, err = upsertOAuthFeishuUser(model.DB, &UserInfo{
		OpenID:   "ou_deleted",
		TenantID: "tenant-a",
		Name:     "Deleted User",
	}, "feishu_ou_deleted")
	if err != errFeishuUserInactive {
		t.Fatalf("deleted user error = %v, want errFeishuUserInactive", err)
	}

	var count int64
	if err := model.DB.Unscoped().Model(&models.FeishuUser{}).Where("open_id = ?", "ou_deleted").Count(&count).Error; err != nil {
		t.Fatalf("count deleted user rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("deleted user rows = %d, want 1", count)
	}
}
