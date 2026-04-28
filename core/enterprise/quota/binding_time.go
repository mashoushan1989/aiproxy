//go:build enterprise

package quota

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

var errBindingExpiredAtInPast = errors.New("expires_at must be in the future")

func activePolicyBindingScope(now time.Time) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.
			Where("(effective_at IS NULL OR effective_at <= ?)", now).
			Where("(expires_at IS NULL OR expires_at > ?)", now)
	}
}

func validateBindingExpiresAt(expiresAt *time.Time) error {
	if expiresAt == nil {
		return nil
	}

	if !expiresAt.After(time.Now()) {
		return errBindingExpiredAtInPast
	}

	return nil
}
