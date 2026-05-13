//go:build enterprise

package analyticsx

import (
	"context"
	"testing"

	enterprisemodels "github.com/labring/aiproxy/core/enterprise/models"
	"github.com/labring/aiproxy/core/model"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupOrgDirectoryDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := model.OpenSQLite(t.TempDir() + "/org-directory.db")
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.Group{},
		&enterprisemodels.OrgUnit{},
		&enterprisemodels.EnterpriseUser{},
		&enterprisemodels.UserOrgUnit{},
	))

	return db
}

func TestOrgDirectoryDescendantsUseCanonicalOrgUnits(t *testing.T) {
	db := setupOrgDirectoryDB(t)
	ctx := context.Background()
	dir := NewGORMOrgDirectory(db)

	require.NoError(t, db.Create([]enterprisemodels.OrgUnit{
		{
			ID:          "ou:default:feishu:root",
			WorkspaceID: "default",
			Provider:    enterprisemodels.ProviderFeishu,
			ExternalID:  "root",
			Path:        "/ou:default:feishu:root",
			Name:        "Root",
			Status:      enterprisemodels.EntityStatusEnabled,
		},
		{
			ID:             "ou:default:feishu:rd",
			WorkspaceID:    "default",
			Provider:       enterprisemodels.ProviderFeishu,
			ExternalID:     "rd",
			ExternalOpenID: "od-rd",
			ParentID:       "ou:default:feishu:root",
			Path:           "/ou:default:feishu:root/ou:default:feishu:rd",
			Name:           "R&D",
			Status:         enterprisemodels.EntityStatusEnabled,
		},
		{
			ID:          "ou:default:feishu:ai",
			WorkspaceID: "default",
			Provider:    enterprisemodels.ProviderFeishu,
			ExternalID:  "ai",
			ParentID:    "ou:default:feishu:rd",
			Path:        "/ou:default:feishu:root/ou:default:feishu:rd/ou:default:feishu:ai",
			Name:        "AI",
			Status:      enterprisemodels.EntityStatusEnabled,
		},
	}).Error)

	got, err := dir.DescendantOrgUnitIDs(ctx, "default", []string{"rd"})
	require.NoError(t, err)

	require.ElementsMatch(t, []string{
		"ou:default:feishu:rd",
		"ou:default:feishu:ai",
	}, got)
}

func TestOrgDirectoryMapsOrgUnitsToGroupIDs(t *testing.T) {
	db := setupOrgDirectoryDB(t)
	ctx := context.Background()
	dir := NewGORMOrgDirectory(db)

	require.NoError(t, db.Create([]enterprisemodels.OrgUnit{
		{
			ID:          "ou-root",
			WorkspaceID: "workspace-a",
			Provider:    enterprisemodels.ProviderManual,
			ExternalID:  "root",
			Path:        "/ou-root",
			Name:        "Root",
			Status:      enterprisemodels.EntityStatusEnabled,
		},
		{
			ID:          "ou-child",
			WorkspaceID: "workspace-a",
			Provider:    enterprisemodels.ProviderManual,
			ExternalID:  "child",
			ParentID:    "ou-root",
			Path:        "/ou-root/ou-child",
			Name:        "Child",
			Status:      enterprisemodels.EntityStatusEnabled,
		},
	}).Error)
	require.NoError(t, db.Create([]model.Group{
		{ID: "group-root", WorkspaceID: "workspace-a", OrgUnitID: "ou-root", Status: model.GroupStatusEnabled},
		{ID: "group-child", WorkspaceID: "workspace-a", OrgUnitID: "ou-child", Status: model.GroupStatusEnabled},
		{ID: "group-disabled", WorkspaceID: "workspace-a", OrgUnitID: "ou-child", Status: model.GroupStatusDisabled},
		{ID: "group-other-workspace", WorkspaceID: "workspace-b", OrgUnitID: "ou-child", Status: model.GroupStatusEnabled},
		{ID: "group-other-org", WorkspaceID: "workspace-a", OrgUnitID: "ou-other", Status: model.GroupStatusEnabled},
	}).Error)

	got, err := dir.GroupIDsForOrgUnits(ctx, "workspace-a", []string{"ou-root"})
	require.NoError(t, err)

	require.ElementsMatch(t, []string{"group-root", "group-child"}, got)
}

func TestOrgDirectoryEmptyOrgUnitsDoNotReturnWorkspaceGroups(t *testing.T) {
	db := setupOrgDirectoryDB(t)
	ctx := context.Background()
	dir := NewGORMOrgDirectory(db)

	require.NoError(t, db.Create(&model.Group{
		ID:          "group-unfiltered",
		WorkspaceID: "default",
		OrgUnitID:   "ou-existing",
		Status:      model.GroupStatusEnabled,
	}).Error)

	got, err := dir.GroupIDsForOrgUnits(ctx, "default", nil)
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestOrgDirectoryAllGroupIDsForWorkspaceRequiresExplicitCall(t *testing.T) {
	db := setupOrgDirectoryDB(t)
	ctx := context.Background()
	dir := NewGORMOrgDirectory(db)

	require.NoError(t, db.Create([]model.Group{
		{ID: "group-enabled", WorkspaceID: "default", Status: model.GroupStatusEnabled},
		{ID: "group-disabled", WorkspaceID: "default", Status: model.GroupStatusDisabled},
		{ID: "group-other", WorkspaceID: "other", Status: model.GroupStatusEnabled},
	}).Error)

	got, err := dir.AllGroupIDsForWorkspace(ctx, "default")
	require.NoError(t, err)
	require.Equal(t, []string{"group-enabled"}, got)
}

func TestOrgDirectoryMapsUsersThroughOrgMembershipsToGroupIDs(t *testing.T) {
	db := setupOrgDirectoryDB(t)
	ctx := context.Background()
	dir := NewGORMOrgDirectory(db)

	require.NoError(t, db.Create(&enterprisemodels.EnterpriseUser{
		ID:             "user-a",
		WorkspaceID:    "default",
		Provider:       enterprisemodels.ProviderManual,
		ExternalOpenID: "open-a",
		Status:         enterprisemodels.EntityStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&enterprisemodels.UserOrgUnit{
		WorkspaceID: "default",
		UserID:      "user-a",
		OrgUnitID:   "ou-a",
		IsPrimary:   true,
	}).Error)
	require.NoError(t, db.Create([]model.Group{
		{ID: "group-owned", WorkspaceID: "default", OwnerUserID: "user-a", Status: model.GroupStatusEnabled},
		{ID: "group-org", WorkspaceID: "default", OrgUnitID: "ou-a", Status: model.GroupStatusEnabled},
		{ID: "group-disabled", WorkspaceID: "default", OrgUnitID: "ou-a", Status: model.GroupStatusDisabled},
	}).Error)

	got, err := dir.GroupIDsForUsers(ctx, "default", []string{"open-a"})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"group-owned", "group-org"}, got)
}

func TestOrgDirectorySupportsFeishuCompatibilityIDs(t *testing.T) {
	db := setupOrgDirectoryDB(t)
	ctx := context.Background()
	dir := NewGORMOrgDirectory(db)

	require.NoError(t, db.Create([]enterprisemodels.OrgUnit{
		{
			ID:             "ou:default:feishu:rd",
			WorkspaceID:    "default",
			Provider:       enterprisemodels.ProviderFeishu,
			ExternalID:     "rd",
			ExternalOpenID: "od-rd",
			Path:           "/ou:default:feishu:rd",
			Name:           "R&D",
			Status:         enterprisemodels.EntityStatusEnabled,
		},
		{
			ID:          "ou:default:feishu:ai",
			WorkspaceID: "default",
			Provider:    enterprisemodels.ProviderFeishu,
			ExternalID:  "ai",
			ParentID:    "ou:default:feishu:rd",
			Path:        "/ou:default:feishu:rd/ou:default:feishu:ai",
			Name:        "AI",
			Status:      enterprisemodels.EntityStatusEnabled,
		},
	}).Error)
	require.NoError(t, db.Create(&model.Group{
		ID:          "group-ai",
		WorkspaceID: "default",
		OrgUnitID:   "ou:default:feishu:ai",
		Status:      model.GroupStatusEnabled,
	}).Error)

	got, err := dir.GroupIDsForOrgUnits(ctx, "default", []string{"od-rd"})
	require.NoError(t, err)

	require.Equal(t, []string{"group-ai"}, got)
}

func TestOrgDirectoryEmptyMatchReturnsEmptyGroups(t *testing.T) {
	db := setupOrgDirectoryDB(t)
	ctx := context.Background()
	dir := NewGORMOrgDirectory(db)

	require.NoError(t, db.Create(&model.Group{
		ID:          "group-unfiltered",
		WorkspaceID: "default",
		OrgUnitID:   "ou-existing",
		Status:      model.GroupStatusEnabled,
	}).Error)

	got, err := dir.GroupIDsForOrgUnits(ctx, "default", []string{"missing-org-unit"})
	require.NoError(t, err)

	if got == nil {
		t.Fatal("missing org unit returned nil, want empty non-nil slice")
	}
	require.Empty(t, got)
}
