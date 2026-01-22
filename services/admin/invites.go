package admin

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

// ============================================================================
// RESPONSE TYPES
// ============================================================================

type InvitationDashboardResponse struct {
	Stats        InvitationStats      `json:"stats"`
	GrowthTrends []GrowthTrendPoint   `json:"growth_trends"`
	TopInviters  []TopInviter         `json:"top_inviters"`
	Invitations  []InvitationListItem `json:"invitations"`
}

type InvitationStats struct {
	TotalInvitationsSent int64   `json:"total_invitations_sent"`
	TotalGrowthPercent   float64 `json:"total_growth_percent"`
	SentToday            int64   `json:"sent_today"`
	SentTodayPercent     float64 `json:"sent_today_percent"`
	Yesterday            int64   `json:"yesterday"`
	ThisWeek             int64   `json:"this_week"`
}

type GrowthTrendPoint struct {
	Month    string `json:"month"`
	Organic  int64  `json:"organic"`
	Referral int64  `json:"referral"`
}

type TopInviter struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	AvatarUrl   string `json:"avatar_url"`
	InviteCount int64  `json:"invite_count"`
	Rank        int    `json:"rank"`
}

type InvitationListItem struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Status   string `json:"status"`
	DateSent string `json:"date_sent"`
}

type InvitationFilter struct {
	Search       string
	Status       string
	IncludeStats bool
	TopLimit     int
}

// ============================================================================
// MAIN FUNCTION
// ============================================================================

func GetInvitationDashboard(db *gorm.DB, c *gin.Context, filter InvitationFilter) (InvitationDashboardResponse, postgresql.PaginationResponse, int, error) {
	var response InvitationDashboardResponse
	pagination := postgresql.GetPagination(c)

	now := time.Now()

	// Calculate Stats (parallel)
	if filter.IncludeStats {
		var wg sync.WaitGroup
		var stats InvitationStats

		startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		startOfYesterday := startOfToday.AddDate(0, 0, -1)
		startOfWeek := now.AddDate(0, 0, -int(now.Weekday()))
		startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		startOfLastMonth := startOfMonth.AddDate(0, -1, 0)

		var currentMonthCount, lastMonthCount int64

		wg.Add(1)
		go func() {
			defer wg.Done()
			db.Model(&models.Invitation{}).Count(&stats.TotalInvitationsSent)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			db.Model(&models.Invitation{}).Where("created_at >= ?", startOfToday).Count(&stats.SentToday)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			db.Model(&models.Invitation{}).Where("created_at >= ? AND created_at < ?", startOfYesterday, startOfToday).Count(&stats.Yesterday)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			db.Model(&models.Invitation{}).Where("created_at >= ?", startOfWeek).Count(&stats.ThisWeek)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			db.Model(&models.Invitation{}).Where("created_at >= ?", startOfMonth).Count(&currentMonthCount)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			db.Model(&models.Invitation{}).Where("created_at >= ? AND created_at < ?", startOfLastMonth, startOfMonth).Count(&lastMonthCount)
		}()

		wg.Wait()

		// Calculate growth percent
		if lastMonthCount > 0 {
			stats.TotalGrowthPercent = float64(currentMonthCount-lastMonthCount) / float64(lastMonthCount) * 100
		} else if currentMonthCount > 0 {
			stats.TotalGrowthPercent = 100
		}

		if stats.Yesterday > 0 {
			stats.SentTodayPercent = float64(stats.SentToday-stats.Yesterday) / float64(stats.Yesterday) * 100
		} else if stats.SentToday > 0 {
			stats.SentTodayPercent = 100
		}

		response.Stats = stats
	}

	// Growth Trends (last 6 months)
	response.GrowthTrends = getGrowthTrends(db, now)

	// Top Inviters
	topLimit := filter.TopLimit
	if topLimit <= 0 {
		topLimit = 5
	}
	response.TopInviters = getTopInviters(db, topLimit)

	// Paginated Invitations List
	invitations, paginationResponse, err := getInvitationsList(db, pagination, filter)
	if err != nil {
		return response, paginationResponse, http.StatusBadRequest, err
	}
	response.Invitations = invitations

	return response, paginationResponse, http.StatusOK, nil
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func getGrowthTrends(db *gorm.DB, now time.Time) []GrowthTrendPoint {
	trends := make([]GrowthTrendPoint, 0, 6)

	for i := 5; i >= 0; i-- {
		monthStart := time.Date(now.Year(), now.Month()-time.Month(i), 1, 0, 0, 0, 0, now.Location())
		monthEnd := monthStart.AddDate(0, 1, 0)
		monthName := monthStart.Format("Jan")

		var organic, referral int64

		// Organic = invitations where invited_by is null (no referrer)
		db.Model(&models.Invitation{}).
			Where("created_at >= ? AND created_at < ?", monthStart, monthEnd).
			Where("invited_by IS NULL").
			Count(&organic)

		// Referral = invitations where invited_by is set
		db.Model(&models.Invitation{}).
			Where("created_at >= ? AND created_at < ?", monthStart, monthEnd).
			Where("invited_by IS NOT NULL").
			Count(&referral)

		trends = append(trends, GrowthTrendPoint{
			Month:    monthName,
			Organic:  organic,
			Referral: referral,
		})
	}

	return trends
}

func getTopInviters(db *gorm.DB, limit int) []TopInviter {
	type leaderRow struct {
		ID        string
		Name      string
		Email     string
		AvatarURL string
		Referrals int64
	}

	var rows []leaderRow

	db.Table("users").
		Select("users.id, users.name, users.email, profiles.avatar_url, COALESCE(COUNT(invitations.id), 0) as referrals").
		Joins("LEFT JOIN invitations ON invitations.invited_by = users.id").
		Joins("LEFT JOIN profiles ON profiles.userid = users.id").
		Where("users.deleted_at IS NULL").
		Group("users.id, profiles.avatar_url").
		Order("referrals desc").
		Limit(limit).
		Find(&rows)

	topInviters := make([]TopInviter, 0, len(rows))
	for idx, r := range rows {
		topInviters = append(topInviters, TopInviter{
			ID:          r.ID,
			Name:        r.Name,
			Email:       r.Email,
			AvatarUrl:   r.AvatarURL,
			InviteCount: r.Referrals,
			Rank:        idx + 1,
		})
	}

	return topInviters
}

func getInvitationsList(db *gorm.DB, pagination postgresql.Pagination, filter InvitationFilter) ([]InvitationListItem, postgresql.PaginationResponse, error) {
	var invitations []models.Invitation

	query := db.Model(&models.Invitation{})

	if filter.Search != "" {
		search := "%" + filter.Search + "%"
		query = query.Where("email ILIKE ?", search)
	}

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		"created_at",
		"desc",
		pagination,
		&invitations,
		"",
	)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, paginationResponse, err
	}

	// Build role lookup
	roleIDs := make([]string, 0, len(invitations))
	for _, inv := range invitations {
		if inv.Role != "" {
			roleIDs = append(roleIDs, inv.Role)
		}
	}

	roleMap := make(map[string]string)
	if len(roleIDs) > 0 {
		var roles []models.OrgRole
		db.Where("id IN ?", roleIDs).Find(&roles)
		for _, r := range roles {
			roleMap[r.ID] = r.Name
		}
	}

	items := make([]InvitationListItem, 0, len(invitations))
	for _, inv := range invitations {
		roleName := roleMap[inv.Role]
		if roleName == "" {
			roleName = "Unknown"
		}

		items = append(items, InvitationListItem{
			ID:       inv.ID,
			Email:    inv.Email,
			Role:     roleName,
			Status:   inv.Status,
			DateSent: inv.CreatedAt.Format("2006-01-02"),
		})
	}

	return items, paginationResponse, nil
}

// ============================================================================
// LEGACY FUNCTION (for backward compatibility)
// ============================================================================

func ListUsersByInvites(db *gorm.DB, orgID *string, limit int) ([]map[string]any, int, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		return nil, http.StatusBadRequest, fmt.Errorf("limit cannot exceed 100")
	}

	type leaderRow struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		Referrals int64  `json:"referrals"`
	}

	var rows []leaderRow

	whereClause := "users.deleted_at IS NULL"
	var whereArgs []any
	if orgID != nil && *orgID != "" {
		whereClause = whereClause + " AND invitations.organisation_id = ?"
		whereArgs = append(whereArgs, *orgID)
	}

	selectQuery := "users.id, users.name, users.email, COALESCE(COUNT(invitations.id), 0) as referrals"
	tx := db.Table("users").
		Select(selectQuery).
		Joins("LEFT JOIN invitations ON invitations.invited_by = users.id").
		Where(whereClause, whereArgs...).
		Group("users.id").
		Order("referrals desc").
		Limit(limit).
		Find(&rows)

	if tx.Error != nil {
		return nil, http.StatusBadRequest, tx.Error
	}

	resp := make([]map[string]any, 0, len(rows))
	for idx, r := range rows {
		resp = append(resp, map[string]any{
			"id":        r.ID,
			"name":      r.Name,
			"email":     r.Email,
			"referrals": int(r.Referrals),
			"rank":      idx + 1,
		})
	}

	return resp, http.StatusOK, nil
}
