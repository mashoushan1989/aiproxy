//go:build enterprise

package analyticsx

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strings"

	enterprisemodels "github.com/labring/aiproxy/core/enterprise/models"
	"gorm.io/gorm"
)

const (
	AuditActionExport  = "export"
	AuditResultSuccess = "success"
	AuditResultFailure = "failure"
)

type ExportAuditInput struct {
	WorkspaceID  string
	ActorGroupID string
	Scope        Scope
	Filter       Filter
	ResultStatus string
	RowCount     int
	Err          error
}

func PersistExportAuditEvent(ctx context.Context, db *gorm.DB, input ExportAuditInput) error {
	status := input.ResultStatus
	if status == "" {
		status = AuditResultSuccess
		if input.Err != nil {
			status = AuditResultFailure
		}
	}

	event := enterprisemodels.AnalyticsAuditEvent{
		WorkspaceID:    input.WorkspaceID,
		ActorGroupHash: actorGroupHash(input.ActorGroupID),
		Action:         AuditActionExport,
		ScopeSummary:   scopeSummary(input.Scope),
		FilterJSON:     filterJSON(input.Filter),
		ResultStatus:   status,
		RowCount:       input.RowCount,
		ErrorMessage:   sanitizeError(input.Err),
	}
	if event.WorkspaceID == "" {
		event.WorkspaceID = input.Scope.WorkspaceID
	}
	if event.ActorGroupHash == "" {
		event.ActorGroupHash = actorGroupHash(input.Scope.CallerGroupID)
	}

	return db.WithContext(ctx).Create(&event).Error
}

func actorGroupHash(groupID string) string {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(groupID))
	return hex.EncodeToString(sum[:])
}

func scopeSummary(scope Scope) string {
	parts := []string{
		"workspace=" + scope.WorkspaceID,
		"role=" + scope.Role,
	}
	if scope.AllGroups {
		parts = append(parts, "groups=all")
	} else {
		parts = append(parts, "groups="+itoa(len(compactStrings(scope.AllowedGroupIDs))))
	}
	if scope.AllOrgUnits {
		parts = append(parts, "org_units=all")
	} else {
		parts = append(parts, "org_units="+itoa(len(compactStrings(scope.AllowedOrgUnitIDs))))
	}
	if scope.AllUsers {
		parts = append(parts, "users=all")
	} else {
		parts = append(parts, "users="+itoa(len(compactStrings(scope.AllowedUserIDs))))
	}
	if len(compactStrings(scope.AllowedModels)) > 0 {
		parts = append(parts, "models="+itoa(len(compactStrings(scope.AllowedModels))))
	}
	return strings.Join(parts, ";")
}

type auditFilter struct {
	StartTimestamp int64    `json:"start_timestamp"`
	EndTimestamp   int64    `json:"end_timestamp"`
	Granularity    string   `json:"granularity,omitempty"`
	OrgUnitCount   int      `json:"org_unit_count"`
	GroupCount     int      `json:"group_count"`
	UserCount      int      `json:"user_count"`
	ModelCount     int      `json:"model_count"`
	Models         []string `json:"models,omitempty"`
	Limit          int      `json:"limit,omitempty"`
	Page           int      `json:"page,omitempty"`
	PerPage        int      `json:"per_page,omitempty"`
}

func filterJSON(filter Filter) string {
	models := compactStrings(filter.Models)
	out := auditFilter{
		StartTimestamp: filter.StartTimestamp,
		EndTimestamp:   filter.EndTimestamp,
		Granularity:    filter.Granularity,
		OrgUnitCount:   len(compactStrings(filter.OrgUnitIDs)),
		GroupCount:     len(compactStrings(filter.GroupIDs)),
		UserCount:      len(compactStrings(filter.UserIDs)),
		ModelCount:     len(models),
		Models:         sanitizeModels(models),
		Limit:          filter.Limit,
		Page:           filter.Page,
		PerPage:        filter.PerPage,
	}

	data, err := json.Marshal(out)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func sanitizeModels(models []string) []string {
	out := make([]string, 0, len(models))
	for _, modelName := range models {
		if looksSecret(modelName) {
			out = append(out, "[redacted]")
			continue
		}
		out = append(out, modelName)
	}
	return out
}

func sanitizeError(err error) string {
	if err == nil {
		return ""
	}
	return secretPattern.ReplaceAllString(err.Error(), "[redacted]")
}

var secretPattern = regexp.MustCompile(`(?i)(bearer\s+[^\s,;]+|sk-[a-z0-9_-]+|token[=:]\s*[^\s,;]+|secret[=:]\s*[^\s,;]+)`)

func looksSecret(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(value, "sk-") ||
		strings.Contains(value, "secret") ||
		strings.Contains(value, "token") ||
		strings.Contains(value, "bearer")
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}

	var buf [20]byte
	i := len(buf)
	for value > 0 {
		i--
		buf[i] = byte('0' + value%10)
		value /= 10
	}
	return string(buf[i:])
}
