//go:build enterprise

package identitysource

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	enterprisemodels "github.com/labring/aiproxy/core/enterprise/models"
	"github.com/labring/aiproxy/core/model"
	"github.com/stretchr/testify/require"
)

func setupHandlerTest(t *testing.T) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	db := setupConfigTestDB(t)
	model.DB = db

	r := gin.New()
	r.PUT("/identity-sources/:provider", UpdateConfigHandler)
	r.POST("/identity-sources/:provider/check", CheckHandler)

	return r
}

func TestUpdateConfigRejectsReservedProviders(t *testing.T) {
	r := setupHandlerTest(t)

	req := httptest.NewRequest(http.MethodPut, "/identity-sources/wecom", strings.NewReader(`{"app_id":"corp"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCheckHandlerDoesNotPersistLastCheckResult(t *testing.T) {
	r := setupHandlerTest(t)
	require.NoError(t, model.DB.Create(&enterprisemodels.IdentitySource{
		WorkspaceID:     enterprisemodels.WorkspaceDefaultID,
		Provider:        enterprisemodels.ProviderFeishu,
		LastCheckStatus: "previous",
		LastCheckResult: `{"previous":true}`,
	}).Error)

	req := httptest.NewRequest(http.MethodPost, "/identity-sources/feishu/check", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var src enterprisemodels.IdentitySource
	require.NoError(t, model.DB.Where("provider = ?", enterprisemodels.ProviderFeishu).First(&src).Error)
	require.Equal(t, "previous", src.LastCheckStatus)
	require.Equal(t, `{"previous":true}`, src.LastCheckResult)
}
