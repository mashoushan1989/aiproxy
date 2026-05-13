//go:build enterprise

package model

func init() {
	enterpriseLogMigrator = MigrateEnterpriseAnalyticsxAggregates
}
