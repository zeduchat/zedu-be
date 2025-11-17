package models

import (
	"context"
	"fmt"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/hngprojects/telex_be/pkg/repository/storage"
)

type SavedMessagesRemainderJobArgs struct {
	ChannelsID string  `json:"channels_id"`
	OrgId      string  `json:"org_id"`
	UserID     string  `json:"user_id"`
	Type       string  `json:"type"`
	MessageID  *string `json:"message_id,omitempty"`
	ThreadID   string  `json:"thread_id"`
}

func (SavedMessagesRemainderJobArgs) Kind() string { return "remainder_job" }

func (w *SavedMessagesRemainderJobArgs) InsertRemainderJob(ctx context.Context, db *storage.Database, remainderTime time.Time) (*rivertype.JobInsertResult, error) {
	client := storage.DB.River
	insertRes, err := client.Insert(ctx, w, &river.InsertOpts{
		MaxAttempts: 5,
		ScheduledAt: remainderTime,
		Priority:    2,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to insert remainder job: %w", err)
	}

	return insertRes, nil
}
