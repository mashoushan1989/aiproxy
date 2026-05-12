//go:build enterprise

package orgsync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	enterprisemodels "github.com/labring/aiproxy/core/enterprise/models"
	"github.com/labring/aiproxy/core/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func OrgUnitID(workspaceID, provider, externalID string) string {
	return boundedID("ou", workspaceID, provider, externalID)
}

func EnterpriseUserID(workspaceID, provider, externalOpenID string) string {
	return boundedID("eu", workspaceID, provider, externalOpenID)
}

func personalGroupID(provider, externalOpenID string) string {
	raw := provider + "_" + externalOpenID
	if len(raw) <= 64 {
		return raw
	}

	sum := sha256.Sum256([]byte(raw))
	prefix := provider
	if len(prefix) > 31 {
		prefix = prefix[:31]
	}

	return prefix + "_" + hex.EncodeToString(sum[:])[:32]
}

func boundedID(parts ...string) string {
	raw := strings.Join(parts, ":")
	if len(raw) <= 64 {
		return raw
	}

	sum := sha256.Sum256([]byte(raw))
	prefix := strings.Join(parts[:len(parts)-1], ":")
	if len(prefix) > 24 {
		prefix = prefix[:24]
	}

	return prefix + ":" + hex.EncodeToString(sum[:])[:32]
}

func SyncSnapshot(ctx context.Context, db *gorm.DB, snapshot Snapshot) error {
	if snapshot.WorkspaceID == "" {
		snapshot.WorkspaceID = enterprisemodels.WorkspaceDefaultID
	}
	if snapshot.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	if err := validateSnapshotLengths(snapshot); err != nil {
		return err
	}
	if err := validateOrgUnitGraph(snapshot.OrgUnits); err != nil {
		return err
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := syncOrgUnits(tx, snapshot); err != nil {
			return err
		}

		return syncUsers(tx, snapshot)
	})
}

func validateSnapshotLengths(snapshot Snapshot) error {
	if len(snapshot.WorkspaceID) > 64 {
		return fmt.Errorf("workspace id length exceeds 64")
	}
	if len(snapshot.Provider) > 32 {
		return fmt.Errorf("provider length exceeds 32")
	}

	for _, rec := range snapshot.OrgUnits {
		if len(rec.ExternalID) > 128 {
			return fmt.Errorf("org unit external id length exceeds 128: %s", rec.ExternalID[:128])
		}
		if len(rec.ExternalOpenID) > 128 {
			return fmt.Errorf("org unit external open id length exceeds 128: %s", rec.ExternalID)
		}
		if len(rec.ParentExternalID) > 128 {
			return fmt.Errorf("org unit parent external id length exceeds 128: %s", rec.ExternalID)
		}
		if len(rec.Name) > 256 {
			return fmt.Errorf("org unit name length exceeds 256: %s", rec.ExternalID)
		}
	}

	for _, rec := range snapshot.Users {
		if len(rec.ExternalUserID) > 128 {
			return fmt.Errorf("user external user id length exceeds 128: %s", rec.ExternalOpenID)
		}
		if len(rec.ExternalOpenID) > 128 {
			return fmt.Errorf("user external open id length exceeds 128")
		}
		if len(rec.ExternalUnionID) > 128 {
			return fmt.Errorf("user external union id length exceeds 128: %s", rec.ExternalOpenID)
		}
		if len(rec.Name) > 128 {
			return fmt.Errorf("user name length exceeds 128: %s", rec.ExternalOpenID)
		}
		if len(rec.Email) > 256 {
			return fmt.Errorf("user email length exceeds 256: %s", rec.ExternalOpenID)
		}
		if len(rec.Avatar) > 512 {
			return fmt.Errorf("user avatar length exceeds 512: %s", rec.ExternalOpenID)
		}
		if len(rec.PrimaryOrgUnitExternalID) > 128 {
			return fmt.Errorf("user primary org unit external id length exceeds 128: %s", rec.ExternalOpenID)
		}
		for _, orgUnitExternalID := range rec.OrgUnitExternalIDs {
			if len(orgUnitExternalID) > 128 {
				return fmt.Errorf("user org unit external id length exceeds 128: %s", rec.ExternalOpenID)
			}
		}
	}

	return nil
}

func validateOrgUnitGraph(records []OrgUnitRecord) error {
	recordsByExternalID := make(map[string]OrgUnitRecord, len(records))
	for _, rec := range records {
		if rec.ExternalID == "" {
			return fmt.Errorf("org unit external id is required")
		}
		if _, exists := recordsByExternalID[rec.ExternalID]; exists {
			return fmt.Errorf("duplicate org unit external id: %s", rec.ExternalID)
		}
		recordsByExternalID[rec.ExternalID] = rec
	}

	for _, rec := range records {
		if rec.ParentExternalID != "" {
			if _, ok := recordsByExternalID[rec.ParentExternalID]; !ok {
				return fmt.Errorf("org unit %s references missing parent %s", rec.ExternalID, rec.ParentExternalID)
			}
		}
		if err := detectOrgUnitCycle(rec.ExternalID, recordsByExternalID, map[string]struct{}{}); err != nil {
			return err
		}
	}

	return nil
}

func detectOrgUnitCycle(
	externalID string,
	records map[string]OrgUnitRecord,
	visiting map[string]struct{},
) error {
	if _, ok := visiting[externalID]; ok {
		return fmt.Errorf("org unit cycle detected at %s", externalID)
	}

	rec := records[externalID]
	if rec.ParentExternalID == "" {
		return nil
	}

	visiting[externalID] = struct{}{}
	defer delete(visiting, externalID)

	return detectOrgUnitCycle(rec.ParentExternalID, records, visiting)
}

func syncOrgUnits(tx *gorm.DB, snapshot Snapshot) error {
	recordsByExternalID := make(map[string]OrgUnitRecord, len(snapshot.OrgUnits))
	for _, rec := range snapshot.OrgUnits {
		if rec.ExternalID == "" {
			return fmt.Errorf("org unit external id is required")
		}
		recordsByExternalID[rec.ExternalID] = rec
	}

	for _, rec := range snapshot.OrgUnits {
		id := OrgUnitID(snapshot.WorkspaceID, snapshot.Provider, rec.ExternalID)
		parentID := ""
		if rec.ParentExternalID != "" {
			parentID = OrgUnitID(snapshot.WorkspaceID, snapshot.Provider, rec.ParentExternalID)
		}

		name := rec.Name
		if name == "" {
			name = rec.ExternalID
		}

		unit := enterprisemodels.OrgUnit{
			ID:             id,
			WorkspaceID:    snapshot.WorkspaceID,
			Provider:       snapshot.Provider,
			ExternalID:     rec.ExternalID,
			ExternalOpenID: rec.ExternalOpenID,
			ParentID:       parentID,
			Path:           buildOrgUnitPath(snapshot.WorkspaceID, snapshot.Provider, rec.ExternalID, recordsByExternalID),
			Depth:          orgUnitDepth(rec.ExternalID, recordsByExternalID),
			Name:           name,
			Order:          rec.Order,
			MemberCount:    rec.MemberCount,
			Status:         enterprisemodels.EntityStatusEnabled,
			Raw:            marshalRaw(rec.Raw),
		}

		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"workspace_id",
				"provider",
				"external_id",
				"external_open_id",
				"parent_id",
				"path",
				"depth",
				"name",
				"order",
				"member_count",
				"status",
				"raw",
				"updated_at",
			}),
		}).Create(&unit).Error; err != nil {
			return err
		}
	}

	return nil
}

func syncUsers(tx *gorm.DB, snapshot Snapshot) error {
	for _, rec := range snapshot.Users {
		if rec.ExternalOpenID == "" {
			return fmt.Errorf("user external open id is required")
		}

		userID := EnterpriseUserID(snapshot.WorkspaceID, snapshot.Provider, rec.ExternalOpenID)
		groupID := personalGroupID(snapshot.Provider, rec.ExternalOpenID)
		primaryOrgUnitID := ""
		if rec.PrimaryOrgUnitExternalID != "" {
			primaryOrgUnitID = OrgUnitID(snapshot.WorkspaceID, snapshot.Provider, rec.PrimaryOrgUnitExternalID)
		}

		user := enterprisemodels.EnterpriseUser{
			ID:               userID,
			WorkspaceID:      snapshot.WorkspaceID,
			Provider:         snapshot.Provider,
			ExternalUserID:   rec.ExternalUserID,
			ExternalOpenID:   rec.ExternalOpenID,
			ExternalUnionID:  rec.ExternalUnionID,
			Name:             rec.Name,
			Email:            rec.Email,
			Avatar:           rec.Avatar,
			Status:           enterprisemodels.EntityStatusEnabled,
			DefaultGroupID:   groupID,
			PrimaryOrgUnitID: primaryOrgUnitID,
			Raw:              marshalRaw(rec.Raw),
		}

		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"workspace_id",
				"provider",
				"external_user_id",
				"external_open_id",
				"external_union_id",
				"name",
				"email",
				"avatar",
				"status",
				"default_group_id",
				"primary_org_unit_id",
				"raw",
				"updated_at",
			}),
		}).Create(&user).Error; err != nil {
			return err
		}

		if err := ensurePersonalGroup(tx, snapshot.WorkspaceID, groupID, userID, rec.ExternalOpenID, primaryOrgUnitID, rec.Name); err != nil {
			return err
		}

		if err := replaceUserOrgUnits(tx, snapshot.WorkspaceID, userID, snapshot.Provider, rec); err != nil {
			return err
		}
	}

	return nil
}

func ensurePersonalGroup(
	tx *gorm.DB,
	workspaceID string,
	groupID string,
	userID string,
	openID string,
	orgUnitID string,
	name string,
) error {
	group := model.Group{
		ID:          groupID,
		Name:        name,
		WorkspaceID: workspaceID,
		Type:        model.GroupTypePersonal,
		OwnerUserID: userID,
		OwnerOpenID: openID,
		OrgUnitID:   orgUnitID,
		Status:      model.GroupStatusEnabled,
	}

	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"name",
			"workspace_id",
			"type",
			"owner_user_id",
			"owner_open_id",
			"org_unit_id",
			"status",
		}),
	}).Create(&group).Error
}

func replaceUserOrgUnits(tx *gorm.DB, workspaceID, userID, provider string, rec UserRecord) error {
	if err := tx.
		Where("workspace_id = ? AND user_id = ?", workspaceID, userID).
		Delete(&enterprisemodels.UserOrgUnit{}).Error; err != nil {
		return err
	}

	externalIDs := append([]string(nil), rec.OrgUnitExternalIDs...)
	if rec.PrimaryOrgUnitExternalID != "" {
		externalIDs = append([]string{rec.PrimaryOrgUnitExternalID}, externalIDs...)
	}

	seen := make(map[string]struct{}, len(externalIDs))
	for _, externalID := range externalIDs {
		if externalID == "" {
			continue
		}
		if _, ok := seen[externalID]; ok {
			continue
		}
		seen[externalID] = struct{}{}

		membership := enterprisemodels.UserOrgUnit{
			WorkspaceID: workspaceID,
			UserID:      userID,
			OrgUnitID:   OrgUnitID(workspaceID, provider, externalID),
			IsPrimary:   externalID == rec.PrimaryOrgUnitExternalID,
		}
		if err := tx.Create(&membership).Error; err != nil {
			return err
		}
	}

	return nil
}

func marshalRaw(raw map[string]any) string {
	if len(raw) == 0 {
		return ""
	}

	data, err := json.Marshal(raw)
	if err != nil {
		return ""
	}

	return string(data)
}

func buildOrgUnitPath(workspaceID, provider, externalID string, records map[string]OrgUnitRecord) string {
	segments := buildOrgUnitPathSegments(workspaceID, provider, externalID, records, map[string]struct{}{})
	return "/" + strings.Join(segments, "/")
}

func buildOrgUnitPathSegments(
	workspaceID string,
	provider string,
	externalID string,
	records map[string]OrgUnitRecord,
	visiting map[string]struct{},
) []string {
	if _, ok := visiting[externalID]; ok {
		return []string{OrgUnitID(workspaceID, provider, externalID)}
	}
	visiting[externalID] = struct{}{}

	rec, ok := records[externalID]
	if !ok || rec.ParentExternalID == "" {
		return []string{OrgUnitID(workspaceID, provider, externalID)}
	}

	parent := buildOrgUnitPathSegments(workspaceID, provider, rec.ParentExternalID, records, visiting)
	return append(parent, OrgUnitID(workspaceID, provider, externalID))
}

func orgUnitDepth(externalID string, records map[string]OrgUnitRecord) int {
	return orgUnitDepthWithVisited(externalID, records, map[string]struct{}{})
}

func orgUnitDepthWithVisited(externalID string, records map[string]OrgUnitRecord, visiting map[string]struct{}) int {
	if _, ok := visiting[externalID]; ok {
		return 0
	}
	visiting[externalID] = struct{}{}

	rec, ok := records[externalID]
	if !ok || rec.ParentExternalID == "" {
		return 0
	}

	return orgUnitDepthWithVisited(rec.ParentExternalID, records, visiting) + 1
}
