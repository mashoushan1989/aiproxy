//go:build enterprise

package quota

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	entmodels "github.com/labring/aiproxy/core/enterprise/models"
	"github.com/labring/aiproxy/core/middleware"
	"github.com/labring/aiproxy/core/model"
)

func setupPromotedModelHandlerRouter(t *testing.T) (*gin.Engine, entmodels.QuotaPolicy) {
	t.Helper()

	db := setupPromotedModelPolicyTestDB(t)
	policy := seedPromotedModelFixtures(t, db)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("username", "admin")
		c.Next()
	})
	RegisterRoutes(router.Group("/api/enterprise"), map[string]gin.HandlerFunc{
		"quota_manage_view":   func(c *gin.Context) { c.Next() },
		"quota_manage_manage": func(c *gin.Context) { c.Next() },
	})

	return router, policy
}

func requestJSON(t *testing.T, router http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return w
}

func TestPromotedModelHandlersCreateListAndAudit(t *testing.T) {
	router, policy := setupPromotedModelHandlerRouter(t)

	createResp := requestJSON(t, router, http.MethodPost, "/api/enterprise/quota/policies/1/promoted-models", gin.H{
		"model":           "pa/gpt-5.5",
		"display_name":    "GPT-5.5",
		"recommend_badge": "Recommended",
		"enabled":         true,
		"override_price": gin.H{
			"input_price":       0.0000145,
			"input_price_unit":  1,
			"output_price":      0.000087,
			"output_price_unit": 1,
		},
	})
	if createResp.Code != http.StatusOK {
		t.Fatalf("create status = %d, body = %s", createResp.Code, createResp.Body.String())
	}

	listResp := requestJSON(t, router, http.MethodGet, "/api/enterprise/quota/policies/1/promoted-models", nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listResp.Code, listResp.Body.String())
	}

	var listBody struct {
		Success bool `json:"success"`
		Data    struct {
			Entries []entmodels.PromotedModelPolicy `json:"entries"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if !listBody.Success || len(listBody.Data.Entries) != 1 || listBody.Data.Entries[0].QuotaPolicyID != policy.ID {
		t.Fatalf("unexpected list body: %#v", listBody)
	}

	auditResp := requestJSON(t, router, http.MethodGet, "/api/enterprise/quota/policies/1/promoted-models/audit", nil)
	if auditResp.Code != http.StatusOK {
		t.Fatalf("audit status = %d, body = %s", auditResp.Code, auditResp.Body.String())
	}

	var auditBody struct {
		Success bool `json:"success"`
		Data    struct {
			Audits []entmodels.PromotedModelPolicyAudit `json:"audits"`
		} `json:"data"`
	}
	if err := json.Unmarshal(auditResp.Body.Bytes(), &auditBody); err != nil {
		t.Fatalf("decode audit: %v", err)
	}
	if !auditBody.Success || len(auditBody.Data.Audits) != 1 || auditBody.Data.Audits[0].OperatorID != "admin" {
		t.Fatalf("unexpected audit body: %#v", auditBody)
	}
}

func TestPromotedModelHandlerRejectsLockedPriceWithoutOverride(t *testing.T) {
	router, policy := setupPromotedModelHandlerRouter(t)

	price, err := commercialPriceFromModelPrice(model.Price{
		InputPrice:     model.ZeroNullFloat64(0.1),
		InputPriceUnit: model.ZeroNullInt64(1),
	})
	if err != nil {
		t.Fatalf("convert price: %v", err)
	}

	entry := entmodels.PromotedModelPolicy{
		QuotaPolicyID: policy.ID,
		Model:         "pa/gpt-5.5",
		Enabled:       true,
		PriceLocked:   true,
		OverridePrice: price,
	}
	if err := model.DB.Create(&entry).Error; err != nil {
		t.Fatalf("seed entry: %v", err)
	}

	resp := requestJSON(t, router, http.MethodPut, "/api/enterprise/quota/policies/1/promoted-models/1", gin.H{
		"enabled": true,
		"override_price": gin.H{
			"input_price":      0.2,
			"input_price_unit": 1,
		},
	})

	if resp.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", resp.Code, resp.Body.String())
	}
}

func TestPromotedModelHandlerScopesEntryMutationsToPolicy(t *testing.T) {
	router, policy := setupPromotedModelHandlerRouter(t)

	otherPolicy := entmodels.QuotaPolicy{Name: "Sales"}
	if err := model.DB.Create(&otherPolicy).Error; err != nil {
		t.Fatalf("seed other policy: %v", err)
	}

	price, err := commercialPriceFromModelPrice(model.Price{
		InputPrice:     model.ZeroNullFloat64(0.1),
		InputPriceUnit: model.ZeroNullInt64(1),
	})
	if err != nil {
		t.Fatalf("convert price: %v", err)
	}

	entry := entmodels.PromotedModelPolicy{
		QuotaPolicyID: otherPolicy.ID,
		Model:         "pa/gpt-5.5",
		Enabled:       true,
		OverridePrice: price,
	}
	if err := model.DB.Create(&entry).Error; err != nil {
		t.Fatalf("seed entry: %v", err)
	}

	resp := requestJSON(t, router, http.MethodPut, "/api/enterprise/quota/policies/1/promoted-models/1", gin.H{
		"enabled": true,
		"override_price": gin.H{
			"input_price":      0.2,
			"input_price_unit": 1,
		},
	})
	if resp.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", resp.Code, resp.Body.String())
	}

	var unchanged entmodels.PromotedModelPolicy
	if err := model.DB.First(&unchanged, entry.ID).Error; err != nil {
		t.Fatalf("load entry: %v", err)
	}
	if unchanged.QuotaPolicyID != otherPolicy.ID || unchanged.OverridePrice.InputPrice != price.InputPrice {
		t.Fatalf("entry mutated through wrong policy path: %#v, original policy %d", unchanged, policy.ID)
	}
}

var _ = middleware.APIResponse{}
