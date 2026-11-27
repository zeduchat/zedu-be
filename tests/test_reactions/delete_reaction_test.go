package test_reactions

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/reactions"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

// Ziki #1590: DELETE /reactions/:reaction_id/thread/:thread_id has no channel id in the
// path, so the service must publish the update to the reaction's stored channel instead
// of an empty id (which failed every delete with 500 "empty channel_id supplied").
func TestDeleteThreadReactionPublishesToStoredChannel(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	authCtrl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	reactionCtrl := reactions.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	id := utility.GenerateUUID()
	user := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("reactdel%v@qa.team", id),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "test",
		LastName:    "user",
		Password:    "password",
		UserName:    fmt.Sprintf("reactdel_%v", id),
	}
	tst.SignupUser(t, gin.Default(), authCtrl, user, false)
	token := tst.GetLoginToken(t, gin.Default(), authCtrl, models.LoginRequestModel{Email: user.Email, Password: user.Password})
	userID := tst.GetUserIDFromToken(t, token, db)

	channelID := utility.GenerateUUID()
	thread := models.ThreadDocument{
		ID:         utility.GenerateUUID(),
		ChannelsID: channelID,
		UserId:     userID,
		Type:       "thread",
		Status:     "success",
		Content:    "react to me",
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	if err := thread.CreateThread(db, logger); err != nil {
		t.Skipf("elasticsearch unavailable, cannot index the thread: %v", err)
	}
	t.Cleanup(func() {
		db.Postgresql.Where("channels_id = ?", channelID).Delete(&models.Reaction{})
	})

	reaction := models.Reaction{
		ID:         utility.GenerateUUID(),
		ChannelsID: channelID,
		UserID:     userID,
		ThreadID:   thread.ID,
		Reaction:   "x",
		ReactionID: utility.GenerateUUIDFromSeed("x"),
		Type:       "thread",
	}
	if err := db.Postgresql.Create(&reaction).Error; err != nil {
		t.Fatalf("create reaction: %v", err)
	}

	r := gin.New()
	r.DELETE("/api/v1/reactions/:reaction_id/thread/:thread_id", middleware.Authorize(db.Postgresql), reactionCtrl.DeleteThreadReactions)

	// The thread document is indexed asynchronously; retry briefly until it is searchable.
	var rr *httptest.ResponseRecorder
	for i := 0; i < 10; i++ {
		req, _ := http.NewRequest(http.MethodDelete, "/api/v1/reactions/"+reaction.ReactionID+"/thread/"+thread.ID, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if !strings.Contains(rr.Body.String(), "thread does not exist") {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	var left int64
	db.Postgresql.Model(&models.Reaction{}).Where("id = ?", reaction.ID).Count(&left)
	if left != 0 {
		t.Fatalf("reaction row should be deleted, %d left (response %d %s)", left, rr.Code, rr.Body.String())
	}

	// Before the fix the publish to "" failed after the row was already gone and the
	// handler answered 500 (the body is masked as "internal server error").
	tst.AssertStatusCode(t, rr.Code, http.StatusOK)
}
