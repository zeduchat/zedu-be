package test_message

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
	"github.com/hngprojects/telex_be/pkg/controller/forwardedMessage"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

// TestForwardMessageAccess covers #1356: forwarding must only read from a channel or DM
// the caller belongs to and only post into a destination the caller belongs to. The
// access checks run before the Elasticsearch message lookup, so they are testable here
// without a working ES; the success path and the thread/channel mismatch need ES.
func TestForwardMessageAccess(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	authCtrl := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	names := []string{"a", "b", "c"}
	tokens := make([]string, len(names))
	userIDs := make([]string, len(names))
	emails := make([]string, len(names))
	for i, n := range names {
		ud := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("fwdacc_%s_%v@qa.team", n, currUUID),
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
			FirstName:   "Fwd",
			LastName:    "User" + n,
			Password:    "password",
			UserName:    fmt.Sprintf("fwdacc_%s_%v", n, currUUID),
		}
		r := gin.Default()
		tst.SignupUser(t, r, authCtrl, ud, false)
		tokens[i] = tst.GetLoginToken(t, r, authCtrl, models.LoginRequestModel{Email: ud.Email, Password: ud.Password})
		userIDs[i] = tst.GetUserIDFromToken(t, tokens[i], db)
		emails[i] = ud.Email
	}
	userA, userB, userC := userIDs[0], userIDs[1], userIDs[2]

	// Channel P owned by A; B and C are not members.
	r := gin.Default()
	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}
	channelCtrl := channel.Controller{Db: db, Validator: validatorRef, Logger: logger}
	orgID, _, _ := tst.CreateOrganisation(t, r, db, orgCtrl, models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("FwdAccess%s", currUUID),
		Description: "forward access test org",
		Email:       emails[0],
		Type:        "type1",
		Location:    "wakanda",
		Country:     "wakanda",
	}, tokens[0])
	channelP, _ := tst.CreateChannels(t, r, channelCtrl, db, models.CreateChannelsRequest{
		Name:           fmt.Sprintf("fwd-%s", currUUID[:8]),
		Username:       "fwdaccess",
		OrganisationID: orgID,
		Description:    "forward access",
	}, tokens[0])
	if channelP == "" {
		t.Fatal("failed to create channel")
	}

	// 1:1 DM between C and B (A is not in it), and one between A and B.
	dmCB := utility.GenerateUUID()
	dmAB := utility.GenerateUUID()
	for _, dm := range []models.DmChannels{
		{ID: utility.GenerateUUID(), UserId: userC, ParticipantId: &userB, ChannelId: dmCB, OrgId: orgID, ChannelType: "dm", ChatType: "user"},
		{ID: utility.GenerateUUID(), UserId: userA, ParticipantId: &userB, ChannelId: dmAB, OrgId: orgID, ChannelType: "dm", ChatType: "user"},
	} {
		dm := dm
		if err := db.Postgresql.Create(&dm).Error; err != nil {
			t.Fatalf("failed to create dm: %v", err)
		}
	}

	fwdCtrl := forwardedMessage.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	fr := gin.Default()
	g := fr.Group("/api/v1/channels", middleware.Authorize(db.Postgresql))
	g.POST("/:channelId/thread/forward", fwdCtrl.ForwardThreadMessage)
	g.POST("/:channelId/message/forward", fwdCtrl.ForwardReplyMessage)

	do := func(path, token string, body any) int {
		var b bytes.Buffer
		_ = json.NewEncoder(&b).Encode(body)
		req, _ := http.NewRequest(http.MethodPost, path, &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		fr.ServeHTTP(rr, req)
		return rr.Code
	}

	threadID := utility.GenerateUUID()
	replyID := utility.GenerateUUID()
	missing := utility.GenerateUUID()
	tokenA, tokenC := tokens[0], tokens[2]

	cases := []struct {
		name, token, source, dest string
	}{
		{"outsider reads from channel P into own DM", tokenC, channelP, dmCB},
		{"outsider reads from someone else's DM", tokenC, dmAB, dmCB},
		{"member posts into a DM they are not in", tokenA, channelP, dmCB},
		{"outsider posts into channel P", tokenC, dmCB, channelP},
		{"source channel does not exist", tokenA, missing, dmAB},
		{"destination does not exist", tokenA, channelP, missing},
	}
	for _, tc := range cases {
		path := "/api/v1/channels/" + tc.source
		if got := do(path+"/thread/forward", tc.token, map[string]string{"thread_id": threadID, "forwarded_to_channel_id": tc.dest}); got != http.StatusNotFound {
			t.Errorf("thread forward, %s: got %d, want 404", tc.name, got)
		}
		if got := do(path+"/message/forward", tc.token, map[string]string{"thread_id": threadID, "message_id": replyID, "forwarded_to_channel_id": tc.dest}); got != http.StatusNotFound {
			t.Errorf("reply forward, %s: got %d, want 404", tc.name, got)
		}
	}

	// A member of both sides passes the access checks and reaches the message lookup,
	// which fails for an unknown thread id (400 from the ES lookup), not 404.
	if got := do("/api/v1/channels/"+channelP+"/thread/forward", tokenA, map[string]string{"thread_id": threadID, "forwarded_to_channel_id": dmAB}); got == http.StatusNotFound {
		t.Errorf("member forward: got 404, want the access checks to pass")
	}
}
