//go:build enterprise

package identitysource

import (
	"errors"
	"os"
	"strings"

	enterprisemodels "github.com/labring/aiproxy/core/enterprise/models"
	"gorm.io/gorm"
)

const (
	SourceDB  = "db"
	SourceEnv = "env"
)

// EffectiveConfig is the resolved identity provider configuration used by
// diagnostics and future runtime cutovers.
type EffectiveConfig struct {
	WorkspaceID    string
	Provider       string
	Source         string
	ExternalOrgID  string
	AppID          string
	AppSecret      string
	RedirectURI    string
	FrontendURL    string
	AllowedTenants []string
	SyncEnabled    bool
	Enabled        bool
	DBConfigured   bool
	HasSecret      bool
}

type UpdateRequest struct {
	ExternalOrgID string `json:"external_org_id"`
	AppID         string `json:"app_id"`
	AppSecret     string `json:"app_secret"`
	RedirectURI   string `json:"redirect_uri"`
	FrontendURL   string `json:"frontend_url"`
	SyncEnabled   bool   `json:"sync_enabled"`
	Enabled       bool   `json:"enabled"`
}

type ConfigResponse struct {
	ID              int    `json:"id"`
	WorkspaceID     string `json:"workspace_id"`
	Provider        string `json:"provider"`
	ExternalOrgID   string `json:"external_org_id"`
	AppID           string `json:"app_id"`
	AppSecretMask   string `json:"app_secret_mask"`
	RedirectURI     string `json:"redirect_uri"`
	FrontendURL     string `json:"frontend_url"`
	SyncEnabled     bool   `json:"sync_enabled"`
	Enabled         bool   `json:"enabled"`
	EffectiveSource string `json:"effective_source"`
	DBConfigured    bool   `json:"db_configured"`
	HasSecret       bool   `json:"has_secret"`
	LastCheckStatus string `json:"last_check_status"`
	LastCheckResult string `json:"last_check_result,omitempty"`
}

func ResolveFeishuConfig(db *gorm.DB) (EffectiveConfig, error) {
	return ResolveConfig(db, enterprisemodels.WorkspaceDefaultID, enterprisemodels.ProviderFeishu)
}

func ResolveConfig(db *gorm.DB, workspaceID string, provider string) (EffectiveConfig, error) {
	if workspaceID == "" {
		workspaceID = enterprisemodels.WorkspaceDefaultID
	}

	env := envConfig(workspaceID, provider)

	var src enterprisemodels.IdentitySource
	err := db.Where("workspace_id = ? AND provider = ?", workspaceID, provider).First(&src).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return env, nil
	}
	if err != nil {
		return EffectiveConfig{}, err
	}

	env.DBConfigured = true
	if !src.Enabled || src.AppID == "" || src.AppSecret == "" {
		return env, nil
	}

	return EffectiveConfig{
		WorkspaceID:   src.WorkspaceID,
		Provider:      src.Provider,
		Source:        SourceDB,
		ExternalOrgID: src.ExternalOrgID,
		AppID:         src.AppID,
		AppSecret:     src.AppSecret,
		RedirectURI:   src.RedirectURI,
		FrontendURL:   src.FrontendURL,
		SyncEnabled:   src.SyncEnabled,
		Enabled:       src.Enabled,
		DBConfigured:  true,
		HasSecret:     src.AppSecret != "",
	}, nil
}

func GetIdentitySource(db *gorm.DB, provider string) (enterprisemodels.IdentitySource, error) {
	var src enterprisemodels.IdentitySource
	err := db.Where("workspace_id = ? AND provider = ?", enterprisemodels.WorkspaceDefaultID, provider).First(&src).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return enterprisemodels.IdentitySource{
			WorkspaceID: enterprisemodels.WorkspaceDefaultID,
			Provider:    provider,
		}, nil
	}

	return src, err
}

func SaveIdentitySource(db *gorm.DB, provider string, req UpdateRequest) (enterprisemodels.IdentitySource, error) {
	if provider == "" {
		provider = enterprisemodels.ProviderFeishu
	}

	var src enterprisemodels.IdentitySource
	err := db.Where("workspace_id = ? AND provider = ?", enterprisemodels.WorkspaceDefaultID, provider).First(&src).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		src = enterprisemodels.IdentitySource{
			WorkspaceID: enterprisemodels.WorkspaceDefaultID,
			Provider:    provider,
		}
	} else if err != nil {
		return enterprisemodels.IdentitySource{}, err
	}

	src.ExternalOrgID = strings.TrimSpace(req.ExternalOrgID)
	src.AppID = strings.TrimSpace(req.AppID)
	if strings.TrimSpace(req.AppSecret) != "" {
		src.AppSecret = strings.TrimSpace(req.AppSecret)
	}
	src.RedirectURI = strings.TrimSpace(req.RedirectURI)
	src.FrontendURL = strings.TrimSpace(req.FrontendURL)
	src.SyncEnabled = req.SyncEnabled
	src.Enabled = req.Enabled

	if err := db.Save(&src).Error; err != nil {
		return enterprisemodels.IdentitySource{}, err
	}

	return src, nil
}

func NewConfigResponse(src enterprisemodels.IdentitySource, effective EffectiveConfig) ConfigResponse {
	hasSecret := src.AppSecret != "" || effective.HasSecret
	mask := ""
	if hasSecret {
		mask = "********"
	}

	return ConfigResponse{
		ID:              src.ID,
		WorkspaceID:     defaultString(src.WorkspaceID, enterprisemodels.WorkspaceDefaultID),
		Provider:        defaultString(src.Provider, effective.Provider),
		ExternalOrgID:   src.ExternalOrgID,
		AppID:           src.AppID,
		AppSecretMask:   mask,
		RedirectURI:     src.RedirectURI,
		FrontendURL:     src.FrontendURL,
		SyncEnabled:     src.SyncEnabled,
		Enabled:         src.Enabled,
		EffectiveSource: effective.Source,
		DBConfigured:    effective.DBConfigured,
		HasSecret:       hasSecret,
		LastCheckStatus: src.LastCheckStatus,
		LastCheckResult: src.LastCheckResult,
	}
}

func envConfig(workspaceID string, provider string) EffectiveConfig {
	if provider != enterprisemodels.ProviderFeishu {
		return EffectiveConfig{
			WorkspaceID: workspaceID,
			Provider:    provider,
			Source:      SourceEnv,
		}
	}

	frontendURL := os.Getenv("FEISHU_FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}

	appSecret := os.Getenv("FEISHU_APP_SECRET")

	return EffectiveConfig{
		WorkspaceID:    workspaceID,
		Provider:       provider,
		Source:         SourceEnv,
		AppID:          os.Getenv("FEISHU_APP_ID"),
		AppSecret:      appSecret,
		RedirectURI:    os.Getenv("FEISHU_REDIRECT_URI"),
		FrontendURL:    frontendURL,
		AllowedTenants: splitCSV(os.Getenv("FEISHU_ALLOWED_TENANTS")),
		Enabled:        os.Getenv("FEISHU_APP_ID") != "" && appSecret != "",
		HasSecret:      appSecret != "",
	}
}

func splitCSV(v string) []string {
	if v == "" {
		return nil
	}

	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}

	return out
}

func defaultString(v string, fallback string) string {
	if v != "" {
		return v
	}

	return fallback
}
