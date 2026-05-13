//go:build enterprise

package analyticsx

import "testing"

func TestConfigDefaultsKeepAnalyticsV2Disabled(t *testing.T) {
	t.Setenv("ENTERPRISE_ANALYTICS_V2_ENABLED", "")
	t.Setenv("ENTERPRISE_ANALYTICS_SCOPE_ENFORCED", "")
	t.Setenv("ENTERPRISE_ANALYTICS_AGGREGATES_ENABLED", "")
	t.Setenv("ENTERPRISE_ANALYTICS_ASYNC_EXPORT_ENABLED", "")

	cfg := LoadConfig()

	if cfg.V2Enabled {
		t.Fatal("expected analytics v2 to be disabled by default")
	}
	if cfg.ScopeEnforced {
		t.Fatal("expected scope enforcement to be disabled by default")
	}
	if cfg.AggregatesEnabled {
		t.Fatal("expected aggregates to be disabled by default")
	}
	if cfg.AsyncExportEnabled {
		t.Fatal("expected async export to be disabled by default")
	}
}

func TestConfigReadsAnalyticsV2Enabled(t *testing.T) {
	t.Setenv("ENTERPRISE_ANALYTICS_V2_ENABLED", "true")

	cfg := LoadConfig()

	if !cfg.V2Enabled {
		t.Fatal("expected analytics v2 to be enabled")
	}
}

func TestConfigReadsScopeEnforced(t *testing.T) {
	t.Setenv("ENTERPRISE_ANALYTICS_SCOPE_ENFORCED", "true")

	cfg := LoadConfig()

	if !cfg.ScopeEnforced {
		t.Fatal("expected scope enforcement to be enabled")
	}
}
