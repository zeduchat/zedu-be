package models

import (
	"errors"
	"net/http"
	"time"

	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
)

// ErrEditLockedByReplies is returned when a message or thread can no longer be
// edited because someone other than its author has replied to it. Editing it
// would change what those replies appear to answer.
var ErrEditLockedByReplies = errors.New("this message has replies from other people and can no longer be edited")

// EditFailureStatus is the HTTP status for a failed message edit: 409 when
// replies lock the message, 400 for everything else.
func EditFailureStatus(err error) int {
	if errors.Is(err, ErrEditLockedByReplies) {
		return http.StatusConflict
	}
	return http.StatusBadRequest
}

// RepliedToByOthers reports whether anyone other than authorID has posted in
// the thread after the given time. A zero time counts every reply in the
// thread, which is the rule for the thread's opening message. The author's own
// follow-ups never lock their message.
func RepliedToByOthers(threadID, authorID string, after time.Time) (bool, error) {
	if storage.DB == nil || storage.DB.Elastic == nil {
		return false, errors.New("elastic db connection is nil")
	}

	filter := []map[string]any{
		{"term": map[string]any{"thread_id": threadID}},
	}
	if !after.IsZero() {
		filter = append(filter, map[string]any{
			"range": map[string]any{
				"created_at": map[string]any{"gt": after.UTC().Format(time.RFC3339Nano)},
			},
		})
	}

	query := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"filter":   filter,
				"must_not": []map[string]any{{"term": map[string]any{"user_id": authorID}}},
			},
		},
		"size": 1,
	}

	var messageData any
	if err := elastic.SelectAll(storage.DB.Elastic, MessageIndexName, query, &messageData); err != nil {
		return false, err
	}

	replies, err := UnmarshalMessageResponse(messageData)
	if err != nil {
		return false, err
	}

	return len(replies) > 0, nil
}
