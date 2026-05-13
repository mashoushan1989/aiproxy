//go:build enterprise

package analyticsx

import (
	"context"
	"errors"
	"testing"

	enterprisemodels "github.com/labring/aiproxy/core/enterprise/models"
	"github.com/labring/aiproxy/core/model"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupServiceDBs(t *testing.T) (*gorm.DB, *gorm.DB) {
	t.Helper()

	db, err := model.OpenSQLite(t.TempDir() + "/service.db")
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.Group{},
		&enterprisemodels.OrgUnit{},
		&enterprisemodels.EnterpriseUser{},
		&enterprisemodels.UserOrgUnit{},
	))

	logDB, err := model.OpenSQLite(t.TempDir() + "/service-log.db")
	require.NoError(t, err)
	require.NoError(t, logDB.AutoMigrate(&model.GroupSummary{}))

	return db, logDB
}

func newTestService(db, logDB *gorm.DB) Service {
	return Service{
		DB:           db,
		LogDB:        logDB,
		OrgDirectory: NewGORMOrgDirectory(db),
	}
}

func seedServiceOrg(t *testing.T, db *gorm.DB) {
	t.Helper()

	require.NoError(t, db.Create([]enterprisemodels.OrgUnit{
		{
			ID:          "dept-a",
			WorkspaceID: "default",
			Provider:    enterprisemodels.ProviderManual,
			ExternalID:  "dept-a",
			Path:        "/dept-a",
			Name:        "Department A",
			MemberCount: 2,
			Status:      enterprisemodels.EntityStatusEnabled,
		},
		{
			ID:          "dept-b",
			WorkspaceID: "default",
			Provider:    enterprisemodels.ProviderManual,
			ExternalID:  "dept-b",
			Path:        "/dept-b",
			Name:        "Department B",
			MemberCount: 1,
			Status:      enterprisemodels.EntityStatusEnabled,
		},
	}).Error)
	require.NoError(t, db.Create([]model.Group{
		{ID: "group-a", WorkspaceID: "default", OrgUnitID: "dept-a", OwnerUserID: "user-a", Status: model.GroupStatusEnabled},
		{ID: "group-b", WorkspaceID: "default", OrgUnitID: "dept-b", OwnerUserID: "user-b", Status: model.GroupStatusEnabled},
		{ID: "group-other-workspace", WorkspaceID: "other", OrgUnitID: "dept-a", OwnerUserID: "user-c", Status: model.GroupStatusEnabled},
	}).Error)
	require.NoError(t, db.Create([]enterprisemodels.EnterpriseUser{
		{
			ID:               "user-a",
			WorkspaceID:      "default",
			Provider:         enterprisemodels.ProviderManual,
			ExternalOpenID:   "open-a",
			Name:             "Alice",
			DefaultGroupID:   "group-a",
			PrimaryOrgUnitID: "dept-a",
			Status:           enterprisemodels.EntityStatusEnabled,
		},
		{
			ID:               "user-b",
			WorkspaceID:      "default",
			Provider:         enterprisemodels.ProviderManual,
			ExternalOpenID:   "open-b",
			Name:             "Bob",
			DefaultGroupID:   "group-b",
			PrimaryOrgUnitID: "dept-b",
			Status:           enterprisemodels.EntityStatusEnabled,
		},
	}).Error)
	require.NoError(t, db.Create([]enterprisemodels.UserOrgUnit{
		{WorkspaceID: "default", UserID: "user-a", OrgUnitID: "dept-a", IsPrimary: true},
		{WorkspaceID: "default", UserID: "user-b", OrgUnitID: "dept-b", IsPrimary: true},
	}).Error)
}

func seedGroupSummary(t *testing.T, logDB *gorm.DB, groupID, modelName string, amount float64, requests int64) {
	t.Helper()

	require.NoError(t, logDB.Create(&model.GroupSummary{
		Unique: model.GroupSummaryUnique{
			GroupID:       groupID,
			TokenName:     "token",
			Model:         modelName,
			HourTimestamp: 3600,
		},
		Data: model.SummaryData{
			SummaryDataSet: model.SummaryDataSet{
				Count: model.Count{
					RequestCount:   requests,
					Status2xxCount: model.ZeroNullInt64(requests),
				},
				Usage: model.Usage{
					InputTokens:  model.ZeroNullInt64(10 * requests),
					OutputTokens: model.ZeroNullInt64(20 * requests),
					TotalTokens:  model.ZeroNullInt64(30 * requests),
				},
				Amount: model.Amount{UsedAmount: amount},
			},
		},
	}).Error)
}

func serviceFilter() Filter {
	return Filter{StartTimestamp: 1, EndTimestamp: 7200}
}

func TestServiceDepartmentSummaryUsesOnlyScopedGroups(t *testing.T) {
	db, logDB := setupServiceDBs(t)
	seedServiceOrg(t, db)
	seedGroupSummary(t, logDB, "group-a", "gpt-4o", 2.5, 2)
	seedGroupSummary(t, logDB, "group-b", "gpt-4o", 9.0, 9)

	got, err := newTestService(db, logDB).DepartmentSummaries(context.Background(), Scope{
		WorkspaceID:     "default",
		AllowedGroupIDs: []string{"group-a"},
	}, serviceFilter())
	require.NoError(t, err)

	require.Len(t, got, 1)
	require.Equal(t, "dept-a", got[0].DepartmentID)
	require.Equal(t, "Department A", got[0].DepartmentName)
	require.Equal(t, int64(2), got[0].RequestCount)
	require.InDelta(t, 2.5, got[0].UsedAmount, 0.0001)
}

func TestServiceUserRankingOutOfScopeDepartmentReturnsEmpty(t *testing.T) {
	db, logDB := setupServiceDBs(t)
	seedServiceOrg(t, db)
	seedGroupSummary(t, logDB, "group-a", "gpt-4o", 2.5, 2)
	seedGroupSummary(t, logDB, "group-b", "gpt-4o", 9.0, 9)

	got, total, err := newTestService(db, logDB).UserRanking(context.Background(), Scope{
		WorkspaceID:     "default",
		AllowedGroupIDs: []string{"group-a"},
	}, Filter{StartTimestamp: 1, EndTimestamp: 7200, OrgUnitIDs: []string{"dept-b"}})
	require.NoError(t, err)
	require.Zero(t, total)
	require.Empty(t, got)
}

func TestServiceModelDistributionNeverFallsBackToGlobal(t *testing.T) {
	db, logDB := setupServiceDBs(t)
	seedServiceOrg(t, db)
	seedGroupSummary(t, logDB, "group-a", "gpt-4o", 2.5, 2)
	seedGroupSummary(t, logDB, "group-b", "claude-3-5", 9.0, 9)

	got, err := newTestService(db, logDB).ModelDistribution(context.Background(), Scope{
		WorkspaceID:     "default",
		AllowedGroupIDs: []string{"group-a"},
	}, Filter{StartTimestamp: 1, EndTimestamp: 7200, GroupIDs: []string{"group-b"}})
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestServiceModelDistributionAllGroupsStaysInWorkspace(t *testing.T) {
	db, logDB := setupServiceDBs(t)
	seedServiceOrg(t, db)
	seedGroupSummary(t, logDB, "group-a", "gpt-4o", 2.5, 2)
	seedGroupSummary(t, logDB, "group-other-workspace", "leaked-model", 50.0, 50)

	got, err := newTestService(db, logDB).ModelDistribution(context.Background(), Scope{
		WorkspaceID: "default",
		AllGroups:   true,
	}, serviceFilter())
	require.NoError(t, err)

	require.Len(t, got, 1)
	require.Equal(t, "gpt-4o", got[0].Model)
}

func TestServiceModelDistributionAppliesAllowedModels(t *testing.T) {
	db, logDB := setupServiceDBs(t)
	seedServiceOrg(t, db)
	seedGroupSummary(t, logDB, "group-a", "gpt-4o", 2.5, 2)
	seedGroupSummary(t, logDB, "group-a", "blocked-model", 7.5, 7)

	got, err := newTestService(db, logDB).ModelDistribution(context.Background(), Scope{
		WorkspaceID:     "default",
		AllowedGroupIDs: []string{"group-a"},
		AllowedModels:   []string{"gpt-4o"},
	}, serviceFilter())
	require.NoError(t, err)

	require.Len(t, got, 1)
	require.Equal(t, "gpt-4o", got[0].Model)
}

func TestServiceModelDistributionDeniedModelIntersectionReturnsEmpty(t *testing.T) {
	db, logDB := setupServiceDBs(t)
	seedServiceOrg(t, db)
	seedGroupSummary(t, logDB, "group-a", "gpt-4o", 2.5, 2)
	seedGroupSummary(t, logDB, "group-a", "blocked-model", 7.5, 7)

	got, err := newTestService(db, logDB).ModelDistribution(context.Background(), Scope{
		WorkspaceID:     "default",
		AllowedGroupIDs: []string{"group-a"},
		AllowedModels:   []string{"gpt-4o"},
	}, Filter{
		StartTimestamp: 1,
		EndTimestamp:   7200,
		Models:         []string{"blocked-model"},
	})
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestServiceQueriesUseContextTimeout(t *testing.T) {
	db, logDB := setupServiceDBs(t)
	seedServiceOrg(t, db)
	seedGroupSummary(t, logDB, "group-a", "gpt-4o", 2.5, 2)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := newTestService(db, logDB).DepartmentSummaries(ctx, Scope{
		WorkspaceID:     "default",
		AllowedGroupIDs: []string{"group-a"},
	}, serviceFilter())
	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled), "got %v", err)
}
