package models

import (
	"context"
	"fmt"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/hngprojects/telex_be/pkg/repository/storage"
)

type ClearUserStatusJobArgs struct {
	UserID string `json:"user_id"`
	OrgID  string `json:"org_id"`
}

func (ClearUserStatusJobArgs) Kind() string { return "clear_user_status" }

func (w *ClearUserStatusJobArgs) InsertClearStatusJob(ctx context.Context, db *storage.Database, scheduledAt time.Time) (*rivertype.JobInsertResult, error) {
	client := storage.DB.River
	if client == nil {
		return nil, nil
	}
	insertRes, err := client.Insert(ctx, w, &river.InsertOpts{
		MaxAttempts: 3,
		ScheduledAt: scheduledAt,
		Priority:    3,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to insert clear status job: %w", err)
	}
	return insertRes, nil
}
