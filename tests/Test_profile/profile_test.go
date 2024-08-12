package test_profile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/profile"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func StringPtr(s string) *string {
	return &s
}

func TestUserProfileFlow(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

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

	authController := auth.Controller{Db: db, Validator: validatorRef, Logger: logger}
	r := gin.Default()

	tst.SignupUser(t, r, authController, userSignUpData, false)

	token := tst.GetLoginToken(t, r, authController, loginData)

	getProfileURI := url.URL{Path: "/api/v1/profile"}
	profileController := profile.Controller{Db: db, Validator: validatorRef, Logger: logger}
	r = gin.Default()
	r.GET(getProfileURI.Path, middleware.Authorize(db.Postgresql),  profileController.GetUserProfile)

	req, _ := http.NewRequest(http.MethodGet, getProfileURI.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	updateProfileURI := url.URL{Path: "/api/v1/profile"}
	updateProfileBody := models.UpdateUserProfileRequest{
		FullName:  StringPtr("Updated Full Name"),
		UserName:  StringPtr("updated_username"),
		Phone:     StringPtr("+2348112345678"),
		AvatarURL: StringPtr("new_avatar_url.png"),
	}

	r = gin.Default()
	r.PATCH(updateProfileURI.Path, middleware.Authorize(db.Postgresql), profileController.UpdateProfile)

	var b bytes.Buffer
	json.NewEncoder(&b).Encode(updateProfileBody)
	req, _ = http.NewRequest(http.MethodPatch, updateProfileURI.String(), &b)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)

	updateResponse := tst.ParseResponse(rr)
	tst.AssertResponseMessage(t, updateResponse["message"].(string), "Profile updated successfully")
}