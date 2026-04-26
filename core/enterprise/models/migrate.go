//go:build enterprise

package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/labring/aiproxy/core/common"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// PPIOSyncHistory mirrors ppio.SyncHistory for migration purposes.
// Defined here to avoid circular dependency (ppio → model → models → ppio).
type PPIOSyncHistory struct {
	ID          int64     `gorm:"primaryKey"`
	SyncedAt    time.Time `gorm:"autoCreateTime;index"`
	Operator    string
	SyncOptions string
	Result      string
	Status      string
	CreatedAt   time.Time `gorm:"autoCreateTime"`
}

func (PPIOSyncHistory) TableName() string {
	return "ppio_sync_history"
}

// NovitaSyncHistory mirrors novita.SyncHistory for migration purposes.
// Defined here to avoid circular dependency (novita → model → models → novita).
type NovitaSyncHistory struct {
	ID          int64     `gorm:"primaryKey"`
	SyncedAt    time.Time `gorm:"autoCreateTime;index"`
	Operator    string
	SyncOptions string
	Result      string
	Status      string
	CreatedAt   time.Time `gorm:"autoCreateTime"`
}

func (NovitaSyncHistory) TableName() string {
	return "novita_sync_history"
}

// EnterpriseAutoMigrate runs database migrations for all enterprise tables.
func EnterpriseAutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&FeishuUser{},
		&FeishuDepartment{},
		&QuotaPolicy{},
		&GroupQuotaPolicy{},
		&TenantWhitelist{},
		&TenantWhitelistConfig{},
		&DepartmentQuotaPolicy{},
		&UserQuotaPolicy{},
		&PPIOSyncHistory{},
		&NovitaSyncHistory{},
		&FeishuSyncHistory{},
		&RejectedTenantLogin{},
		&RolePermission{},
		&QuotaAlertHistory{},
		&ReportTemplate{},
	); err != nil {
		return err
	}

	// Migrate unique indexes to partial indexes (soft-delete compatible)
	migratePartialUniqueIndexes(db)

	// Seed default role permissions if table is empty
	seedDefaultRolePermissions(db)

	// Migrate V1 (single key) → V2 (view/manage split)
	migratePermissionsV2(db)

	// Tag legacy model_configs rows with synced_from based on Owner.
	// One-shot bootstrap so the new sync ownership rules can manage them.
	migrateModelConfigSyncedFrom(db)

	return nil
}

// channelModels is a narrow projection of channels.models used only by the
// synced_from backfill below. Declared as a named type (not anonymous) so
// GORM's `serializer:fastjson` tag is reliably picked up on Scan — anonymous
// struct fields can be skipped by the GORM tag scanner in some versions.
type channelModels struct {
	Models []string `gorm:"serializer:fastjson;type:text"`
}

// migrateModelConfigSyncedFrom assigns the synced_from tag to rows that
// pre-date the field. Strategy: rows with owner='ppio'/'novita' AND whose
// model name appears in some active channel.Models are claimed by the
// corresponding sync. Manually-added alias rows (e.g. claude-opus-4-6 used
// only via channel.model_mapping) and virtual rows (web-search/tavily) keep
// synced_from='' — sync MUST NOT touch them per the SyncedFrom contract.
//
// Idempotent: only touches rows where synced_from is empty/NULL.
//
// Cross-dialect: filters routed-models in Go rather than via PG-specific
// jsonb_array_elements_text so SQLite tests behave identically to production.
func migrateModelConfigSyncedFrom(db *gorm.DB) {
	// Build the set of models actually referenced by an active channel.
	// We only need the Models field; selecting that column keeps the read cheap.
	var channels []channelModels
	if err := db.Table("channels").
		Where("status = ?", 1).
		Select("models").
		Scan(&channels).Error; err != nil {
		log.Errorf("failed to read channels for synced_from backfill: %v", err)
		return
	}

	if len(channels) == 0 {
		// First-time install with no channels yet — nothing routable to claim.
		return
	}

	seen := make(map[string]struct{})
	for _, c := range channels {
		for _, m := range c.Models {
			seen[m] = struct{}{}
		}
	}

	// Surface enough info for ops to verify the projection worked. If channels
	// existed but Models was empty (fastjson misread), we'd skip the UPDATE
	// silently — this log makes the case observable.
	log.Infof(
		"synced_from backfill: %d active channels, %d distinct routed models",
		len(channels),
		len(seen),
	)

	if len(seen) == 0 {
		return
	}

	routedModels := make([]string, 0, len(seen))
	for m := range seen {
		routedModels = append(routedModels, m)
	}

	// Owner / SyncedFrom values intentionally inlined as string literals: the
	// natural sources (model.ModelOwner* and synccommon.SyncedFrom*) live in
	// packages that already depend on this one, so importing them here would
	// create a cycle. Keep these in sync with:
	//   model.ModelOwnerPPIO   = "ppio"   / synccommon.SyncedFromPPIO   = "ppio"
	//   model.ModelOwnerNovita = "novita" / synccommon.SyncedFromNovita = "novita"
	updates := []struct {
		owner, syncedFrom string
	}{
		{"ppio", "ppio"},
		{"novita", "novita"},
	}

	for _, u := range updates {
		res := db.Exec(
			"UPDATE model_configs SET synced_from = ? "+
				"WHERE owner = ? AND (synced_from IS NULL OR synced_from = '') "+
				"AND model IN ?",
			u.syncedFrom, u.owner, routedModels,
		)
		if res.Error != nil {
			log.Errorf("failed to backfill model_configs.synced_from for owner=%s: %v",
				u.owner, res.Error)
			continue
		}

		if res.RowsAffected > 0 {
			log.Infof("backfilled synced_from='%s' for %d model_configs rows",
				u.syncedFrom, res.RowsAffected)
		}
	}
}

// seedDefaultRolePermissions inserts default permissions if the table is empty.
func seedDefaultRolePermissions(db *gorm.DB) {
	var count int64
	db.Model(&RolePermission{}).Count(&count)

	if count > 0 {
		return
	}

	var records []RolePermission
	for role, perms := range DefaultRolePermissions {
		for _, perm := range perms {
			records = append(records, RolePermission{
				Role:       role,
				Permission: perm,
			})
		}
	}

	if err := db.Create(&records).Error; err != nil {
		log.Errorf("failed to seed default role permissions: %v", err)
		return
	}

	log.Infof("seeded %d default role permissions", len(records))
}

// migratePermissionsV2 converts old single-key permissions (e.g. "dashboard")
// to the new view/manage split (e.g. "dashboard_view" + "dashboard_manage").
// It only runs when old-format records exist but new-format ones do not.
func migratePermissionsV2(db *gorm.DB) {
	// Check if old format exists
	var oldCount int64
	db.Model(&RolePermission{}).Where("permission = ?", "dashboard").Count(&oldCount)

	if oldCount == 0 {
		return // already migrated or fresh install
	}

	// Check if new format already exists
	var newCount int64
	db.Model(&RolePermission{}).Where("permission = ?", "dashboard_view").Count(&newCount)

	if newCount > 0 {
		return // already migrated
	}

	log.Info("migrating permissions from V1 to V2 (view/manage split)...")

	var oldPerms []RolePermission
	db.Find(&oldPerms)

	var newRecords []RolePermission
	for _, op := range oldPerms {
		// Each old permission becomes both view and manage
		newRecords = append(newRecords,
			RolePermission{Role: op.Role, Permission: ViewPermission(op.Permission)},
			RolePermission{Role: op.Role, Permission: ManagePermission(op.Permission)},
		)
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		// Delete all old records
		if err := tx.Where("1=1").Delete(&RolePermission{}).Error; err != nil {
			return err
		}

		// Insert new records
		if len(newRecords) > 0 {
			if err := tx.Create(&newRecords).Error; err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		log.Errorf("failed to migrate permissions to V2: %v", err)
		return
	}

	log.Infof(
		"migrated %d old permission records to %d new records",
		len(oldPerms),
		len(newRecords),
	)
}

// migratePartialUniqueIndexes replaces full unique indexes with partial unique indexes
// (WHERE deleted_at IS NULL) on soft-delete tables. This prevents soft-deleted rows
// from blocking inserts of new rows with the same natural key.
//
// Idempotent: checks index definition before acting; safe to run on every startup.
func migratePartialUniqueIndexes(db *gorm.DB) {
	migrations := []struct {
		table    string
		oldIndex string
		column   string
	}{
		{"feishu_users", "idx_feishu_users_open_id", "open_id"},
		{"feishu_departments", "idx_feishu_departments_department_id", "department_id"},
	}

	for _, m := range migrations {
		if isPartialUniqueIndex(db, m.table, m.oldIndex) {
			continue
		}

		// Atomic: drop old + create partial unique in one transaction to avoid indexless window
		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec("DROP INDEX IF EXISTS " + m.oldIndex).Error; err != nil {
				return err
			}

			return tx.Exec(fmt.Sprintf(
				"CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s (%s) WHERE deleted_at IS NULL",
				m.oldIndex, m.table, m.column,
			)).Error
		}); err != nil {
			log.Errorf("failed to migrate index %s on %s: %v", m.oldIndex, m.table, err)
			continue
		}

		log.Infof(
			"migrated index %s on %s to partial unique (WHERE deleted_at IS NULL)",
			m.oldIndex,
			m.table,
		)
	}
}

// isPartialUniqueIndex checks if an index already has a WHERE clause (partial index).
func isPartialUniqueIndex(db *gorm.DB, table, indexName string) bool {
	var (
		indexDef string
		query    string
	)

	if common.UsingSQLite {
		query = "SELECT sql FROM sqlite_master WHERE type = 'index' AND tbl_name = ? AND name = ?"
	} else {
		query = "SELECT indexdef FROM pg_indexes WHERE tablename = ? AND indexname = ?"
	}

	if err := db.Raw(query, table, indexName).Scan(&indexDef).Error; err == nil && indexDef != "" {
		return strings.Contains(strings.ToUpper(indexDef), "WHERE")
	}

	// Index doesn't exist yet — will be created fresh
	return false
}
