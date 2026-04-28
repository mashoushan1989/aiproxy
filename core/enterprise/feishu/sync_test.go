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

	if err := testDB.AutoMigrate(&models.FeishuUser{}); err != nil {
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
