//go:build enterprise

package analyticsx

import "github.com/gin-gonic/gin"

// RegisterRoutes registers analytics v2 routes when the feature flag is enabled.
func RegisterRoutes(group *gin.RouterGroup, permMiddleware map[string]gin.HandlerFunc) {
	dash := group.Group("/v2", permMiddleware["dashboard_view"])
	dash.GET("/department", HandleDepartmentSummary)
	dash.GET("/model/distribution", HandleModelDistribution)

	rank := group.Group("/v2", permMiddleware["ranking_view"])
	rank.GET("/user/ranking", HandleUserRanking)
}
