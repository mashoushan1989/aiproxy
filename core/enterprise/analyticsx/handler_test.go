//go:build enterprise

package analyticsx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegisterRoutesAddsV2RoutesWithPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	analytics := router.Group("/analytics")
	seen := map[string]bool{}
	RegisterRoutes(analytics, map[string]gin.HandlerFunc{
		"dashboard_view": markPermission(seen, "dashboard_view"),
		"ranking_view":   markPermission(seen, "ranking_view"),
		"export_manage":  markPermission(seen, "export_manage"),
	})

	want := map[string]string{
		"/analytics/v2/department":         "dashboard_view",
		"/analytics/v2/user/ranking":       "ranking_view",
		"/analytics/v2/model/distribution": "dashboard_view",
	}

	for path, permission := range want {
		t.Run(path, func(t *testing.T) {
			seen[permission] = false
			req, err := http.NewRequest(http.MethodGet, path+"?granularity=raw_sql", nil)
			if err != nil {
				t.Fatalf("NewRequest() error = %v", err)
			}

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code == http.StatusNotFound {
				t.Fatalf("%s returned 404; route was not registered", path)
			}
			if !seen[permission] {
				t.Fatalf("%s did not run %s middleware", path, permission)
			}
		})
	}
}

func markPermission(seen map[string]bool, permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		seen[permission] = true
		c.Next()
	}
}
