//go:build enterprise

package synccommon

import (
	"errors"
	"fmt"
	"slices"

	"github.com/labring/aiproxy/core/model"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// SyncedFrom tag values. These are the only valid values for ModelConfig.SyncedFrom
// when written by sync code. NULL/empty means "not from sync" (autodiscover, virtual,
// manual admin, yaml overlay).
//
// We use one tag per provider — chat and multimodal sync both write the same
// provider's tag. The lifecycle (missing_count, channel.Models composition) is
// managed per-provider; each ExecuteSync call covers both upstream APIs and
// passes the union of seen models to MarkMissingForSync.
const (
	SyncedFromPPIO   = "ppio"
	SyncedFromNovita = "novita"
)

// SyncMissingThreshold is how many consecutive sync runs may miss a model
// before it gets dropped from channel.Models. The model_configs row itself
// is preserved (admin can manually re-add). 7 ≈ one week of daily sync.
const SyncMissingThreshold = 7

// Advisory-lock keys used by AcquireSyncLock / ReleaseSyncLock. Keep these
// values stable across releases — changing one would let two old/new processes
// hold "different" locks on the same logical sync. Use distinct, large-enough
// constants to avoid collisions with other advisory-lock users in the same DB.
const (
	advisoryLockKeyPPIOSync   int64 = 0x4149505052494F00 // "AIPPPRIO\0"
	advisoryLockKeyNovitaSync int64 = 0x4149504E4F564954 // "AIPNOVIT"
)

// AdvisoryLockKey returns the integer advisory-lock key for a given syncedFrom
// tag. Returns 0 for unknown tags (caller treats 0 as "skip locking").
func AdvisoryLockKey(syncedFrom string) int64 {
	switch syncedFrom {
	case SyncedFromPPIO:
		return advisoryLockKeyPPIOSync
	case SyncedFromNovita:
		return advisoryLockKeyNovitaSync
	}

	return 0
}

// AcquireSyncLock tries to obtain a Postgres advisory lock for a given sync
// across the entire shared database (i.e. across nodes that share PG via the
// WireGuard tunnel). Returns true if locked, false if another node holds it.
// On non-PostgreSQL backends (SQLite test/dev) this is a no-op returning true
// — single-process locking is already enforced by the per-process syncMu.
//
// The lock is automatically released when the session ends, so a crashed
// node won't leave a stale lock. Pair every Acquire with ReleaseSyncLock for
// faster handoff under normal completion.
func AcquireSyncLock(db *gorm.DB, syncedFrom string) (bool, error) {
	key := AdvisoryLockKey(syncedFrom)
	if key == 0 {
		return false, fmt.Errorf("AcquireSyncLock: unknown syncedFrom %q", syncedFrom)
	}

	// SQLite (dev/test) doesn't support advisory locks; rely on syncMu only.
	if db.Name() == "sqlite" {
		return true, nil
	}

	var got bool
	if err := db.Raw("SELECT pg_try_advisory_lock(?)", key).Scan(&got).Error; err != nil {
		return false, fmt.Errorf("AcquireSyncLock(%s): %w", syncedFrom, err)
	}

	return got, nil
}

// ReleaseSyncLock releases a previously-acquired advisory lock. Idempotent;
// safe to call even if the caller didn't actually hold the lock (Postgres
// returns false but does not error).
func ReleaseSyncLock(db *gorm.DB, syncedFrom string) error {
	key := AdvisoryLockKey(syncedFrom)
	if key == 0 {
		return nil
	}

	if db.Name() == "sqlite" {
		return nil
	}

	var ok bool

	return db.Raw("SELECT pg_advisory_unlock(?)", key).Scan(&ok).Error
}

// CanSyncOwn reports whether the sync identified by mySyncedFrom may write
// to a row whose current SyncedFrom value is existing.
//
// Rules:
//   - existing == ""           → SKIP. Empty means non-sync (autodiscover, virtual,
//     manual admin, yaml overlay). Sync MUST NOT touch
//     these rows. They are preserved indefinitely.
//   - existing == mySyncedFrom → write/update freely (own row).
//   - any other value          → SKIP. Owned by a different sync.
//
// Bootstrap note: existing rows that pre-date the synced_from field have
// SyncedFrom = "" but are conceptually owned by their original Owner. The
// data-migration step in the deploy plan tags them (UPDATE … SET synced_from
// = 'ppio'/'novita' WHERE owner = …) so this function never sees an
// "ownerless legacy row" in production.
func CanSyncOwn(existing, mySyncedFrom string) bool {
	return existing == mySyncedFrom && mySyncedFrom != ""
}

// MarkMissingForSync increments missing_count by 1 for all model_configs rows
// owned by syncedFrom but NOT in upstreamModels. Models in upstreamModels are
// reset to missing_count=0 by the caller's upsert path (createModelConfigV2 etc.)
// so this function only handles the "not seen this run" case.
//
// Pass an empty upstreamModels slice for "no models seen" (e.g. upstream API
// failure) — but the caller should generally skip this update on API failure
// to avoid a spurious decay.
func MarkMissingForSync(db *gorm.DB, syncedFrom string, upstreamModels []string) error {
	if syncedFrom == "" {
		return errors.New("MarkMissingForSync: syncedFrom must be non-empty")
	}

	q := db.Model(&model.ModelConfig{}).
		Where("synced_from = ?", syncedFrom)

	if len(upstreamModels) > 0 {
		q = q.Where("model NOT IN ?", upstreamModels)
	}

	return q.UpdateColumn("missing_count", gorm.Expr("missing_count + 1")).Error
}

// bookkeepSeenWarnThreshold caps the seen-list size that BookkeepMissing
// considers "normal". Above this, we emit a warn so ops notice before we
// approach PG's 65535 parameter limit on the underlying `model NOT IN ?` query.
const bookkeepSeenWarnThreshold = 5000

// BookkeepMissing wraps the per-sync missing-count bookkeeping with the safety
// gate both providers need: only run when chat upstream returned data AND
// multimodal either succeeded or wasn't attempted. A failed multimodal API
// would otherwise leave multimodal names absent from the seen union, falsely
// aging owned multimodal rows out of channel.Models after ~7 transient
// failures (SyncMissingThreshold).
//
// Returns the error from MarkMissingForSync when invoked, or nil when gated out.
func BookkeepMissing(
	db *gorm.DB,
	syncedFrom string,
	chatModels, multimodalNames []string,
	multimodalOK bool,
) error {
	if len(chatModels) == 0 || !multimodalOK {
		return nil
	}

	seen := make([]string, 0, len(chatModels)+len(multimodalNames))
	seen = append(seen, chatModels...)
	seen = append(seen, multimodalNames...)

	if len(seen) > bookkeepSeenWarnThreshold {
		log.Warnf(
			"BookkeepMissing: seen list size %d (chat=%d multimodal=%d) for %s —"+
				" approaching PG IN-clause parameter limit (65535);"+
				" batch MarkMissingForSync if this fires repeatedly",
			len(seen),
			len(chatModels),
			len(multimodalNames),
			syncedFrom,
		)
	}

	return MarkMissingForSync(db, syncedFrom, seen)
}

// MergeChannelModels composes the new value for channel.Models such that:
//
//  1. All upstream models for this channel are present (latest truth).
//  2. Models owned by this sync (synced_from == syncedFrom) that the upstream
//     missed transiently — but with missing_count < SyncMissingThreshold —
//     are kept (resilience to API hiccup).
//  3. Models currently in channel.Models that are NOT owned by this sync
//     (synced_from is empty or different) are preserved untouched. This is
//     how autodiscover, virtual model injection, manual admin entries, and
//     other syncs' models survive a sync run.
//
// Models owned by this sync that have aged past SyncMissingThreshold are
// dropped from the returned list (their model_configs row stays for manual
// recovery; admin can re-trigger discovery or add them back).
//
// upstreamSubsetForChannel is the slice of upstream-returned models that
// belong in *this* channel type (e.g. anthropicModels for the Anthropic
// channel, openaiModels for the OpenAI channel). Caller is responsible for
// the channel-type filtering.
func MergeChannelModels(
	db *gorm.DB,
	syncedFrom string,
	upstreamSubsetForChannel []string,
	currentChannelModels []string,
) ([]string, error) {
	if syncedFrom == "" {
		return nil, errors.New("MergeChannelModels: syncedFrom must be non-empty")
	}

	upstreamSet := make(map[string]struct{}, len(upstreamSubsetForChannel))
	for _, m := range upstreamSubsetForChannel {
		upstreamSet[m] = struct{}{}
	}

	// Look up SyncedFrom + MissingCount for every model currently in this channel
	// (one query, scoped to the channel's existing list — bounded by channel size).
	configBySf := make(map[string]string, len(currentChannelModels))

	configByMc := make(map[string]int, len(currentChannelModels))
	if len(currentChannelModels) > 0 {
		var rows []model.ModelConfig
		if err := db.Select("model, synced_from, missing_count").
			Where("model IN ?", currentChannelModels).
			Find(&rows).Error; err != nil {
			return nil, fmt.Errorf("MergeChannelModels: load current configs: %w", err)
		}

		for _, r := range rows {
			configBySf[r.Model] = r.SyncedFrom
			configByMc[r.Model] = r.MissingCount
		}
	}

	result := make(map[string]struct{}, len(upstreamSubsetForChannel)+len(currentChannelModels))

	// (1) upstream truth.
	for _, m := range upstreamSubsetForChannel {
		result[m] = struct{}{}
	}

	// (2) + (3) examine current entries.
	for _, m := range currentChannelModels {
		if _, inUpstream := upstreamSet[m]; inUpstream {
			continue // already in result
		}

		sf, hasConfig := configBySf[m]
		if !hasConfig {
			// No model_config row at all (uncommon: stale entry, or row was deleted
			// out-of-band). Preserve to avoid surprising data loss; a follow-up admin
			// or autodiscover run can clean it up.
			result[m] = struct{}{}
			continue
		}

		if sf == syncedFrom {
			// Owned by this sync but not in upstream this run — keep iff aging<threshold.
			if configByMc[m] < SyncMissingThreshold {
				result[m] = struct{}{}
			}
			continue
		}

		// sf != syncedFrom (empty or another sync) — preserve untouched.
		result[m] = struct{}{}
	}

	out := make([]string, 0, len(result))
	for m := range result {
		out = append(out, m)
	}

	slices.Sort(out)

	return out, nil
}
