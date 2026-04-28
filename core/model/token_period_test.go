package model

import (
	"testing"
	"time"
)

func TestTokenPeriodStartCalendarAligned(t *testing.T) {
	loc := time.FixedZone("CST", 8*60*60)
	now := time.Date(2026, time.May, 13, 15, 30, 45, 123, loc) // Wednesday

	tests := []struct {
		name       string
		periodType EmptyNullString
		want       time.Time
	}{
		{
			name:       "monthly starts on first day midnight",
			periodType: EmptyNullString(PeriodTypeMonthly),
			want:       time.Date(2026, time.May, 1, 0, 0, 0, 0, loc),
		},
		{
			name:       "empty defaults to monthly",
			periodType: "",
			want:       time.Date(2026, time.May, 1, 0, 0, 0, 0, loc),
		},
		{
			name:       "weekly starts on monday midnight",
			periodType: EmptyNullString(PeriodTypeWeekly),
			want:       time.Date(2026, time.May, 11, 0, 0, 0, 0, loc),
		},
		{
			name:       "daily starts on current day midnight",
			periodType: EmptyNullString(PeriodTypeDaily),
			want:       time.Date(2026, time.May, 13, 0, 0, 0, 0, loc),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tokenPeriodStart(now, tt.periodType)
			if err != nil {
				t.Fatalf("tokenPeriodStart() error = %v", err)
			}

			if !got.Equal(tt.want) {
				t.Fatalf("tokenPeriodStart() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestTokenNeedsPeriodResetUsesCalendarBoundary(t *testing.T) {
	now := time.Now()
	currentWeeklyStart := mustCurrentTokenPeriodStart(t, EmptyNullString(PeriodTypeWeekly))
	currentDailyStart := mustCurrentTokenPeriodStart(t, EmptyNullString(PeriodTypeDaily))

	tests := []struct {
		name       string
		periodType EmptyNullString
		lastUpdate time.Time
		want       bool
	}{
		{
			name:       "monthly resets after natural month boundary",
			periodType: EmptyNullString(PeriodTypeMonthly),
			lastUpdate: time.Date(now.Year(), now.Month()-1, 15, 12, 0, 0, 0, now.Location()),
			want:       true,
		},
		{
			name:       "monthly does not reset inside current natural month",
			periodType: EmptyNullString(PeriodTypeMonthly),
			lastUpdate: time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()),
			want:       false,
		},
		{
			name:       "weekly resets after monday boundary even before seven days",
			periodType: EmptyNullString(PeriodTypeWeekly),
			lastUpdate: currentWeeklyStart.Add(-24 * time.Hour),
			want:       true,
		},
		{
			name:       "daily resets after midnight boundary",
			periodType: EmptyNullString(PeriodTypeDaily),
			lastUpdate: currentDailyStart.Add(-time.Minute),
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := Token{
				PeriodType:           tt.periodType,
				PeriodLastUpdateTime: tt.lastUpdate,
			}

			got, err := token.NeedsPeriodReset()
			if err != nil {
				t.Fatalf("NeedsPeriodReset() error = %v", err)
			}

			if got != tt.want {
				t.Fatalf("NeedsPeriodReset() = %v, want %v", got, tt.want)
			}
		})
	}
}

func mustCurrentTokenPeriodStart(t *testing.T, periodType EmptyNullString) time.Time {
	t.Helper()

	start, err := currentTokenPeriodStart(periodType)
	if err != nil {
		t.Fatalf("currentTokenPeriodStart() error = %v", err)
	}

	return start
}
