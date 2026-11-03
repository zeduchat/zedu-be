package test_channel

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/elastic/go-elasticsearch/v8"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/thread"
)

func TestChannelStarStatusPropagatesStorageErrors(t *testing.T) {
	storageErr := errors.New("star storage unavailable")
	for _, queryErr := range []error{storageErr, gorm.ErrRecordNotFound} {
		db, err := gorm.Open(postgres.New(postgres.Config{DSN: "sslmode=disable"}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
		if err != nil {
			t.Fatal(err)
		}
		sqlDB, err := db.DB()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { sqlDB.Close() })
		if err := db.Callback().Query().Before("gorm:query").Register("test:star_error", func(tx *gorm.DB) { tx.AddError(queryErr) }); err != nil {
			t.Fatal(err)
		}
		starred, err := models.IsChannelStarred(db, "user", "channel")
		if starred {
			t.Fatal("a missing or unavailable star must not report starred")
		}
		if errors.Is(queryErr, gorm.ErrRecordNotFound) {
			if err != nil {
				t.Fatalf("missing star should be unstarred without error: %v", err)
			}
		} else if !errors.Is(err, storageErr) {
			t.Fatalf("storage failure must reach the caller: %v", err)
		}
	}
}

func TestNoticeMergePreservesParticipantsOnProfileError(t *testing.T) {
	profileErr := errors.New("profile storage unavailable")
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "sslmode=disable"}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := db.Callback().Query().Before("gorm:query").Register("test:profile_error", func(tx *gorm.DB) { tx.AddError(profileErr) }); err != nil {
		t.Fatal(err)
	}

	for _, action := range []string{"join", "leave"} {
		t.Run(action, func(t *testing.T) {
			message := "Previous User joined this channel"
			if action == "leave" {
				message = "Previous User left this channel"
			}
			writes := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Elastic-Product", "Elasticsearch")
				w.Header().Set("Content-Type", "application/json")
				if !strings.HasSuffix(r.URL.Path, "/_search") {
					writes++
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				json.NewEncoder(w).Encode(map[string]any{"hits": map[string]any{"hits": []any{
					map[string]any{"_source": map[string]any{
						"thread_id": "existing-notice", "user_id": "previous-user",
						"message":  message,
						"mentions": []any{map[string]any{"id": "previous-user", "type": "user"}},
					}},
				}}})
			}))
			defer server.Close()
			client, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{server.URL}})
			if err != nil {
				t.Fatal(err)
			}
			store := &storage.Database{Postgresql: db, Elastic: client}
			participants := []models.UserMention{{UserID: "new-user", Username: "New User"}}
			if action == "join" {
				err = thread.SaveOrMergeJoinSystemMessage(store, nil, "channel", "general", "org", participants, nil)
			} else {
				err = thread.SaveOrMergeLeaveSystemMessage(store, nil, "channel", "general", "org", participants, nil)
			}
			if !errors.Is(err, profileErr) {
				t.Fatalf("profile failure must reach the merge caller: %v", err)
			}
			if writes != 0 {
				t.Fatalf("profile failure must not overwrite a stored notice; got %d writes", writes)
			}
		})
	}
}
