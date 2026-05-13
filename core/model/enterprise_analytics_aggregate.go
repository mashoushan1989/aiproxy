//go:build enterprise

package model

import (
	"time"

	"gorm.io/gorm"
)

type EnterpriseAnalyticsxHourly struct {
	ID            int         `gorm:"primaryKey"`
	GroupID       string      `gorm:"size:64;not null;uniqueIndex:idx_analyticsx_hourly_unique,priority:1"`
	TokenName     string      `gorm:"size:32;not null;uniqueIndex:idx_analyticsx_hourly_unique,priority:2"`
	Model         string      `gorm:"size:64;not null;uniqueIndex:idx_analyticsx_hourly_unique,priority:3"`
	HourTimestamp int64       `gorm:"not null;uniqueIndex:idx_analyticsx_hourly_unique,priority:4;index"`
	Data          SummaryData `gorm:"embedded"`
	CreatedAt     time.Time   `gorm:"autoCreateTime"`
	UpdatedAt     time.Time   `gorm:"autoUpdateTime"`
}

func (EnterpriseAnalyticsxHourly) TableName() string {
	return "enterprise_analyticsx_hourly"
}

type EnterpriseAnalyticsxDaily struct {
	ID           int         `gorm:"primaryKey"`
	GroupID      string      `gorm:"size:64;not null;uniqueIndex:idx_analyticsx_daily_unique,priority:1"`
	TokenName    string      `gorm:"size:32;not null;uniqueIndex:idx_analyticsx_daily_unique,priority:2"`
	Model        string      `gorm:"size:64;not null;uniqueIndex:idx_analyticsx_daily_unique,priority:3"`
	DayTimestamp int64       `gorm:"not null;uniqueIndex:idx_analyticsx_daily_unique,priority:4;index"`
	Data         SummaryData `gorm:"embedded"`
	CreatedAt    time.Time   `gorm:"autoCreateTime"`
	UpdatedAt    time.Time   `gorm:"autoUpdateTime"`
}

func (EnterpriseAnalyticsxDaily) TableName() string {
	return "enterprise_analyticsx_daily"
}

type EnterpriseAnalyticsxMonthly struct {
	ID             int         `gorm:"primaryKey"`
	GroupID        string      `gorm:"size:64;not null;uniqueIndex:idx_analyticsx_monthly_unique,priority:1"`
	TokenName      string      `gorm:"size:32;not null;uniqueIndex:idx_analyticsx_monthly_unique,priority:2"`
	Model          string      `gorm:"size:64;not null;uniqueIndex:idx_analyticsx_monthly_unique,priority:3"`
	MonthTimestamp int64       `gorm:"not null;uniqueIndex:idx_analyticsx_monthly_unique,priority:4;index"`
	Data           SummaryData `gorm:"embedded"`
	CreatedAt      time.Time   `gorm:"autoCreateTime"`
	UpdatedAt      time.Time   `gorm:"autoUpdateTime"`
}

func (EnterpriseAnalyticsxMonthly) TableName() string {
	return "enterprise_analyticsx_monthly"
}

func MigrateEnterpriseAnalyticsxAggregates(db *gorm.DB) error {
	return db.AutoMigrate(
		&EnterpriseAnalyticsxHourly{},
		&EnterpriseAnalyticsxDaily{},
		&EnterpriseAnalyticsxMonthly{},
	)
}
