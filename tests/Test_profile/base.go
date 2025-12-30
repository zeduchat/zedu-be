package test_profile

import (
	"fmt"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/profile"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/minio"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func initialise(currUUID string, t *testing.T, r *gin.Engine, user auth.Controller, status bool) string {
	userSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testuser%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "test",
		LastName:    "user",
		Password:    "password",
		UserName:    fmt.Sprintf("test_username%v", currUUID),
	}
	loginData := models.LoginRequestModel{
		Email:    userSignUpData.Email,
		Password: userSignUpData.Password,
	}

	tst.SignupUser(t, r, user, userSignUpData, status)

	token := tst.GetLoginToken(t, r, user, loginData)

	return token
}

func SetupProfileTestRouter() (*gin.Engine, *profile.Controller) {
	gin.SetMode(gin.TestMode)

	logger := tst.Setup()
	db := storage.Connection()
	config := config.Setup(logger, "../../app")
	minio.ConnectToMinio(logger, config.Minio)
	validator := validator.New()

	profileController := &profile.Controller{
		Db:        db,
		Validator: validator,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	r := gin.Default()
	SetupProfileRoutes(r, profileController)
	return r, profileController
}

func SetupProfileRoutes(r *gin.Engine, profileController *profile.Controller) {
	profileUrl := r.Group("/api/v1",
		middleware.Authorize(profileController.Db.Postgresql))
	profileUrl.PATCH("/profile", profileController.UpdateProfile)
	profileUrl.GET("/profile", profileController.GetUserProfile)
	profileUrl.DELETE("/profile/image", profileController.DeleteUserProfileImage)
}
