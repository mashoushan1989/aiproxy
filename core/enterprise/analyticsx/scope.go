//go:build enterprise

package analyticsx

import (
	"errors"

	"github.com/gin-gonic/gin"
	enterprisemodels "github.com/labring/aiproxy/core/enterprise/models"
)

type Scope struct {
	WorkspaceID       string
	Role              string
	CallerGroupID     string
	CallerUserID      string
	AllowedOrgUnitIDs []string
	AllowedUserIDs    []string
	AllowedGroupIDs   []string
	AllowedModels     []string
	AllOrgUnits       bool
	AllUsers          bool
	AllGroups         bool
}

type GroupSelection struct {
	All bool
	IDs []string
}

type Resolver interface {
	Resolve(c *gin.Context) (Scope, error)
}

type ContextResolver struct{}

func (ContextResolver) Resolve(c *gin.Context) (Scope, error) {
	return ResolveScope(c)
}

func ResolveScope(c *gin.Context) (Scope, error) {
	role, _ := c.Get("enterprise_role")
	roleName, _ := role.(string)

	scope := Scope{
		WorkspaceID: enterprisemodels.WorkspaceDefaultID,
		Role:        roleName,
	}

	userValue, ok := c.Get("enterprise_user")
	if !ok {
		if roleName == enterprisemodels.RoleAdmin {
			scope.AllOrgUnits = true
			scope.AllUsers = true
			scope.AllGroups = true
			return scope, nil
		}
		return Scope{}, errors.New("enterprise user is required")
	}

	if user, ok := userValue.(*enterprisemodels.FeishuUser); ok {
		scope.CallerGroupID = user.GroupID
		scope.CallerUserID = user.EnterpriseUserID
		if scope.CallerUserID == "" {
			scope.CallerUserID = user.OpenID
		}
		if user.WorkspaceID != "" {
			scope.WorkspaceID = user.WorkspaceID
		}
	}

	if roleName == enterprisemodels.RoleAdmin {
		scope.AllOrgUnits = true
		scope.AllUsers = true
		scope.AllGroups = true
		return scope, nil
	}

	if scope.CallerGroupID != "" {
		scope.AllowedGroupIDs = []string{scope.CallerGroupID}
	}
	if scope.CallerUserID != "" {
		scope.AllowedUserIDs = []string{scope.CallerUserID}
	}

	return scope, nil
}

func SelectGroupIDs(scope Scope, requested []string) GroupSelection {
	if scope.AllGroups {
		if len(requested) == 0 {
			return GroupSelection{All: true}
		}
		return GroupSelection{IDs: cloneStringSlice(requested)}
	}

	return GroupSelection{IDs: IntersectGroupIDs(scope, requested)}
}

func IntersectGroupIDs(scope Scope, requested []string) []string {
	if scope.AllGroups {
		if len(requested) == 0 {
			return nil
		}
		return requested
	}

	if len(requested) == 0 {
		return cloneStringSlice(scope.AllowedGroupIDs)
	}

	allowed := make(map[string]struct{}, len(scope.AllowedGroupIDs))
	for _, groupID := range scope.AllowedGroupIDs {
		allowed[groupID] = struct{}{}
	}

	intersected := make([]string, 0, len(requested))
	for _, groupID := range requested {
		if _, ok := allowed[groupID]; ok {
			intersected = append(intersected, groupID)
		}
	}

	return intersected
}

func cloneStringSlice(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}

	cloned := make([]string, len(values))
	copy(cloned, values)

	return cloned
}
