package admin

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

type ActivityStats struct {
	TotalActivitiesMonth       int64   `json:"total_activities_month"`
	TotalActivitiesMonthChange float64 `json:"total_activities_month_change"`
	AdminActionsMonth          int64   `json:"admin_actions_month"`
	AdminActionsMonthChange    float64 `json:"admin_actions_month_change"`
	UserActionsMonth           int64   `json:"user_actions_month"`
	UserActionsMonthChange     float64 `json:"user_actions_month_change"`
	FailedActionsMonth         int64   `json:"failed_actions_month"`
	FailedActionsMonthChange   float64 `json:"failed_actions_month_change"`
}

type ActivityListItem struct {
	ActivityId string          `json:"activity_id"`
	User       ActivityUser    `json:"user"`
	Role       string          `json:"role"`
	Action     string          `json:"action"`
	Resource   string          `json:"resource"`
	Status     string          `json:"status"`
	Details    ActivityDetails `json:"details"`
}

type ActivityUser struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarUrl string `json:"avatar_url"`
}

type ActivityDetails struct {
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
}

type AppActivityResponse struct {
	Stats      ActivityStats      `json:"activity_stats"`
	Activities []ActivityListItem `json:"activities"`
}

type ActivityFilter struct {
	Search       string
	Role         string
	Status       string
	Action       string
	Duration     string
	StartDate    string
	EndDate      string
	IncludeStats bool
}

func ListAppActivity(db *gorm.DB, c *gin.Context, filter ActivityFilter) (AppActivityResponse, postgresql.PaginationResponse, int, error) {
	var logs []models.AuditLog
	pagination := postgresql.GetPagination(c)

	query := db.Model(&models.AuditLog{})

	if filter.Search != "" {
		search := "%" + filter.Search + "%"
		query = query.Where(
			"actor_email ILIKE ? OR actor_role ILIKE ? OR description ILIKE ? OR action ILIKE ?",
			search, search, search, search,
		)
	}

	if filter.Role != "" {
		query = query.Where("actor_role = ?", filter.Role)
	}

	if filter.Status != "" {
		// Status is inferred from action prefix: "failed_" => failed, else success
		// We store status via a naming convention or a dedicated column if available.
		// Using action string prefix convention: actions starting with "failed" are failed.
		if filter.Status == "failed" {
			query = query.Where("action ILIKE 'failed%'")
		} else if filter.Status == "success" {
			query = query.Where("action NOT ILIKE 'failed%'")
		}
	}

	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}

	now := time.Now()
	switch filter.Duration {
	case "last_3_months":
		filter.StartDate = now.AddDate(0, -3, 0).Format("2006-01-02")
	case "last_month":
		filter.StartDate = now.AddDate(0, -1, 0).Format("2006-01-02")
	case "last_year":
		filter.StartDate = now.AddDate(-1, 0, 0).Format("2006-01-02")
	case "custom":
		// use provided StartDate/EndDate
	default:
		// all_time or empty — no date filter
	}

	if filter.StartDate != "" {
		query = query.Where("created_at >= ?", filter.StartDate)
	}
	if filter.EndDate != "" {
		query = query.Where("created_at <= ?", filter.EndDate)
	}

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		"created_at",
		"desc",
		pagination,
		&logs,
		"",
	)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return AppActivityResponse{}, paginationResponse, http.StatusNoContent, nil
		}
		return AppActivityResponse{}, paginationResponse, http.StatusBadRequest, err
	}

	var stats ActivityStats

	if filter.IncludeStats {
		var wg sync.WaitGroup

		last30Days := now.AddDate(0, 0, -30)
		last60Days := now.AddDate(0, 0, -60)

		var prevTotal, prevAdmin, prevUser, prevFailed int64

		wg.Add(1)
		go func() {
			defer wg.Done()
			db.Model(&models.AuditLog{}).
				Where("created_at >= ?", last30Days).
				Count(&stats.TotalActivitiesMonth)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			db.Model(&models.AuditLog{}).
				Where("created_at >= ? AND created_at < ?", last60Days, last30Days).
				Count(&prevTotal)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			db.Model(&models.AuditLog{}).
				Where("created_at >= ? AND actor_role IN ?", last30Days, []string{"admin", "superadmin"}).
				Count(&stats.AdminActionsMonth)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			db.Model(&models.AuditLog{}).
				Where("created_at >= ? AND created_at < ? AND actor_role IN ?", last60Days, last30Days, []string{"admin", "superadmin"}).
				Count(&prevAdmin)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			db.Model(&models.AuditLog{}).
				Where("created_at >= ? AND actor_role = ?", last30Days, "user").
				Count(&stats.UserActionsMonth)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			db.Model(&models.AuditLog{}).
				Where("created_at >= ? AND created_at < ? AND actor_role = ?", last60Days, last30Days, "user").
				Count(&prevUser)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			db.Model(&models.AuditLog{}).
				Where("created_at >= ? AND action ILIKE 'failed%'", last30Days).
				Count(&stats.FailedActionsMonth)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			db.Model(&models.AuditLog{}).
				Where("created_at >= ? AND created_at < ? AND action ILIKE 'failed%'", last60Days, last30Days).
				Count(&prevFailed)
		}()

		wg.Wait()

		stats.TotalActivitiesMonthChange = percentChange(stats.TotalActivitiesMonth, prevTotal)
		stats.AdminActionsMonthChange = percentChange(stats.AdminActionsMonth, prevAdmin)
		stats.UserActionsMonthChange = percentChange(stats.UserActionsMonth, prevUser)
		stats.FailedActionsMonthChange = percentChange(stats.FailedActionsMonth, prevFailed)
	}

	if len(logs) == 0 {
		return AppActivityResponse{Stats: stats, Activities: []ActivityListItem{}}, paginationResponse, http.StatusOK, nil
	}

	// Collect unique actor IDs to batch-fetch user/admin info
	actorIDSet := make(map[string]struct{})
	for _, l := range logs {
		actorIDSet[l.ActorID] = struct{}{}
	}
	actorIDs := make([]string, 0, len(actorIDSet))
	for id := range actorIDSet {
		actorIDs = append(actorIDs, id)
	}

	// Try fetching from users first, then admins
	type actorInfo struct {
		ID        string
		Name      string
		Email     string
		AvatarURL string
	}

	actorMap := make(map[string]actorInfo)

	var userActors []models.User
	if err := db.Preload("Profile").Where("id IN ?", actorIDs).Find(&userActors).Error; err == nil {
		for _, u := range userActors {
			actorMap[u.ID] = actorInfo{
				ID:        u.ID,
				Name:      u.Name,
				Email:     u.Email,
				AvatarURL: u.Profile.AvatarURL,
			}
		}
	}

	var adminActors []models.Admin
	if err := db.Where("id IN ?", actorIDs).Find(&adminActors).Error; err == nil {
		for _, a := range adminActors {
			if _, exists := actorMap[a.ID]; !exists {
				actorMap[a.ID] = actorInfo{
					ID:    a.ID,
					Name:  a.Name,
					Email: a.Email,
				}
			}
		}
	}

	activities := make([]ActivityListItem, 0, len(logs))
	for _, l := range logs {
		actor := actorMap[l.ActorID]

		status := "success"
		if len(l.Action) >= 6 && string(l.Action)[:6] == "failed" {
			status = "failed"
		}

		actionParts := strings.Split(string(l.Action), ".")
		actionVerb := actionParts[len(actionParts)-1]

		activities = append(activities, ActivityListItem{
			ActivityId: l.ID,
			User: ActivityUser{
				ID:        actor.ID,
				Name:      actor.Name,
				Email:     actor.Email,
				AvatarUrl: actor.AvatarURL,
			},
			Role:     l.ActorRole,
			Action:   actionVerb,
			Resource: string(l.ResourceType),
			Status:   status,
			Details: ActivityDetails{
				Description: l.Description,
				Timestamp:   l.CreatedAt,
			},
		})
	}

	return AppActivityResponse{
		Stats:      stats,
		Activities: activities,
	}, paginationResponse, http.StatusOK, nil
}

func percentChange(current, previous int64) float64 {
	if previous > 0 {
		return float64(current-previous) / float64(previous) * 100
	}
	if current > 0 {
		return 100
	}
	return 0
}
