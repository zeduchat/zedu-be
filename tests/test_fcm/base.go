package test_fcm

import (
	"fmt"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	fcmcontroller "github.com/hngprojects/telex_be/pkg/controller/fcmTokens"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func SetupFCMTestRouter() (*gin.Engine, *fcmcontroller.Controller) {
	gin.SetMode(gin.TestMode)

	db := storage.Connection()
	logger := tst.Setup()
	validate := validator.New()

	fcmCtrl := &fcmcontroller.Controller{
		Db:        db,
		Validator: validate,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	r := gin.Default()
	fcmGroup := r.Group("/api/v1/fcmtoken", middleware.Authorize(db.Postgresql))
	{
		fcmGroup.POST("", fcmCtrl.CreateFcmToken)
		fcmGroup.DELETE("", fcmCtrl.DeleteFcmToken)
	}

	return r, fcmCtrl
}

func CreateTestUser(t *testing.T, db *storage.Database) models.User {
	uid := utility.GenerateUUID()
	user := models.User{
		ID:       uid,
		Email:    fmt.Sprintf("fcm_test_%v@example.com", uid),
		Name:     fmt.Sprintf("fcmuser_%v", uid),
		Password: "password123",
		Role:     int(models.RoleIdentity.User),
	}

	if err := db.Postgresql.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return user
}

func GenerateTestToken(t *testing.T, db *storage.Database, userID string) string {
	cfg := config.GetConfig()
	accessUUID := utility.GenerateUUID()
	claims := jwt.MapClaims{
		"user_id":     userID,
		"org_id":      "00000000-0000-0000-0000-000000000000",
		"access_uuid": accessUUID,
		"exp":         time.Now().Add(time.Hour * 24).Unix(),
		"authorised":  true,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.Server.Secret))
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	accessToken := models.AccessToken{
		ID:               accessUUID,
		LoginAccessToken: tokenString,
		OwnerID:          userID,
		IsLive:           true,
	}
	if err := db.Postgresql.Create(&accessToken).Error; err != nil {
		t.Fatalf("Failed to create access token: %v", err)
	}

	return tokenString
}
