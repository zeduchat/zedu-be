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

	var loginErr error
	if req.Email == cfg.SUPER_ADMIN_EMAIL && req.Password == cfg.SUPER_ADMIN_PASSWORD {
		loginErr = admin.GetOrCreateSuperAdmin(db, cfg)
		if loginErr != nil {
			return responseData, http.StatusInternalServerError, fmt.Errorf("error creating super admin: %w", loginErr)
		}
	} else {
		exists := postgresql.CheckExists(db, &admin, "email = ?", strings.ToLower(req.Email))
		if !exists {
			if auditErr := audit_utility.CreateAuditLog(
				db,
				"",
				req.Email,
				"",
				models.ActionAdminLogin,
				models.ResourceAdmin,
				"",
				"",
				"",
				fmt.Sprintf("Admin login failed: no account found for %s", req.Email),
				audit_utility.GetClientIP(c),
				c.GetHeader("User-Agent"),
				false,
			); auditErr != nil {
				fmt.Printf("failed to create audit log for admin login failure: %v\n", auditErr)
			}
			return responseData, http.StatusBadRequest, fmt.Errorf("invalid credentials")
		}

		if !utility.CompareHash(req.Password, admin.Password) {
			if auditErr := audit_utility.CreateAuditLog(
				db,
				admin.ID,
				admin.Email,
				admin.Role,
				models.ActionAdminLogin,
				models.ResourceAdmin,
				admin.ID,
				"",
				"",
				fmt.Sprintf("Admin login failed: incorrect password for %s", admin.Email),
				audit_utility.GetClientIP(c),
				c.GetHeader("User-Agent"),
				false,
			); auditErr != nil {
				fmt.Printf("failed to create audit log for admin login failure: %v\n", auditErr)
			}
			return responseData, http.StatusBadRequest, fmt.Errorf("invalid credentials")
		}
	}

	if !admin.IsActive {
		// Audit the attempt against a deactivated account.
		if auditErr := audit_utility.CreateAuditLog(
			db,
			admin.ID,
			admin.Email,
			admin.Role,
			models.ActionAdminLogin,
			models.ResourceAdmin,
			admin.ID,
			"",
			"",
			fmt.Sprintf("Admin login failed: account deactivated for %s", admin.Email),
			audit_utility.GetClientIP(c),
			c.GetHeader("User-Agent"),
			false,
		); auditErr != nil {
			fmt.Printf("failed to create audit log for admin login failure: %v\n", auditErr)
		}
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

	if auditErr := audit_utility.CreateAuditLog(
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
		true,
	); auditErr != nil {
		fmt.Printf("failed to create audit log for admin login: %v\n", auditErr)
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

	plaintextPass, err := GenerateStrongPassword(16)
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

	requesterID, requesterEmail, requesterRole := "", "", ""
	if requester != nil {
		requesterID = requester.ID
		requesterEmail = requester.Email
		requesterRole = requester.Role
	}

	createErr := admin.CreateAdmin(db.Postgresql)

	success := createErr == nil
	description := fmt.Sprintf("Superadmin %s created new admin account for %s", requesterEmail, admin.Email)
	if !success {
		description = fmt.Sprintf("Superadmin %s failed to create admin account for %s: %v", requesterEmail, admin.Email, createErr)
	}

	if auditErr := audit_utility.CreateAuditLog(
		db.Postgresql,
		requesterID,
		requesterEmail,
		requesterRole,
		models.ActionAdminCreate,
		models.ResourceAdmin,
		admin.ID,
		"",
		"",
		description,
		audit_utility.GetClientIP(c),
		c.GetHeader("User-Agent"),
		success,
	); auditErr != nil {
		fmt.Printf("failed to create audit log for admin creation: %v\n", auditErr)
	}

	if createErr != nil {
		return nil, createErr
	}

	// Broadcast audit event only on success, and only when requester is known.
	if requester != nil {
		telexaudit.CreateAdminAudit(db, logger, requester.Email, admin.Email)
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
