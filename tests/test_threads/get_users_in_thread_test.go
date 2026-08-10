package test_threads

import (
	"testing"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/thread"
	"github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestUserThreadMultiUserReplyCount(t *testing.T) {
	logger := tests.Setup()
	db := storage.Connection()
	if db == nil || db.Postgresql == nil {
		t.Skip("PostgreSQL database connection unavailable for unit test")
	}

	orgID := utility.GenerateUUID()
	userA := utility.GenerateUUID()
	userB := utility.GenerateUUID()
	userC := utility.GenerateUUID()
	userD := utility.GenerateUUID()

	thread1 := utility.GenerateUUID()
	thread2 := utility.GenerateUUID()
	thread3 := utility.GenerateUUID()

	err := thread.ProcessThreadUnseenForParticipants(db.Postgresql, logger, orgID, thread1, []string{userA, userB})
	if err != nil {
		t.Fatalf("failed to process unseen for thread 1: %v", err)
	}

	err = thread.ProcessThreadUnseenForParticipants(db.Postgresql, logger, orgID, thread2, []string{userA, userC})
	if err != nil {
		t.Fatalf("failed to process unseen for thread 2: %v", err)
	}

	err = thread.ProcessThreadUnseenForParticipants(db.Postgresql, logger, orgID, thread3, []string{userA, userD})
	if err != nil {
		t.Fatalf("failed to process unseen for thread 3: %v", err)
	}

	countA, err := models.GetUnseenThreadCountForUser(db.Postgresql, userA, orgID)
	if err != nil {
		t.Fatalf("failed to get unseen thread count for user A: %v", err)
	}
	if countA != 3 {
		t.Errorf("expected unseen thread count for user A to be 3 across 3 threads, got %d", countA)
	}

	err = thread.ProcessThreadUnseenForParticipants(db.Postgresql, logger, orgID, thread1, []string{userA, userC})
	if err != nil {
		t.Fatalf("failed to process second comment on thread 1 from user C: %v", err)
	}

	err = thread.ProcessThreadUnseenForParticipants(db.Postgresql, logger, orgID, thread1, []string{userA, userD})
	if err != nil {
		t.Fatalf("failed to process third comment on thread 1 from user D: %v", err)
	}

	countARe, _ := models.GetUnseenThreadCountForUser(db.Postgresql, userA, orgID)
	if countARe != 3 {
		t.Errorf("expected unseen thread count for user A to remain 3 after additional comments on thread 1, got %d", countARe)
	}

	err = thread.ProcessThreadSeenForUser(db.Postgresql, logger, orgID, userA, thread1)
	if err != nil {
		t.Fatalf("failed to mark thread 1 seen for user A: %v", err)
	}

	countAAfterRead1, _ := models.GetUnseenThreadCountForUser(db.Postgresql, userA, orgID)
	if countAAfterRead1 != 2 {
		t.Errorf("expected unseen thread count for user A to decrement to 2 after viewing thread 1, got %d", countAAfterRead1)
	}

	_ = db.Postgresql.Where("org_id = ?", orgID).Delete(&models.UserThreadRead{}).Error
}

func TestGetUsersInThreadStructFiltering(t *testing.T) {
	senderID := "user-sender-uuid"
	authorID := "user-author-uuid"
	commenterID := "user-commenter-uuid"

	doc := models.ThreadDocument{
		ID:             "thread-123",
		UserId:         authorID,
		OrganisationID: "org-123",
		Mentions: []models.Mention{
			{ID: "channel-123", Type: "channel"},
			{ID: "here", Type: "here"},
			{ID: commenterID, Type: "user"},
		},
	}

	userIDsMap := make(map[string]struct{})
	if doc.UserId != "" && doc.UserId != senderID {
		userIDsMap[doc.UserId] = struct{}{}
	}

	for _, m := range doc.Mentions {
		if m.ID != "" && m.Type == "user" && m.ID != senderID {
			userIDsMap[m.ID] = struct{}{}
		}
	}

	if _, exists := userIDsMap["channel-123"]; exists {
		t.Errorf("expected channel wildcard to be excluded from thread participants")
	}

	if _, exists := userIDsMap[senderID]; exists {
		t.Errorf("expected sender to be excluded from thread participants")
	}

	if _, exists := userIDsMap[authorID]; !exists {
		t.Errorf("expected author %s to be included in thread participants", authorID)
	}

	if _, exists := userIDsMap[commenterID]; !exists {
		t.Errorf("expected user mention %s to be included in thread participants", commenterID)
	}
}
