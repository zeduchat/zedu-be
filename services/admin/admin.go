package admin

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"crypto/rand"
	"encoding/base64"
	"strings"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/services/user"
	"github.com/hngprojects/telex_be/utility"
)

func LoginAdmin(req models.AdminLoginRequest, db *gorm.DB, c *gin.Context) (gin.H, int, error) {
	var (
		admin        = models.Admin{}
		responseData gin.H
		envEmail     = config.GetConfig().Admin.SUPER_ADMIN_EMAIL
		envName      = config.GetConfig().Admin.SUPER_ADMIN_NAME
		envPassword  = config.GetConfig().Admin.SUPER_ADMIN_PASSWORD
		envRole      = config.GetConfig().Admin.SUPER_ADMIN_ROLE
	)

	if req.Email == envEmail && req.Password == envPassword {
		// Construct a pseudo-admin
		admin = models.Admin{
			ID:       utility.GenerateUUID(),
			Email:    envEmail,
			Name:     envName,
			IsActive: true,
			Role:     envRole,
		}
	} else {
		// Proceed with DB check
		exists := postgresql.CheckExists(db, &admin, "email = ?", req.Email)
		if !exists {
			return responseData, 400, fmt.Errorf("invalid credentials")
		}

		if !utility.CompareHash(req.Password, admin.Password) {
			return responseData, 400, fmt.Errorf("invalid credentials")
		}
	}

	tokenData, err := middleware.CreateAdminToken(admin, c)
	if err != nil {
		return responseData, http.StatusInternalServerError, fmt.Errorf("error saving token: %w", err)
	}

	tokens := map[string]string{
		"access_token": tokenData.AccessToken,
		"exp":          strconv.Itoa(int(tokenData.ExpiresAt.Unix())),
	}

	access_token := models.AccessToken{ID: tokenData.AccessUuid, OwnerID: admin.ID}

	err = access_token.CreateAccessToken(db, tokens)
	if err != nil {
		return responseData, http.StatusInternalServerError, fmt.Errorf("error saving token: %w", err)
	}

	responseData = gin.H{
		"admin": map[string]any{
			"id":        admin.ID,
			"email":     admin.Email,
			"name":      admin.Name,
			"is_active": admin.IsActive,
			"role":      admin.Role,
		},
		"access_token": tokenData.AccessToken,
	}

	return responseData, http.StatusOK, nil
}

func CreateAdmin(db *storage.Database, req models.CreateAdminRequest, c *gin.Context) (gin.H, error) {
	var (
		email        = strings.ToLower(req.Email)
		responseData gin.H
	)

	var existing models.Admin
	if err := db.Postgresql.Where("email = ?", email).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("admin with this email already exists")
	}

	plaintextPass, err := GenerateStrongPassword(16) // 16-char password
	if err != nil {
		return nil, err
	}

	password, err := utility.HashPassword(plaintextPass)
	if err != nil {
		return nil, err
	}

	admin := models.Admin{
		ID:       utility.GenerateUUID(),
		Name:     req.Name,
		Email:    email,
		Role:     req.Role,
		Password: password,
		IsActive: true,
	}

	err = admin.CreateAdmin(db.Postgresql)
	if err != nil {
		return nil, err
	}

	tokenData, err := middleware.CreateAdminToken(admin, c)
	if err != nil {
		return nil, fmt.Errorf("error saving token: %w", err)
	}

	tokens := map[string]string{
		"access_token": tokenData.AccessToken,
		"exp":          strconv.Itoa(int(tokenData.ExpiresAt.Unix())),
	}

	access_token := models.AccessToken{ID: tokenData.AccessUuid, OwnerID: admin.ID}

	err = access_token.CreateAccessToken(db.Postgresql, tokens)
	if err != nil {
		return nil, fmt.Errorf("error saving token: %w", err)
	}

	responseData = gin.H{
		"user": map[string]any{
			"id":       admin.ID,
			"email":    admin.Email,
			"name":     admin.Name,
			"password": plaintextPass, // show generated password to admin once to store it somewhere, for security purpose
		},
		"access_token": tokenData.AccessToken,
	}

	return responseData, nil
}

func ChangeAdminRole(db *storage.Database, targetAdminId, newRole, requesterId string) error {

	var (
		admin       models.Admin
		targetAdmin *models.Admin
		err         error
	)

	targetAdmin, err = models.GetAdminById(db.Postgresql, targetAdminId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("admin with ID %s not found", targetAdminId)
		}
		return fmt.Errorf("failed to fetch admin: %w", err)
	}

	oldRole := targetAdmin.Role

	if oldRole == newRole {
		return errors.New("admin already has this role")
	}

	if oldRole == models.RoleSuperAdmin && newRole == models.RoleAdmin {
		var superadminCount int64
		if err := db.Postgresql.Model(&models.Admin{}).
			Where("role = ? AND is_active = ? AND id != ?", models.RoleSuperAdmin, true, targetAdminId).
			Count(&superadminCount).Error; err != nil {
			return fmt.Errorf("failed to verify superadmin count: %w", err)
		}

		if superadminCount == 0 {
			return errors.New("cannot downgrade the last active superadmin")
		}
	}

	err = admin.ChangeRole(db.Postgresql, newRole, targetAdminId)
	if err != nil {
		return fmt.Errorf("failed to change role: %w", err)
	}

	// TODO: verify this part
	// Invalidate all active sessions for security
	// Force re-authentication with new role
	var accessToken models.AccessToken
	if err := db.Postgresql.Model(&accessToken).
		Where("owner_id = ? AND is_live = ?", targetAdminId, true).
		Updates(map[string]interface{}{"is_live": false}).Error; err != nil {
		// Log error but don't fail - role change already succeeded
		// TODO: log this
	}

	// TODO: Create audit log entry
	// auditLog := models.AuditLog{
	// 	ID:          utility.GenerateUUID(),
	// 	AdminID:     requesterID,
	// 	Action:      "CHANGE_ADMIN_ROLE",
	// 	TargetID:    targetAdminID,
	// 	TargetType:  "admin",
	// 	OldValue:    oldRole,
	// 	NewValue:    newRole,
	// 	Timestamp:   time.Now(),
	// 	IPAddress:   c.ClientIP(), // Pass from controller
	// }
	// if err := auditLog.Create(db.Postgresql); err != nil {
	// 	// Log but don't fail
	// }

	return nil
}

func GenerateStrongPassword(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	password := base64.URLEncoding.EncodeToString(bytes)

	if len(password) > length {
		password = password[:length]
	}

	return password, nil
}

type UserListItem struct {
	ID                 string  `json:"id"`
	Email              string  `json:"email"`
	Name               string  `json:"name"`
	AvatarUrl          string  `json:"avatar_url"`
	CreatedAt          string  `json:"created_at"`
	LastLogInAt        string  `json:"last_log_in_at"`
	LastActivityAt     string  `json:"last_activity_at"`
	ActivityLength     string  `json:"activity_length"`
	Referrals          int64   `json:"referrals"`
	CreditUsed         float64 `json:"credit_used"`
	AmountSpent        float64 `json:"amount_spent"`
	SubscriptionStatus string  `json:"subscription_status"`
}

func ListUsers(db *gorm.DB, c *gin.Context) ([]UserListItem, postgresql.PaginationResponse, int, error) {
	var users []models.User
	pagination := postgresql.GetPagination(c)

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		db.Preload("Profile"),
		"created_at",
		"desc",
		pagination,
		&users,
		"deleted_at IS NULL",
	)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, paginationResponse, http.StatusNoContent, nil
		}
		return nil, paginationResponse, http.StatusBadRequest, err
	}

	if len(users) == 0 {
		return []UserListItem{}, paginationResponse, http.StatusOK, nil
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
		// If user has any paid subscription, mark as "Paid"
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

	return resp, paginationResponse, http.StatusOK, nil
}

func getUserIDs(users []models.User) []string {
	userIDs := make([]string, len(users))
	for i, u := range users {
		userIDs[i] = u.ID
	}
	return userIDs
}

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
