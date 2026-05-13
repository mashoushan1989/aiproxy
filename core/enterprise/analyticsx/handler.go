//go:build enterprise

package analyticsx

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/middleware"
	"github.com/labring/aiproxy/core/model"
)

const defaultServiceTimeout = 30 * time.Second

// HandleDepartmentSummary returns scoped department-level analytics.
func HandleDepartmentSummary(c *gin.Context) {
	scope, filter, ok := resolveHandlerInputs(c)
	if !ok {
		return
	}

	summaries, err := newService().DepartmentSummaries(c.Request.Context(), scope, filter)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.SuccessResponse(c, gin.H{
		"departments": summaries,
		"total":       len(summaries),
	})
}

// HandleUserRanking returns scoped user usage rankings.
func HandleUserRanking(c *gin.Context) {
	scope, filter, ok := resolveHandlerInputs(c)
	if !ok {
		return
	}

	ranking, total, err := newService().UserRanking(c.Request.Context(), scope, filter)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.SuccessResponse(c, gin.H{
		"ranking": ranking,
		"total":   total,
	})
}

// HandleModelDistribution returns scoped model usage distribution.
func HandleModelDistribution(c *gin.Context) {
	scope, filter, ok := resolveHandlerInputs(c)
	if !ok {
		return
	}

	distribution, err := newService().ModelDistribution(c.Request.Context(), scope, filter)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.SuccessResponse(c, gin.H{
		"distribution": distribution,
		"total":        len(distribution),
	})
}

// HandleExport returns a scoped analytics export and records an audit event.
func HandleExport(c *gin.Context) {
	scope, filter, ok := resolveHandlerInputs(c)
	if !ok {
		return
	}

	dataset, err := BuildExportDataset(c.Request.Context(), newService(), scope, filter)
	if err != nil {
		_ = PersistExportAuditEvent(c.Request.Context(), model.DB, ExportAuditInput{
			Scope:        scope,
			Filter:       filter,
			ResultStatus: AuditResultFailure,
			Err:          err,
		})
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	workbook, err := BuildExportWorkbook(dataset, filter)
	if err != nil {
		_ = PersistExportAuditEvent(c.Request.Context(), model.DB, ExportAuditInput{
			Scope:        scope,
			Filter:       filter,
			RowCount:     dataset.TotalRows,
			ResultStatus: AuditResultFailure,
			Err:          err,
		})
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	if err := PersistExportAuditEvent(c.Request.Context(), model.DB, ExportAuditInput{
		Scope:        scope,
		Filter:       filter,
		ResultStatus: AuditResultSuccess,
		RowCount:     dataset.TotalRows,
	}); err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	filename := fmt.Sprintf(
		"enterprise_analytics_v2_%s_%s.xlsx",
		time.Unix(filter.StartTimestamp, 0).Format("20060102"),
		time.Unix(filter.EndTimestamp, 0).Format("20060102"),
	)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename="+filename)

	if err := workbook.Write(c.Writer); err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, "failed to write excel file")
	}
}

func resolveHandlerInputs(c *gin.Context) (Scope, Filter, bool) {
	scope, err := ResolveScope(c)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusForbidden, err.Error())
		return Scope{}, Filter{}, false
	}

	filter, err := ParseRequest(c)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return Scope{}, Filter{}, false
	}

	return scope, filter, true
}

func newService() Service {
	return Service{
		DB:           model.DB,
		LogDB:        model.LogDB,
		OrgDirectory: NewGORMOrgDirectory(model.DB),
		Timeout:      defaultServiceTimeout,
	}
}
