package admin

import (
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"
)

type ChartDataPoint struct {
	Date      string `json:"date"`
	FreeCount int64  `json:"free"`
	PaidCount int64  `json:"paid"`
}

type FreeVsPaidStatsResponse struct {
	Filter string           `json:"filter"`
	Data   []ChartDataPoint `json:"data"`
}

func GetFreeVsPaidUserStats(db *gorm.DB, duration string) (FreeVsPaidStatsResponse, int, error) {
	var resp FreeVsPaidStatsResponse
	resp.Filter = duration

	now := time.Now()
	var startDate time.Time
	var dateFormat string
	var step time.Duration

	switch duration {
	case "last_week":
		startDate = now.AddDate(0, 0, -7)
		dateFormat = "2006-01-02"
		step = 24 * time.Hour
	case "last_month":
		startDate = now.AddDate(0, -1, 0)
		dateFormat = "2006-01-02"
		step = 24 * time.Hour
	case "last_year":
		startDate = now.AddDate(-1, 0, 0)
		dateFormat = "2006-01"
		step = 0 // handled differently
	default:
		// Default to last month
		duration = "last_month"
		resp.Filter = duration
		startDate = now.AddDate(0, -1, 0)
		dateFormat = "2006-01-02"
		step = 24 * time.Hour
	}

	dataMap := make(map[string]*ChartDataPoint)

	// This ensures the chart has no gaps
	current := startDate
	for !current.After(now) {
		dateStr := current.Format(dateFormat)
		dataMap[dateStr] = &ChartDataPoint{Date: dateStr, FreeCount: 0, PaidCount: 0}

		if duration == "last_year" {
			current = current.AddDate(0, 1, 0)
		} else {
			current = current.Add(step)
		}
	}

	type Result struct {
		Date  string
		Count int64
	}

	var totalResults []Result
	var paidResults []Result

	dbDateFormat := "YYYY-MM-DD"
	if duration == "last_year" {
		dbDateFormat = "YYYY-MM"
	}

	totalQuery := db.Table("users").
		Select(fmt.Sprintf("TO_CHAR(created_at, '%s') as date, COUNT(*) as count", dbDateFormat)).
		Where("created_at >= ? AND deleted_at IS NULL", startDate).
		Group(fmt.Sprintf("TO_CHAR(created_at, '%s')", dbDateFormat))

	if err := totalQuery.Scan(&totalResults).Error; err != nil {
		return resp, http.StatusInternalServerError, err
	}

	paidQuery := db.Table("users").
		Select(fmt.Sprintf("TO_CHAR(users.created_at, '%s') as date, COUNT(distinct users.id) as count", dbDateFormat)).
		Joins("JOIN organisations ON organisations.owner_id = users.id").
		Where("organisations.subscription_plan_id != 'free' AND organisations.subscription_plan_id != ''").
		Where("users.created_at >= ? AND users.deleted_at IS NULL", startDate).
		Group(fmt.Sprintf("TO_CHAR(users.created_at, '%s')", dbDateFormat))

	if err := paidQuery.Scan(&paidResults).Error; err != nil {
		return resp, http.StatusInternalServerError, err
	}

	for _, r := range totalResults {
		if pt, ok := dataMap[r.Date]; ok {
			pt.FreeCount = r.Count 
		}
	}

	for _, r := range paidResults {
		if pt, ok := dataMap[r.Date]; ok {
			pt.PaidCount = r.Count
			pt.FreeCount -= r.Count 
			if pt.FreeCount < 0 {
				pt.FreeCount = 0
			}
		}
	}

	// Re-iterate time range to preserve order
	current = startDate
	for !current.After(now) {
		dateStr := current.Format(dateFormat)
		if pt, ok := dataMap[dateStr]; ok {
			resp.Data = append(resp.Data, *pt)
		}

		if duration == "last_year" {
			current = current.AddDate(0, 1, 0)
		} else {
			current = current.Add(step)
		}
	}

	return resp, http.StatusOK, nil
}
