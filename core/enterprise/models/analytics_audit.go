//go:build enterprise

package models

import "time"

type AnalyticsAuditEvent struct {
	ID             int       `json:"id"             gorm:"primaryKey"`
	WorkspaceID    string    `json:"workspace_id"   gorm:"size:64;index"`
	ActorGroupHash string    `json:"actor_group_hash" gorm:"size:64;index"`
	Action         string    `json:"action"         gorm:"size:64;index"`
	ScopeSummary   string    `json:"scope_summary"  gorm:"type:text"`
	FilterJSON     string    `json:"filter_json"    gorm:"type:text"`
	ResultStatus   string    `json:"result_status"  gorm:"size:32;index"`
	RowCount       int       `json:"row_count"`
	ErrorMessage   string    `json:"error_message"  gorm:"type:text"`
	CreatedAt      time.Time `json:"created_at"     gorm:"autoCreateTime;index"`
}

func (AnalyticsAuditEvent) TableName() string {
	return "enterprise_analyticsx_audit_events"
}
