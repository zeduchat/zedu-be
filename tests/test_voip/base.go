package test_voip

import (
	"fmt"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/user"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func SetupVoIPTestRouter() (*gin.Engine, *user.Controller) {
	gin.SetMode(gin.TestMode)

	db := storage.Connection()
	logger := tst.Setup()
	validator := validator.New()

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
	SetupVoIPRoutes(r, userController, db)
	return r, userController
}

func SetupVoIPRoutes(r *gin.Engine, userController *user.Controller, db *storage.Database) {
	userUrl := r.Group("/api/v1", middleware.Authorize(db.Postgresql))
	{
		userUrl.PUT("/users/voip-push-token", userController.UpdateVoIPToken)
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
