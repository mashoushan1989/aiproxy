//go:build enterprise

package quota

import (
	"time"

	"github.com/labring/aiproxy/core/enterprise/models"
)

// PeriodStartByType returns the calendar-aligned start of the current period
// for a given policy PeriodType (1=daily, 2=weekly, 3=monthly).
func PeriodStartByType(periodType int) time.Time {
	return models.PeriodStartByType(periodType)
}
