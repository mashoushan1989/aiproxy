//go:build enterprise

package analyticsx

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	DefaultRangeDays        = 7
	MaxInteractiveRangeDays = 90
	MaxHourlyRangeDays      = 31
	MaxLimit                = 1000
	MaxPageSize             = 500

	defaultPerPage = 50
)

type Filter struct {
	StartTimestamp int64
	EndTimestamp   int64
	Granularity    string
	OrgUnitIDs     []string
	GroupIDs       []string
	UserIDs        []string
	Models         []string
	Limit          int
	Page           int
	PerPage        int
}

func ParseRequest(c *gin.Context) (Filter, error) {
	now := time.Now().Unix()
	granularity := strings.ToLower(c.DefaultQuery("granularity", "daily"))
	if !validGranularity(granularity) {
		return Filter{}, fmt.Errorf("invalid granularity: %s", granularity)
	}

	filter := Filter{
		StartTimestamp: now - daysToSeconds(DefaultRangeDays),
		EndTimestamp:   now,
		Granularity:    granularity,
		OrgUnitIDs:     c.QueryArray("org_unit_id"),
		GroupIDs:       c.QueryArray("group_id"),
		UserIDs:        c.QueryArray("user_id"),
		Models:         c.QueryArray("model"),
		Page:           1,
		PerPage:        defaultPerPage,
	}

	var err error
	if filter.StartTimestamp, err = parseOptionalInt64(c, "start_timestamp", filter.StartTimestamp); err != nil {
		return Filter{}, err
	}
	if filter.EndTimestamp, err = parseOptionalInt64(c, "end_timestamp", filter.EndTimestamp); err != nil {
		return Filter{}, err
	}
	if filter.Limit, err = parseOptionalInt(c, "limit", 0); err != nil {
		return Filter{}, err
	}
	if filter.Page, err = parseOptionalInt(c, "page", filter.Page); err != nil {
		return Filter{}, err
	}
	if filter.PerPage, err = parseOptionalInt(c, "per_page", filter.PerPage); err != nil {
		return Filter{}, err
	}

	if filter.EndTimestamp < filter.StartTimestamp {
		return Filter{}, fmt.Errorf("end_timestamp cannot be before start_timestamp")
	}

	rangeSeconds := filter.EndTimestamp - filter.StartTimestamp
	if rangeSeconds > daysToSeconds(MaxInteractiveRangeDays) {
		return Filter{}, fmt.Errorf("time range cannot exceed %d days", MaxInteractiveRangeDays)
	}
	if filter.Granularity == "hourly" && rangeSeconds > daysToSeconds(MaxHourlyRangeDays) {
		return Filter{}, fmt.Errorf("hourly granularity cannot exceed %d days; use daily granularity", MaxHourlyRangeDays)
	}

	filter.Limit = clampPositiveMax(filter.Limit, MaxLimit)
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage <= 0 {
		filter.PerPage = defaultPerPage
	}
	if filter.PerPage > MaxPageSize {
		filter.PerPage = MaxPageSize
	}

	return filter, nil
}

func parseOptionalInt64(c *gin.Context, key string, fallback int64) (int64, error) {
	value := c.Query(key)
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}

	return parsed, nil
}

func parseOptionalInt(c *gin.Context, key string, fallback int) (int, error) {
	value := c.Query(key)
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}

	return parsed, nil
}

func clampPositiveMax(value, maxValue int) int {
	if value <= 0 {
		return 0
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func daysToSeconds(days int) int64 {
	return int64(days) * int64(24*time.Hour/time.Second)
}

func validGranularity(value string) bool {
	switch value {
	case "hourly", "daily", "monthly":
		return true
	default:
		return false
	}
}
