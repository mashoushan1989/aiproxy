//go:build enterprise

package models

import "time"

const (
	IdentitySourceStatusPassed  = "passed"
	IdentitySourceStatusWarning = "warning"
	IdentitySourceStatusFailed  = "failed"
)

// IdentitySource stores per-workspace external identity provider app settings.
// Secrets are never returned by API handlers; callers should use HasSecret or
// a fixed mask when presenting configuration state.
type IdentitySource struct {
	ID              int       `json:"id"                         gorm:"primaryKey"`
	WorkspaceID     string    `json:"workspace_id"               gorm:"size:64;index;not null;uniqueIndex:idx_identity_source_workspace_provider,priority:1"`
	Provider        string    `json:"provider"                   gorm:"size:32;index;not null;uniqueIndex:idx_identity_source_workspace_provider,priority:2"`
	ExternalOrgID   string    `json:"external_org_id"            gorm:"size:128;index"`
	AppID           string    `json:"app_id"                     gorm:"size:128"`
	AppSecret       string    `json:"-"                          gorm:"type:text"`
	RedirectURI     string    `json:"redirect_uri"               gorm:"size:512"`
	FrontendURL     string    `json:"frontend_url"               gorm:"size:512"`
	SyncEnabled     bool      `json:"sync_enabled"               gorm:"default:false"`
	Enabled         bool      `json:"enabled"                    gorm:"default:false;index"`
	LastCheckStatus string    `json:"last_check_status"          gorm:"size:32"`
	LastCheckResult string    `json:"last_check_result,omitempty" gorm:"type:text"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (IdentitySource) TableName() string {
	return "enterprise_identity_sources"
}
