//go:build enterprise

package models

import (
	"testing"
	"time"
)

func TestPeriodStartAtCalendarAligned(t *testing.T) {
	loc := time.FixedZone("CST", 8*60*60)
	now := time.Date(2026, time.May, 13, 15, 30, 45, 123, loc) // Wednesday

	tests := []struct {
		name       string
		periodType int
		want       time.Time
	}{
		{
			name:       "daily starts at local midnight",
			periodType: PeriodTypeDaily,
			want:       time.Date(2026, time.May, 13, 0, 0, 0, 0, loc),
		},
		{
			name:       "weekly starts on monday midnight",
			periodType: PeriodTypeWeekly,
			want:       time.Date(2026, time.May, 11, 0, 0, 0, 0, loc),
		},
		{
			name:       "monthly starts on first day midnight",
			periodType: PeriodTypeMonthly,
			want:       time.Date(2026, time.May, 1, 0, 0, 0, 0, loc),
		},
		{
			name:       "unknown defaults to monthly",
			periodType: 0,
			want:       time.Date(2026, time.May, 1, 0, 0, 0, 0, loc),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PeriodStartAt(now, tt.periodType)
			if !got.Equal(tt.want) {
				t.Fatalf("PeriodStartAt() = %s, want %s", got, tt.want)
			}
		})
	}
}
