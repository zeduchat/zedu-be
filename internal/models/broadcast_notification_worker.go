package models

import (
	"context"
	"fmt"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/hngprojects/telex_be/pkg/repository/storage"
)

type BroadcastNotificationJobArgs struct {
	BroadcastLogID string `json:"broadcast_log_id"`
	AdminID        string `json:"admin_id"`
	AdminEmail     string `json:"admin_email"`
	Title          string `json:"title"`
	Message        string `json:"message"`
	AvatarUrl      string `json:"avatar_url"`
	IPAddress      string `json:"ip_address"`
	UserAgent      string `json:"user_agent"`
}

func (BroadcastNotificationJobArgs) Kind() string { return "broadcast_notification_job" }

func (b *BroadcastNotificationJobArgs) InsertBroadcastNotificationJob(ctx context.Context, db *storage.Database) (*rivertype.JobInsertResult, error) {
	client := storage.DB.River
	if client == nil {
		return nil, nil
	}
	insertRes, err := client.Insert(ctx, b, &river.InsertOpts{
		MaxAttempts: 3,
		Priority:    1,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to insert broadcast notification job: %w", err)
	}

	return insertRes, nil
}
