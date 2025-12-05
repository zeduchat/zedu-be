package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"

	"context"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/internal/models/migrations"
	"github.com/hngprojects/telex_be/internal/models/seed"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/buzz"
	"github.com/hngprojects/telex_be/pkg/controller/channel"
	"github.com/hngprojects/telex_be/pkg/controller/invitation"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/agora"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	riverqueueBg "github.com/hngprojects/telex_be/pkg/repository/riverqueue"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/pkg/repository/storage/redis"
	"github.com/hngprojects/telex_be/pkg/repository/storage/typesense"
	"github.com/hngprojects/telex_be/utility"
)

func Setup() *utility.Logger {
	logger := utility.NewLogger()
	config := config.Setup(logger, "../../app")

	postgresql.ConnectToDatabase(logger, config.TestDatabase)
	redis.ConnectToRedis(logger, config.Redis)
	typesense.ConnectToTypeSense(logger, config.TypeSense)
	centrifuge.NewCentrifugoService(logger, config.Centrifuge)
	elastic.ConnectToElastic(logger, config.Elastic)
	agora.NewAgoraService(logger, config.Agora)
	db := storage.Connection()
	if config.TestDatabase.Migrate {
		migrations.RunAllMigrations(db)
		seed.SeedRolesAndPermissions(logger, db.Postgresql)
		seed.SeedPlans(logger, db.Postgresql)
	}

	// Initialize River client for background jobs
	ctx := context.Background()
	riverClient, err := riverqueueBg.SetupRiver(ctx, config.TestDatabase, logger, db)
	if err != nil {
		logger.Error("Failed to initialize River client: ", err)
	} else {
		db.River = riverClient
	}

	return logger
}

func ParseResponse(w *httptest.ResponseRecorder) map[string]any {
	res := make(map[string]any)
	json.NewDecoder(w.Body).Decode(&res)
	return res
}

func AssertStatusCode(t *testing.T, got, expected int) {
	if got != expected {
		t.Errorf("handler returned wrong status code: got status %d expected status %d", got, expected)
	}
}

func AssertResponseMessage(t *testing.T, got, expected string) {
	if got != expected {
		t.Errorf("handler returned wrong message: got message: %q expected: %q", got, expected)
	}
}
func AssertBool(t *testing.T, got, expected bool) {
	if got != expected {
		t.Errorf("handler returned wrong boolean: got %v expected %v", got, expected)
	}
}

func AssertValidationError(t *testing.T, response map[string]any, field string, expectedMessage string) {
	errors, ok := response["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'error' field in response")
	}

	errorMessage, exists := errors[field]
	if !exists {
		t.Fatalf("expected validation error message for field '%s'", field)
	}

	if errorMessage != expectedMessage {
		t.Errorf("unexpected error message for field '%s': got %v, want %v", field, errorMessage, expectedMessage)
	}
}

func SignupUser(t *testing.T, r *gin.Engine, auth auth.Controller, userSignUpData models.CreateUserRequestModel, admin bool) {
	var (
		signupPath = "/api/v1/auth/register"
		signupURI  = url.URL{Path: signupPath}
	)

	r.POST(signupPath, auth.RegisterUser)

	var b bytes.Buffer
	json.NewEncoder(&b).Encode(userSignUpData)
	req, err := http.NewRequest(http.MethodPost, signupURI.String(), &b)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
		t.Logf("Registration failed with status %d. Response: %s", rr.Code, rr.Body.String())
	}
}

func GetLoginToken(t *testing.T, r *gin.Engine, auth auth.Controller, loginData models.LoginRequestModel) string {
	var (
		loginPath = "/api/v1/auth/login"
		loginURI  = url.URL{Path: loginPath}
	)
	r.POST(loginPath, auth.LoginUser)
	var b bytes.Buffer
	json.NewEncoder(&b).Encode(loginData)
	req, err := http.NewRequest(http.MethodPost, loginURI.String(), &b)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		return ""
	}

	data := ParseResponse(rr)
	dataM := data["data"].(map[string]any)
	token := dataM["access_token"].(string)

	return token
}

func CreateChannels(t *testing.T, r *gin.Engine, channel channel.Controller, db *storage.Database, CreateData models.CreateChannelsRequest, token string) (string, string) {
	var (
		createPath = "/api/v1/channels/"
		createURI  = url.URL{Path: createPath}
	)

	channelUrl := r.Group(fmt.Sprintf("%v", "/api/v1/channels"), middleware.Authorize(db.Postgresql))
	{
		channelUrl.POST("/", channel.CreateChannel)
	}

	var b bytes.Buffer
	json.NewEncoder(&b).Encode(CreateData)
	req, err := http.NewRequest(http.MethodPost, createURI.String(), &b)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		return "", ""
	}

	data := ParseResponse(rr)
	dataM := data["data"].(map[string]any)
	channelID := dataM["channels_id"].(string)
	channelName := dataM["name"].(string)

	return channelID, channelName
}

func CreateOrganisation(t *testing.T, r *gin.Engine, db *storage.Database, org organisation.Controller, orgData models.CreateOrgRequestModel, token string) (string, string, string) {
	var (
		orgPath = "/api/v1/organisations"
		orgURI  = url.URL{Path: orgPath}
	)
	orgUrl := r.Group(fmt.Sprintf("%v", "/api/v1"), middleware.Authorize(db.Postgresql))
	{
		orgUrl.POST("/organisations", org.CreateOrganisation)
	}
	var b bytes.Buffer
	json.NewEncoder(&b).Encode(orgData)
	req, err := http.NewRequest(http.MethodPost, orgURI.String(), &b)

	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	//get the response
	data := ParseResponse(rr)
	dataM := data["data"].(map[string]any)
	orgID := dataM["id"].(string)
	orgName := dataM["name"].(string)
	ownerID := dataM["owner_id"].(string)
	return orgID, orgName, ownerID
}

func CreateInvitation(t *testing.T, r *gin.Engine, db *storage.Database, invite invitation.Controller, invitereq models.InvitationCreateReq, token string) (string, string) {
	var (
		invitePath = "/api/v1/invite"
		inviteURI  = url.URL{Path: invitePath}
		invitation models.Invitation
	)
	inviteUrl := r.Group(fmt.Sprintf("%v", "/api/v1"))
	{
		inviteUrl.POST("/invite", middleware.Authorize(db.Postgresql), invite.OrganisationInviteMany)
	}

	var b bytes.Buffer
	json.NewEncoder(&b).Encode(invitereq)
	req, err := http.NewRequest(http.MethodPost, inviteURI.String(), &b)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	data := ParseResponse(rr)
	dataM := data["data"].(map[string]any)
	if dataM["errors"] != nil {
		t.Fatal(dataM["errors"])
	}

	err = invitation.GetInvitationByEmail(db.Postgresql, invitereq.Emails[0], invitereq.OrganisationID)
	if err != nil {
		t.Fatal(err)
	}

	invite_token := invitation.Token
	invite_id := invitation.ID

	return invite_token, invite_id
}

func CreateBuzz(t *testing.T, r *gin.Engine, buzzContoller buzz.Controller, db *storage.Database, createBuzzReq models.CreateBuzzRequest, token string) (string, string) {
	var (
		createBuzzPath = "/api/v1/buzz/create"
		createBuzzURI  = url.URL{Path: createBuzzPath}
	)
	buzzUrl := r.Group(fmt.Sprintf("%v", "/api/buzz/create"), middleware.Authorize(db.Postgresql))
	{
		buzzUrl.POST("/", buzzContoller.Create)
	}

	var b bytes.Buffer
	json.NewEncoder(&b).Encode(createBuzzReq)
	req, err := http.NewRequest(http.MethodPost, createBuzzURI.String(), &b)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		return "", ""
	}

	data := ParseResponse(rr)
	dataM := data["data"].(map[string]any)
	buzzID := dataM["buzz_id"].(string)
	hostID := dataM["host_id"].(string)
	return buzzID, hostID
}
