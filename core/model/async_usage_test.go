package model_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/labring/aiproxy/core/model"
)

func TestAsyncUsageBackoffDelay(t *testing.T) {
	tests := []struct {
		name       string
		retryCount int
		want       time.Duration
	}{
		{name: "first retry", retryCount: 1, want: model.AsyncUsageDefaultPollDelay},
		{name: "second retry", retryCount: 2, want: 2 * model.AsyncUsageDefaultPollDelay},
		{name: "caps maximum", retryCount: 20, want: model.AsyncUsageMaxPollDelay},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := model.AsyncUsageBackoffDelay(tt.retryCount); got != tt.want {
				t.Fatalf("expected %s, got %s", tt.want, got)
			}
		})
	}
}

func TestUpdateLogUsageByRequestIDIsIdempotent(t *testing.T) {
	db, err := model.OpenSQLite(filepath.Join(t.TempDir(), "logs.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	prevLogDB := model.LogDB
	model.LogDB = db
	t.Cleanup(func() {
		model.LogDB = prevLogDB
	})

	if err := db.AutoMigrate(&model.Log{}, &model.RequestDetail{}); err != nil {
		t.Fatalf("migrate log db: %v", err)
	}

	now := time.Unix(1777052048, 0)
	if err := model.RecordConsumeLog(
		"req_async_final",
		now,
		now,
		time.Time{},
		now,
		"test-group",
		200,
		1,
		"gpt-5.4",
		2,
		"test-token",
		"/v1/responses",
		"",
		1,
		"127.0.0.1",
		0,
		nil,
		model.Usage{},
		model.Price{},
		model.Amount{},
		"",
		nil,
		"",
		"resp_async_final",
		"default",
		model.AsyncUsageStatusPending,
	); err != nil {
		t.Fatalf("record consume log: %v", err)
	}

	usage := model.Usage{InputTokens: 100, OutputTokens: 50, TotalTokens: 150}
	amount := model.Amount{InputAmount: 0.1, OutputAmount: 0.2, UsedAmount: 0.3}

	if err := model.UpdateLogUsageByRequestID("req_async_final", usage, amount); err != nil {
		t.Fatalf("first update log usage: %v", err)
	}

	if err := model.UpdateLogUsageByRequestID("req_async_final", usage, amount); err != nil {
		t.Fatalf("second update log usage: %v", err)
	}

	var got model.Log
	if err := db.Where("request_id = ?", "req_async_final").First(&got).Error; err != nil {
		t.Fatalf("query log: %v", err)
	}

	if got.Amount.UsedAmount != amount.UsedAmount {
		t.Fatalf("expected used amount %f, got %f", amount.UsedAmount, got.Amount.UsedAmount)
	}

	if got.Usage.TotalTokens != usage.TotalTokens {
		t.Fatalf("expected total tokens %d, got %d", usage.TotalTokens, got.Usage.TotalTokens)
	}

	if got.AsyncUsageStatus != model.AsyncUsageStatusCompleted {
		t.Fatalf("expected async usage status completed, got %d", got.AsyncUsageStatus)
	}
}
