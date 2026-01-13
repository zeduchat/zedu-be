package admin

import (
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
	"github.com/hngprojects/telex_be/services/invitation"
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
	AvatarUrl          *string `json:"avatar_url"`
	CreatedAt          string  `json:"created_at"`
	LastLogInAt        *string `json:"last_log_in_at"`
	LastActivityAt     *string `json:"last_activity_at"`
	ActivityLength     *string `json:"activity_length"`
	Referrals          int64   `json:"referrals"`      //TODO
	CreditUsed         int     `json:"credit_used"`    //TODO: follow up on credit usage implementation (Tobi)
	AmountSpent        int     `json:"amount_spent"`   //TODO
	SubscriptionStatus string  `json:"payment_status"` //TODO: follow up on the the plan-split endpoint implementation
}

func ListUsers(db *gorm.DB, c *gin.Context) ([]map[string]any, postgresql.PaginationResponse, int, error) {
	var users []models.User
	pagination := postgresql.GetPagination(c)

	paginationResponse, err := postgresql.SelectAllFromDbOrderByPaginated(
		db,
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
		return []map[string]any{}, paginationResponse, http.StatusOK, nil
	}

	resp := make([]map[string]any, 0, len(users))
	for _, u := range users {
		var lastLogInAt *string
		if u.LastLogInAt != nil {
			formatted := u.LastLogInAt.Format("2006-01-02T15:04:05Z07:00")
			lastLogInAt = &formatted
		}

		var lastActivityAt *string
		if u.LastActivityAt != nil {
			formatted := u.LastActivityAt.Format("2006-01-02T15:04:05Z07:00")
			lastActivityAt = &formatted
		}

		var avatarUrl *string
		if u.Profile.AvatarURL != "" {
			avatarUrl = &u.Profile.AvatarURL
		}

		activityLength := user.GetActivityLength(u)

		referrals, _ := invitation.CountInvitesByUser(db, u.ID)

		resp = append(resp, map[string]any{
			"id":               u.ID,
			"email":            u.Email,
			"name":             u.Name,
			"avatar_url":       avatarUrl,
			"created_at":       u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			"last_log_in_at":   lastLogInAt,
			"last_activity_at": lastActivityAt,
			"activity_length":  activityLength,
			"referrals":        referrals,
		})
	}

	return resp, paginationResponse, http.StatusOK, nil
}

func ListUsersByInvites(db *gorm.DB, c *gin.Context) ([]map[string]any, postgresql.PaginationResponse, int, error) {
	pagination := postgresql.GetPagination(c)

	type leaderRow struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		Referrals int64  `json:"referrals"`
	}

	var rows []leaderRow

	selectQuery := "users.id, users.name, users.email, COALESCE(COUNT(invitations.id), 0) as referrals"
	paginationResponse, err := postgresql.RawSelectAllFromByGroup(db, "referrals", "desc", &pagination, &models.User{}, &rows, "invitations.invited_by", selectQuery, "users.deleted_at IS NULL")
	if err != nil {
		return nil, paginationResponse, http.StatusBadRequest, err
	}

	resp := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		resp = append(resp, map[string]any{
			"id":        r.ID,
			"name":      r.Name,
			"email":     r.Email,
			"referrals": r.Referrals,
		})
	}

	return resp, paginationResponse, http.StatusOK, nil
}
