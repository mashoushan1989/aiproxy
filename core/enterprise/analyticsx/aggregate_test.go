//go:build enterprise

package analyticsx

import (
	"context"
	"testing"

	"github.com/labring/aiproxy/core/model"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupAggregateDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := model.OpenSQLite(t.TempDir() + "/aggregate.db")
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.GroupSummary{}))
	require.NoError(t, AutoMigrateAggregates(db))

	return db
}

func seedAggregateGroupSummary(
	t *testing.T,
	db *gorm.DB,
	groupID string,
	tokenName string,
	modelName string,
	hourTimestamp int64,
	requests int64,
	usedAmount float64,
) {
	t.Helper()

	require.NoError(t, db.Create(&model.GroupSummary{
		Unique: model.GroupSummaryUnique{
			GroupID:       groupID,
			TokenName:     tokenName,
			Model:         modelName,
			HourTimestamp: hourTimestamp,
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
				Amount: model.Amount{UsedAmount: usedAmount},
			},
		},
	}).Error)
}

func readHourlyAggregate(t *testing.T, db *gorm.DB, groupID, tokenName, modelName string, hourTimestamp int64) EnterpriseAnalyticsxHourly {
	t.Helper()

	var got EnterpriseAnalyticsxHourly
	require.NoError(t, db.Where(
		"group_id = ? AND token_name = ? AND model = ? AND hour_timestamp = ?",
		groupID,
		tokenName,
		modelName,
		hourTimestamp,
	).First(&got).Error)

	return got
}

func readDailyAggregate(t *testing.T, db *gorm.DB, groupID, tokenName, modelName string, dayTimestamp int64) EnterpriseAnalyticsxDaily {
	t.Helper()

	var got EnterpriseAnalyticsxDaily
	require.NoError(t, db.Where(
		"group_id = ? AND token_name = ? AND model = ? AND day_timestamp = ?",
		groupID,
		tokenName,
		modelName,
		dayTimestamp,
	).First(&got).Error)

	return got
}

func readMonthlyAggregate(t *testing.T, db *gorm.DB, groupID, tokenName, modelName string, monthTimestamp int64) EnterpriseAnalyticsxMonthly {
	t.Helper()

	var got EnterpriseAnalyticsxMonthly
	require.NoError(t, db.Where(
		"group_id = ? AND token_name = ? AND model = ? AND month_timestamp = ?",
		groupID,
		tokenName,
		modelName,
		monthTimestamp,
	).First(&got).Error)

	return got
}

func TestAggregateHourlyIsIdempotent(t *testing.T) {
	db := setupAggregateDB(t)
	seedAggregateGroupSummary(t, db, "group-a", "token-a", "gpt-4o", 3600, 2, 1.5)

	require.NoError(t, AggregateHourly(context.Background(), db, 0, 7200))
	require.NoError(t, AggregateHourly(context.Background(), db, 0, 7200))

	var count int64
	require.NoError(t, db.Model(&EnterpriseAnalyticsxHourly{}).Count(&count).Error)
	require.Equal(t, int64(1), count)

	got := readHourlyAggregate(t, db, "group-a", "token-a", "gpt-4o", 3600)
	require.Equal(t, int64(2), got.Data.RequestCount)
	require.Equal(t, model.ZeroNullInt64(60), got.Data.TotalTokens)
	require.InDelta(t, 1.5, got.Data.UsedAmount, 0.0001)
}

func TestAggregateDailyRollsUpHourlyRows(t *testing.T) {
	db := setupAggregateDB(t)
	seedAggregateGroupSummary(t, db, "group-a", "token-a", "gpt-4o", 3600, 2, 1.5)
	seedAggregateGroupSummary(t, db, "group-a", "token-a", "gpt-4o", 7200, 3, 2.25)

	require.NoError(t, AggregateHourly(context.Background(), db, 0, 86400))
	require.NoError(t, AggregateDaily(context.Background(), db, 0, 86400))
	require.NoError(t, AggregateDaily(context.Background(), db, 0, 86400))

	var count int64
	require.NoError(t, db.Model(&EnterpriseAnalyticsxDaily{}).Count(&count).Error)
	require.Equal(t, int64(1), count)

	got := readDailyAggregate(t, db, "group-a", "token-a", "gpt-4o", 0)
	require.Equal(t, int64(5), got.Data.RequestCount)
	require.Equal(t, model.ZeroNullInt64(150), got.Data.TotalTokens)
	require.InDelta(t, 3.75, got.Data.UsedAmount, 0.0001)
}

func TestAggregateMonthlyRollsUpDailyRows(t *testing.T) {
	db := setupAggregateDB(t)
	seedAggregateGroupSummary(t, db, "group-a", "token-a", "gpt-4o", 3600, 2, 1.5)
	seedAggregateGroupSummary(t, db, "group-a", "token-a", "gpt-4o", 86400, 3, 2.25)

	require.NoError(t, AggregateHourly(context.Background(), db, 0, 2678400))
	require.NoError(t, AggregateDaily(context.Background(), db, 0, 2678400))
	require.NoError(t, AggregateMonthly(context.Background(), db, 0, 2678400))
	require.NoError(t, AggregateMonthly(context.Background(), db, 0, 2678400))

	var count int64
	require.NoError(t, db.Model(&EnterpriseAnalyticsxMonthly{}).Count(&count).Error)
	require.Equal(t, int64(1), count)

	got := readMonthlyAggregate(t, db, "group-a", "token-a", "gpt-4o", 0)
	require.Equal(t, int64(5), got.Data.RequestCount)
	require.Equal(t, model.ZeroNullInt64(150), got.Data.TotalTokens)
	require.InDelta(t, 3.75, got.Data.UsedAmount, 0.0001)
}

func TestAggregateDoesNotModifyGroupSummary(t *testing.T) {
	db := setupAggregateDB(t)
	seedAggregateGroupSummary(t, db, "group-a", "token-a", "gpt-4o", 3600, 2, 1.5)
	seedAggregateGroupSummary(t, db, "group-b", "token-b", "claude-3-5", 7200, 4, 3.5)

	var before []model.GroupSummary
	require.NoError(t, db.Order("id").Find(&before).Error)

	require.NoError(t, AggregateHourly(context.Background(), db, 0, 86400))
	require.NoError(t, AggregateDaily(context.Background(), db, 0, 86400))
	require.NoError(t, AggregateMonthly(context.Background(), db, 0, 2678400))

	var after []model.GroupSummary
	require.NoError(t, db.Order("id").Find(&after).Error)
	require.Equal(t, before, after)
}
