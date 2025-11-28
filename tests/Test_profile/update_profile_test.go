package test_profile

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestUpdateProfileAddsNewFields(t *testing.T) {
	_, profileController := SetupProfileTestRouter()
	db := profileController.Db.Postgresql

	currUUID := utility.GenerateUUID()
	password, _ := utility.HashPassword("password")

	regularUser := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "Regular User",
		Email:    "user" + currUUID + "@qa.team",
		Password: password,
	}

	db.Create(&regularUser)

	setup := func() (*gin.Engine, *auth.Controller) {
		router, profileController := SetupProfileTestRouter()
		authController := auth.Controller{
			Db:        profileController.Db,
			Validator: profileController.Validator,
			Logger:    profileController.Logger,
			ExtReq:    profileController.ExtReq,
		}
		return router, &authController
	}

	router, authController := setup()

	loginData := models.LoginRequestModel{
		Email:    regularUser.Email,
		Password: "password",
	}

	token := tst.GetLoginToken(t, router, *authController, loginData)

	// create profile
	prof := models.Profile{ID: utility.GenerateUUID(), Userid: regularUser.ID}
	db.Create(&prof)

	// profile variables
	fullName := "New Full Name"
	workspaceID := "Workspace 123"
	track := "Engineering"

	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	writer.WriteField("full_name", fullName)
	writer.WriteField("workspace_id", workspaceID)
	writer.WriteField("track", track)

	links := []string{"https://portfolio.example/user", "https://linkedin.com/in/user"}
	for _, l := range links {
		writer.WriteField("links", l)
	}

	writer.Close()

	req, _ := http.NewRequest(http.MethodPatch, "/api/v1/profile", &b)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	tst.AssertStatusCode(t, resp.Code, http.StatusOK)

	// fetch profile and compare returned values
	getReq, _ := http.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	getReq.Header.Set("Content-Type", "application/json")
	getReq.Header.Set("Authorization", "Bearer "+token)

	getRespRec := httptest.NewRecorder()
	router.ServeHTTP(getRespRec, getReq)

	tst.AssertStatusCode(t, getRespRec.Code, http.StatusOK)

	var getBody struct {
		Data models.ProfileSummary `json:"data"`
	}
	if err := json.Unmarshal(getRespRec.Body.Bytes(), &getBody); err != nil {
		t.Fatalf("failed to unmarshal profile GET response: %v", err)
	}

	returned := getBody.Data

	if returned.WorkspaceID != workspaceID {
		t.Fatalf("workspace_id mismatch: want %q got %q", workspaceID, returned.WorkspaceID)
	}
	if returned.Track != track {
		t.Fatalf("track mismatch: want %q got %q", track, returned.Track)
	}

	t.Logf("returned links: %v", returned.Links)
	if len(returned.Links) != len(links) {
		t.Fatalf("links length mismatch: want %d got %d", len(links), len(returned.Links))
	}
	for i, l := range links {
		if returned.Links[i] != l {
			t.Fatalf("links[%d] mismatch: want %q got %q", i, l, returned.Links[i])
		}
	}
}
