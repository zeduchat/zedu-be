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

func TestRepliedToByOthers(t *testing.T) {
	logger := tst.Setup()
	db := storage.Connection()

	if db == nil || db.Elastic == nil {
		t.Skip("ElasticSearch not configured in test environment; skipping ElasticSearch integration test")
		return
	}

	threadID := uuid.Must(uuid.NewV7()).String()
	author := uuid.Must(uuid.NewV7()).String()
	other := uuid.Must(uuid.NewV7()).String()
	start := time.Now().UTC().Truncate(time.Second).Add(-time.Hour)

	var added []string
	t.Cleanup(func() {
		for _, id := range added {
			_ = elastic.DeleteDocument(db.Elastic, models.MessageIndexName, id)
		}
	})

	reply := func(userID string, at time.Time) {
		id := uuid.Must(uuid.NewV7()).String()
		doc := map[string]any{
			"id":         id,
			"thread_id":  threadID,
			"user_id":    userID,
			"message":    "reply",
			"created_at": at.Format(time.RFC3339),
		}
		if err := elastic.AddDocument(db.Elastic, models.MessageIndexName, id, doc, logger); err != nil {
			t.Fatalf("failed to add reply: %v", err)
		}
		added = append(added, id)
	}

	expect := func(name, userID string, after time.Time, want bool) {
		t.Helper()
		got, err := models.RepliedToByOthers(threadID, userID, after)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", name, err)
		}
		if got != want {
			t.Errorf("%s: RepliedToByOthers = %v, want %v", name, got, want)
		}
	}

	expect("a thread with no replies stays editable", author, time.Time{}, false)

	ownReplyAt := start.Add(1 * time.Minute)
	reply(author, ownReplyAt)
	expect("the author's own follow-up does not lock the thread", author, time.Time{}, false)

	otherReplyAt := start.Add(2 * time.Minute)
	reply(other, otherReplyAt)
	expect("another user's reply locks the thread", author, time.Time{}, true)
	expect("another user's later reply locks the author's earlier reply", author, ownReplyAt, true)

	laterOwnReplyAt := start.Add(3 * time.Minute)
	reply(author, laterOwnReplyAt)
	expect("an author reply posted after the other user's stays editable", author, laterOwnReplyAt, false)
	expect("the other user's reply is locked once the author answers it", other, otherReplyAt, true)
}

func TestRepliedToByOthersNilDB(t *testing.T) {
	origDB := storage.DB
	storage.DB = nil
	defer func() { storage.DB = origDB }()

	locked, err := models.RepliedToByOthers(uuid.Must(uuid.NewV7()).String(), uuid.Must(uuid.NewV7()).String(), time.Time{})
	if err == nil {
		t.Errorf("expected an error when storage.DB is nil, got nil")
	}
	if locked {
		t.Errorf("expected locked = false when the lookup fails")
	}
}
