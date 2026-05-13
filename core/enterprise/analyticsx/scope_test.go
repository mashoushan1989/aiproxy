//go:build enterprise

package analyticsx

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	enterprisemodels "github.com/labring/aiproxy/core/enterprise/models"
)

func TestResolveScopeAdminIsGlobal(t *testing.T) {
	c := newScopeContext()
	c.Set("enterprise_role", enterprisemodels.RoleAdmin)

	scope, err := ResolveScope(c)
	if err != nil {
		t.Fatalf("ResolveScope() error = %v", err)
	}

	if !scope.AllGroups || !scope.AllOrgUnits || !scope.AllUsers {
		t.Fatalf("admin scope = %#v, want global access", scope)
	}
}

func TestResolveScopeMemberUsesOwnGroup(t *testing.T) {
	c := newScopeContext()
	c.Set("enterprise_role", enterprisemodels.RoleViewer)
	c.Set("enterprise_user", &enterprisemodels.FeishuUser{
		OpenID:  "open-a",
		GroupID: "group-a",
	})

	scope, err := ResolveScope(c)
	if err != nil {
		t.Fatalf("ResolveScope() error = %v", err)
	}

	want := []string{"group-a"}
	if !reflect.DeepEqual(scope.AllowedGroupIDs, want) {
		t.Fatalf("AllowedGroupIDs = %#v, want %#v", scope.AllowedGroupIDs, want)
	}
}

func TestAdminScopeIsGlobal(t *testing.T) {
	scope := Scope{
		Role:      "admin",
		AllGroups: true,
	}

	if got := SelectGroupIDs(scope, nil); !got.All || got.IDs != nil {
		t.Fatalf("empty request for global scope = %#v, want all selection", got)
	}

	requested := []string{"group-a", "group-b"}
	if got := SelectGroupIDs(scope, requested); got.All || !reflect.DeepEqual(got.IDs, requested) {
		t.Fatalf("requested groups for global scope = %#v, want restricted requested groups", got)
	}
}

func TestMemberScopeUsesOwnGroup(t *testing.T) {
	scope := Scope{
		Role:            "member",
		CallerGroupID:   "group-a",
		AllowedGroupIDs: []string{"group-a"},
	}

	want := []string{"group-a"}
	if got := IntersectGroupIDs(scope, nil); !reflect.DeepEqual(got, want) {
		t.Fatalf("empty request for member scope = %#v, want %#v", got, want)
	}
}

func TestIntersectGroupsDoesNotWidenAccess(t *testing.T) {
	scope := Scope{
		Role:            "member",
		AllowedGroupIDs: []string{"group-a", "group-b"},
	}

	want := []string{"group-b"}
	if got := IntersectGroupIDs(scope, []string{"group-b", "group-c"}); !reflect.DeepEqual(got, want) {
		t.Fatalf("requested intersection = %#v, want %#v", got, want)
	}

	got := IntersectGroupIDs(scope, []string{"group-c"})
	if got == nil {
		t.Fatal("empty intersection returned nil, want empty non-nil slice")
	}
	if len(got) != 0 {
		t.Fatalf("disallowed request = %#v, want empty slice", got)
	}
}

func TestIntersectGroupsEmptyRequestMeansAllowedGroups(t *testing.T) {
	scope := Scope{
		Role:            "member",
		AllowedGroupIDs: []string{"group-a", "group-b"},
	}

	want := []string{"group-a", "group-b"}
	if got := IntersectGroupIDs(scope, []string{}); !reflect.DeepEqual(got, want) {
		t.Fatalf("empty request = %#v, want %#v", got, want)
	}
}

func newScopeContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/analytics", nil)
	return c
}
