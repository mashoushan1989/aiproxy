//go:build enterprise

package analytics

import "testing"

func TestNormalizeCustomReportLimit(t *testing.T) {
	tests := []struct {
		name  string
		limit int
		want  int
	}{
		{name: "default", limit: 0, want: defaultCustomReportLimit},
		{name: "negative", limit: -1, want: defaultCustomReportLimit},
		{name: "keeps requested limit", limit: 2500, want: 2500},
		{name: "clamps to max", limit: maxCustomReportLimit + 1, want: maxCustomReportLimit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeCustomReportLimit(tt.limit); got != tt.want {
				t.Fatalf("normalizeCustomReportLimit(%d) = %d, want %d", tt.limit, got, tt.want)
			}
		})
	}
}
