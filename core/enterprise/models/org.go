//go:build enterprise

package models

import (
	"time"

	"gorm.io/gorm"
)

const (
	ProviderFeishu   = "feishu"
	ProviderWeCom    = "wecom"
	ProviderDingTalk = "dingtalk"
	ProviderManual   = "manual"

	EntityStatusEnabled  = 1
	EntityStatusDisabled = 2
)

type OrgUnit struct {
	ID             string         `json:"id"               gorm:"size:96;primaryKey"`
	WorkspaceID    string         `json:"workspace_id"     gorm:"size:64;index;not null;uniqueIndex:idx_org_unit_provider_external,priority:1"`
	Provider       string         `json:"provider"         gorm:"size:32;index;not null;uniqueIndex:idx_org_unit_provider_external,priority:2"`
	ExternalID     string         `json:"external_id"      gorm:"size:128;index;not null;uniqueIndex:idx_org_unit_provider_external,priority:3"`
	ExternalOpenID string         `json:"external_open_id" gorm:"size:128;index"`
	ParentID       string         `json:"parent_id"        gorm:"size:96;index"`
	Path           string         `json:"path"             gorm:"type:text;index:,length:191"`
	Depth          int            `json:"depth"            gorm:"index"`
	Name           string         `json:"name"             gorm:"size:256;not null"`
	Order          int            `json:"order"            gorm:"default:0"`
	MemberCount    int            `json:"member_count"     gorm:"default:0"`
	Status         int            `json:"status"           gorm:"default:1;index"`
	Raw            string         `json:"raw,omitempty"    gorm:"type:text"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-"                gorm:"index"`
}

func (OrgUnit) TableName() string { return "enterprise_org_units" }

type EnterpriseUser struct {
	ID               string         `json:"id"                  gorm:"size:96;primaryKey"`
	WorkspaceID      string         `json:"workspace_id"        gorm:"size:64;index;not null;uniqueIndex:idx_enterprise_user_provider_open,priority:1"`
	Provider         string         `json:"provider"            gorm:"size:32;index;not null;uniqueIndex:idx_enterprise_user_provider_open,priority:2"`
	ExternalUserID   string         `json:"external_user_id"    gorm:"size:128;index"`
	ExternalOpenID   string         `json:"external_open_id"    gorm:"size:128;index;not null;uniqueIndex:idx_enterprise_user_provider_open,priority:3"`
	ExternalUnionID  string         `json:"external_union_id"   gorm:"size:128;index"`
	Name             string         `json:"name"                gorm:"size:128"`
	Email            string         `json:"email"               gorm:"size:256"`
	Avatar           string         `json:"avatar"              gorm:"size:512"`
	Status           int            `json:"status"              gorm:"default:1;index"`
	DefaultGroupID   string         `json:"default_group_id"    gorm:"size:64;index"`
	PrimaryOrgUnitID string         `json:"primary_org_unit_id" gorm:"size:96;index"`
	Raw              string         `json:"raw,omitempty"       gorm:"type:text"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `json:"-"                   gorm:"index"`
}

func (EnterpriseUser) TableName() string { return "enterprise_users" }

type UserOrgUnit struct {
	ID          int       `json:"id"           gorm:"primaryKey"`
	WorkspaceID string    `json:"workspace_id" gorm:"size:64;index;not null;uniqueIndex:idx_user_org_unit,priority:1"`
	UserID      string    `json:"user_id"      gorm:"size:96;index;not null;uniqueIndex:idx_user_org_unit,priority:2"`
	OrgUnitID   string    `json:"org_unit_id"  gorm:"size:96;index;not null;uniqueIndex:idx_user_org_unit,priority:3"`
	IsPrimary   bool      `json:"is_primary"   gorm:"default:false;index"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (UserOrgUnit) TableName() string { return "enterprise_user_org_units" }
