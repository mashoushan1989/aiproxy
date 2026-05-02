package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/labring/aiproxy/core/common"
	"gorm.io/gorm"
)

func TestOptimizeLogSkipsExplicitVacuumForNonSQLite(t *testing.T) {
	prevUsingSQLite := common.UsingSQLite
	prevLogDB := LogDB
	t.Cleanup(func() {
		common.UsingSQLite = prevUsingSQLite
		LogDB = prevLogDB
	})

	common.UsingSQLite = false
	LogDB = nil

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("optimizeLog should not touch LogDB for non-SQLite databases, panicked: %v", r)
		}
	}()

	if err := optimizeLog(); err != nil {
		t.Fatalf("expected non-SQLite optimizeLog to be a no-op, got %v", err)
	}
}

func TestOptimizeLogKeepsSQLiteVacuum(t *testing.T) {
	prevUsingSQLite := common.UsingSQLite
	prevLogDB := LogDB
	t.Cleanup(func() {
		common.UsingSQLite = prevUsingSQLite
		LogDB = prevLogDB
	})

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite database: %v", err)
	}

	common.UsingSQLite = true
	LogDB = db

	if err := optimizeLog(); err != nil {
		t.Fatalf("expected SQLite VACUUM to succeed, got %v", err)
	}
}
