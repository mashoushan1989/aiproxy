//go:build enterprise

package identitysource

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/labring/aiproxy/core/enterprise/models"
	"github.com/labring/aiproxy/core/middleware"
	"github.com/labring/aiproxy/core/model"
)

func GetConfigHandler(c *gin.Context) {
	provider := c.Param("provider")
	if !isSupportedProvider(provider) {
		middleware.ErrorResponse(c, http.StatusBadRequest, "unsupported identity source provider")
		return
	}

	src, err := GetIdentitySource(model.DB, provider)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	effective, err := ResolveConfig(model.DB, models.WorkspaceDefaultID, provider)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.SuccessResponse(c, NewConfigResponse(src, effective))
}

func UpdateConfigHandler(c *gin.Context) {
	provider := c.Param("provider")
	if !isSupportedProvider(provider) {
		middleware.ErrorResponse(c, http.StatusBadRequest, "unsupported identity source provider")
		return
	}
	if provider != models.ProviderFeishu {
		middleware.ErrorResponse(c, http.StatusBadRequest, "identity source provider is reserved")
		return
	}

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	src, err := SaveIdentitySource(model.DB, provider, req)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	effective, err := ResolveConfig(model.DB, models.WorkspaceDefaultID, provider)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.SuccessResponse(c, NewConfigResponse(src, effective))
}

func CheckHandler(c *gin.Context) {
	provider := c.Param("provider")
	if provider != models.ProviderFeishu {
		middleware.ErrorResponse(c, http.StatusBadRequest, "identity source doctor currently supports feishu only")
		return
	}

	effective, err := ResolveConfig(model.DB, models.WorkspaceDefaultID, provider)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	middleware.SuccessResponse(c, RunDoctor(c.Request.Context(), effective, nil))
}

func isSupportedProvider(provider string) bool {
	switch provider {
	case models.ProviderFeishu, models.ProviderWeCom, models.ProviderDingTalk:
		return true
	default:
		return false
	}
}
