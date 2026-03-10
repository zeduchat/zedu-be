package admin

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	telexaudit "github.com/hngprojects/telex_be/services/telexAudit"
	"github.com/hngprojects/telex_be/utility"
	"github.com/hngprojects/telex_be/utility/audit_utility"
)

func LoginAdmin(req models.AdminLoginRequest, db *gorm.DB, c *gin.Context) (gin.H, int, error) {
	var (
		admin        = models.Admin{}
		responseData gin.H
		cfg          = config.GetConfig().Admin
	)

	if req.Email == cfg.SUPER_ADMIN_EMAIL && req.Password == cfg.SUPER_ADMIN_PASSWORD {
		err := admin.GetOrCreateSuperAdmin(db, cfg)
		if err != nil {
			return responseData, http.StatusInternalServerError, fmt.Errorf("error creating super admin: %w", err)
		}
	} else {
		exists := postgresql.CheckExists(db, &admin, "email = ?", strings.ToLower(req.Email))
		if !exists {
			return responseData, http.StatusBadRequest, fmt.Errorf("invalid credentials")
		}

		if !utility.CompareHash(req.Password, admin.Password) {
			return responseData, http.StatusBadRequest, fmt.Errorf("invalid credentials")
		}
	}

	if !admin.IsActive {
		return responseData, http.StatusForbidden, fmt.Errorf("admin account is deactivated")
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

	// Create audit log for admin login
	if err := audit_utility.CreateAuditLog(
		db,
		admin.ID,
		admin.Email,
		admin.Role,
		models.ActionAdminLogin,
		models.ResourceAdmin,
		admin.ID,
		"",
		"",
		fmt.Sprintf("Admin %s logged in", admin.Email),
		audit_utility.GetClientIP(c),
		c.GetHeader("User-Agent"),
	); err != nil {
		// Log error but don't fail the request
		fmt.Printf("failed to create audit log for admin login: %v\n", err)
	}

	return responseData, http.StatusOK, nil
}

func CreateAdmin(db *storage.Database, req models.CreateAdminRequest, c *gin.Context, logger *utility.Logger) (gin.H, error) {
	var (
		email        = strings.ToLower(req.Email)
		responseData gin.H
	)

	var existing models.Admin
	if err := db.Postgresql.Where("email = ?", email).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("admin with this email already exists")
	}

	var user models.User
	if err := db.Postgresql.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, fmt.Errorf("email is not registered as a Telex user")
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
		Role:     models.RoleAdmin,
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

	// Get requester information for audit logging
	var requester *models.Admin
	if claims, exists := c.Get("adminClaims"); exists {
		if claimsMap, ok := claims.(jwt.MapClaims); ok {
			if adminID, ok := claimsMap["admin_id"].(string); ok {
				if reqAdmin, err := models.GetAdminById(db.Postgresql, adminID); err == nil {
					requester = reqAdmin
				}
			}
		}
	}

	// Create audit log for admin creation (only if requester was found)
	if requester != nil {
		if err := audit_utility.CreateAuditLog(
			db.Postgresql,
			requester.ID,
			requester.Email,
			requester.Role,
			models.ActionAdminCreate,
			models.ResourceAdmin,
			admin.ID,
			"",
			"",
			fmt.Sprintf("Superadmin %s created new admin account for %s", requester.Email, admin.Email),
			audit_utility.GetClientIP(c),
			c.GetHeader("User-Agent"),
		); err != nil {
			// Log error but don't fail the request
			fmt.Printf("failed to create audit log for admin creation: %v\n", err)
		}

		// Broadcast audit event
		telexaudit.CreateAdminAudit(db, logger, requester.Email, admin.Email)
	}

	responseData = gin.H{
		"user": map[string]any{
			"id":       admin.ID,
			"email":    admin.Email,
			"name":     admin.Name,
			"password": plaintextPass,
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
