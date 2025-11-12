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

	existingSavedMessage, err := savedMessage.GetByUserAndThread(
		db.Postgresql,
		req.UserId,
		req.OrgId,
		req.ChannelsId,
		req.ThreadId,
	)

	if err == nil && existingSavedMessage != nil && existingSavedMessage.RiverJobID != nil {
		logger.Info("Found existing remainder job %d for user %s, cancelling it", *existingSavedMessage.RiverJobID, req.UserId)

		_, cancelErr := db.River.JobCancel(ctx, *existingSavedMessage.RiverJobID)
		if cancelErr != nil {
			logger.Error("failed to cancel existing remainder job %d: %v", *existingSavedMessage.RiverJobID, cancelErr)
		} else {
			logger.Info("Successfully cancelled existing remainder job %d", *existingSavedMessage.RiverJobID)
		}
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

	insertRes, err := jobArgs.InsertRemainderJob(ctx, db, req.RemainderAt)
	if err != nil {
		logger.Error("failed to insert remainder job: %v", err)
		return 400, err
	}

	update := map[string]any{
		"remainder_at": req.RemainderAt,
		"river_job_id": insertRes.Job.ID,
	}

	resp, err := savedMessage.UpdateSavedMessageRemainder(db.Postgresql, req, update)
	if err != nil {
		logger.Error("failed to set remainder: %v", err)
		if _, cancelErr := db.River.JobCancel(ctx, insertRes.Job.ID); cancelErr != nil {
			logger.Error("failed to rollback job %d after DB error: %v", insertRes.Job.ID, cancelErr)
		}
		return resp, err
	}

	return resp, nil
}
