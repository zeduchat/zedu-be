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
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/controller/pinnedmessages"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

// TestPinnedMessagesAccess covers #1189: the pinned-message routes must only let
// channel members and DM / group DM participants read, pin and unpin.
func TestPinnedMessagesAccess(t *testing.T) {
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
			Email:       fmt.Sprintf("pinacc_%s_%v@qa.team", n, currUUID),
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
			FirstName:   "Pin",
			LastName:    "User" + n,
			Password:    "password",
			UserName:    fmt.Sprintf("pinacc_%s_%v", n, currUUID),
		}
		r := gin.Default()
		tst.SignupUser(t, r, authCtrl, ud, false)
		tokens[i] = tst.GetLoginToken(t, r, authCtrl, models.LoginRequestModel{Email: ud.Email, Password: ud.Password})
		userIDs[i] = tst.GetUserIDFromToken(t, tokens[i], db)
		emails[i] = ud.Email
	}
	userA, userB, userC := userIDs[0], userIDs[1], userIDs[2]

	// Regular channel owned by A.
	r := gin.Default()
	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}
	channelCtrl := channel.Controller{Db: db, Validator: validatorRef, Logger: logger}
	orgID, _, _ := tst.CreateOrganisation(t, r, db, orgCtrl, models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("PinAccess%s", currUUID),
		Description: "pinned access test org",
		Email:       emails[0],
		Type:        "type1",
		Location:    "wakanda",
		Country:     "wakanda",
	}, tokens[0])
	channelID, _ := tst.CreateChannels(t, r, channelCtrl, db, models.CreateChannelsRequest{
		Name:           fmt.Sprintf("pin-%s", currUUID[:8]),
		Username:       "pinaccess",
		OrganisationID: orgID,
		Description:    "pinned access",
	}, tokens[0])
	if channelID == "" {
		t.Fatal("failed to create channel")
	}

	// 1:1 DM between A and B.
	dmID := utility.GenerateUUID()
	if err := db.Postgresql.Create(&models.DmChannels{
		ID: utility.GenerateUUID(), UserId: userA, ParticipantId: &userB, ChannelId: dmID,
		OrgId: orgID, ChannelType: "dm", ChatType: "user",
	}).Error; err != nil {
		t.Fatalf("failed to create dm: %v", err)
	}

	// Group DM created by A, where only B is still a participant (A has left).
	groupID := utility.GenerateUUID()
	if err := db.Postgresql.Create(&models.DmChannels{
		ID: utility.GenerateUUID(), UserId: userA, ChannelId: groupID,
		OrgId: orgID, ChannelType: "group_dm", ChatType: "user",
	}).Error; err != nil {
		t.Fatalf("failed to create group dm: %v", err)
	}
	if err := db.Postgresql.Create(&models.ChannelParticipant{
		ID: utility.GenerateUUID(), ChannelId: groupID, UserId: userB, OrgId: orgID,
	}).Error; err != nil {
		t.Fatalf("failed to add group dm participant: %v", err)
	}

	t.Run("UserCanAccessChannel", func(t *testing.T) {
		cases := []struct {
			name      string
			channelID string
			userID    string
			want      bool
		}{
			{"channel member", channelID, userA, true},
			{"channel non-member", channelID, userC, false},
			{"dm owner", dmID, userA, true},
			{"dm participant", dmID, userB, true},
			{"dm outsider", dmID, userC, false},
			{"group dm participant", groupID, userB, true},
			{"group dm creator who left", groupID, userA, false},
			{"group dm outsider", groupID, userC, false},
			{"invalid channel id", "not-a-uuid", userA, false},
			{"invalid user id", dmID, "not-a-uuid", false},
		}
		for _, tc := range cases {
			got, err := models.UserCanAccessChannel(db.Postgresql, tc.channelID, tc.userID)
			if err != nil {
				t.Fatalf("%s: unexpected error: %v", tc.name, err)
			}
			if got != tc.want {
				t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
			}
		}
	})

	// Pinned thread and reply in the DM, and a pinned thread in the channel.
	dmThreadID := utility.GenerateUUID()
	dmReplyID := utility.GenerateUUID()
	chThreadID := utility.GenerateUUID()
	pins := []models.PinnedMessage{
		{ID: utility.GenerateUUID(), ChannelsID: dmID, UserID: userA, ThreadID: dmThreadID, Type: "thread", Pinned: true},
		{ID: utility.GenerateUUID(), ChannelsID: dmID, UserID: userA, ThreadID: dmThreadID, Type: "reply", MessageID: &dmReplyID, Pinned: true},
		{ID: utility.GenerateUUID(), ChannelsID: channelID, UserID: userA, ThreadID: chThreadID, Type: "thread", Pinned: true},
	}
	for i := range pins {
		if err := db.Postgresql.Create(&pins[i]).Error; err != nil {
			t.Fatalf("failed to create pin: %v", err)
		}
	}

	pinCtrl := pinnedmessages.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	pr := gin.Default()
	g := pr.Group("/api/v1/channels", middleware.Authorize(db.Postgresql))
	g.POST("/pin/:channel_id/thread", pinCtrl.PinThreadMessage)
	g.POST("/pin/:channel_id/message", pinCtrl.PinReplyMessage)
	g.GET("/pin/:channel_id", pinCtrl.GetAllPinnedMessages)
	g.DELETE("/pin/:channel_id/thread/:threadId", pinCtrl.UnPinThreadMessage)
	g.DELETE("/pin/:channel_id/message/:messageId", pinCtrl.UnPinReplyMessage)

	do := func(method, path, token string, body any) int {
		var b bytes.Buffer
		if body != nil {
			_ = json.NewEncoder(&b).Encode(body)
		}
		req, _ := http.NewRequest(method, path, &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		pr.ServeHTTP(rr, req)
		return rr.Code
	}

	pinCount := func(channel string) int64 {
		var n int64
		db.Postgresql.Model(&models.PinnedMessage{}).Where("channels_id = ?", channel).Count(&n)
		return n
	}

	tokenC := tokens[2]
	outsider := []struct {
		name, method, path string
		body               any
		want               int
	}{
		{"list dm pins", http.MethodGet, "/api/v1/channels/pin/" + dmID, nil, http.StatusNotFound},
		{"list group dm pins", http.MethodGet, "/api/v1/channels/pin/" + groupID, nil, http.StatusNotFound},
		{"pin thread in dm", http.MethodPost, "/api/v1/channels/pin/" + dmID + "/thread", map[string]string{"thread_id": dmThreadID}, http.StatusForbidden},
		{"pin reply in dm", http.MethodPost, "/api/v1/channels/pin/" + dmID + "/message", map[string]string{"thread_id": dmThreadID, "message_id": dmReplyID}, http.StatusForbidden},
		{"unpin thread in dm", http.MethodDelete, "/api/v1/channels/pin/" + dmID + "/thread/" + dmThreadID, nil, http.StatusNotFound},
		{"unpin reply in dm", http.MethodDelete, "/api/v1/channels/pin/" + dmID + "/message/" + dmReplyID, nil, http.StatusNotFound},
		{"unpin thread in channel", http.MethodDelete, "/api/v1/channels/pin/" + channelID + "/thread/" + chThreadID, nil, http.StatusNotFound},
	}
	for _, tc := range outsider {
		if got := do(tc.method, tc.path, tokenC, tc.body); got != tc.want {
			t.Errorf("outsider %s: got %d, want %d", tc.name, got, tc.want)
		}
	}

	if n := pinCount(dmID); n != 2 {
		t.Errorf("dm pins changed by outsider: got %d, want 2", n)
	}
	if n := pinCount(channelID); n != 1 {
		t.Errorf("channel pins changed by outsider: got %d, want 1", n)
	}
}
