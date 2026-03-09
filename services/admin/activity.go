package admin

import "time"

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
