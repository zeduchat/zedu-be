package test_reactions

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
	"github.com/hngprojects/telex_be/pkg/controller/reactions"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

// Ziki #1360: only members of a channel or DM may add reactions in it.
func TestCreateReactionRequiresChannelAccess(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	authCtrl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}
	channelCtrl := channel.Controller{Db: db, Validator: validatorRef, Logger: logger}
	reactionCtrl := reactions.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	newUser := func(tag string) (string, string) {
		id := utility.GenerateUUID()
		data := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("react%s%v@qa.team", tag, id),
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
			FirstName:   "test",
			LastName:    tag,
			Password:    "password",
			UserName:    fmt.Sprintf("react_%s%v", tag, id),
		}
		tst.SignupUser(t, gin.Default(), authCtrl, data, false)
		token := tst.GetLoginToken(t, gin.Default(), authCtrl, models.LoginRequestModel{Email: data.Email, Password: data.Password})
		return token, tst.GetUserIDFromToken(t, token, db)
	}

	ownerToken, ownerID := newUser("owner")
	outsiderToken, _ := newUser("outsider")
	dmToken, dmUserID := newUser("dm")

	orgID, _, _ := tst.CreateOrganisation(t, gin.Default(), db, orgCtrl, models.CreateOrgRequestModel{
		Name: fmt.Sprintf("ReactOrg%s", utility.GenerateUUID()), Description: "reactions", Email: fmt.Sprintf("reactorg%v@qa.team", utility.GenerateUUID()),
		Type: "type1", Location: "wakanda", Country: "wakanda",
	}, ownerToken)
	channelID, _ := tst.CreateChannels(t, gin.Default(), channelCtrl, db, models.CreateChannelsRequest{
		Name: fmt.Sprintf("reactpriv%s", utility.GenerateUUID()), Username: fmt.Sprintf("reactpriv%s", utility.GenerateUUID()),
		OrganisationID: orgID, IsPrivate: true,
	}, ownerToken)
	if channelID == "" {
		t.Fatal("channel setup failed")
	}

	// A 1:1 DM between the owner and dmUser.
	dmChannelID := utility.GenerateUUID()
	dm := models.DmChannels{ID: utility.GenerateUUID(), UserId: ownerID, ChannelId: dmChannelID, OrgId: orgID, ParticipantId: &dmUserID, ParticipantHash: utility.GenerateUUID(), ChannelType: "dm"}
	if err := db.Postgresql.Create(&dm).Error; err != nil {
		t.Fatalf("create dm: %v", err)
	}

	r := gin.New()
	r.POST("/api/v1/reactions/:channel_id", middleware.Authorize(db.Postgresql), reactionCtrl.CreateReaction)

	// The thread id is never indexed, so a caller who passes the access check gets the
	// existing 400 "thread does not exist" and no reaction row is written either way.
	threadID := utility.GenerateUUID()
	react := func(channel, token string) (int, string) {
		body, _ := json.Marshal(map[string]string{"thread_id": threadID, "reaction": "x", "type": "thread"})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/reactions/"+channel, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		errMsg, _ := tst.ParseResponse(rr)["error"].(string)
		return rr.Code, errMsg
	}

	countRows := func(channel string) int64 {
		var n int64
		db.Postgresql.Model(&models.Reaction{}).Where("channels_id = ?", channel).Count(&n)
		return n
	}

	t.Run("non-member gets 404 on a real private channel", func(t *testing.T) {
		code, msg := react(channelID, outsiderToken)
		tst.AssertStatusCode(t, code, http.StatusNotFound)
		tst.AssertResponseMessage(t, msg, "channel not found")
		if n := countRows(channelID); n != 0 {
			t.Errorf("expected no reaction rows, got %d", n)
		}
	})

	t.Run("non-member gets the same 404 on a made-up channel", func(t *testing.T) {
		code, msg := react(utility.GenerateUUID(), outsiderToken)
		tst.AssertStatusCode(t, code, http.StatusNotFound)
		tst.AssertResponseMessage(t, msg, "channel not found")
	})

	t.Run("non-member gets 404 on someone else's DM", func(t *testing.T) {
		code, _ := react(dmChannelID, outsiderToken)
		tst.AssertStatusCode(t, code, http.StatusNotFound)
	})

	t.Run("channel member passes the access check", func(t *testing.T) {
		code, msg := react(channelID, ownerToken)
		tst.AssertStatusCode(t, code, http.StatusBadRequest)
		tst.AssertResponseMessage(t, msg, "failed to save reaction , error: thread does not exist")
	})

	t.Run("DM participant passes the access check", func(t *testing.T) {
		code, msg := react(dmChannelID, dmToken)
		tst.AssertStatusCode(t, code, http.StatusBadRequest)
		tst.AssertResponseMessage(t, msg, "failed to save reaction , error: thread does not exist")
	})
}
