//go:build enterprise

package analyticsx

import "github.com/gin-gonic/gin"

// RegisterRoutes registers analytics v2 routes when the feature flag is enabled.
func RegisterRoutes(group *gin.RouterGroup, permMiddleware map[string]gin.HandlerFunc) {
}
