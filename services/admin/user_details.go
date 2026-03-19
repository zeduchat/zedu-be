package admin

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/avatar"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/user"
)

type UserDetailResponse struct {
	Profile       UserProfileInfo   `json:"profile"`
	CreditStats   UserCreditStats   `json:"credit_stats"`
	ActivityInfo  UserActivityInfo  `json:"activity_info"`
	AppUsage      UserAppUsage      `json:"app_usage"`
}

type UserProfileInfo struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Email              string `json:"email"`
	AvatarUrl          string `json:"avatar_url"`
	DefaultAvatarUrl   string `json:"default_avatar_url"`
	SubscriptionStatus string `json:"subscription_status"`
}

type UserCreditStats struct {
	TotalCreditsUsed       float64 `json:"total_credits_used"`
	CreditsUsedChange      float64 `json:"credits_used_change"`
	TotalAmountSpent       float64 `json:"total_amount_spent"`
	AmountSpentChange      float64 `json:"amount_spent_change"`
}

type UserActivityInfo struct {
	LastActive     string `json:"last_active"`
	LastLogin      string `json:"last_login"`
	ActivityLength string `json:"activity_length"`
	Referrals      int64  `json:"referrals"`
}

type UserAppUsage struct {
	TotalItemsCreated        int64   `json:"total_items_created"`
	TotalItemsCreatedChange  float64 `json:"total_items_created_change"`
	SessionsInitiated        int64   `json:"sessions_initiated"`
	SessionsInitiatedChange  float64 `json:"sessions_initiated_change"`
	AvgSessionDuration       string  `json:"avg_session_duration"`
	AvgSessionDurationChange float64 `json:"avg_session_duration_change"`
	KeyActionsPerformed      int64   `json:"key_actions_performed"`
	KeyActionsChange         float64 `json:"key_actions_change"`
}

func GetUserDetails(db *gorm.DB, userID string) (UserDetailResponse, int, error) {
	var u models.User
	if err := db.Preload("Profile").Preload("Organisations").Where("id = ? AND deleted_at IS NULL", userID).First(&u).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return UserDetailResponse{}, http.StatusNotFound, fmt.Errorf("user not found")
		}
		return UserDetailResponse{}, http.StatusInternalServerError, err
	}

	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)
	twoHoursAgo := now.Add(-2 * time.Hour)
	last30Days := now.AddDate(0, 0, -30)
	last60Days := now.AddDate(0, 0, -60)

	var (
		wg sync.WaitGroup

		// Credit stats
		totalCreditsUsed     float64
		creditsUsedLastHour  float64
		creditsUsedPrevHour  float64
		totalAmountSpent     float64
		amountSpentLastHour  float64
		amountSpentPrevHour  float64

		// Referrals
		referrals int64

		// App usage
		channelsCreated       int64
		integrationsCreated   int64
		prevChannelsCreated   int64
		prevIntegrationsCreated int64
		sessionsLast30        int64
		sessionsPrev30        int64
		avgSessionThis        float64
		avgSessionPrev        float64
		actionsLast30         int64
		actionsPrev30         int64
	)

	// Collect org IDs for the user
	orgIDs := make([]string, 0, len(u.Organisations))
	for _, org := range u.Organisations {
		orgIDs = append(orgIDs, org.ID)
	}

	// Total credits used (across all orgs the user belongs to)
	wg.Add(1)
	go func() {
		defer wg.Done()
		db.Model(&models.CreditUsage{}).
			Select("COALESCE(SUM(amount), 0)").
			Where("user_id = ?", userID).
			Scan(&totalCreditsUsed)
	}()

	// Credits used in last hour
	wg.Add(1)
	go func() {
		defer wg.Done()
		db.Model(&models.CreditUsage{}).
			Select("COALESCE(SUM(amount), 0)").
			Where("user_id = ? AND created_at >= ?", userID, oneHourAgo).
			Scan(&creditsUsedLastHour)
	}()

	// Credits used in previous hour (for % change)
	wg.Add(1)
	go func() {
		defer wg.Done()
		db.Model(&models.CreditUsage{}).
			Select("COALESCE(SUM(amount), 0)").
			Where("user_id = ? AND created_at >= ? AND created_at < ?", userID, twoHoursAgo, oneHourAgo).
			Scan(&creditsUsedPrevHour)
	}()

	// Total amount spent (credit transactions via orgs owned by user)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if len(orgIDs) > 0 {
			db.Table("credit_transactions").
				Select("COALESCE(SUM(amount), 0)").
				Where("organisation_id IN ?", orgIDs).
				Scan(&totalAmountSpent)
		}
	}()

	// Amount spent last hour
	wg.Add(1)
	go func() {
		defer wg.Done()
		if len(orgIDs) > 0 {
			db.Table("credit_transactions").
				Select("COALESCE(SUM(amount), 0)").
				Where("organisation_id IN ? AND created_at >= ?", orgIDs, oneHourAgo).
				Scan(&amountSpentLastHour)
		}
	}()

	// Amount spent previous hour
	wg.Add(1)
	go func() {
		defer wg.Done()
		if len(orgIDs) > 0 {
			db.Table("credit_transactions").
				Select("COALESCE(SUM(amount), 0)").
				Where("organisation_id IN ? AND created_at >= ? AND created_at < ?", orgIDs, twoHoursAgo, oneHourAgo).
				Scan(&amountSpentPrevHour)
		}
	}()

	// Referrals count
	wg.Add(1)
	go func() {
		defer wg.Done()
		db.Model(&models.Invitation{}).
			Where("invited_by = ?", userID).
			Count(&referrals)
	}()

	// Channels created (last 30 days)
	wg.Add(1)
	go func() {
		defer wg.Done()
		db.Model(&models.Channels{}).
			Where("owner_id = ? AND created_at >= ?", userID, last30Days).
			Count(&channelsCreated)
	}()

	// Channels created (prev 30 days)
	wg.Add(1)
	go func() {
		defer wg.Done()
		db.Model(&models.Channels{}).
			Where("owner_id = ? AND created_at >= ? AND created_at < ?", userID, last60Days, last30Days).
			Count(&prevChannelsCreated)
	}()

	// Integrations created (last 30 days)
	wg.Add(1)
	go func() {
		defer wg.Done()
		db.Model(&models.OrganisationIntegrations{}).
			Where("owner_id = ? AND created_at >= ?", userID, last30Days).
			Count(&integrationsCreated)
	}()

	// Integrations created (prev 30 days)
	wg.Add(1)
	go func() {
		defer wg.Done()
		db.Model(&models.OrganisationIntegrations{}).
			Where("owner_id = ? AND created_at >= ? AND created_at < ?", userID, last60Days, last30Days).
			Count(&prevIntegrationsCreated)
	}()

	// Sessions (access tokens) last 30 days
	wg.Add(1)
	go func() {
		defer wg.Done()
		db.Model(&models.AccessToken{}).
			Where("owner_id = ? AND created_at >= ?", userID, last30Days).
			Count(&sessionsLast30)
	}()

	// Sessions prev 30 days
	wg.Add(1)
	go func() {
		defer wg.Done()
		db.Model(&models.AccessToken{}).
			Where("owner_id = ? AND created_at >= ? AND created_at < ?", userID, last60Days, last30Days).
			Count(&sessionsPrev30)
	}()

	// Avg session duration last 30 days (seconds)
	wg.Add(1)
	go func() {
		defer wg.Done()
		db.Model(&models.AccessToken{}).
			Select("COALESCE(AVG(EXTRACT(EPOCH FROM updated_at - created_at)), 0)").
			Where("owner_id = ? AND created_at >= ?", userID, last30Days).
			Scan(&avgSessionThis)
	}()

	// Avg session duration prev 30 days
	wg.Add(1)
	go func() {
		defer wg.Done()
		db.Model(&models.AccessToken{}).
			Select("COALESCE(AVG(EXTRACT(EPOCH FROM updated_at - created_at)), 0)").
			Where("owner_id = ? AND created_at >= ? AND created_at < ?", userID, last60Days, last30Days).
			Scan(&avgSessionPrev)
	}()

	// Key actions performed (audit log entries) last 30 days
	wg.Add(1)
	go func() {
		defer wg.Done()
		db.Model(&models.AuditLog{}).
			Where("actor_id = ? AND created_at >= ?", userID, last30Days).
			Count(&actionsLast30)
	}()

	// Key actions prev 30 days
	wg.Add(1)
	go func() {
		defer wg.Done()
		db.Model(&models.AuditLog{}).
			Where("actor_id = ? AND created_at >= ? AND created_at < ?", userID, last60Days, last30Days).
			Count(&actionsPrev30)
	}()

	wg.Wait()

	// Build profile
	subscriptionStatus := SubscriptionStatusFree
	for _, org := range u.Organisations {
		if org.SubscriptionPlanId != "free" && org.SubscriptionPlanId != "" {
			subscriptionStatus = SubscriptionStatusPaid
			break
		}
	}

	var lastActive, lastLogin string
	if u.LastActivityAt != nil {
		lastActive = u.LastActivityAt.Format("2006-01-02T15:04:05Z07:00")
	}
	if u.LastLogInAt != nil {
		lastLogin = u.LastLogInAt.Format("2006-01-02T15:04:05Z07:00")
	}

	var activityLength string
	if al := user.GetActivityLength(u); al != nil {
		activityLength = *al
	}

	totalItemsCreated := channelsCreated + integrationsCreated
	prevTotalItemsCreated := prevChannelsCreated + prevIntegrationsCreated

	return UserDetailResponse{
		Profile: UserProfileInfo{
			ID:                 u.ID,
			Name:               u.Name,
			Email:              u.Email,
			AvatarUrl:          u.Profile.AvatarURL,
			DefaultAvatarUrl:   avatar.GenerateDefaultAvatarURL(u.ID),
			SubscriptionStatus: subscriptionStatus,
		},
		CreditStats: UserCreditStats{
			TotalCreditsUsed:  totalCreditsUsed,
			CreditsUsedChange: percentChange(int64(creditsUsedLastHour), int64(creditsUsedPrevHour)),
			TotalAmountSpent:  totalAmountSpent,
			AmountSpentChange: percentChange(int64(amountSpentLastHour), int64(amountSpentPrevHour)),
		},
		ActivityInfo: UserActivityInfo{
			LastActive:     lastActive,
			LastLogin:      lastLogin,
			ActivityLength: activityLength,
			Referrals:      referrals,
		},
		AppUsage: UserAppUsage{
			TotalItemsCreated:        totalItemsCreated,
			TotalItemsCreatedChange:  percentChange(totalItemsCreated, prevTotalItemsCreated),
			SessionsInitiated:        sessionsLast30,
			SessionsInitiatedChange:  percentChange(sessionsLast30, sessionsPrev30),
			AvgSessionDuration:       formatSecs(avgSessionThis),
			AvgSessionDurationChange: percentChangeFloat(avgSessionThis, avgSessionPrev),
			KeyActionsPerformed:      actionsLast30,
			KeyActionsChange:         percentChange(actionsLast30, actionsPrev30),
		},
	}, http.StatusOK, nil
}

func percentChangeFloat(current, previous float64) float64 {
	if previous > 0 {
		return (current - previous) / previous * 100
	}
	if current > 0 {
		return 100
	}
	return 0
}
