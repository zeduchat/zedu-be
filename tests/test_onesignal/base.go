package test_onesignal

import (
	"fmt"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/user"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/pushNotifications/onesignal"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func SetupOnesignalTestRouter() (*gin.Engine, *user.Controller) {
	gin.SetMode(gin.TestMode)

	db := storage.Connection()
	logger := tst.Setup()
	validator := validator.New()

	onesignal.ConnectOneSignal(logger, config.GetConfig().OneSignal)

	userController := &user.Controller{
		Db:        db,
		Validator: validator,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	r := gin.Default()
	SetupOnesignalRoutes(r, userController, db)
	return r, userController
}

func SetupOnesignalRoutes(r *gin.Engine, userController *user.Controller, db *storage.Database) {
	userUrl := r.Group("/api/v1", middleware.Authorize(db.Postgresql))
	{
		userUrl.POST("/users/onesignal-subscription-id", userController.RegisterOneSignalSubscriptionID)
		userUrl.PUT("/users/onesignal-subscription-id", userController.UpdateOneSignalSubscriptionID)
		userUrl.GET("/users/onesignal-subscription-id", userController.GetOneSignalSubscriptionID)
	}
}

func CreateTestUser(t *testing.T, db *storage.Database) models.User {
	uuid := utility.GenerateUUID()
	user := models.User{
		ID:       uuid,
		Email:    fmt.Sprintf("test%v@example.com", uuid),
		Name:     fmt.Sprintf("testuser%v", uuid),
		Password: "password123",
		Role:     int(models.RoleIdentity.User),
	}

	if err := db.Postgresql.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return user
}
