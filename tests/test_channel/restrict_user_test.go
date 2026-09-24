package test_channel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/channel"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/controller/thread"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestGetUserDefaultPermissions(t *testing.T) {
	perms := models.GetUserDefaultPermissions()
	if perms.CanCreateChannels {
		t.Errorf("expected CanCreateChannels to be false for default user permissions, got true")
	}
}

func TestRestrictUserFlowEndToEnd(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	// 1. Create Owner User
	ownerSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("owner_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Owner",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("owner_%v", currUUID),
	}

	authCtrl := auth.Controller{
		Db: db, Validator: validatorRef, Logger: logger,
		ExtReq: request.ExternalRequest{Logger: logger, Test: true},
	}
	channelCtrl := channel.Controller{Db: db, Validator: validatorRef, Logger: logger}
	threadCtrl := thread.Controller{Db: db, Validator: validatorRef, Logger: logger}
	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}

	r := gin.Default()

	// Register router groups
	channelUrl := r.Group("/api/v1/channels", middleware.Authorize(db.Postgresql))
	{
		channelUrl.POST("", channelCtrl.CreateChannel)
		channelUrl.GET("/:channelId", channelCtrl.GetChannel)
		channelUrl.GET("/:channelId/users", channelCtrl.GetUsersInChannel)
		channelUrl.POST("/:channelId/join", channelCtrl.JoinChannels)
		channelUrl.POST("/:channelId/messages", channelCtrl.ReplyThreadMessage)
		channelUrl.PATCH("/:channelId/users/:userId/restrict", channelCtrl.RestrictUser)
		channelUrl.PATCH("/:channelId/users/restrict-all", channelCtrl.RestrictAllUsers)
	}

	threadUrl := r.Group("/api/v1/threads", middleware.Authorize(db.Postgresql))
	{
		threadUrl.POST("/:channel_id", threadCtrl.AddAThread)
	}

	orgUrl := r.Group("/api/v1/organisations", middleware.Authorize(db.Postgresql))
	{
		orgUrl.GET("/:org_id/user-channels", channelCtrl.GetUserChannels)
	}

	tst.SignupUser(t, r, authCtrl, ownerSignUpData, false)
	ownerToken := tst.GetLoginToken(t, r, authCtrl, models.LoginRequestModel{
		Email:    ownerSignUpData.Email,
		Password: ownerSignUpData.Password,
	})

	// Create Organisation
	createOrgData := models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("OrgRestrictTest%s", currUUID),
		Description: "Org description",
		Email:       ownerSignUpData.Email,
		Type:        "test",
		Location:    "test",
		Country:     "test",
	}
	orgId, _, _ := tst.CreateOrganisation(t, r, db, orgCtrl, createOrgData, ownerToken)

	// Create Channel
	createChanData := models.CreateChannelsRequest{
		Name:           fmt.Sprintf("chan_%s", currUUID),
		Username:       fmt.Sprintf("chan_user_%s", currUUID),
		OrganisationID: orgId,
		Description:    "Test channel for restrictions",
	}
	channelID, _ := tst.CreateChannels(t, r, channelCtrl, db, createChanData, ownerToken)

	// 2. Create User 1 & User 2
	user1SignUp := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("user1_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "User1",
		LastName:    "Test",
		Password:    "password",
		UserName:    fmt.Sprintf("user1_%v", currUUID),
	}
	tst.SignupUser(t, r, authCtrl, user1SignUp, false)
	token1 := tst.GetLoginToken(t, r, authCtrl, models.LoginRequestModel{
		Email:    user1SignUp.Email,
		Password: user1SignUp.Password,
	})

	var user1Model models.User
	db.Postgresql.Where("email = ?", user1SignUp.Email).First(&user1Model)
	db.Postgresql.Create(&models.OrgUserManagement{
		UserID:         user1Model.ID,
		OrganisationID: orgId,
		Status:         "active",
		RoleID:         utility.GenerateUUID(),
	})

	user2SignUp := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("user2_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "User2",
		LastName:    "Test",
		Password:    "password",
		UserName:    fmt.Sprintf("user2_%v", currUUID),
	}
	tst.SignupUser(t, r, authCtrl, user2SignUp, false)
	token2 := tst.GetLoginToken(t, r, authCtrl, models.LoginRequestModel{
		Email:    user2SignUp.Email,
		Password: user2SignUp.Password,
	})

	var user2Model models.User
	db.Postgresql.Where("email = ?", user2SignUp.Email).First(&user2Model)
	db.Postgresql.Create(&models.OrgUserManagement{
		UserID:         user2Model.ID,
		OrganisationID: orgId,
		Status:         "active",
		RoleID:         utility.GenerateUUID(),
	})

	// User 1 & User 2 Join Channel
	performPostRequest(t, r, fmt.Sprintf("/api/v1/channels/%s/join", channelID), token1, map[string]any{}, http.StatusOK)
	performPostRequest(t, r, fmt.Sprintf("/api/v1/channels/%s/join", channelID), token2, map[string]any{}, http.StatusOK)

	// 3. Verify initial non-restricted state in channel users list
	respGetUsers := performGetRequest(t, r, fmt.Sprintf("/api/v1/channels/%s/users", channelID), ownerToken, http.StatusOK)
	dataMap := tst.ParseResponse(respGetUsers)
	if dataMap["status"] != "success" {
		t.Fatalf("expected success status fetching channel users")
	}

	// 4. Test unauthorized restriction attempt by User 1 (non-owner) targeting User 2
	restrictPayload := map[string]bool{"restricted": true}
	performPatchRequest(t, r, fmt.Sprintf("/api/v1/channels/%s/users/%s/restrict", channelID, user2Model.ID), token1, restrictPayload, http.StatusForbidden)

	// 5. Test restriction of User 1 by Channel Owner
	performPatchRequest(t, r, fmt.Sprintf("/api/v1/channels/%s/users/%s/restrict", channelID, user1Model.ID), ownerToken, restrictPayload, http.StatusOK)

	// 6. Test User 1 top-level thread creation -> Expect 403 Forbidden
	threadReq := models.CreateThreadMsgReq{
		Content: "<p>Top level thread by restricted user</p>",
	}
	performPostRequest(t, r, fmt.Sprintf("/api/v1/threads/%s", channelID), token1, threadReq, http.StatusForbidden)

	// 7. Test User 2 (unrestricted) top-level thread creation -> Expect 201 Created
	threadReq2 := models.CreateThreadMsgReq{
		Content: "<p>Top level thread by unrestricted user</p>",
	}
	respThread2 := performPostRequest(t, r, fmt.Sprintf("/api/v1/threads/%s", channelID), token2, threadReq2, http.StatusCreated)
	threadData2 := tst.ParseResponse(respThread2)
	threadDataMap, ok := threadData2["data"].(map[string]any)
	if !ok {
		t.Fatalf("failed to parse created thread data")
	}
	createdThreadID := threadDataMap["thread_id"].(string)

	// 8. Test Restricted User 1 replying to thread -> Expect 201 Created
	replyReq := models.CreateMessageRequest{
		Content:    "<p>Reply by restricted user</p>",
		ChannelsId: channelID,
		ThreadId:   createdThreadID,
	}
	performPostRequest(t, r, fmt.Sprintf("/api/v1/channels/%s/messages", channelID), token1, replyReq, http.StatusCreated)

	// 9. Test Bulk Restrict All Users by Owner
	performPatchRequest(t, r, fmt.Sprintf("/api/v1/channels/%s/users/restrict-all", channelID), ownerToken, map[string]bool{"restricted": true}, http.StatusOK)

	// User 2 (now restricted via restrict-all) attempts top-level thread -> Expect 403 Forbidden
	performPostRequest(t, r, fmt.Sprintf("/api/v1/threads/%s", channelID), token2, threadReq2, http.StatusForbidden)

	// 10. Test New Member User 3 joining restricted channel -> Inherits restriction
	user3SignUp := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("user3_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "User3",
		LastName:    "Test",
		Password:    "password",
		UserName:    fmt.Sprintf("user3_%v", currUUID),
	}
	tst.SignupUser(t, r, authCtrl, user3SignUp, false)
	token3 := tst.GetLoginToken(t, r, authCtrl, models.LoginRequestModel{
		Email:    user3SignUp.Email,
		Password: user3SignUp.Password,
	})

	var user3Model models.User
	db.Postgresql.Where("email = ?", user3SignUp.Email).First(&user3Model)
	db.Postgresql.Create(&models.OrgUserManagement{
		UserID:         user3Model.ID,
		OrganisationID: orgId,
		Status:         "active",
		RoleID:         utility.GenerateUUID(),
	})

	performPostRequest(t, r, fmt.Sprintf("/api/v1/channels/%s/join", channelID), token3, map[string]any{}, http.StatusOK)
	// User 3 should be restricted
	performPostRequest(t, r, fmt.Sprintf("/api/v1/threads/%s", channelID), token3, threadReq, http.StatusForbidden)

	// 11. Bulk Unrestrict All Users by Owner
	performPatchRequest(t, r, fmt.Sprintf("/api/v1/channels/%s/users/restrict-all", channelID), ownerToken, map[string]bool{"restricted": false}, http.StatusOK)

	// User 1, User 2, User 3 can now post top-level threads
	performPostRequest(t, r, fmt.Sprintf("/api/v1/threads/%s", channelID), token1, threadReq, http.StatusCreated)
	performPostRequest(t, r, fmt.Sprintf("/api/v1/threads/%s", channelID), token3, threadReq, http.StatusCreated)
}

func TestSingleUserEnableDisableFlow(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	ownerSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("single_owner_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "SingleOwner",
		LastName:    "User",
		Password:    "password",
		UserName:    fmt.Sprintf("single_owner_%v", currUUID),
	}

	authCtrl := auth.Controller{
		Db: db, Validator: validatorRef, Logger: logger,
		ExtReq: request.ExternalRequest{Logger: logger, Test: true},
	}
	channelCtrl := channel.Controller{Db: db, Validator: validatorRef, Logger: logger}
	threadCtrl := thread.Controller{Db: db, Validator: validatorRef, Logger: logger}
	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}

	r := gin.Default()

	channelUrl := r.Group("/api/v1/channels", middleware.Authorize(db.Postgresql))
	{
		channelUrl.POST("", channelCtrl.CreateChannel)
		channelUrl.GET("/:channelId", channelCtrl.GetChannel)
		channelUrl.GET("/:channelId/users", channelCtrl.GetUsersInChannel)
		channelUrl.POST("/:channelId/join", channelCtrl.JoinChannels)
		channelUrl.PATCH("/:channelId/users/:userId/restrict", channelCtrl.RestrictUser)
	}

	threadUrl := r.Group("/api/v1/threads", middleware.Authorize(db.Postgresql))
	{
		threadUrl.POST("/:channel_id", threadCtrl.AddAThread)
	}

	tst.SignupUser(t, r, authCtrl, ownerSignUpData, false)
	ownerToken := tst.GetLoginToken(t, r, authCtrl, models.LoginRequestModel{
		Email:    ownerSignUpData.Email,
		Password: ownerSignUpData.Password,
	})

	var ownerModel models.User
	db.Postgresql.Where("email = ?", ownerSignUpData.Email).First(&ownerModel)

	createOrgData := models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("OrgSingleTest%s", currUUID),
		Description: "Org description",
		Email:       ownerSignUpData.Email,
		Type:        "test",
		Location:    "test",
		Country:     "test",
	}
	orgId, _, _ := tst.CreateOrganisation(t, r, db, orgCtrl, createOrgData, ownerToken)

	createChanData := models.CreateChannelsRequest{
		Name:           fmt.Sprintf("chan_single_%s", currUUID),
		Username:       fmt.Sprintf("chan_user_single_%s", currUUID),
		OrganisationID: orgId,
		Description:    "Test channel for single user restriction",
	}
	channelID, _ := tst.CreateChannels(t, r, channelCtrl, db, createChanData, ownerToken)

	user1SignUp := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("single_user1_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "SingleUser1",
		LastName:    "Test",
		Password:    "password",
		UserName:    fmt.Sprintf("single_user1_%v", currUUID),
	}
	tst.SignupUser(t, r, authCtrl, user1SignUp, false)
	token1 := tst.GetLoginToken(t, r, authCtrl, models.LoginRequestModel{
		Email:    user1SignUp.Email,
		Password: user1SignUp.Password,
	})

	var user1Model models.User
	db.Postgresql.Where("email = ?", user1SignUp.Email).First(&user1Model)
	db.Postgresql.Create(&models.OrgUserManagement{
		UserID:         user1Model.ID,
		OrganisationID: orgId,
		Status:         "active",
		RoleID:         utility.GenerateUUID(),
	})

	performPostRequest(t, r, fmt.Sprintf("/api/v1/channels/%s/join", channelID), token1, map[string]any{}, http.StatusOK)

	// 1. Initially User 1 can post top-level thread
	threadReq := models.CreateThreadMsgReq{
		Content: "<p>Top level thread by User 1</p>",
	}
	performPostRequest(t, r, fmt.Sprintf("/api/v1/threads/%s", channelID), token1, threadReq, http.StatusCreated)

	// 2. Owner restricts User 1 (disable write access)
	performPatchRequest(t, r, fmt.Sprintf("/api/v1/channels/%s/users/%s/restrict", channelID, user1Model.ID), ownerToken, map[string]bool{"restricted": true}, http.StatusOK)

	// 3. User 1 attempts top-level thread -> Expect 403 Forbidden
	performPostRequest(t, r, fmt.Sprintf("/api/v1/threads/%s", channelID), token1, threadReq, http.StatusForbidden)

	// 4. Owner unrestricts/enables User 1 (re-enable write access)
	performPatchRequest(t, r, fmt.Sprintf("/api/v1/channels/%s/users/%s/restrict", channelID, user1Model.ID), ownerToken, map[string]bool{"restricted": false}, http.StatusOK)

	// 5. User 1 posts top-level thread again -> Expect 201 Created
	performPostRequest(t, r, fmt.Sprintf("/api/v1/threads/%s", channelID), token1, threadReq, http.StatusCreated)

	// 6. Owner attempts to restrict themselves -> Expect 400 Bad Request
	performPatchRequest(t, r, fmt.Sprintf("/api/v1/channels/%s/users/%s/restrict", channelID, ownerModel.ID), ownerToken, map[string]bool{"restricted": true}, http.StatusBadRequest)
}

func performPostRequest(t *testing.T, r *gin.Engine, path, token string, body any, expectedCode int) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req, err := http.NewRequest(http.MethodPost, path, &buf)
	if err != nil {
		t.Fatalf("failed to create POST request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	tst.AssertStatusCode(t, rr.Code, expectedCode)
	return rr
}

func performPatchRequest(t *testing.T, r *gin.Engine, path, token string, body any, expectedCode int) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req, err := http.NewRequest(http.MethodPatch, path, &buf)
	if err != nil {
		t.Fatalf("failed to create PATCH request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	tst.AssertStatusCode(t, rr.Code, expectedCode)
	return rr
}

func performGetRequest(t *testing.T, r *gin.Engine, path, token string, expectedCode int) *httptest.ResponseRecorder {
	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		t.Fatalf("failed to create GET request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	tst.AssertStatusCode(t, rr.Code, expectedCode)
	return rr
}
