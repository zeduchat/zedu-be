package test_threads

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestEditThreadAfterReplyFromAnotherUser(t *testing.T) {
	_, threadController := SetupThreadsTestRouter()
	db := threadController.Db

	if db == nil || db.Elastic == nil {
		t.Skip("ElasticSearch not configured in test environment; skipping ElasticSearch integration test")
		return
	}

	router := gin.Default()
	router.PUT("/api/v1/threads/:thread_id/channels/:channel_id", middleware.Authorize(db.Postgresql),
		threadController.UpdateThreadMessage)
	authController := auth.Controller{
		Db:        db,
		Validator: threadController.Validator,
		Logger:    threadController.Logger,
		ExtReq:    threadController.ExtReq,
	}

	currUUID := utility.GenerateUUID()
	password, _ := utility.HashPassword("password")
	author := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "Thread Author",
		Email:    fmt.Sprintf("author%v@qa.team", currUUID),
		Password: password,
		Role:     int(models.RoleIdentity.SuperAdmin),
	}
	org := models.Organisation{
		ID:      utility.GenerateUUID(),
		Name:    fmt.Sprintf("Org edit lock %v", currUUID),
		Email:   fmt.Sprintf("orgeditlock%v@qa.team", currUUID),
		OwnerID: author.ID,
	}
	channel := models.Channels{
		ID:             utility.GenerateUUID(),
		Name:           fmt.Sprintf("edit-lock-%v", currUUID),
		Description:    "Edit lock channel",
		OwnerId:        author.ID,
		OrganisationID: org.ID,
	}
	db.Postgresql.Create(&author)
	db.Postgresql.Create(&org)
	db.Postgresql.Create(&channel)

	const original = "Can I submit the assignment on Friday?"
	thread := models.ThreadDocument{
		ID:             utility.GenerateUUID(),
		ChannelsID:     channel.ID,
		OrganisationID: org.ID,
		Username:       author.Name,
		Content:        original,
		UserId:         author.ID,
		Type:           "thread",
		CreatedAt:      time.Now().UTC().Add(-time.Hour),
	}
	if err := thread.CreateThread(db, threadController.Logger); err != nil {
		t.Fatalf("failed to create thread: %v", err)
	}

	replyID := utility.GenerateUUID()
	reply := map[string]any{
		"id":          replyID,
		"thread_id":   thread.ID,
		"channels_id": channel.ID,
		"user_id":     utility.GenerateUUID(),
		"message":     "Yes, that's fine.",
		"created_at":  time.Now().UTC().Format(time.RFC3339),
	}
	if err := elastic.AddDocument(db.Elastic, models.MessageIndexName, replyID, reply, threadController.Logger); err != nil {
		t.Fatalf("failed to add reply: %v", err)
	}
	t.Cleanup(func() {
		_ = elastic.DeleteDocument(db.Elastic, models.MessageIndexName, replyID)
		_ = elastic.DeleteDocument(db.Elastic, models.ThreadIndexName, thread.ID)
	})

	token := tests.GetLoginToken(t, router, authController, models.LoginRequestModel{
		Email:    author.Email,
		Password: "password",
	})

	body, _ := json.Marshal(map[string]any{"content": "Can I skip the assignment entirely?"})
	req, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/threads/%s/channels/%s", thread.ID, channel.ID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	tests.AssertStatusCode(t, resp.Code, http.StatusConflict)
	response := tests.ParseResponse(resp)
	tests.AssertResponseMessage(t, response["message"].(string), models.ErrEditLockedByReplies.Error())

	var stored models.ThreadDocument
	if err := stored.GetThreadById(thread.ID); err != nil {
		t.Fatalf("failed to read thread back: %v", err)
	}
	if stored.Content != original {
		t.Errorf("thread content changed to %q; it must stay %q once another user has replied", stored.Content, original)
	}
	if stored.Edited {
		t.Errorf("thread was marked edited although the edit was refused")
	}
}
