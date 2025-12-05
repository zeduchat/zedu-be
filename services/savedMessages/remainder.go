package savedMessages

import (
	"context"
	"errors"
	"fmt"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
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
		req.Type,
		req.MessageId,
	)
	if err != nil {
		logger.Error("failed to fetch saved message for setting remainder: %v", err)
		return 400, errors.New("failed to fetch saved message: " + err.Error())
	}

	if existingSavedMessage.Completed {
		logger.Info("Saved message for user %s, thread %s is already completed; cannot set remainder", req.UserId, req.ThreadId)
		return 200, nil
	}

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

func MarkCompleteSavedMessage(req models.MarkCompleteSavedMessageRequest, db *storage.Database, logger *utility.Logger) (int, error) {
	var (
		savedMessage models.SavedMessage
		ctx          = context.Background()
	)

	update := map[string]any{
		"completed": req.Completed,
	}

	resp, err := savedMessage.UpdateSavedMessageCompletion(db.Postgresql, logger, req, update)
	if err != nil {
		logger.Error("failed to mark saved message as complete: %v", err)
		return resp, err
	}

	if *req.Completed && savedMessage.RiverJobID != nil && savedMessage.RemainderAt != nil {
		_, cancelErr := db.River.JobCancel(ctx, *savedMessage.RiverJobID)
		if cancelErr != nil {
			logger.Error("failed to cancel remainder job %d for saved message %s: %v", *savedMessage.RiverJobID, req.SavedMessageID, cancelErr)
			return resp, cancelErr
		}
	}

	//publish notification to centrifuge
	if !*req.Completed {
		notification := models.Notification[models.SavedMessageCompletionEvent]
		if savedMessage.Type == "thread" {
			notification.SectionType = models.ThreadSection
		}
		notification.SectionType = models.ReplySection

		err = centrifuge.PublishChannel(logger, fmt.Sprintf("%s/%s", req.OrgId, req.UserId), notification)
		if err != nil {
			logger.Error("Error Publishing saved message completion event to with destination id: %s error: %v", req.SavedMessageID, err.Error())
			return resp, errors.New("failed to publish data: " + err.Error())
		}

		logger.Info("Successfully Published Saved Message Completion Notification to with destination id: %s", req.SavedMessageID)
	}

	return resp, nil
}

func ArchiveSavedMessage(req models.ArchiveSavedMessageRequest, db *storage.Database, logger *utility.Logger) (int, error) {
	var (
		savedMessage models.SavedMessage
		notification models.Content
	)

	update := map[string]any{
		"archived": req.Archived,
	}

	resp, err := savedMessage.UpdateSavedMessageArchiveStatus(db.Postgresql, req, update)
	if err != nil {
		logger.Error("failed to update archive status of saved message: %v", err)
		return resp, err
	}

	//publish notification to centrifuge
	if *req.Archived {
		notification = models.Notification[models.ArchiveSavedMessage]
	} else {
		notification = models.Notification[models.UnArchiveSavedMessage]
	}
	if savedMessage.Type == "thread" {
		notification.SectionType = models.ThreadSection
	}
	notification.SectionType = models.ReplySection

	err = centrifuge.PublishChannel(logger, fmt.Sprintf("%s/%s", req.OrgId, req.UserId), notification)
	if err != nil {
		logger.Error("Error Publishing saved message completion event to with destination id: %s error: %v", req.SavedMessageID, err.Error())
		return resp, errors.New("failed to publish data: " + err.Error())
	}

	return resp, nil
}

func GetCompletedSavedMessages(db *storage.Database, logger *utility.Logger, ids models.IDS, pagination postgresql.Pagination) ([]models.SavedMessagesResp, postgresql.PaginationResponse, error) {
	var savedMessage models.SavedMessage

	messages, paginationResponse, err := savedMessage.GetCompletedSavedMessages(db.Postgresql, ids.OrganisationID, ids.UserID, pagination)
	if err != nil {
		logger.Error("An error occurred while fetching completed saved messages from Postgres: %v", err)
		return nil, postgresql.PaginationResponse{}, err
	}

	return messages, paginationResponse, nil
}

func GetArchivedSavedMessages(db *storage.Database, logger *utility.Logger, ids models.IDS, pagination postgresql.Pagination) ([]models.SavedMessagesResp, postgresql.PaginationResponse, error) {
	var savedMessage models.SavedMessage

	messages, paginationResponse, err := savedMessage.GetArchivedSavedMessages(db.Postgresql, ids.OrganisationID, ids.UserID, pagination)
	if err != nil {
		logger.Error("An error occurred while fetching completed saved messages from Postgres: %v", err)
		return nil, postgresql.PaginationResponse{}, err
	}

	return messages, paginationResponse, nil
}
