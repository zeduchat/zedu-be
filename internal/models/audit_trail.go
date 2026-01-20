package models

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

type LoginActivity struct {
	ID             string         `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	UserID         string         `gorm:"type:varchar(100);not null;index" json:"user_id"`
	OrganisationID *string        `gorm:"type:varchar(100);null;index" json:"-"`
	AccessID       *uuid.UUID     `gorm:"type:uuid;null;index" json:"access_token_id"`
	LoginAt        time.Time      `gorm:"type:timestamp;null" json:"login_at"`
	IPAddress      string         `gorm:"type:varchar(45);not null" json:"ip_address"`
	Location       string         `gorm:"type:varchar(100)" json:"location"`
	Device         string         `gorm:"type:varchar(50)" json:"device"`
	CreatedAt      time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime" json:"-"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	IsLive         bool           `gorm:"column:is_live" json:"is_live"`
}

type AuditLog struct {
	ID        string    `gorm:"type:uuid;primaryKey;unique;not null" json:"id"`
	AdminID   string    `gorm:"type:uuid;not null;index" json:"admin_id"` // The one who performed the action
	Action    string    `gorm:"type:varchar(100);not null" json:"action"`
	TargetID  string    `gorm:"type:uuid;not null;index" json:"target_id"` // The admin whose role was changed
	OldValue  string    `gorm:"type:varchar(50)" json:"old_value"`
	NewValue  string    `gorm:"type:varchar(50)" json:"new_value"`
	IPAddress string    `gorm:"type:varchar(45)" json:"ip_address"`
	CreatedAt time.Time `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
}

func (l *LoginActivity) Create(db *gorm.DB) error {
	err := postgresql.CreateOneRecord(db, &l)

	if err != nil {
		return err
	}

	return nil
}

func (la *LoginActivity) GetLoginActivityByIDsAdmin(db *gorm.DB, c *gin.Context, userID, requesterID string, isSuperAdmin bool) ([]LoginActivity, postgresql.PaginationResponse, error) {
	var (
		loginActivities = []LoginActivity{}
	)

	var isOwner bool
	err := db.Model(&Organisation{}).
		Select("count(*) > 0").
		Where("owner_id = ? AND id IN (SELECT organisation_id FROM user_organisations WHERE user_id = ?)", requesterID, userID).
		Find(&isOwner).
		Error
	if err != nil {
		return loginActivities, postgresql.PaginationResponse{}, err
	}

	// Check if user has permission to access the login activities
	hasPermission := isOwner || isSuperAdmin || requesterID == userID
	if !hasPermission {
		return loginActivities, postgresql.PaginationResponse{}, errors.New("login activities not found")
	}

	query := db.Model(&LoginActivity{}).
		Select("login_activities.*, (access_tokens.is_live) AS is_live").
		Joins("LEFT JOIN access_tokens ON access_tokens.id = login_activities.access_id")

	if isOwner || isSuperAdmin {
		query = query.Where("login_activities.user_id = ?", userID)
	} else if requesterID == userID {
		query = query.Where("login_activities.user_id = ? AND login_activities.user_id = ?", userID, requesterID)
	}

	pagination := postgresql.GetPagination(c)
	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		query,
		"login_activities.login_at",
		"desc",
		pagination,
		&loginActivities,
		nil,
	)
	if err != nil {
		return loginActivities, postgresql.PaginationResponse{}, err
	}

	return loginActivities, paginationResponse, nil
}

func (a *AuditLog) GetAuditHistory(db *gorm.DB, c *gin.Context) ([]AuditLog, postgresql.PaginationResponse, error) {
	var (
		logs       []AuditLog
		conditions []string
		values     []any
	)

	conditions = append(conditions, "action = ?")
	values = append(values, "ROLE_CHANGE_CONFIRMED")

	if targetID := c.Query("target_id"); targetID != "" {
		conditions = append(conditions, "target_id = ?")
		values = append(values, targetID)
	}

	if date := c.Query("date"); date != "" {
		conditions = append(conditions, "DATE(created_at) = ?")
		values = append(values, date)
	}

	queryStr := ""
	if len(conditions) > 0 {
		queryStr = conditions[0]
		for i := 1; i < len(conditions); i++ {
			queryStr += " AND " + conditions[i]
		}
	}

	pagination := postgresql.GetPagination(c)

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		db,
		"created_at",
		"desc",
		pagination,
		&logs,
		queryStr,
		values...,
	)

	if err != nil {
		return nil, paginationResponse, err
	}

	return logs, paginationResponse, nil
}
