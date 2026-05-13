//go:build enterprise

package identitysource

import (
	"testing"

	enterprisemodels "github.com/labring/aiproxy/core/enterprise/models"
	"github.com/labring/aiproxy/core/model"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupConfigTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := model.OpenSQLite(t.TempDir() + "/identity_source.db")
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&enterprisemodels.IdentitySource{}))

	return db
}

func TestResolveFeishuConfigFallsBackToEnvWhenDBMissing(t *testing.T) {
	db := setupConfigTestDB(t)
	t.Setenv("FEISHU_APP_ID", "env-app")
	t.Setenv("FEISHU_APP_SECRET", "dummy-env-value")
	t.Setenv("FEISHU_REDIRECT_URI", "https://env.example.com/callback")
	t.Setenv("FEISHU_FRONTEND_URL", "https://env.example.com")
	t.Setenv("FEISHU_ALLOWED_TENANTS", "tenant-a,tenant-b")

	cfg, err := ResolveFeishuConfig(db)

	require.NoError(t, err)
	require.Equal(t, SourceEnv, cfg.Source)
	require.Equal(t, "env-app", cfg.AppID)
	require.Equal(t, "dummy-env-value", cfg.AppSecret)
	require.Equal(t, "https://env.example.com/callback", cfg.RedirectURI)
	require.Equal(t, "https://env.example.com", cfg.FrontendURL)
	require.Equal(t, []string{"tenant-a", "tenant-b"}, cfg.AllowedTenants)
	require.False(t, cfg.DBConfigured)
	require.True(t, cfg.HasSecret)
}

func TestResolveConfigDoesNotUseFeishuEnvForOtherProviders(t *testing.T) {
	db := setupConfigTestDB(t)
	t.Setenv("FEISHU_APP_ID", "env-app")
	t.Setenv("FEISHU_APP_SECRET", "dummy-env-value")
	t.Setenv("FEISHU_REDIRECT_URI", "https://env.example.com/callback")
	t.Setenv("FEISHU_FRONTEND_URL", "https://env.example.com")

	cfg, err := ResolveConfig(db, enterprisemodels.WorkspaceDefaultID, enterprisemodels.ProviderWeCom)

	require.NoError(t, err)
	require.Equal(t, SourceEnv, cfg.Source)
	require.Equal(t, enterprisemodels.ProviderWeCom, cfg.Provider)
	require.Empty(t, cfg.AppID)
	require.Empty(t, cfg.AppSecret)
	require.False(t, cfg.Enabled)
	require.False(t, cfg.HasSecret)
}

func TestResolveFeishuConfigUsesEnabledDBConfig(t *testing.T) {
	db := setupConfigTestDB(t)
	t.Setenv("FEISHU_APP_ID", "env-app")
	t.Setenv("FEISHU_APP_SECRET", "dummy-env-value")

	require.NoError(t, db.Create(&enterprisemodels.IdentitySource{
		WorkspaceID:   enterprisemodels.WorkspaceDefaultID,
		Provider:      enterprisemodels.ProviderFeishu,
		ExternalOrgID: "tenant-db",
		AppID:         "db-app",
		AppSecret:     "dummy-db-value",
		RedirectURI:   "https://db.example.com/callback",
		FrontendURL:   "https://db.example.com",
		SyncEnabled:   true,
		Enabled:       true,
	}).Error)

	cfg, err := ResolveFeishuConfig(db)

	require.NoError(t, err)
	require.Equal(t, SourceDB, cfg.Source)
	require.Equal(t, "tenant-db", cfg.ExternalOrgID)
	require.Equal(t, "db-app", cfg.AppID)
	require.Equal(t, "dummy-db-value", cfg.AppSecret)
	require.Equal(t, "https://db.example.com/callback", cfg.RedirectURI)
	require.Equal(t, "https://db.example.com", cfg.FrontendURL)
	require.True(t, cfg.SyncEnabled)
	require.True(t, cfg.DBConfigured)
	require.True(t, cfg.HasSecret)
}

func TestResolveFeishuConfigFallsBackToEnvWhenDBDisabled(t *testing.T) {
	db := setupConfigTestDB(t)
	t.Setenv("FEISHU_APP_ID", "env-app")
	t.Setenv("FEISHU_APP_SECRET", "dummy-env-value")

	require.NoError(t, db.Create(&enterprisemodels.IdentitySource{
		WorkspaceID: enterprisemodels.WorkspaceDefaultID,
		Provider:    enterprisemodels.ProviderFeishu,
		AppID:       "db-app",
		AppSecret:   "dummy-db-value",
		Enabled:     false,
	}).Error)

	cfg, err := ResolveFeishuConfig(db)

	require.NoError(t, err)
	require.Equal(t, SourceEnv, cfg.Source)
	require.Equal(t, "env-app", cfg.AppID)
	require.Equal(t, "dummy-env-value", cfg.AppSecret)
	require.True(t, cfg.DBConfigured)
}

func TestSaveIdentitySourcePreservesSecretWhenRequestSecretEmpty(t *testing.T) {
	db := setupConfigTestDB(t)
	require.NoError(t, db.Create(&enterprisemodels.IdentitySource{
		WorkspaceID: enterprisemodels.WorkspaceDefaultID,
		Provider:    enterprisemodels.ProviderFeishu,
		AppID:       "old-app",
		AppSecret:   "dummy-old-value",
		Enabled:     true,
	}).Error)

	saved, err := SaveIdentitySource(db, enterprisemodels.ProviderFeishu, UpdateRequest{
		AppID:       "new-app",
		AppSecret:   "",
		RedirectURI: "https://new.example.com/callback",
		Enabled:     true,
	})

	require.NoError(t, err)
	require.Equal(t, "new-app", saved.AppID)
	require.Equal(t, "dummy-old-value", saved.AppSecret)

	cfg, err := ResolveFeishuConfig(db)
	require.NoError(t, err)
	require.Equal(t, "dummy-old-value", cfg.AppSecret)
}

func TestIdentitySourceResponseMasksSecret(t *testing.T) {
	src := enterprisemodels.IdentitySource{
		WorkspaceID: enterprisemodels.WorkspaceDefaultID,
		Provider:    enterprisemodels.ProviderFeishu,
		AppID:       "app-id",
		AppSecret:   "dummy-sensitive-value",
		Enabled:     true,
	}

	resp := NewConfigResponse(src, EffectiveConfig{Source: SourceDB, HasSecret: true, DBConfigured: true})

	require.True(t, resp.HasSecret)
	require.Equal(t, "********", resp.AppSecretMask)
	require.NotContains(t, resp.AppSecretMask, "dummy-sensitive-value")
}
