package admin

import (
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
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

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	switch duration {
	case "last_week":
		startDate = today.AddDate(0, 0, -7)
		dateFormat = "2006-01-02"
		step = 24 * time.Hour
	case "last_month":
		startDate = today.AddDate(0, -1, 0)
		dateFormat = "2006-01-02"
		step = 24 * time.Hour
	case "last_year":
		startDate = time.Date(now.Year()-1, now.Month(), 1, 0, 0, 0, 0, now.Location())
		dateFormat = "2006-01"
		step = 0
	case "all_time":
		var oldestUser models.User
		if err := db.Order("created_at asc").First(&oldestUser).Error; err == nil {
			startDate = time.Date(oldestUser.CreatedAt.Year(), oldestUser.CreatedAt.Month(), 1, 0, 0, 0, 0, now.Location())
		} else {
			startDate = today.AddDate(-1, 0, 0)
		}
		dateFormat = "2006-01"
		step = 0
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
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	switch duration {
	case "weekly":
		startDate = today.AddDate(0, 0, -7)
		dateFormat = "2006-01-02"
		dbDateFormat = "YYYY-MM-DD"
		step = 24 * time.Hour
	case "monthly":
		startDate = today.AddDate(0, -1, 0)
		dateFormat = "2006-01-02"
		dbDateFormat = "YYYY-MM-DD"
		step = 24 * time.Hour
	case "yearly":
		startDate = time.Date(now.Year()-1, now.Month(), 1, 0, 0, 0, 0, now.Location())
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

type Metric struct {
	Value            float64 `json:"value"`
	PercentageChange float64 `json:"percentage_change"`
}

type OverviewStatsResponse struct {
	Revenue          Metric  `json:"revenue"`
	NewUsers         Metric  `json:"new_users"`
	TotalUsers       Metric  `json:"total_users"`
	ActiveUsers      Metric  `json:"active_users"`
	AvailableCredits float64 `json:"available_credits"`
}

func GetDashboardOverviewStats(db *gorm.DB) (OverviewStatsResponse, int, error) {
	var resp OverviewStatsResponse
	now := time.Now()

	// Dates for Current Month
	currentYear, currentMonth, _ := now.Date()
	currentLocation := now.Location()

	startOfCurrentMonth := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, currentLocation)
	startOfNextMonth := startOfCurrentMonth.AddDate(0, 1, 0)

	// Dates for Last Month
	startOfLastMonth := startOfCurrentMonth.AddDate(0, -1, 0)

	// 1. Revenue
	var currentRevenue float64
	var lastMonthRevenue float64

	db.Model(&models.CreditTransaction{}).
		Where("created_at >= ? AND created_at < ? AND type = ?", startOfCurrentMonth, startOfNextMonth, "Top-up").
		Select("COALESCE(SUM(amount), 0)").Scan(&currentRevenue)

	db.Model(&models.CreditTransaction{}).
		Where("created_at >= ? AND created_at < ? AND type = ?", startOfLastMonth, startOfCurrentMonth, "Top-up").
		Select("COALESCE(SUM(amount), 0)").Scan(&lastMonthRevenue)

	resp.Revenue.Value = currentRevenue
	if lastMonthRevenue > 0 {
		resp.Revenue.PercentageChange = ((currentRevenue - lastMonthRevenue) / lastMonthRevenue) * 100
	} else if currentRevenue > 0 {
		resp.Revenue.PercentageChange = 100
	}

	// 2. New Users
	var currentNewUsers int64
	var lastMonthNewUsers int64

	db.Model(&models.User{}).
		Where("created_at >= ? AND created_at < ?", startOfCurrentMonth, startOfNextMonth).
		Count(&currentNewUsers)

	db.Model(&models.User{}).
		Where("created_at >= ? AND created_at < ?", startOfLastMonth, startOfCurrentMonth).
		Count(&lastMonthNewUsers)

	resp.NewUsers.Value = float64(currentNewUsers)
	if lastMonthNewUsers > 0 {
		resp.NewUsers.PercentageChange = (float64(currentNewUsers-lastMonthNewUsers) / float64(lastMonthNewUsers)) * 100
	} else if currentNewUsers > 0 {
		resp.NewUsers.PercentageChange = 100
	}

	// 3. Total Users
	var totalUsers int64
	db.Model(&models.User{}).Count(&totalUsers)
	resp.TotalUsers.Value = float64(totalUsers)

	totalStartOfMonth := totalUsers - currentNewUsers
	if totalStartOfMonth > 0 {
		resp.TotalUsers.PercentageChange = (float64(currentNewUsers) / float64(totalStartOfMonth)) * 100
	}

	// 4. Active Users
	startOf30DaysAgo := now.Add(-30 * 24 * time.Hour)
	var activeUsers int64
	db.Model(&models.User{}).Where("updated_at >= ?", startOf30DaysAgo).Count(&activeUsers)
	resp.ActiveUsers.Value = float64(activeUsers)

	resp.ActiveUsers.PercentageChange = 0

	// 5. Available Credits
	metrics, err := models.GetPlatformCreditSummary(db)
	if err == nil {
		resp.AvailableCredits = metrics.TotalBalance
	}

	return resp, http.StatusOK, nil
}

type UserGrowthParams struct {
	Preset   string
	From     *time.Time
	To       *time.Time
	GroupBy  string
	Timezone string
}

type UserGrowthResponse struct {
	Period     string                `json:"period,omitempty"`
	StartDate  string                `json:"start_date,omitempty"`
	EndDate    string                `json:"end_date,omitempty"`
	TotalCount int64                 `json:"total_count"`
	Breakdown  []UserGrowthBreakdown `json:"breakdown,omitempty"`
}

type UserGrowthBreakdown struct {
	Period string `json:"period"`
	Date   string `json:"date"`
	Count  int64  `json:"count"`
}

func GetUserGrowthMetrics(db *gorm.DB, params UserGrowthParams) (*UserGrowthResponse, error) {
	// Defaults to Nigerian time if timezone param is not provided
	location, err := time.LoadLocation("Africa/Lagos")
	if err != nil {
		location = time.FixedZone("WAT", 1*60*60)
	}

	// Override with provided timezone if specified
	if params.Timezone != "" {
		loc, locErr := time.LoadLocation(params.Timezone)
		if locErr != nil {
			return nil, fmt.Errorf("invalid timezone: %w", locErr)
		}
		location = loc
	}

	now := time.Now().In(location)
	var startDate, endDate time.Time

	// Determine date range based on preset or custom dates
	if params.Preset != "" {
		startDate, endDate = getPresetDateRange(params.Preset, now)
	} else if params.From != nil && params.To != nil {
		startDate = time.Date(params.From.Year(), params.From.Month(), params.From.Day(), 0, 0, 0, 0, location)
		endDate = time.Date(params.To.Year(), params.To.Month(), params.To.Day(), 23, 59, 59, 999999999, location)
	} else {
		return nil, fmt.Errorf("either preset or from/to dates must be provided")
	}

	response := &UserGrowthResponse{
		StartDate: startDate.Format("2006-01-02"),
		EndDate:   endDate.Format("2006-01-02"),
	}

	if params.Preset != "" {
		response.Period = params.Preset
	}

	// Get total count
	var totalCount int64
	if err := db.Model(&models.User{}).
		Where("created_at >= ? AND created_at <= ? AND deleted_at IS NULL", startDate.UTC(), endDate.UTC()).
		Count(&totalCount).Error; err != nil {
		return nil, fmt.Errorf("failed to get total user count: %w", err)
	}
	response.TotalCount = totalCount

	// If group_by is specified, get breakdown
	if params.GroupBy != "" {
		breakdown, err := getBreakdown(db, startDate, endDate, params.GroupBy, location)
		if err != nil {
			return nil, fmt.Errorf("failed to get breakdown: %w", err)
		}
		response.Breakdown = breakdown
	}

	return response, nil
}

func getPresetDateRange(preset string, now time.Time) (time.Time, time.Time) {
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())

	switch preset {
	case "today":
		return todayStart, todayEnd

	case "yesterday":
		yesterdayStart := todayStart.AddDate(0, 0, -1)
		yesterdayEnd := yesterdayStart.Add(24*time.Hour - time.Nanosecond)
		return yesterdayStart, yesterdayEnd

	case "last_7_days":
		startDate := todayStart.AddDate(0, 0, -6) // Last 7 days including today
		return startDate, todayEnd

	case "last_30_days":
		startDate := todayStart.AddDate(0, 0, -29) // Last 30 days including today
		return startDate, todayEnd

	case "this_month":
		monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return monthStart, todayEnd

	case "this_year":
		yearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		return yearStart, todayEnd

	default:
		return todayStart, todayEnd
	}
}

func getBreakdown(db *gorm.DB, startDate, endDate time.Time, groupBy string, location *time.Location) ([]UserGrowthBreakdown, error) {
	switch groupBy {
	case "day":
		return getDailyBreakdown(db, startDate, endDate, location)
	case "week":
		return getWeeklyBreakdown(db, startDate, endDate, location)
	case "month":
		return getMonthlyBreakdown(db, startDate, endDate, location)
	default:
		return nil, fmt.Errorf("invalid group_by value: %s", groupBy)
	}
}

func getDailyBreakdown(db *gorm.DB, startDate, endDate time.Time, location *time.Location) ([]UserGrowthBreakdown, error) {
	var breakdown []UserGrowthBreakdown

	// Normalize dates to compare only date parts, not timestamps
	currentDate := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, location)
	endDateOnly := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, location)

	// Fixed: Use !After instead of Before/Equal to properly include the last day
	for !currentDate.After(endDateOnly) {
		dayStart := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(), 0, 0, 0, 0, location)
		dayEnd := dayStart.Add(24*time.Hour - time.Nanosecond)

		var count int64
		if err := db.Model(&models.User{}).
			Where("created_at >= ? AND created_at <= ? AND deleted_at IS NULL", dayStart.UTC(), dayEnd.UTC()).
			Count(&count).Error; err != nil {
			return nil, fmt.Errorf("failed to count users for day %s: %w", dayStart.Format("2006-01-02"), err)
		}

		breakdown = append(breakdown, UserGrowthBreakdown{
			Period: "day",
			Date:   dayStart.Format("2006-01-02"),
			Count:  count,
		})

		currentDate = currentDate.AddDate(0, 0, 1)
	}

	return breakdown, nil
}

func getWeeklyBreakdown(db *gorm.DB, startDate, endDate time.Time, location *time.Location) ([]UserGrowthBreakdown, error) {
	var breakdown []UserGrowthBreakdown

	// Start from the beginning of the week containing startDate
	weekday := int(startDate.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weekStart := startDate.AddDate(0, 0, -(weekday - 1))
	weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, location)

	for weekStart.Before(endDate) || weekStart.Equal(endDate) {
		weekEnd := weekStart.AddDate(0, 0, 6)
		weekEnd = time.Date(weekEnd.Year(), weekEnd.Month(), weekEnd.Day(), 23, 59, 59, 999999999, location)

		// Adjust if week extends beyond endDate
		if weekEnd.After(endDate) {
			weekEnd = endDate
		}

		var count int64
		if err := db.Model(&models.User{}).
			Where("created_at >= ? AND created_at <= ? AND deleted_at IS NULL", weekStart.UTC(), weekEnd.UTC()).
			Count(&count).Error; err != nil {
			return nil, err
		}

		breakdown = append(breakdown, UserGrowthBreakdown{
			Period: "week",
			Date:   weekStart.Format("2006-01-02"),
			Count:  count,
		})

		weekStart = weekStart.AddDate(0, 0, 7)
	}

	return breakdown, nil
}

func getMonthlyBreakdown(db *gorm.DB, startDate, endDate time.Time, location *time.Location) ([]UserGrowthBreakdown, error) {
	var breakdown []UserGrowthBreakdown

	// Start from the beginning of the month containing startDate
	monthStart := time.Date(startDate.Year(), startDate.Month(), 1, 0, 0, 0, 0, location)

	for monthStart.Before(endDate) || monthStart.Equal(endDate) {
		// Calculate the last day of the month
		nextMonth := monthStart.AddDate(0, 1, 0)
		monthEnd := nextMonth.Add(-time.Nanosecond)

		// Adjust if month extends beyond endDate
		if monthEnd.After(endDate) {
			monthEnd = endDate
		}

		var count int64
		if err := db.Model(&models.User{}).
			Where("created_at >= ? AND created_at <= ? AND deleted_at IS NULL", monthStart.UTC(), monthEnd.UTC()).
			Count(&count).Error; err != nil {
			return nil, err
		}

		breakdown = append(breakdown, UserGrowthBreakdown{
			Period: "month",
			Date:   monthStart.Format("2006-01"),
			Count:  count,
		})

		monthStart = nextMonth
	}

	return breakdown, nil
}
