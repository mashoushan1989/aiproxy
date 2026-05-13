//go:build enterprise

package analyticsx

import (
	"context"
	"errors"
	"strings"
	"testing"

	enterprisemodels "github.com/labring/aiproxy/core/enterprise/models"
	"github.com/labring/aiproxy/core/model"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupAuditDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := model.OpenSQLite(t.TempDir() + "/audit.db")
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&enterprisemodels.AnalyticsAuditEvent{}))
	return db
}

func TestAuditEventPersistsExportSuccess(t *testing.T) {
	db := setupAuditDB(t)

	err := PersistExportAuditEvent(context.Background(), db, ExportAuditInput{
		WorkspaceID:  "workspace-a",
		ActorGroupID: "group-a",
		Scope: Scope{
			WorkspaceID:     "workspace-a",
			AllowedGroupIDs: []string{"group-a"},
		},
		Filter:       Filter{StartTimestamp: 1, EndTimestamp: 2, Models: []string{"gpt-4o"}},
		ResultStatus: AuditResultSuccess,
		RowCount:     7,
	})
	require.NoError(t, err)

	var got enterprisemodels.AnalyticsAuditEvent
	require.NoError(t, db.First(&got).Error)
	require.Equal(t, "workspace-a", got.WorkspaceID)
	require.NotEmpty(t, got.ActorGroupHash)
	require.NotEqual(t, "group-a", got.ActorGroupHash)
	require.Equal(t, AuditActionExport, got.Action)
	require.Contains(t, got.ScopeSummary, "workspace=workspace-a")
	require.Contains(t, got.ScopeSummary, "groups=1")
	require.Contains(t, got.FilterJSON, `"models":["gpt-4o"]`)
	require.Equal(t, AuditResultSuccess, got.ResultStatus)
	require.Equal(t, 7, got.RowCount)
	require.Empty(t, got.ErrorMessage)
	require.False(t, got.CreatedAt.IsZero())
}

func TestAuditEventPersistsExportFailure(t *testing.T) {
	db := setupAuditDB(t)

	err := PersistExportAuditEvent(context.Background(), db, ExportAuditInput{
		WorkspaceID:  "workspace-a",
		ActorGroupID: "group-a",
		Scope:        Scope{WorkspaceID: "workspace-a", AllGroups: true},
		Filter:       Filter{StartTimestamp: 1, EndTimestamp: 2},
		ResultStatus: AuditResultFailure,
		RowCount:     0,
		Err:          errors.New("query export dataset: boom"),
	})
	require.NoError(t, err)

	var got enterprisemodels.AnalyticsAuditEvent
	require.NoError(t, db.First(&got).Error)
	require.Equal(t, AuditResultFailure, got.ResultStatus)
	require.Equal(t, 0, got.RowCount)
	require.Equal(t, "query export dataset: boom", got.ErrorMessage)
}

func TestAuditEventDoesNotStoreSecrets(t *testing.T) {
	db := setupAuditDB(t)

	err := PersistExportAuditEvent(context.Background(), db, ExportAuditInput{
		WorkspaceID:  "workspace-secret",
		ActorGroupID: "group-secret",
		Scope: Scope{
			WorkspaceID:     "workspace-secret",
			CallerUserID:    "user-secret",
			AllowedGroupIDs: []string{"group-secret"},
		},
		Filter: Filter{
			StartTimestamp: 1,
			EndTimestamp:   2,
			GroupIDs:       []string{"group-secret"},
			UserIDs:        []string{"user-secret"},
			Models:         []string{"sk-test-secret", "gpt-4o"},
		},
		ResultStatus: AuditResultSuccess,
		RowCount:     1,
		Err:          errors.New("upstream Authorization: Bearer top-secret-token failed"),
	})
	require.NoError(t, err)

	var got enterprisemodels.AnalyticsAuditEvent
	require.NoError(t, db.First(&got).Error)
	stored := strings.Join([]string{got.ActorGroupHash, got.ScopeSummary, got.FilterJSON, got.ErrorMessage}, " ")
	require.NotContains(t, stored, "user-secret")
	require.NotContains(t, stored, "group-secret")
	require.NotContains(t, stored, "sk-test-secret")
	require.NotContains(t, stored, "top-secret-token")
	require.NotContains(t, stored, "Bearer")
	require.Contains(t, got.FilterJSON, `"model_count":2`)
	require.Contains(t, got.ErrorMessage, "[redacted]")
}

func TestExportDatasetUsesServiceScopedFilters(t *testing.T) {
	db, logDB := setupServiceDBs(t)
	seedServiceOrg(t, db)
	seedGroupSummary(t, logDB, "group-a", "gpt-4o", 2.5, 2)
	seedGroupSummary(t, logDB, "group-b", "blocked-model", 9.0, 9)

	dataset, err := BuildExportDataset(context.Background(), newTestService(db, logDB), Scope{
		WorkspaceID:     "default",
		AllowedGroupIDs: []string{"group-a"},
		AllowedModels:   []string{"gpt-4o"},
	}, Filter{
		StartTimestamp: 1,
		EndTimestamp:   7200,
		GroupIDs:       []string{"group-b"},
		Models:         []string{"blocked-model"},
	})
	require.NoError(t, err)
	require.Empty(t, dataset.DepartmentSummaries)
	require.Empty(t, dataset.UserRanking)
	require.Empty(t, dataset.ModelDistribution)
	require.Zero(t, dataset.DepartmentSummaryCount)
	require.Zero(t, dataset.UserRankingCount)
	require.Zero(t, dataset.ModelDistributionCount)

	dataset, err = BuildExportDataset(context.Background(), newTestService(db, logDB), Scope{
		WorkspaceID:     "default",
		AllowedGroupIDs: []string{"group-a"},
		AllowedModels:   []string{"gpt-4o"},
	}, Filter{StartTimestamp: 1, EndTimestamp: 7200})
	require.NoError(t, err)
	require.Len(t, dataset.DepartmentSummaries, 1)
	require.Len(t, dataset.UserRanking, 1)
	require.Len(t, dataset.ModelDistribution, 1)
	require.Equal(t, 1, dataset.DepartmentSummaryCount)
	require.Equal(t, 1, dataset.UserRankingCount)
	require.Equal(t, 1, dataset.ModelDistributionCount)
	require.Equal(t, 3, dataset.TotalRows)
	require.Equal(t, "dept-a", dataset.DepartmentSummaries[0].DepartmentID)
	require.Equal(t, "group-a", dataset.UserRanking[0].GroupID)
	require.Equal(t, "gpt-4o", dataset.ModelDistribution[0].Model)
}
