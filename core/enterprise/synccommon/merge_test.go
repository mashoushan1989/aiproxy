//go:build enterprise

package synccommon

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/model"
)

func setupMergeTestDB(t *testing.T) {
	t.Helper()

	prevDB := model.DB
	prevUsingSQLite := common.UsingSQLite

	testDB, err := model.OpenSQLite(filepath.Join(t.TempDir(), "merge.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	model.DB = testDB
	common.UsingSQLite = true
	t.Cleanup(func() {
		model.DB = prevDB
		common.UsingSQLite = prevUsingSQLite
	})

	if err := testDB.AutoMigrate(&model.ModelConfig{}); err != nil {
		t.Fatalf("migrate model_configs: %v", err)
	}
}

func seedConfig(t *testing.T, modelName, syncedFrom string, missingCount int) {
	t.Helper()

	mc := model.ModelConfig{
		Model:        modelName,
		SyncedFrom:   syncedFrom,
		MissingCount: missingCount,
	}
	if err := model.DB.Create(&mc).Error; err != nil {
		t.Fatalf("seed %s: %v", modelName, err)
	}
}

// (1) Upstream returned, brand new — should be added.
func TestMergeChannelModels_UpstreamNew(t *testing.T) {
	setupMergeTestDB(t)

	got, err := MergeChannelModels(model.DB, SyncedFromPPIO,
		[]string{"new-model"},
		nil)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}

	if !slices.Equal(got, []string{"new-model"}) {
		t.Fatalf("got %v", got)
	}
}

// (2) Upstream returned a model previously owned and aging — missing_count
// reset is the caller's responsibility (during upsert), so merge just adds it.
func TestMergeChannelModels_UpstreamReturnsAgingModel(t *testing.T) {
	setupMergeTestDB(t)

	seedConfig(t, "back-from-the-dead", SyncedFromPPIO, 5)

	got, err := MergeChannelModels(model.DB, SyncedFromPPIO,
		[]string{"back-from-the-dead"},
		[]string{"back-from-the-dead"})
	if err != nil {
		t.Fatalf("merge: %v", err)
	}

	if !slices.Equal(got, []string{"back-from-the-dead"}) {
		t.Fatalf("got %v", got)
	}
}

// (3) Owned by this sync, missed this run, missing_count below threshold —
// stays in channel.Models (transient API hiccup tolerance).
func TestMergeChannelModels_OwnedAgingBelowThreshold(t *testing.T) {
	setupMergeTestDB(t)

	seedConfig(t, "transient-miss", SyncedFromPPIO, SyncMissingThreshold-1)

	got, err := MergeChannelModels(model.DB, SyncedFromPPIO,
		[]string{"upstream-model"},
		[]string{"transient-miss", "upstream-model"})
	if err != nil {
		t.Fatalf("merge: %v", err)
	}

	want := []string{"transient-miss", "upstream-model"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

// (4) Owned by this sync, missed this run, missing_count >= threshold —
// dropped from channel.Models (model_configs row is NOT deleted by merge —
// that's a separate concern).
func TestMergeChannelModels_OwnedAgingAtThreshold(t *testing.T) {
	setupMergeTestDB(t)

	seedConfig(t, "long-gone", SyncedFromPPIO, SyncMissingThreshold)

	got, err := MergeChannelModels(model.DB, SyncedFromPPIO,
		nil,
		[]string{"long-gone"})
	if err != nil {
		t.Fatalf("merge: %v", err)
	}

	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}

// (5) Other sync's row in current channel — preserve untouched.
func TestMergeChannelModels_PreserveOtherSync(t *testing.T) {
	setupMergeTestDB(t)

	seedConfig(t, "novita-model", SyncedFromNovita, 0)

	got, err := MergeChannelModels(model.DB, SyncedFromPPIO,
		[]string{"ppio-model"},
		[]string{"novita-model"})
	if err != nil {
		t.Fatalf("merge: %v", err)
	}

	want := []string{"novita-model", "ppio-model"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

// (6) Non-sync row (NULL/empty SyncedFrom) — preserve forever.
// This protects autodiscover, virtual models, manual admin, yaml-overlay entries.
func TestMergeChannelModels_PreserveNonSync(t *testing.T) {
	setupMergeTestDB(t)

	seedConfig(t, "autodiscover-model", "", 0)
	seedConfig(t, "virtual-web-search", "", 0)

	got, err := MergeChannelModels(model.DB, SyncedFromPPIO,
		[]string{"new-upstream"},
		[]string{"autodiscover-model", "virtual-web-search"})
	if err != nil {
		t.Fatalf("merge: %v", err)
	}

	want := []string{"autodiscover-model", "new-upstream", "virtual-web-search"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

// Edge case: model in channel.Models with no model_config row.
// Should be preserved (defensive — admin or autodiscover may rebuild it).
func TestMergeChannelModels_NoConfigRow(t *testing.T) {
	setupMergeTestDB(t)

	got, err := MergeChannelModels(model.DB, SyncedFromPPIO,
		[]string{"upstream-x"},
		[]string{"orphan-name"})
	if err != nil {
		t.Fatalf("merge: %v", err)
	}

	want := []string{"orphan-name", "upstream-x"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestMarkMissingForSync_IncrementsOnlyOwnedAndMissing(t *testing.T) {
	setupMergeTestDB(t)

	seedConfig(t, "ppio-seen", SyncedFromPPIO, 0)
	seedConfig(t, "ppio-missed", SyncedFromPPIO, 2)
	seedConfig(t, "novita-seen", SyncedFromNovita, 0)
	seedConfig(t, "novita-missed", SyncedFromNovita, 0)
	seedConfig(t, "manual", "", 0)

	if err := MarkMissingForSync(model.DB, SyncedFromPPIO,
		[]string{"ppio-seen"}); err != nil {
		t.Fatalf("mark missing: %v", err)
	}

	check := func(name string, want int) {
		t.Helper()

		var mc model.ModelConfig
		if err := model.DB.Where("model = ?", name).First(&mc).Error; err != nil {
			t.Fatalf("load %s: %v", name, err)
		}

		if mc.MissingCount != want {
			t.Errorf("%s: missing_count = %d, want %d", name, mc.MissingCount, want)
		}
	}

	check("ppio-seen", 0)     // in upstream → untouched
	check("ppio-missed", 3)   // not in upstream, owned → +1
	check("novita-seen", 0)   // not owned by ppio → untouched
	check("novita-missed", 0) // not owned by ppio → untouched
	check("manual", 0)        // not owned by any sync → untouched
}

func TestMarkMissingForSync_EmptyUpstream(t *testing.T) {
	setupMergeTestDB(t)

	seedConfig(t, "ppio-a", SyncedFromPPIO, 0)
	seedConfig(t, "ppio-b", SyncedFromPPIO, 1)

	if err := MarkMissingForSync(model.DB, SyncedFromPPIO, nil); err != nil {
		t.Fatalf("mark missing: %v", err)
	}

	for _, m := range []string{"ppio-a", "ppio-b"} {
		var mc model.ModelConfig
		if err := model.DB.Where("model = ?", m).First(&mc).Error; err != nil {
			t.Fatalf("load %s: %v", m, err)
		}

		if mc.MissingCount == 0 {
			t.Errorf("%s: expected increment, got 0", m)
		}
	}
}

func TestMarkMissingForSync_EmptySyncedFromError(t *testing.T) {
	setupMergeTestDB(t)

	if err := MarkMissingForSync(model.DB, "", []string{"x"}); err == nil {
		t.Fatal("expected error for empty syncedFrom")
	}
}

func TestCanSyncOwn(t *testing.T) {
	cases := []struct {
		existing, mine string
		want           bool
	}{
		{"", SyncedFromPPIO, false}, // unowned → never claim
		{SyncedFromPPIO, SyncedFromPPIO, true},
		{SyncedFromNovita, SyncedFromPPIO, false},
		{SyncedFromPPIO, SyncedFromNovita, false},
		{"", "", false}, // defensive: empty mine never wins
		{SyncedFromPPIO, "", false},
	}
	for _, c := range cases {
		if got := CanSyncOwn(c.existing, c.mine); got != c.want {
			t.Errorf("CanSyncOwn(%q,%q) = %v, want %v", c.existing, c.mine, got, c.want)
		}
	}
}
