//go:build enterprise

package orgsync

import (
	"context"
	"strings"
	"testing"

	enterprisemodels "github.com/labring/aiproxy/core/enterprise/models"
	"github.com/labring/aiproxy/core/model"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupOrgSyncDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := model.OpenSQLite(t.TempDir() + "/orgsync.db")
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.Group{},
		&enterprisemodels.Workspace{},
		&enterprisemodels.OrgUnit{},
		&enterprisemodels.EnterpriseUser{},
		&enterprisemodels.UserOrgUnit{},
	))
	require.NoError(t, enterprisemodels.EnsureDefaultWorkspace(db))

	return db
}

func TestSyncSnapshotSupportsDeepOrgTreeAndDescendants(t *testing.T) {
	db := setupOrgSyncDB(t)

	snapshot := Snapshot{
		WorkspaceID: enterprisemodels.WorkspaceDefaultID,
		Provider:    enterprisemodels.ProviderFeishu,
		OrgUnits: []OrgUnitRecord{
			{ExternalID: "root", Name: "Root"},
			{ExternalID: "l1", ParentExternalID: "root", Name: "Level 1"},
			{ExternalID: "l2", ParentExternalID: "l1", Name: "Level 2"},
			{ExternalID: "l3", ParentExternalID: "l2", Name: "Level 3"},
			{ExternalID: "l4", ParentExternalID: "l3", Name: "Level 4"},
		},
	}

	require.NoError(t, SyncSnapshot(context.Background(), db, snapshot))

	l2ID := OrgUnitID(enterprisemodels.WorkspaceDefaultID, enterprisemodels.ProviderFeishu, "l2")
	desc, err := GetOrgUnitDescendantIDs(context.Background(), db, enterprisemodels.WorkspaceDefaultID, l2ID)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{
		OrgUnitID(enterprisemodels.WorkspaceDefaultID, enterprisemodels.ProviderFeishu, "l2"),
		OrgUnitID(enterprisemodels.WorkspaceDefaultID, enterprisemodels.ProviderFeishu, "l3"),
		OrgUnitID(enterprisemodels.WorkspaceDefaultID, enterprisemodels.ProviderFeishu, "l4"),
	}, desc)
}

func TestSyncSnapshotCreatesUserMembershipsAndPersonalGroup(t *testing.T) {
	db := setupOrgSyncDB(t)

	snapshot := Snapshot{
		Provider: enterprisemodels.ProviderFeishu,
		OrgUnits: []OrgUnitRecord{
			{ExternalID: "root", Name: "Root"},
			{ExternalID: "rd", ParentExternalID: "root", Name: "R&D"},
			{ExternalID: "ai", ParentExternalID: "rd", Name: "AI"},
		},
		Users: []UserRecord{
			{
				ExternalOpenID:           "ou_1",
				ExternalUserID:           "user_1",
				Name:                     "User One",
				Email:                    "u1@example.com",
				PrimaryOrgUnitExternalID: "ai",
				OrgUnitExternalIDs:       []string{"ai", "rd", "ai"},
				Raw:                      map[string]any{"active": true},
			},
		},
	}

	require.NoError(t, SyncSnapshot(context.Background(), db, snapshot))

	userID := EnterpriseUserID(enterprisemodels.WorkspaceDefaultID, enterprisemodels.ProviderFeishu, "ou_1")
	var user enterprisemodels.EnterpriseUser
	require.NoError(t, db.First(&user, "id = ?", userID).Error)
	require.Equal(t, "feishu_ou_1", user.DefaultGroupID)
	require.Equal(t, OrgUnitID(enterprisemodels.WorkspaceDefaultID, enterprisemodels.ProviderFeishu, "ai"), user.PrimaryOrgUnitID)
	require.Equal(t, enterprisemodels.EntityStatusEnabled, user.Status)
	require.JSONEq(t, `{"active":true}`, user.Raw)

	var memberships []enterprisemodels.UserOrgUnit
	require.NoError(t, db.Where("user_id = ?", userID).Find(&memberships).Error)
	require.Len(t, memberships, 2)

	primaryID := OrgUnitID(enterprisemodels.WorkspaceDefaultID, enterprisemodels.ProviderFeishu, "ai")
	for _, membership := range memberships {
		if membership.OrgUnitID == primaryID {
			require.True(t, membership.IsPrimary)
		} else {
			require.False(t, membership.IsPrimary)
		}
	}

	var group model.Group
	require.NoError(t, db.First(&group, "id = ?", "feishu_ou_1").Error)
	require.Equal(t, enterprisemodels.WorkspaceDefaultID, group.WorkspaceID)
	require.Equal(t, model.GroupTypePersonal, group.Type)
	require.Equal(t, userID, group.OwnerUserID)
	require.Equal(t, "ou_1", group.OwnerOpenID)
	require.Equal(t, primaryID, group.OrgUnitID)
	require.Equal(t, model.GroupStatusEnabled, group.Status)
	require.Equal(t, "User One", group.Name)

	groupIDs, err := GetGroupIDsForOrgUnits(context.Background(), db, enterprisemodels.WorkspaceDefaultID, []string{
		OrgUnitID(enterprisemodels.WorkspaceDefaultID, enterprisemodels.ProviderFeishu, "rd"),
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"feishu_ou_1"}, groupIDs)
}

func TestSyncSnapshotRejectsInvalidOrgGraphs(t *testing.T) {
	db := setupOrgSyncDB(t)

	err := SyncSnapshot(context.Background(), db, Snapshot{
		Provider: enterprisemodels.ProviderFeishu,
		OrgUnits: []OrgUnitRecord{
			{ExternalID: "child", ParentExternalID: "missing", Name: "Child"},
		},
	})
	require.ErrorContains(t, err, "references missing parent")

	err = SyncSnapshot(context.Background(), db, Snapshot{
		Provider: enterprisemodels.ProviderFeishu,
		OrgUnits: []OrgUnitRecord{
			{ExternalID: "a", ParentExternalID: "b", Name: "A"},
			{ExternalID: "b", ParentExternalID: "a", Name: "B"},
		},
	})
	require.ErrorContains(t, err, "cycle detected")
}

func TestSyncSnapshotUsesBoundedStableIDs(t *testing.T) {
	db := setupOrgSyncDB(t)

	longWorkspaceID := strings.Repeat("w", 64)
	longExternalID := strings.Repeat("d", 128)
	longOpenID := strings.Repeat("u", 128)

	require.NoError(t, SyncSnapshot(context.Background(), db, Snapshot{
		WorkspaceID: longWorkspaceID,
		Provider:    enterprisemodels.ProviderFeishu,
		OrgUnits: []OrgUnitRecord{
			{ExternalID: longExternalID, Name: "Long Department"},
		},
		Users: []UserRecord{
			{
				ExternalOpenID:           longOpenID,
				Name:                     "Long User",
				PrimaryOrgUnitExternalID: longExternalID,
			},
		},
	}))

	orgUnitID := OrgUnitID(longWorkspaceID, enterprisemodels.ProviderFeishu, longExternalID)
	userID := EnterpriseUserID(longWorkspaceID, enterprisemodels.ProviderFeishu, longOpenID)
	groupID := personalGroupID(enterprisemodels.ProviderFeishu, longOpenID)
	maxProviderGroupID := personalGroupID(strings.Repeat("p", 32), longOpenID)

	require.LessOrEqual(t, len(orgUnitID), 64)
	require.LessOrEqual(t, len(userID), 64)
	require.LessOrEqual(t, len(groupID), 64)
	require.LessOrEqual(t, len(maxProviderGroupID), 64)

	var unit enterprisemodels.OrgUnit
	require.NoError(t, db.First(&unit, "id = ?", orgUnitID).Error)
	require.Equal(t, longExternalID, unit.ExternalID)

	var user enterprisemodels.EnterpriseUser
	require.NoError(t, db.First(&user, "id = ?", userID).Error)
	require.Equal(t, groupID, user.DefaultGroupID)

	var group model.Group
	require.NoError(t, db.First(&group, "id = ?", groupID).Error)
	require.Equal(t, userID, group.OwnerUserID)
}

func TestSyncSnapshotRejectsOverLimitRawFields(t *testing.T) {
	db := setupOrgSyncDB(t)

	err := SyncSnapshot(context.Background(), db, Snapshot{
		WorkspaceID: strings.Repeat("w", 65),
		Provider:    enterprisemodels.ProviderFeishu,
	})
	require.ErrorContains(t, err, "workspace id length exceeds 64")

	err = SyncSnapshot(context.Background(), db, Snapshot{
		Provider: enterprisemodels.ProviderFeishu,
		OrgUnits: []OrgUnitRecord{
			{ExternalID: strings.Repeat("d", 129), Name: "Department"},
		},
	})
	require.ErrorContains(t, err, "org unit external id length exceeds 128")

	err = SyncSnapshot(context.Background(), db, Snapshot{
		Provider: enterprisemodels.ProviderFeishu,
		Users: []UserRecord{
			{ExternalOpenID: strings.Repeat("u", 129), Name: "User"},
		},
	})
	require.ErrorContains(t, err, "user external open id length exceeds 128")
}
