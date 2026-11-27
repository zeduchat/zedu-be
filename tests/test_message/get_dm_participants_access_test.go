package test_message

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	dmCtrl "github.com/hngprojects/telex_be/pkg/controller/directMessage"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

// TestGetDmParticipantsAccess checks that only members of a DM or group DM can
// read its participants (#1232).
func TestGetDmParticipantsAccess(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	authController := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	// creator, participant, removed participant, same-org outsider, other-org outsider
	names := []string{"creator", "member", "removed", "outsider", "other"}
	users := map[string]models.User{}
	tokens := map[string]string{}
	for _, n := range names {
		signup := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("dmpart_%s_%v@qa.team", n, currUUID),
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
			FirstName:   "Dm",
			LastName:    n,
			Password:    "password",
			UserName:    fmt.Sprintf("dmpart_%s_%v", n, currUUID),
		}
		r := gin.Default()
		tst.SignupUser(t, r, authController, signup, false)
		tokens[n] = tst.GetLoginToken(t, r, authController, models.LoginRequestModel{Email: signup.Email, Password: signup.Password})
		var u models.User
		if err := db.Postgresql.Where("email = ?", signup.Email).First(&u).Error; err != nil {
			t.Fatalf("failed to load %s: %v", n, err)
		}
		users[n] = u
	}

	var orgA, orgOther models.Organisation
	if err := db.Postgresql.Where("owner_id = ?", users["creator"].ID).First(&orgA).Error; err != nil {
		t.Fatalf("failed to load creator org: %v", err)
	}
	if err := db.Postgresql.Where("owner_id = ?", users["other"].ID).First(&orgOther).Error; err != nil {
		t.Fatalf("failed to load other org: %v", err)
	}
	for _, n := range []string{"member", "removed", "outsider"} {
		if err := db.Postgresql.Exec("INSERT INTO user_organisations (user_id, organisation_id) VALUES (?, ?) ON CONFLICT DO NOTHING", users[n].ID, orgA.ID).Error; err != nil {
			t.Fatalf("failed to add %s to org: %v", n, err)
		}
	}

	groupID := utility.GenerateUUID()
	if err := db.Postgresql.Create(&models.DmChannels{
		ID:              utility.GenerateUUID(),
		UserId:          users["creator"].ID,
		ChannelId:       groupID,
		OrgId:           orgA.ID,
		ParticipantHash: utility.GenerateUUID(),
		ChatType:        "user",
		ChannelType:     "group_dm",
	}).Error; err != nil {
		t.Fatalf("failed to create group DM: %v", err)
	}
	for _, n := range []string{"creator", "member", "removed"} {
		if err := db.Postgresql.Create(&models.ChannelParticipant{
			ID:        utility.GenerateUUID(),
			ChannelId: groupID,
			UserId:    users[n].ID,
			OrgId:     orgA.ID,
		}).Error; err != nil {
			t.Fatalf("failed to add participant %s: %v", n, err)
		}
	}
	if err := db.Postgresql.Where("channel_id = ? AND user_id = ?", groupID, users["removed"].ID).Delete(&models.ChannelParticipant{}).Error; err != nil {
		t.Fatalf("failed to remove participant: %v", err)
	}

	dmID := utility.GenerateUUID()
	memberID := users["member"].ID
	if err := db.Postgresql.Create(&models.DmChannels{
		ID:            utility.GenerateUUID(),
		UserId:        users["creator"].ID,
		ChannelId:     dmID,
		OrgId:         orgA.ID,
		ParticipantId: &memberID,
		ChatType:      "user",
		ChannelType:   "dm",
	}).Error; err != nil {
		t.Fatalf("failed to create DM: %v", err)
	}

	controller := dmCtrl.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	r := gin.Default()
	r.GET("/api/v1/organisations/:org_id/dms/participants/:channel_id",
		middleware.Authorize(db.Postgresql), middleware.RequireOrgMember(db.Postgresql), controller.GetDmParticipants)

	memberEmail := users["member"].Email
	cases := []struct {
		name     string
		caller   string
		orgID    string
		channel  string
		wantCode int
	}{
		{"group DM creator", "creator", orgA.ID, groupID, http.StatusOK},
		{"group DM participant without dm_channels row", "member", orgA.ID, groupID, http.StatusOK},
		{"same-org user not in group DM", "outsider", orgA.ID, groupID, http.StatusNotFound},
		{"removed participant", "removed", orgA.ID, groupID, http.StatusNotFound},
		{"other-org user with org A in path", "other", orgA.ID, groupID, http.StatusForbidden},
		{"other-org user with own org in path", "other", orgOther.ID, groupID, http.StatusNotFound},
		{"1:1 DM creator", "creator", orgA.ID, dmID, http.StatusOK},
		{"same-org user not in 1:1 DM", "outsider", orgA.ID, dmID, http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/dms/participants/%s", tc.orgID, tc.channel), nil)
			req.Header.Set("Authorization", "Bearer "+tokens[tc.caller])
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			if rr.Code != tc.wantCode {
				t.Fatalf("expected %d, got %d: %s", tc.wantCode, rr.Code, rr.Body.String())
			}
			body := rr.Body.String()
			if tc.wantCode == http.StatusOK && tc.channel == groupID && !strings.Contains(body, memberEmail) {
				t.Errorf("expected participant list to include the member: %s", body)
			}
			if tc.wantCode != http.StatusOK && strings.Contains(body, memberEmail) {
				t.Errorf("rejected response leaks participant email: %s", body)
			}
		})
	}
}
