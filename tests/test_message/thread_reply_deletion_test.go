package test_message

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	tst "github.com/hngprojects/telex_be/tests"
)

func TestThreadReplyDeletionCountAndMessagesArray(t *testing.T) {
	logger := tst.Setup()
	db := storage.Connection()

	if db == nil || db.Elastic == nil {
		t.Skip("ElasticSearch not configured in test environment; skipping ElasticSearch integration test")
		return
	}

	threadID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7()).String()
	orgID := uuid.Must(uuid.NewV7()).String()

	msg1ID := uuid.Must(uuid.NewV7()).String()
	msg2ID := uuid.Must(uuid.NewV7()).String()

	msg1Doc := map[string]any{
		"id":         msg1ID,
		"thread_id":  threadID.String(),
		"user_id":    userID,
		"org_id":     orgID,
		"message":    "First reply message by user",
		"created_at": time.Now().Add(-10 * time.Minute).Format(time.RFC3339),
	}

	msg2Doc := map[string]any{
		"id":         msg2ID,
		"thread_id":  threadID.String(),
		"user_id":    userID,
		"org_id":     orgID,
		"message":    "Second reply message by user",
		"created_at": time.Now().Format(time.RFC3339),
	}

	threadDoc := map[string]any{
		"id":            threadID.String(),
		"thread_id":     threadID.String(),
		"user_id":       userID,
		"org_id":        orgID,
		"message_count": 2,
		"messages": []map[string]any{
			msg2Doc,
		},
	}

	_ = elastic.AddDocument(db.Elastic, models.ThreadIndexName, threadID.String(), threadDoc, logger)
	_ = elastic.AddDocument(db.Elastic, models.MessageIndexName, msg1ID, msg1Doc, logger)
	_ = elastic.AddDocument(db.Elastic, models.MessageIndexName, msg2ID, msg2Doc, logger)

	time.Sleep(1 * time.Second)

	msg2ToDelete := models.MessageDocument{
		ID:        msg2ID,
		ThreadID:  threadID,
		UserID:    userID,
		CreatedAt: time.Now(),
	}

	_, err := msg2ToDelete.DeleteMessage(db.Postgresql, logger)
	if err != nil {
		t.Logf("DeleteMessage msg2 returned: %v", err)
	}

	time.Sleep(1 * time.Second)

	var threadAfterFirstDelete models.ThreadDocument
	if err := threadAfterFirstDelete.GetThreadById(threadID.String()); err == nil {
		if threadAfterFirstDelete.MessageCount != 1 {
			t.Errorf("expected message_count = 1 after deleting 1 of 2 replies, got %d", threadAfterFirstDelete.MessageCount)
		}
		if len(threadAfterFirstDelete.Messages) != 1 {
			t.Errorf("expected messages array to still have 1 entry (msg1), got %d", len(threadAfterFirstDelete.Messages))
		} else if threadAfterFirstDelete.Messages[0].ID != msg1ID {
			t.Errorf("expected remaining message in messages array to be msg1 (%s), got %s", msg1ID, threadAfterFirstDelete.Messages[0].ID)
		}
	}

	msg1ToDelete := models.MessageDocument{
		ID:        msg1ID,
		ThreadID:  threadID,
		UserID:    userID,
		CreatedAt: time.Now(),
	}

	_, err = msg1ToDelete.DeleteMessage(db.Postgresql, logger)
	if err != nil {
		t.Logf("DeleteMessage msg1 returned: %v", err)
	}

	time.Sleep(1 * time.Second)

	var threadAfterSecondDelete models.ThreadDocument
	if err := threadAfterSecondDelete.GetThreadById(threadID.String()); err == nil {
		if threadAfterSecondDelete.MessageCount != 0 {
			t.Errorf("expected message_count = 0 after deleting all replies, got %d", threadAfterSecondDelete.MessageCount)
		}
		if len(threadAfterSecondDelete.Messages) != 0 {
			t.Errorf("expected messages array to be empty after deleting all replies, got %d", len(threadAfterSecondDelete.Messages))
		}
	}

	// Cleanup documents from ES
	_ = elastic.DeleteDocument(db.Elastic, models.MessageIndexName, msg1ID)
	_ = elastic.DeleteDocument(db.Elastic, models.MessageIndexName, msg2ID)
	_ = elastic.DeleteDocument(db.Elastic, models.ThreadIndexName, threadID.String())
}

func TestGetLatestMessageHelpersNilDB(t *testing.T) {
	origDB := storage.DB
	storage.DB = nil
	defer func() { storage.DB = origDB }()

	threadID := uuid.Must(uuid.NewV7()).String()
	userID := uuid.Must(uuid.NewV7()).String()

	msg, err := models.GetLatestMessageByThreadAndUser(threadID, userID)
	if err == nil {
		t.Errorf("expected error when storage.DB is nil, got nil error")
	}
	if msg != nil {
		t.Errorf("expected nil message when storage.DB is nil, got %v", msg)
	}

	msg2, err2 := models.GetLatestMessageByThread(threadID)
	if err2 == nil {
		t.Errorf("expected error when storage.DB is nil, got nil error")
	}
	if msg2 != nil {
		t.Errorf("expected nil message when storage.DB is nil, got %v", msg2)
	}
}
