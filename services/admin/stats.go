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

type AICreditDataPoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

type AICreditStatsResponse struct {
	Filter           string              `json:"filter"`
	Unit             string              `json:"unit"`
	TotalConsumed    float64             `json:"total_consumed"`
	PercentageChange float64             `json:"percentage_change"`
	Data             []AICreditDataPoint `json:"data"`
}

func GetAICreditUsageStats(db *gorm.DB, duration, unit string) (AICreditStatsResponse, int, error) {
	var resp AICreditStatsResponse
	resp.Filter = duration
	resp.Unit = unit

	now := time.Now()
	var startDate time.Time
	var dateFormat string
	var step time.Duration
	var dbDateFormat string

	// 1. Determine Date Range
	switch duration {
	case "weekly":
		startDate = now.AddDate(0, 0, -7)
		dateFormat = "2006-01-02"
		dbDateFormat = "YYYY-MM-DD"
		step = 24 * time.Hour
	case "monthly":
		startDate = now.AddDate(0, -1, 0)
		dateFormat = "2006-01-02"
		dbDateFormat = "YYYY-MM-DD"
		step = 24 * time.Hour
	case "yearly":
		startDate = now.AddDate(-1, 0, 0)
		dateFormat = "2006-01"
		dbDateFormat = "YYYY-MM"
		step = 0
	case "daily":
		startDate = now.Add(-24 * time.Hour)
		dateFormat = "15:04"
		dbDateFormat = "HH24:00"
		step = 1 * time.Hour
	default:
		duration = "monthly"
		resp.Filter = duration
		startDate = now.AddDate(0, -1, 0)
		dateFormat = "2006-01-02"
		dbDateFormat = "YYYY-MM-DD"
		step = 24 * time.Hour
	}

	dataMap := make(map[string]float64)
	current := startDate

	for !current.After(now) {
		dateStr := current.Format(dateFormat)
		dataMap[dateStr] = 0

		switch duration {
		case "yearly":
			current = current.AddDate(0, 1, 0)
		case "daily":
			current = current.Add(time.Hour)
		default:
			current = current.Add(step)
		}
	}

	type Result struct {
		Date  string
		Total float64
	}
	var results []Result

	query := db.Table("credit_usages").
		Select(fmt.Sprintf("TO_CHAR(created_at, '%s') as date, COALESCE(SUM(amount), 0) as total", dbDateFormat)).
		Where("created_at >= ?", startDate).
		Group(fmt.Sprintf("TO_CHAR(created_at, '%s')", dbDateFormat))

	if err := query.Scan(&results).Error; err != nil {
		return resp, http.StatusInternalServerError, err
	}

	// 4. Fill Data
	for _, r := range results {
		if _, ok := dataMap[r.Date]; ok {
			dataMap[r.Date] = r.Total
		}
	}

	// 5. Calculate Stats (Total & Trend)
	var totalCurrent float64
	for _, v := range dataMap {
		totalCurrent += v
	}
	resp.TotalConsumed = totalCurrent

	// Calculate Previous Period Total for Trend
	var previousTotal float64
	var prevStartDate time.Time

	switch duration {
	case "weekly":
		prevStartDate = startDate.AddDate(0, 0, -7)
	case "monthly":
		prevStartDate = startDate.AddDate(0, -1, 0)
	case "yearly":
		prevStartDate = startDate.AddDate(-1, 0, 0)
	case "daily":
		prevStartDate = startDate.Add(-24 * time.Hour)
	}

	err := db.Table("credit_usages").
		Select("COALESCE(SUM(amount), 0)").
		Where("created_at >= ? AND created_at < ?", prevStartDate, startDate).
		Scan(&previousTotal).Error

	if err == nil {
		if previousTotal > 0 {
			resp.PercentageChange = ((totalCurrent - previousTotal) / previousTotal) * 100
		} else if totalCurrent > 0 {
			resp.PercentageChange = 100
		}
	}

	// 6. Handle Price Unit Conversion
	multiplier := 1.0
	if unit == "price" {
		multiplier = 0.02
		resp.TotalConsumed *= multiplier
		// Percentage change stays same
	}

	// 7. Format Output sorted
	current = startDate
	for !current.After(now) {
		dateStr := current.Format(dateFormat)
		if val, ok := dataMap[dateStr]; ok {
			finalVal := val * multiplier
			resp.Data = append(resp.Data, AICreditDataPoint{Date: dateStr, Value: finalVal})
		}

		switch duration {
		case "yearly":
			current = current.AddDate(0, 1, 0)
		case "daily":
			current = current.Add(time.Hour)
		default:
			current = current.Add(step)
		}
	}

	return resp, http.StatusOK, nil
}
