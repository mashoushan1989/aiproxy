//go:build enterprise

package models

import "time"

const (
	WorkspaceDefaultID = "default"

	WorkspaceKindPrimary  = "primary"
	WorkspaceKindDemo     = "demo"
	WorkspaceKindSandbox  = "sandbox"
	WorkspaceKindInternal = "internal"

	WorkspaceStatusEnabled  = 1
	WorkspaceStatusDisabled = 2
)

type Workspace struct {
	ID              string    `json:"id"               gorm:"size:64;primaryKey"`
	Name            string    `json:"name"             gorm:"size:128;not null"`
	Slug            string    `json:"slug"             gorm:"size:128;uniqueIndex;not null"`
	Kind            string    `json:"kind"             gorm:"size:32;index;not null"`
	Status          int       `json:"status"           gorm:"default:1;index"`
	DefaultProvider string    `json:"default_provider" gorm:"size:32;index"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (Workspace) TableName() string { return "enterprise_workspaces" }

type WorkspaceProviderBinding struct {
	ID               int       `json:"id"                 gorm:"primaryKey"`
	WorkspaceID      string    `json:"workspace_id"       gorm:"size:64;index;not null;uniqueIndex:idx_workspace_provider_binding,priority:1"`
	Provider         string    `json:"provider"           gorm:"size:32;index;not null;uniqueIndex:idx_workspace_provider_binding,priority:2"`
	ExternalTenantID string    `json:"external_tenant_id" gorm:"size:128;index;uniqueIndex:idx_workspace_provider_binding,priority:3"`
	ExternalCorpID   string    `json:"external_corp_id"   gorm:"size:128;index"`
	ExternalAppID    string    `json:"external_app_id"    gorm:"size:128;index"`
	DisplayName      string    `json:"display_name"       gorm:"size:255"`
	Status           int       `json:"status"             gorm:"default:1;index"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (WorkspaceProviderBinding) TableName() string {
	return "enterprise_workspace_provider_bindings"
}
