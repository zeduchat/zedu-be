package test_channel

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/gofrs/uuid"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/channel"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

// setupJoinNoticeTestEnv signs up an owner and a joiner, creates an org and
// a channel, and pins the joiner's profile names to the given values.
// Returns the router, db, controller, owner token, joiner token, channel ID
// and the joiner's user ID.
func setupJoinNoticeTestEnv(t *testing.T, fullName, userName, displayName string) (*gin.Engine, *storage.Database, channel.Controller, string, string, string, string) {
	t.Helper()
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	ownerSignup := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("joindisplayowner_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Owner",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("joindisplayowner_%v", currUUID),
	}
	joinerSignup := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("joindisplayjoiner_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Joiner",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("joindisplayjoiner_%v", currUUID),
	}

	authController := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}
	channelController := channel.Controller{Db: db, Validator: validatorRef, Logger: logger}
	orgController := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}

	r := gin.Default()
	tst.SignupUser(t, r, authController, ownerSignup, false)
	tst.SignupUser(t, r, authController, joinerSignup, false)

	ownerToken := tst.GetLoginToken(t, r, authController, models.LoginRequestModel{Email: ownerSignup.Email, Password: ownerSignup.Password})
	joinerToken := tst.GetLoginToken(t, r, authController, models.LoginRequestModel{Email: joinerSignup.Email, Password: joinerSignup.Password})

	orgReq := models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("JoinDisplayOrg_%s", currUUID),
		Description: "Org for join notice display-name tests",
		Email:       ownerSignup.Email,
		Type:        "test",
		Location:    "lagos",
		Country:     "ng",
	}
	orgID, _, _ := tst.CreateOrganisation(t, r, db, orgController, orgReq, ownerToken)

	var joinerModel models.User
	if err := db.Postgresql.Where("email = ?", joinerSignup.Email).First(&joinerModel).Error; err != nil {
		t.Fatalf("failed to load joiner user: %v", err)
	}

	// Add the joiner to the org so their profile resolves for this org.
	db.Postgresql.Create(&models.OrgUserManagement{
		UserID:         joinerModel.ID,
		OrganisationID: orgID,
		Status:         "active",
		RoleID:         utility.GenerateUUID(),
	})

	// Point the joiner's active organisation at the org they just joined so
	// their token is org-scoped, mirroring a client that switched organisation.
	joinerModel.CurrentOrg = uuid.FromStringOrNil(orgID)
	if err := db.Postgresql.Save(&joinerModel).Error; err != nil {
		t.Fatalf("failed to update joiner current org: %v", err)
	}

	// Pin the joiner's profile identity fields explicitly.
	var prof models.Profile
	if err := prof.GetProfileByUserId(db.Postgresql, joinerModel.ID, orgID); err != nil {
		t.Fatalf("failed to load joiner profile: %v", err)
	}
	if err := db.Postgresql.Model(&models.Profile{}).Where("id = ?", prof.ID).Updates(map[string]any{
		"full_name":    fullName,
		"user_name":    userName,
		"display_name": displayName,
	}).Error; err != nil {
		t.Fatalf("failed to update joiner profile: %v", err)
	}

	// Re-issue tokens now that both users are scoped to the created org, so the
	// signed-in org_id in the claims matches the org the resources live in.
	ownerToken = tst.GetLoginToken(t, r, authController, models.LoginRequestModel{Email: ownerSignup.Email, Password: ownerSignup.Password})
	joinerToken = tst.GetLoginToken(t, r, authController, models.LoginRequestModel{Email: joinerSignup.Email, Password: joinerSignup.Password})

	chanReq := models.CreateChannelsRequest{
		Name:           fmt.Sprintf("joindisplay-%s", currUUID),
		Username:       ownerSignup.UserName,
		OrganisationID: orgID,
		Description:    "Channel for join notice display-name tests",
	}
	channelID, _ := tst.CreateChannels(t, r, channelController, db, chanReq, ownerToken)

	return r, db, channelController, ownerToken, joinerToken, channelID, joinerModel.ID
}

func joinChannelWithBody(t *testing.T, r *gin.Engine, db *storage.Database, channelController channel.Controller, channelID, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	joinRouter := gin.Default()
	channelUrl := joinRouter.Group("/api/v1/channels", middleware.Authorize(db.Postgresql))
	{
		channelUrl.POST("/:channelId/join", channelController.JoinChannels)
	}
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/channels/%s/join", channelID), strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	joinRouter.ServeHTTP(rr, req)
	return rr
}
func TestChannelJoinNoticeUsesDisplayName(t *testing.T) {
	logger := tst.Setup()
	r, db, channelController, ownerToken, joinerToken, channelID, _ :=
		setupJoinNoticeTestEnv(t, "Pauline Banye", "pauline.banye", "Lynn B")

	defer tst.Cleanup(db)

	rr := joinChannelWithBody(t, r, db, channelController, channelID, joinerToken, `{"username":"pauline.banye"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 for join, got %d: %s", rr.Code, rr.Body.String())
	}

	time.Sleep(2 * time.Second)

	sysMsg := tst.FetchSystemMessage(t, db, logger, channelID, ownerToken)
	if sysMsg == nil {
		t.Fatal("expected a join system message to exist")
	}
	content, ok := sysMsg["message"].(string)
	if !ok {
		t.Fatal("join system message has no message content")
	}
	if !strings.Contains(content, "Lynn B") {
		t.Errorf("expected join notice to contain display name %q, got %q", "Lynn B", content)
	}
	if strings.Contains(content, "Pauline Banye") {
		t.Errorf("expected join notice to omit full name, got %q", content)
	}
	var records any
	query := map[string]any{"query": map[string]any{"term": map[string]any{"channels_id": channelID}}}
	if err := elastic.SelectAll(db.Elastic, models.ThreadIndexName, query, &records); err != nil {
		t.Fatal(err)
	}
	hits := records.(map[string]any)["hits"].(map[string]any)["hits"].([]any)
	creationCount := 0
	for _, hit := range hits {
		source := hit.(map[string]any)["_source"].(map[string]any)
		if strings.Contains(utility.GetString(source, "message"), " created #") {
			creationCount++
		}
	}
	if creationCount != 1 {
		t.Fatalf("joining must preserve the channel creation notice; got %d creation notices", creationCount)
	}
}

func TestChannelJoinNoticeIgnoresClientSuppliedUsername(t *testing.T) {
	logger := tst.Setup()
	r, db, channelController, ownerToken, joinerToken, channelID, _ :=
		setupJoinNoticeTestEnv(t, "Pauline Banye", "pauline.banye", "Lynn B")

	defer tst.Cleanup(db)

	// The legacy frontend sends whatever string it has locally, which may be
	// the user's full name. The backend must not use it as the notice label.
	rr := joinChannelWithBody(t, r, db, channelController, channelID, joinerToken, `{"username":"Pauline Banye"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 for join, got %d: %s", rr.Code, rr.Body.String())
	}

	time.Sleep(2 * time.Second)

	sysMsg := tst.FetchSystemMessage(t, db, logger, channelID, ownerToken)
	if sysMsg == nil {
		t.Fatal("expected a join system message to exist")
	}
	content, ok := sysMsg["message"].(string)
	if !ok {
		t.Fatal("join system message has no message content")
	}
	if !strings.Contains(content, "Lynn B") {
		t.Errorf("expected join notice to contain display name %q, got %q", "Lynn B", content)
	}
	if strings.Contains(content, "Pauline Banye") {
		t.Errorf("expected join notice to ignore client-supplied full name, got %q", content)
	}
}

func TestChannelJoinNoticeFallsBackToUsername(t *testing.T) {
	logger := tst.Setup()
	r, db, channelController, ownerToken, joinerToken, channelID, _ :=
		setupJoinNoticeTestEnv(t, "Pauline Banye", "pauline.banye", "")

	defer tst.Cleanup(db)

	rr := joinChannelWithBody(t, r, db, channelController, channelID, joinerToken, `{"username":"Pauline Banye"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 for join, got %d: %s", rr.Code, rr.Body.String())
	}

	time.Sleep(2 * time.Second)

	sysMsg := tst.FetchSystemMessage(t, db, logger, channelID, ownerToken)
	if sysMsg == nil {
		t.Fatal("expected a join system message to exist")
	}
	content, ok := sysMsg["message"].(string)
	if !ok {
		t.Fatal("join system message has no message content")
	}
	if !strings.Contains(content, "pauline.banye") {
		t.Errorf("expected join notice to fall back to username %q, got %q", "pauline.banye", content)
	}
	if strings.Contains(content, "Pauline Banye") {
		t.Errorf("expected join notice to omit full name, got %q", content)
	}
}

func TestChannelJoinNoticeMentionReferencesUserID(t *testing.T) {
	logger := tst.Setup()
	r, db, channelController, ownerToken, joinerToken, channelID, joinerID :=
		setupJoinNoticeTestEnv(t, "Pauline Banye", "pauline.banye", "Lynn B")

	defer tst.Cleanup(db)

	rr := joinChannelWithBody(t, r, db, channelController, channelID, joinerToken, `{"username":"pauline.banye"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 for join, got %d: %s", rr.Code, rr.Body.String())
	}

	time.Sleep(2 * time.Second)

	sysMsg := tst.FetchSystemMessage(t, db, logger, channelID, ownerToken)
	if sysMsg == nil {
		t.Fatal("expected a join system message to exist")
	}
	content, ok := sysMsg["message"].(string)
	if !ok {
		t.Fatal("join system message has no message content")
	}
	if !strings.Contains(content, fmt.Sprintf(`data-id="%s"`, joinerID)) {
		t.Errorf("expected join notice mention to reference user id %q, got %q", joinerID, content)
	}
}
