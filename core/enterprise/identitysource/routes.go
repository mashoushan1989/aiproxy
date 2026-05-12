//go:build enterprise

package identitysource

import (
	"github.com/gin-gonic/gin"

	"github.com/labring/aiproxy/core/enterprise/models"
)

type Middleware struct {
	UserManageView   gin.HandlerFunc
	UserManageManage gin.HandlerFunc
	AdminOnly        gin.HandlerFunc
}

func RegisterRoutes(group *gin.RouterGroup, mw Middleware) {
	view := group.Group("/identity-sources", mw.UserManageView)
	view.GET("/:provider", GetConfigHandler)

	manage := group.Group("/identity-sources", mw.UserManageManage, mw.AdminOnly)
	manage.PUT("/:provider", UpdateConfigHandler)
	manage.POST("/:provider/check", CheckHandler)
}

func NewMiddleware(permMW map[string]gin.HandlerFunc, adminOnly gin.HandlerFunc) Middleware {
	return Middleware{
		UserManageView:   permMW[models.PermUserManageView],
		UserManageManage: permMW[models.PermUserManageManage],
		AdminOnly:        adminOnly,
	}
}
