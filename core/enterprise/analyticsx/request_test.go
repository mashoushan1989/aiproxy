//go:build enterprise

package analyticsx

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func newParseRequestContext(t *testing.T, values url.Values) *gin.Context {
	t.Helper()

	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest(http.MethodGet, "/analytics?"+values.Encode(), nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	return c
}

func TestParseRequestDefaultsToSevenDays(t *testing.T) {
	before := time.Now().Unix()

	filter, err := ParseRequest(newParseRequestContext(t, nil))
	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}

	after := time.Now().Unix()
	wantRange := int64(DefaultRangeDays * 24 * 60 * 60)

	if filter.EndTimestamp < before || filter.EndTimestamp > after {
		t.Fatalf("EndTimestamp = %d, want between %d and %d", filter.EndTimestamp, before, after)
	}
	if got := filter.EndTimestamp - filter.StartTimestamp; got != wantRange {
		t.Fatalf("range = %d seconds, want %d", got, wantRange)
	}
	if filter.Page != 1 {
		t.Fatalf("Page = %d, want 1", filter.Page)
	}
	if filter.PerPage <= 0 || filter.PerPage > MaxPageSize {
		t.Fatalf("PerPage = %d, want positive value no greater than %d", filter.PerPage, MaxPageSize)
	}
}

func TestParseRequestRejectsEndBeforeStart(t *testing.T) {
	query := url.Values{}
	query.Set("start_timestamp", "200")
	query.Set("end_timestamp", "100")

	if _, err := ParseRequest(newParseRequestContext(t, query)); err == nil {
		t.Fatal("ParseRequest() error = nil, want error")
	}
}

func TestParseRequestRejectsInteractiveRangeOverNinetyDays(t *testing.T) {
	query := url.Values{}
	query.Set("start_timestamp", "0")
	query.Set("end_timestamp", "7862401")
	query.Set("granularity", "daily")

	if _, err := ParseRequest(newParseRequestContext(t, query)); err == nil {
		t.Fatal("ParseRequest() error = nil, want error")
	}
}

func TestParseRequestRequiresDailyGranularityForLongHourlyRange(t *testing.T) {
	query := url.Values{}
	query.Set("start_timestamp", "0")
	query.Set("end_timestamp", "2678401")
	query.Set("granularity", "hourly")

	if _, err := ParseRequest(newParseRequestContext(t, query)); err == nil {
		t.Fatal("ParseRequest() error = nil, want error")
	}
}

func TestParseRequestRejectsUnknownGranularity(t *testing.T) {
	query := url.Values{}
	query.Set("granularity", "raw_sql")

	if _, err := ParseRequest(newParseRequestContext(t, query)); err == nil {
		t.Fatal("ParseRequest() error = nil, want error")
	}
}

func TestParseRequestClampsPagination(t *testing.T) {
	query := url.Values{}
	query.Set("limit", "5000")
	query.Set("page", "0")
	query.Set("per_page", "5000")

	filter, err := ParseRequest(newParseRequestContext(t, query))
	if err != nil {
		t.Fatalf("ParseRequest() error = %v", err)
	}

	if filter.Limit != MaxLimit {
		t.Fatalf("Limit = %d, want %d", filter.Limit, MaxLimit)
	}
	if filter.Page != 1 {
		t.Fatalf("Page = %d, want 1", filter.Page)
	}
	if filter.PerPage != MaxPageSize {
		t.Fatalf("PerPage = %d, want %d", filter.PerPage, MaxPageSize)
	}
}
