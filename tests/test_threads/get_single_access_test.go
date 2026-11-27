package test_threads

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

// TestGetUserSingleThreadsAccess covers #1359: reading (and editing or deleting) a
// thread through /threads/:thread_id/channels/:channel_id or the /group-dms alias
// requires access to :channel_id. The membership gate runs before any Elasticsearch
// lookup, so these cases don't need ES.
func TestGetUserSingleThreadsAccess(t *testing.T) {
	_, threadController := SetupThreadsTestRouter()
	db := threadController.Db.Postgresql
	currUUID := utility.GenerateUUID()
	password, _ := utility.HashPassword("password")

	newUser := func(tag string) models.User {
		u := models.User{
			ID:       utility.GenerateUUID(),
			Name:     "Thread " + tag,
			Email:    fmt.Sprintf("thracc_%s_%v@qa.team", tag, currUUID),
			Password: password,
		}
		if err := db.Create(&u).Error; err != nil {
			t.Fatalf("create user %s: %v", tag, err)
		}
		return u
	}
	owner := newUser("owner")
	sameOrg := newUser("sameorg")
	otherOrg := newUser("otherorg")

	org := models.Organisation{
		ID:      utility.GenerateUUID(),
		Name:    fmt.Sprintf("ThreadAccess%v", currUUID),
		Email:   fmt.Sprintf("thracc_org_%v@qa.team", currUUID),
		OwnerID: owner.ID,
	}
	channel := models.Channels{
		ID:             utility.GenerateUUID(),
		Name:           fmt.Sprintf("private-%s", currUUID[:8]),
		Description:    "private channel",
		OwnerId:        owner.ID,
		OrganisationID: org.ID,
		IsPrivate:      true,
	}
	for _, rec := range []any{
		&org,
		&channel,
		&models.UserChannels{UserID: owner.ID, ChannelsID: channel.ID, Username: owner.Name},
	} {
		if err := db.Create(rec).Error; err != nil {
			t.Fatalf("create fixture: %v", err)
		}
	}
	if err := db.Exec("INSERT INTO user_organisations (user_id, organisation_id) VALUES (?, ?), (?, ?) ON CONFLICT DO NOTHING",
		owner.ID, org.ID, sameOrg.ID, org.ID).Error; err != nil {
		t.Fatalf("add org members: %v", err)
	}

	// 1:1 DM between owner and sameOrg, and a group DM with owner and sameOrg.
	dmID := utility.GenerateUUID()
	groupID := utility.GenerateUUID()
	fixtures := []any{
		&models.DmChannels{ID: utility.GenerateUUID(), UserId: owner.ID, ParticipantId: &sameOrg.ID, ChannelId: dmID, OrgId: org.ID, ChannelType: "dm", ChatType: "user"},
		&models.DmChannels{ID: utility.GenerateUUID(), UserId: owner.ID, ChannelId: groupID, OrgId: org.ID, ChannelType: "group_dm", ChatType: "user"},
		&models.ChannelParticipant{ID: utility.GenerateUUID(), ChannelId: groupID, UserId: owner.ID, OrgId: org.ID},
		&models.ChannelParticipant{ID: utility.GenerateUUID(), ChannelId: groupID, UserId: sameOrg.ID, OrgId: org.ID},
	}
	for _, rec := range fixtures {
		if err := db.Create(rec).Error; err != nil {
			t.Fatalf("create dm fixture: %v", err)
		}
	}

	router := gin.Default()
	// Same handlers as pkg/router/thread.go and pkg/router/group_message.go.
	authMw := middleware.Authorize(threadController.Db.Postgresql)
	router.GET("/api/v1/threads/:thread_id/channels/:channel_id", authMw, threadController.GetUserSingleThreads)
	router.PUT("/api/v1/threads/:thread_id/channels/:channel_id", authMw, threadController.UpdateThreadMessage)
	router.DELETE("/api/v1/threads/:thread_id/channels/:channel_id", authMw, threadController.DeleteAThread)
	router.GET("/api/v1/group-dms/thread/:thread_id/channels/:channel_id", authMw, threadController.GetUserSingleThreads)

	authCtrl := auth.Controller{
		Db:        threadController.Db,
		Validator: threadController.Validator,
		Logger:    threadController.Logger,
		ExtReq:    threadController.ExtReq,
	}
	token := func(u models.User) string {
		return tests.GetLoginToken(t, router, authCtrl, models.LoginRequestModel{Email: u.Email, Password: "password"})
	}
	ownerToken, sameOrgToken, otherOrgToken := token(owner), token(sameOrg), token(otherOrg)

	do := func(method, path, tok string, body any) int {
		var b bytes.Buffer
		if body != nil {
			_ = json.NewEncoder(&b).Encode(body)
		}
		req, _ := http.NewRequest(method, path, &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tok)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		return rr.Code
	}

	threadID := utility.GenerateUUID()
	single := func(ch string) string { return fmt.Sprintf("/api/v1/threads/%s/channels/%s", threadID, ch) }
	group := fmt.Sprintf("/api/v1/group-dms/thread/%s/channels/%s", threadID, groupID)

	denied := []struct {
		name, method, path, tok string
		body                    any
	}{
		{"same-org non-member reads private channel thread", http.MethodGet, single(channel.ID), sameOrgToken, nil},
		{"other-org user reads private channel thread", http.MethodGet, single(channel.ID), otherOrgToken, nil},
		{"outsider reads with a random channel id", http.MethodGet, single(utility.GenerateUUID()), otherOrgToken, nil},
		{"outsider reads dm thread", http.MethodGet, single(dmID), otherOrgToken, nil},
		{"outsider reads group dm thread via alias", http.MethodGet, group, otherOrgToken, nil},
		{"non-member edits thread message", http.MethodPut, single(channel.ID), sameOrgToken, map[string]string{"content": "edited by outsider"}},
		{"non-member deletes thread", http.MethodDelete, single(channel.ID), sameOrgToken, nil},
	}
	for _, tc := range denied {
		if got := do(tc.method, tc.path, tc.tok, tc.body); got != http.StatusForbidden {
			t.Errorf("%s: got %d, want %d", tc.name, got, http.StatusForbidden)
		}
	}

	// Members get past the membership gate (the thread lookup decides the rest).
	allowed := []struct{ name, path, tok string }{
		{"channel member", single(channel.ID), ownerToken},
		{"dm participant", single(dmID), sameOrgToken},
		{"group dm participant via alias", group, sameOrgToken},
	}
	for _, tc := range allowed {
		if got := do(http.MethodGet, tc.path, tc.tok, nil); got == http.StatusForbidden {
			t.Errorf("%s: got 403, want access", tc.name)
		}
	}

	var reads int64
	db.Model(&models.UserThreadRead{}).Where("user_id IN ? AND thread_id = ?", []string{sameOrg.ID, otherOrg.ID}, threadID).Count(&reads)
	if reads != 0 {
		t.Errorf("thread marked seen for users without access: %d rows", reads)
	}
}
