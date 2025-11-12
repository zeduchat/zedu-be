package savedMessages

import (
	"context"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func SetRemainder(req models.SetRemainderRequest, db *storage.Database, logger *utility.Logger) (int, error) {
	var (
		savedMessage models.SavedMessage
		jobArgs      models.SavedMessagesRemainderJobArgs
		ctx          = context.Background()
	)

	update := map[string]any{
		"remainder_at": req.RemainderAt,
	}

	resp, err := savedMessage.UpdateSavedMessageRemainder(db.Postgresql, req, update)
	if err != nil {
		logger.Error("failed to set remainder: %v", err)
		return resp, err
	}

	//register riverqueue job to handle remainder notification
	if req.MessageId != nil {
		jobArgs.MessageID = req.MessageId
	}
	jobArgs.ChannelsID = req.ChannelsId
	jobArgs.OrgId = req.OrgId
	jobArgs.UserID = req.UserId
	jobArgs.Type = req.Type
	jobArgs.ThreadID = req.ThreadId

	err = jobArgs.InsertRemainderJob(ctx, db, req.RemainderAt)
	if err != nil {
		logger.Error("failed to insert remainder job: %v", err)
		return resp, err
	}

	return resp, nil
}
