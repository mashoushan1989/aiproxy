//go:build enterprise

package quota

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	entmodels "github.com/labring/aiproxy/core/enterprise/models"
	"github.com/labring/aiproxy/core/middleware"
	"github.com/labring/aiproxy/core/model"
	"gorm.io/gorm"
)

func auditOperatorFromContext(c *gin.Context) AuditOperator {
	username := c.GetString("username")
	return AuditOperator{ID: username, Name: username}
}

func policyIDParam(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, "invalid policy id")
		return 0, false
	}

	return id, true
}

func entryIDParam(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("entry_id"))
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, "invalid promoted model id")
		return 0, false
	}

	return id, true
}

func requirePromotedModelEntryInPolicy(c *gin.Context) (int, int, bool) {
	policyID, ok := policyIDParam(c)
	if !ok {
		return 0, 0, false
	}

	entryID, ok := entryIDParam(c)
	if !ok {
		return 0, 0, false
	}

	var count int64
	if err := model.DB.Model(&entmodels.PromotedModelPolicy{}).
		Where("id = ? AND quota_policy_id = ?", entryID, policyID).
		Count(&count).Error; err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return 0, 0, false
	}
	if count == 0 {
		middleware.ErrorResponse(c, http.StatusNotFound, "promoted model not found")
		return 0, 0, false
	}

	return policyID, entryID, true
}

func ListPromotedModels(c *gin.Context) {
	policyID, ok := policyIDParam(c)
	if !ok {
		return
	}

	var entries []entmodels.PromotedModelPolicy
	if err := model.DB.
		Where("quota_policy_id = ?", policyID).
		Order("sort_order ASC, id DESC").
		Find(&entries).Error; err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.SuccessResponse(c, gin.H{"entries": entries})
}

func CreatePromotedModel(c *gin.Context) {
	policyID, ok := policyIDParam(c)
	if !ok {
		return
	}

	var req CreatePromotedModelEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	req.QuotaPolicyID = policyID

	entry, err := CreatePromotedModelEntry(req, auditOperatorFromContext(c))
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	middleware.SuccessResponse(c, entry)
}

func UpdatePromotedModel(c *gin.Context) {
	_, entryID, ok := requirePromotedModelEntryInPolicy(c)
	if !ok {
		return
	}

	var req struct {
		UpdatePromotedModelEntryRequest
		OverrideLocked bool `json:"override_locked"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	entry, err := UpdatePromotedModelEntry(
		entryID,
		req.UpdatePromotedModelEntryRequest,
		auditOperatorFromContext(c),
		req.OverrideLocked,
	)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, ErrPromotedModelPriceLocked) {
			status = http.StatusConflict
		}
		middleware.ErrorResponse(c, status, err.Error())
		return
	}

	middleware.SuccessResponse(c, entry)
}

func DeletePromotedModel(c *gin.Context) {
	_, entryID, ok := requirePromotedModelEntryInPolicy(c)
	if !ok {
		return
	}

	if err := DeletePromotedModelEntry(entryID, auditOperatorFromContext(c)); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	middleware.SuccessResponse(c, gin.H{"deleted": true})
}

func RollbackPromotedModel(c *gin.Context) {
	_, entryID, ok := requirePromotedModelEntryInPolicy(c)
	if !ok {
		return
	}

	var req struct {
		Version int `json:"version" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	entry, err := RollbackPromotedModelEntry(entryID, req.Version, auditOperatorFromContext(c))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			middleware.ErrorResponse(c, http.StatusNotFound, "rollback version not found")
			return
		}
		middleware.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	middleware.SuccessResponse(c, entry)
}

func ListPromotedModelAudits(c *gin.Context) {
	policyID, ok := policyIDParam(c)
	if !ok {
		return
	}

	var audits []entmodels.PromotedModelPolicyAudit
	if err := model.DB.
		Where("quota_policy_id = ?", policyID).
		Order("id DESC").
		Find(&audits).Error; err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.SuccessResponse(c, gin.H{"audits": audits})
}
