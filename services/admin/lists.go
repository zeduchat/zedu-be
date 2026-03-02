package admin

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/avatar"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/services/user"
)

const (
	SubscriptionStatusFree = "Free"
	SubscriptionStatusPaid = "Paid"
)

type UserListItem struct {
	ID                 string  `json:"id"`
	Email              string  `json:"email"`
	Name               string  `json:"name"`
	AvatarUrl          string  `json:"avatar_url"`
	DefaultAvatarUrl   string  `json:"default_avatar_url"`
	CreatedAt          string  `json:"created_at"`
	LastLogInAt        string  `json:"last_log_in_at"`
	LastActivityAt     string  `json:"last_activity_at"`
	ActivityLength     string  `json:"activity_length"`
	Referrals          int64   `json:"referrals"`
	CreditUsed         float64 `json:"credit_used"`
	AmountSpent        float64 `json:"amount_spent"`
	SubscriptionStatus string  `json:"subscription_status"`
}

type UserStats struct {
	TotalSignups                  int64   `json:"total_signups"`
	TotalGrowthNum                float64 `json:"total_growth_num"`
	NewToday                      int64   `json:"new_today"`
	NewTodayPercent               float64 `json:"new_today_percent"`
	Yesterday                     int64   `json:"yesterday"`
	ThisWeek                      int64   `json:"this_week"`
	FreeUsers                     int64   `json:"free_users"`
	FreeUsersChangePercent        float64 `json:"free_users_change_percent"`
	PaidUsers                     int64   `json:"paid_users"`
	PaidUsersChangePercent        float64 `json:"paid_users_change_percent"`
	NewUsersMonth                 int64   `json:"new_users_month"`
	NewUsersMonthChangePercent    float64 `json:"new_users_month_change_percent"`
	ActiveUsersMonth              int64   `json:"active_users_month"`
	ActiveUsersMonthChangePercent float64 `json:"active_users_month_change_percent"`
	AvgSessionLengthMonth         string  `json:"avg_session_length_month"`          // Last 30 days average session length
	AvgSessionLengthChangePercent float64 `json:"avg_session_length_change_percent"` // 30-day rolling comparison: percent change from previous 30 days
}

type ListUsersResponse struct {
	Stats UserStats      `json:"stats"`
	Users []UserListItem `json:"users"`
}

type UserFilter struct {
	Search       string
	UserType     string
	Duration     string
	StartDate    string
	EndDate      string
	IncludeStats bool
}

func ListUsers(db *gorm.DB, c *gin.Context, filter UserFilter) (ListUsersResponse, postgresql.PaginationResponse, int, error) {
	var users []models.User
	pagination := postgresql.GetPagination(c)

	query := db.Preload("Profile").Model(&models.User{})

	if filter.Search != "" {
		search := "%" + filter.Search + "%"
		profileSubQuery := db.Table("profiles").
			Select("userid").
			Where("first_name ILIKE ? OR last_name ILIKE ?", search, search)

		query = query.Where(
			"users.name ILIKE ? OR users.email ILIKE ? OR users.id IN (?)",
			search, search, profileSubQuery,
		)
	}

	// Duration preset mapping
	now := time.Now()
	switch filter.Duration {
	case "last_3_months":
		filter.StartDate = now.AddDate(0, -3, 0).Format("2006-01-02")
	case "last_month":
		filter.StartDate = now.AddDate(0, -1, 0).Format("2006-01-02")
	case "last_year":
		filter.StartDate = now.AddDate(-1, 0, 0).Format("2006-01-02")
	case "custom":
		// Use provided StartDate/EndDate as-is
	default:
		// "all_time" or empty - no date filter
	}

	if filter.StartDate != "" {
		query = query.Where("created_at >= ?", filter.StartDate)
	}
	if filter.EndDate != "" {
		query = query.Where("created_at <= ?", filter.EndDate)
	}

	switch filter.UserType {
	case "active":
		thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
		query = query.Where("last_activity_at >= ?", thirtyDaysAgo)
	case "paying":
		query = query.Joins("JOIN organisations ON organisations.owner_id = users.id").
			Where("organisations.subscription_plan_id != 'free' AND organisations.subscription_plan_id != ''").
			Group("users.id")
	case "free":
		subQuery := db.Table("organisations").Select("owner_id").Where("subscription_plan_id != 'free' AND subscription_plan_id != ''")
		query = query.Where("users.id NOT IN (?)", subQuery)
	}

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		"created_at",
		"desc",
		pagination,
		&users,
		"users.deleted_at IS NULL",
	)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return ListUsersResponse{}, paginationResponse, http.StatusNoContent, nil
		}
		return ListUsersResponse{}, paginationResponse, http.StatusBadRequest, err
	}

	var stats UserStats
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	if filter.IncludeStats {
		var wg sync.WaitGroup

		startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		startOfYesterday := startOfToday.AddDate(0, 0, -1)
		startOfWeek := now.AddDate(0, 0, -int(now.Weekday()))
		last30Days := now.AddDate(0, 0, -30)
		last60Days := now.AddDate(0, 0, -60)

		var currentMonthCount, lastMonthCount int64
		var lastMonthPaid, lastMonthTotal int64
		var avgThis, avgLast float64

		wg.Add(1)
		go func() {
			defer wg.Done()
			db.Model(&models.User{}).Where("deleted_at IS NULL").Count(&stats.TotalSignups)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			db.Model(&models.User{}).Where("created_at >= ? AND deleted_at IS NULL", last30Days).Count(&currentMonthCount)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			db.Model(&models.User{}).Where("created_at >= ? AND created_at < ? AND deleted_at IS NULL", last60Days, last30Days).Count(&lastMonthCount)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			db.Model(&models.User{}).Where("created_at >= ? AND deleted_at IS NULL", startOfToday).Count(&stats.NewToday)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			db.Model(&models.AccessToken{}).
				Select("COALESCE(AVG(EXTRACT(EPOCH FROM updated_at - created_at)),0)").
				Where("created_at >= ?", last30Days).
				Scan(&avgThis)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			db.Model(&models.AccessToken{}).
				Select("COALESCE(AVG(EXTRACT(EPOCH FROM updated_at - created_at)),0)").
				Where("created_at >= ? AND created_at < ?", last60Days, last30Days).
				Scan(&avgLast)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			db.Model(&models.User{}).Where("created_at >= ? AND created_at < ? AND deleted_at IS NULL", startOfYesterday, startOfToday).Count(&stats.Yesterday)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			db.Model(&models.User{}).Where("created_at >= ? AND deleted_at IS NULL", startOfWeek).Count(&stats.ThisWeek)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			db.Table("users").
				Joins("JOIN organisations ON organisations.owner_id = users.id").
				Where("organisations.subscription_plan_id != 'free' AND organisations.subscription_plan_id != ''").
				Where("users.deleted_at IS NULL").
				Group("users.id").
				Count(&stats.PaidUsers)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			db.Table("users").
				Joins("JOIN organisations ON organisations.owner_id = users.id").
				Where("organisations.subscription_plan_id != 'free' AND organisations.subscription_plan_id != ''").
				Where("users.created_at < ? AND users.deleted_at IS NULL", last30Days).
				Group("users.id").
				Count(&lastMonthPaid)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			db.Model(&models.User{}).Where("created_at < ? AND deleted_at IS NULL", last30Days).Count(&lastMonthTotal)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			db.Model(&models.User{}).Where("last_activity_at >= ? AND deleted_at IS NULL", thirtyDaysAgo).Count(&stats.ActiveUsersMonth)
		}()

		wg.Wait()

		if lastMonthCount > 0 {
			stats.TotalGrowthNum = float64(currentMonthCount-lastMonthCount) / float64(lastMonthCount) * 100
		} else if currentMonthCount > 0 {
			stats.TotalGrowthNum = 100
		}

		if stats.Yesterday > 0 {
			stats.NewTodayPercent = float64(stats.NewToday-stats.Yesterday) / float64(stats.Yesterday) * 100
		} else if stats.NewToday > 0 {
			stats.NewTodayPercent = 100
		}

		stats.FreeUsers = stats.TotalSignups - stats.PaidUsers

		lastMonthFree := lastMonthTotal - lastMonthPaid
		if lastMonthPaid > 0 {
			stats.PaidUsersChangePercent = float64(stats.PaidUsers-lastMonthPaid) / float64(lastMonthPaid) * 100
		}
		if lastMonthFree > 0 {
			stats.FreeUsersChangePercent = float64(stats.FreeUsers-lastMonthFree) / float64(lastMonthFree) * 100
		}

		stats.NewUsersMonth = currentMonthCount
		stats.NewUsersMonthChangePercent = stats.TotalGrowthNum

		stats.AvgSessionLengthMonth = formatSecs(avgThis)
		if avgLast > 0 {
			stats.AvgSessionLengthChangePercent = (avgThis - avgLast) / avgLast * 100
		}
	}

	if len(users) == 0 {
		return ListUsersResponse{Stats: stats, Users: []UserListItem{}}, paginationResponse, http.StatusOK, nil
	}

	userIDs := getUserIDs(users)

	type referralCount struct {
		UserID string
		Count  int64
	}

	var referralCounts []referralCount
	if err := db.Model(&models.Invitation{}).
		Select("invited_by as user_id, COUNT(*) as count").
		Where("invited_by IN ?", userIDs).
		Group("invited_by").
		Scan(&referralCounts).Error; err != nil {
		referralCounts = []referralCount{}
	}

	referralMap := make(map[string]int64)
	for _, rc := range referralCounts {
		referralMap[rc.UserID] = rc.Count
	}

	type creditUsageSum struct {
		UserID string
		Total  float64
	}
	var creditUsages []creditUsageSum
	if err := db.Model(&models.CreditUsage{}).
		Select("user_id, COALESCE(SUM(amount), 0) as total").
		Where("user_id IN ?", userIDs).
		Group("user_id").
		Scan(&creditUsages).Error; err != nil {
		creditUsages = []creditUsageSum{}
	}

	creditUsageMap := make(map[string]float64)
	for _, cu := range creditUsages {
		creditUsageMap[cu.UserID] = cu.Total
	}

	type userSubscription struct {
		UserID             string
		SubscriptionPlanId string
	}
	var userSubscriptions []userSubscription
	if err := db.Table("organisations").
		Select("owner_id as user_id, subscription_plan_id").
		Where("owner_id IN ?", userIDs).
		Where("subscription_plan_id != 'free' AND subscription_plan_id != ''").
		Scan(&userSubscriptions).Error; err != nil {
		userSubscriptions = []userSubscription{}
	}

	subscriptionMap := make(map[string]string)
	for _, us := range userSubscriptions {
		subscriptionMap[us.UserID] = "Paid"
	}

	type amountSpentSum struct {
		UserID string
		Total  float64
	}
	var amountSpents []amountSpentSum
	if err := db.Table("credit_transactions").
		Select("organisations.owner_id as user_id, COALESCE(SUM(credit_transactions.amount), 0) as total").
		Joins("JOIN organisations ON organisations.id = credit_transactions.organisation_id").
		Where("organisations.owner_id IN ?", userIDs).
		Group("organisations.owner_id").
		Scan(&amountSpents).Error; err != nil {
		amountSpents = []amountSpentSum{}
	}

	amountSpentMap := make(map[string]float64)
	for _, as := range amountSpents {
		amountSpentMap[as.UserID] = as.Total
	}

	resp := make([]UserListItem, 0, len(users))
	for _, u := range users {
		var lastLogInAt string
		if u.LastLogInAt != nil {
			lastLogInAt = u.LastLogInAt.Format("2006-01-02T15:04:05Z07:00")
		}

		var lastActivityAt string
		if u.LastActivityAt != nil {
			lastActivityAt = u.LastActivityAt.Format("2006-01-02T15:04:05Z07:00")
		}

		avatarUrl := u.Profile.AvatarURL
		defaultAvatarUrl := avatar.GenerateDefaultAvatarURL(u.ID)

		var activityLength string
		if al := user.GetActivityLength(u); al != nil {
			activityLength = *al
		}

		subscriptionStatus := "Free"
		if status, ok := subscriptionMap[u.ID]; ok {
			subscriptionStatus = status
		}

		resp = append(resp, UserListItem{
			ID:                 u.ID,
			Email:              u.Email,
			Name:               u.Name,
			AvatarUrl:          avatarUrl,
			DefaultAvatarUrl:   defaultAvatarUrl,
			CreatedAt:          u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			LastLogInAt:        lastLogInAt,
			LastActivityAt:     lastActivityAt,
			ActivityLength:     activityLength,
			Referrals:          referralMap[u.ID],
			CreditUsed:         creditUsageMap[u.ID],
			AmountSpent:        amountSpentMap[u.ID],
			SubscriptionStatus: subscriptionStatus,
		})
	}

	return ListUsersResponse{
		Stats: stats,
		Users: resp,
	}, paginationResponse, http.StatusOK, nil
}

// formatSecs converts a duration in seconds to a human-readable string
func formatSecs(sec float64) string {
	if sec <= 0 {
		return "0s"
	}
	d := time.Duration(sec) * time.Second
	mins := int(d.Minutes())
	secs := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", mins, secs)
}

func getUserIDs(users []models.User) []string {
	userIDs := make([]string, len(users))
	for i, u := range users {
		userIDs[i] = u.ID
	}
	return userIDs
}
