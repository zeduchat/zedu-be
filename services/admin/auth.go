package admin

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
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
