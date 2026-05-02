package model_test

import (
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
