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
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

// The archive route must check the signed-in user, not a user_id in the body (Ziki #1591).
func TestArchiveChannelUsesTheSignedInUser(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	authCtrl := auth.Controller{
		Db: db, Validator: validatorRef, Logger: logger,
		ExtReq: request.ExternalRequest{Logger: logger, Test: true},
	}
	channelCtrl := channel.Controller{Db: db, Validator: validatorRef, Logger: logger}
	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}

	r := gin.Default()
	channelUrl := r.Group("/api/v1/channels", middleware.Authorize(db.Postgresql))
	{
		channelUrl.POST("", channelCtrl.CreateChannel)
		channelUrl.POST("/:channelId/join", channelCtrl.JoinChannels)
		channelUrl.PUT("/:channelId/archive", channelCtrl.ArchiveChannel)
	}
	r.GET("/api/v1/organisations/:org_id/channels/archived", middleware.Authorize(db.Postgresql), channelCtrl.GetArchivedChannels)

	signUp := func(prefix string) (models.User, string) {
		data := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("%s_%v@qa.team", prefix, currUUID),
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
			FirstName:   prefix,
			LastName:    "Archive",
			Password:    "password",
			UserName:    fmt.Sprintf("%s_%v", prefix, currUUID),
		}
		tst.SignupUser(t, r, authCtrl, data, false)
		token := tst.GetLoginToken(t, r, authCtrl, models.LoginRequestModel{Email: data.Email, Password: data.Password})
		var user models.User
		db.Postgresql.Where("email = ?", data.Email).First(&user)
		return user, token
	}

	owner, ownerToken := signUp("archowner")
	orgId, _, _ := tst.CreateOrganisation(t, r, db, orgCtrl, models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("OrgArchiveTest%s", currUUID),
		Description: "Org description",
		Email:       owner.Email,
		Type:        "test",
		Location:    "test",
		Country:     "test",
	}, ownerToken)
	channelID, _ := tst.CreateChannels(t, r, channelCtrl, db, models.CreateChannelsRequest{
		Name:           fmt.Sprintf("arch_%s", currUUID),
		Username:       fmt.Sprintf("arch_user_%s", currUUID),
		OrganisationID: orgId,
		Description:    "Channel for archive permission tests",
	}, ownerToken)

	// A member of the organisation whose role has no channel permissions.
	member, memberToken := signUp("archmember")
	db.Postgresql.Create(&models.OrgUserManagement{
		UserID:         member.ID,
		OrganisationID: orgId,
		Status:         "active",
		RoleID:         utility.GenerateUUID(),
	})
	performPostRequest(t, r, fmt.Sprintf("/api/v1/channels/%s/join", channelID), memberToken, map[string]any{}, http.StatusOK)

	// Someone with no link to the organisation at all.
	_, outsiderToken := signUp("archoutsider")

	path := fmt.Sprintf("/api/v1/channels/%s/archive", channelID)
	isArchived := func() bool {
		var ch models.Channels
		db.Postgresql.Raw("SELECT id, COALESCE(archived, false) AS archived FROM channels WHERE id = ?", channelID).Scan(&ch)
		return ch.Archived
	}

	// Naming the owner in the body must not lend the owner's rights.
	performPutRequest(t, r, path, outsiderToken, map[string]any{"archived": true, "user_id": owner.ID}, http.StatusForbidden)
	performPutRequest(t, r, path, memberToken, map[string]any{"archived": true, "user_id": owner.ID}, http.StatusForbidden)
	if isArchived() {
		t.Fatalf("channel was archived by a user without permission")
	}

	// A body without the flag is rejected instead of being read as "unarchive".
	performPutRequest(t, r, path, ownerToken, map[string]any{}, http.StatusUnprocessableEntity)

	// The owner sends only the flag, as the web and desktop clients do.
	performPutRequest(t, r, path, ownerToken, map[string]any{"archived": true}, http.StatusOK)
	if !isArchived() {
		t.Fatalf("expected the owner's request to archive the channel")
	}
	performPutRequest(t, r, path, ownerToken, map[string]any{"archived": true}, http.StatusBadRequest)

	// Outsiders can't unarchive it either.
	performPutRequest(t, r, path, outsiderToken, map[string]any{"archived": false}, http.StatusForbidden)
	if !isArchived() {
		t.Fatalf("channel was unarchived by a user without permission")
	}

	performPutRequest(t, r, path, ownerToken, map[string]any{"archived": false}, http.StatusOK)
	if isArchived() {
		t.Fatalf("expected the owner's request to unarchive the channel")
	}

	performPutRequest(t, r, "/api/v1/channels/not-a-uuid/archive", ownerToken, map[string]any{"archived": true}, http.StatusBadRequest)
	performPutRequest(t, r, fmt.Sprintf("/api/v1/channels/%s/archive", utility.GenerateUUID()), ownerToken, map[string]any{"archived": true}, http.StatusNotFound)
}

// The archived list is for the organisation's members, and private channels only for theirs.
func TestGetArchivedChannelsIsScopedToMembers(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	authCtrl := auth.Controller{
		Db: db, Validator: validatorRef, Logger: logger,
		ExtReq: request.ExternalRequest{Logger: logger, Test: true},
	}
	channelCtrl := channel.Controller{Db: db, Validator: validatorRef, Logger: logger}
	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}

	r := gin.Default()
	channelUrl := r.Group("/api/v1/channels", middleware.Authorize(db.Postgresql))
	{
		channelUrl.POST("", channelCtrl.CreateChannel)
		channelUrl.PUT("/:channelId/archive", channelCtrl.ArchiveChannel)
	}
	r.GET("/api/v1/organisations/:org_id/channels/archived", middleware.Authorize(db.Postgresql), channelCtrl.GetArchivedChannels)

	signUp := func(prefix string) (models.User, string) {
		data := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("%s_%v@qa.team", prefix, currUUID),
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
			FirstName:   prefix,
			LastName:    "Archive",
			Password:    "password",
			UserName:    fmt.Sprintf("%s_%v", prefix, currUUID),
		}
		tst.SignupUser(t, r, authCtrl, data, false)
		token := tst.GetLoginToken(t, r, authCtrl, models.LoginRequestModel{Email: data.Email, Password: data.Password})
		var user models.User
		db.Postgresql.Where("email = ?", data.Email).First(&user)
		return user, token
	}

	owner, ownerToken := signUp("archlistowner")
	orgId, _, _ := tst.CreateOrganisation(t, r, db, orgCtrl, models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("OrgArchiveList%s", currUUID),
		Description: "Org description",
		Email:       owner.Email,
		Type:        "test",
		Location:    "test",
		Country:     "test",
	}, ownerToken)

	createChannel := func(name string, private bool) string {
		rr := performPostRequest(t, r, "/api/v1/channels", ownerToken, models.CreateChannelsRequest{
			Name:           name,
			Username:       name,
			OrganisationID: orgId,
			IsPrivate:      private,
		}, http.StatusCreated)
		data, _ := tst.ParseResponse(rr)["data"].(map[string]any)
		id, _ := data["channels_id"].(string)
		if id == "" {
			t.Fatalf("could not create channel %s", name)
		}
		performPutRequest(t, r, fmt.Sprintf("/api/v1/channels/%s/archive", id), ownerToken, map[string]any{"archived": true}, http.StatusOK)
		return id
	}
	publicID := createChannel(fmt.Sprintf("archpub_%s", currUUID), false)
	privateID := createChannel(fmt.Sprintf("archpriv_%s", currUUID), true)

	member, memberToken := signUp("archlistmember")
	db.Postgresql.Exec("INSERT INTO user_organisations (user_id, organisation_id) VALUES (?, ?)", member.ID, orgId)
	_, outsiderToken := signUp("archlistoutsider")

	path := fmt.Sprintf("/api/v1/organisations/%s/channels/archived", orgId)
	listed := func(token string) map[string]bool {
		rr := performGetRequest(t, r, path, token, http.StatusOK)
		ids := map[string]bool{}
		items, _ := tst.ParseResponse(rr)["data"].([]any)
		for _, item := range items {
			if ch, ok := item.(map[string]any); ok {
				id, _ := ch["channels_id"].(string)
				ids[id] = true
			}
		}
		return ids
	}

	if got := listed(ownerToken); !got[publicID] || !got[privateID] {
		t.Errorf("owner should see both archived channels, got %v", got)
	}
	if got := listed(memberToken); !got[publicID] || got[privateID] {
		t.Errorf("member should see only the public archived channel, got %v", got)
	}
	performGetRequest(t, r, path, outsiderToken, http.StatusForbidden)
}

func performPutRequest(t *testing.T, r *gin.Engine, path, token string, body any, expectedCode int) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req, err := http.NewRequest(http.MethodPut, path, &buf)
	if err != nil {
		t.Fatalf("failed to create PUT request: %v", err)
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
