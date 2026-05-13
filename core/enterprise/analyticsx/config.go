//go:build enterprise

package analyticsx

import "github.com/labring/aiproxy/core/common/env"

// Config controls enterprise analytics v2 behavior.
type Config struct {
	V2Enabled          bool
	ScopeEnforced      bool
	AggregatesEnabled  bool
	AsyncExportEnabled bool
}

// LoadConfig reads enterprise analytics v2 feature flags from the environment.
func LoadConfig() Config {
	return Config{
		V2Enabled:          env.Bool("ENTERPRISE_ANALYTICS_V2_ENABLED", false),
		ScopeEnforced:      env.Bool("ENTERPRISE_ANALYTICS_SCOPE_ENFORCED", false),
		AggregatesEnabled:  env.Bool("ENTERPRISE_ANALYTICS_AGGREGATES_ENABLED", false),
		AsyncExportEnabled: env.Bool("ENTERPRISE_ANALYTICS_ASYNC_EXPORT_ENABLED", false),
	}
}
