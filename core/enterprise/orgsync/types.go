//go:build enterprise

package orgsync

import "context"

type ProviderClient interface {
	Identity(ctx context.Context) (TenantIdentity, error)
	ListOrgUnits(ctx context.Context) ([]OrgUnitRecord, error)
	ListUsers(ctx context.Context) ([]UserRecord, error)
}

type TenantIdentity struct {
	Provider         string
	ExternalTenantID string
	ExternalCorpID   string
	ExternalAppID    string
	DisplayName      string
}

type OrgUnitRecord struct {
	ExternalID       string
	ExternalOpenID   string
	ParentExternalID string
	Name             string
	Order            int
	MemberCount      int
	Raw              map[string]any
}

type UserRecord struct {
	ExternalUserID           string
	ExternalOpenID           string
	ExternalUnionID          string
	Name                     string
	Email                    string
	Avatar                   string
	PrimaryOrgUnitExternalID string
	OrgUnitExternalIDs       []string
	Raw                      map[string]any
}

type Snapshot struct {
	WorkspaceID string
	Provider    string
	OrgUnits    []OrgUnitRecord
	Users       []UserRecord
}
