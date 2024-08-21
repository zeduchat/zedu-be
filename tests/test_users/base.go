package test_users

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/user"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/tests"
)

type UserOrganisation struct {
	UserID         string `gorm:"type:uuid;not null" json:"user_id"`
	OrganisationID string `gorm:"type:uuid;not null" json:"organisation_id"`
}

func SetupUsersTestRouter() (*gin.Engine, *user.Controller) {
	gin.SetMode(gin.TestMode)

	logger := tests.Setup()
	db := storage.Connection()
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
	SetupUsersRoutes(r, userController)
	return r, userController
}

func SetupUsersRoutes(r *gin.Engine, userController *user.Controller) {
	r.PUT("/api/v1/users/switch-roles",
		middleware.Authorize(userController.Db.Postgresql),
		userController.AssignRoleToUser)
	r.GET("/api/v1/users/:user_id/login-audit", middleware.Authorize(userController.Db.Postgresql),
		userController.GetUserLoginAudit)
	r.PUT("/api/v1/users/revoke-session", middleware.Authorize(userController.Db.Postgresql),
		userController.RevokeUserAccessToken)
	r.GET("/api/v1/users/:user_id/organisations",
		middleware.Authorize(userController.Db.Postgresql),
		userController.GetAUserOrganisation)
}
