//go:build enterprise

package analyticsx

import (
	"context"
	"fmt"
	"time"

	"github.com/labring/aiproxy/core/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type EnterpriseAnalyticsxHourly = model.EnterpriseAnalyticsxHourly
type EnterpriseAnalyticsxDaily = model.EnterpriseAnalyticsxDaily
type EnterpriseAnalyticsxMonthly = model.EnterpriseAnalyticsxMonthly

func AutoMigrateAggregates(db *gorm.DB) error {
	return model.MigrateEnterpriseAnalyticsxAggregates(db)
}

var aggregateUpdateColumns = append(summaryDataColumns(), "updated_at")

func summaryDataColumns() []string {
	prefixes := []string{"", "service_tier_flex_", "service_tier_priority_", "claude_long_context_"}
	columns := make([]string, 0, len(prefixes)*37)
	for _, prefix := range prefixes {
		columns = append(columns,
			prefix+"request_count",
			prefix+"retry_count",
			prefix+"exception_count",
			prefix+"status2xx_count",
			prefix+"status4xx_count",
			prefix+"status5xx_count",
			prefix+"status_other_count",
			prefix+"status400_count",
			prefix+"status429_count",
			prefix+"status500_count",
			prefix+"cache_hit_count",
			prefix+"cache_creation_count",
			prefix+"input_tokens",
			prefix+"image_input_tokens",
			prefix+"audio_input_tokens",
			prefix+"output_tokens",
			prefix+"image_output_tokens",
			prefix+"cached_tokens",
			prefix+"cache_creation_tokens",
			prefix+"reasoning_tokens",
			prefix+"total_tokens",
			prefix+"web_search_count",
			prefix+"input_amount",
			prefix+"image_input_amount",
			prefix+"audio_input_amount",
			prefix+"output_amount",
			prefix+"image_output_amount",
			prefix+"reasoning_amount",
			prefix+"cached_amount",
			prefix+"cache_creation_amount",
			prefix+"web_search_amount",
			prefix+"used_amount",
			prefix+"total_time_milliseconds",
			prefix+"total_ttfb_milliseconds",
		)
	}
	return columns
}

func AggregateHourly(ctx context.Context, db *gorm.DB, startTimestamp, endTimestamp int64) error {
	var summaries []model.GroupSummary
	if err := db.WithContext(ctx).
		Model(&model.GroupSummary{}).
		Where("hour_timestamp >= ? AND hour_timestamp <= ?", startTimestamp, endTimestamp).
		Find(&summaries).Error; err != nil {
		return fmt.Errorf("aggregate hourly source: %w", err)
	}

	rows := make(map[aggregateKey]*EnterpriseAnalyticsxHourly)
	for _, summary := range summaries {
		key := aggregateKey{
			groupID:   summary.Unique.GroupID,
			tokenName: summary.Unique.TokenName,
			model:     summary.Unique.Model,
			timestamp: summary.Unique.HourTimestamp,
		}
		row := rows[key]
		if row == nil {
			row = &EnterpriseAnalyticsxHourly{
				GroupID:       summary.Unique.GroupID,
				TokenName:     summary.Unique.TokenName,
				Model:         summary.Unique.Model,
				HourTimestamp: summary.Unique.HourTimestamp,
			}
			rows[key] = row
		}
		row.Data.Add(summary.Data)
	}

	for _, row := range rows {
		if err := upsertAggregate(ctx, db, &row, []clause.Column{
			{Name: "group_id"},
			{Name: "token_name"},
			{Name: "model"},
			{Name: "hour_timestamp"},
		}); err != nil {
			return fmt.Errorf("upsert hourly aggregate: %w", err)
		}
	}

	return nil
}

func AggregateDaily(ctx context.Context, db *gorm.DB, startTimestamp, endTimestamp int64) error {
	var rows []EnterpriseAnalyticsxHourly
	if err := db.WithContext(ctx).
		Where("hour_timestamp >= ? AND hour_timestamp <= ?", startTimestamp, endTimestamp).
		Find(&rows).Error; err != nil {
		return fmt.Errorf("query hourly aggregates: %w", err)
	}

	daily := make(map[aggregateKey]*EnterpriseAnalyticsxDaily)
	for _, row := range rows {
		key := aggregateKey{
			groupID:   row.GroupID,
			tokenName: row.TokenName,
			model:     row.Model,
			timestamp: dayStart(row.HourTimestamp),
		}
		item := daily[key]
		if item == nil {
			item = &EnterpriseAnalyticsxDaily{
				GroupID:      row.GroupID,
				TokenName:    row.TokenName,
				Model:        row.Model,
				DayTimestamp: key.timestamp,
			}
			daily[key] = item
		}
		addHourlyToDaily(item, row)
	}

	for _, row := range daily {
		if err := upsertAggregate(ctx, db, row, []clause.Column{
			{Name: "group_id"},
			{Name: "token_name"},
			{Name: "model"},
			{Name: "day_timestamp"},
		}); err != nil {
			return fmt.Errorf("upsert daily aggregate: %w", err)
		}
	}

	return nil
}

func AggregateMonthly(ctx context.Context, db *gorm.DB, startTimestamp, endTimestamp int64) error {
	var rows []EnterpriseAnalyticsxDaily
	if err := db.WithContext(ctx).
		Where("day_timestamp >= ? AND day_timestamp <= ?", startTimestamp, endTimestamp).
		Find(&rows).Error; err != nil {
		return fmt.Errorf("query daily aggregates: %w", err)
	}

	monthly := make(map[aggregateKey]*EnterpriseAnalyticsxMonthly)
	for _, row := range rows {
		key := aggregateKey{
			groupID:   row.GroupID,
			tokenName: row.TokenName,
			model:     row.Model,
			timestamp: monthStart(row.DayTimestamp),
		}
		item := monthly[key]
		if item == nil {
			item = &EnterpriseAnalyticsxMonthly{
				GroupID:        row.GroupID,
				TokenName:      row.TokenName,
				Model:          row.Model,
				MonthTimestamp: key.timestamp,
			}
			monthly[key] = item
		}
		addDailyToMonthly(item, row)
	}

	for _, row := range monthly {
		if err := upsertAggregate(ctx, db, row, []clause.Column{
			{Name: "group_id"},
			{Name: "token_name"},
			{Name: "model"},
			{Name: "month_timestamp"},
		}); err != nil {
			return fmt.Errorf("upsert monthly aggregate: %w", err)
		}
	}

	return nil
}

type aggregateKey struct {
	groupID   string
	tokenName string
	model     string
	timestamp int64
}

func upsertAggregate(ctx context.Context, db *gorm.DB, value any, columns []clause.Column) error {
	return db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   columns,
			DoUpdates: clause.AssignmentColumns(aggregateUpdateColumns),
		}).
		Create(value).Error
}

func addHourlyToDaily(target *EnterpriseAnalyticsxDaily, source EnterpriseAnalyticsxHourly) {
	target.Data.Add(source.Data)
}

func addDailyToMonthly(target *EnterpriseAnalyticsxMonthly, source EnterpriseAnalyticsxDaily) {
	target.Data.Add(source.Data)
}

func dayStart(timestamp int64) int64 {
	return timestamp - timestamp%int64(24*time.Hour/time.Second)
}

func monthStart(timestamp int64) int64 {
	t := time.Unix(timestamp, 0).UTC()
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC).Unix()
}
