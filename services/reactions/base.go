package reactions

import (
	"errors"
	"fmt"
	"net/http"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

func CreateReaction(req models.ReactionRequest, db *storage.Database, logger *utility.Logger) (int, error) {

	react := models.Reaction{
		ID:         utility.GenerateUUID(),
		ChannelsID: req.ChannelsID,
		UserID:     req.UserID,
		ThreadID:   req.ThreadID,
		ReactionID: utility.GenerateUUIDFromSeed(req.Reaction),
		Reaction:   req.Reaction,
		Type:       req.Type,
	}

	if req.Type == "reply" {
		react.MessageID = &req.MessageID
	}

	createEntry := map[string]func(db *gorm.DB) (int, bool, error){
		"thread": func(db *gorm.DB) (int, bool, error) { return react.CreateThreadReaction(db) },
		"reply":  func(db *gorm.DB) (int, bool, error) { return react.CreateReplyReaction(db) },
	}

	code, updateElastic, createErr := createEntry[req.Type](db.Postgresql)
	if createErr != nil {
		logger.Error("failed to save reaction: %v", createErr)
		return code, errors.New("failed to save reaction , error: " + createErr.Error())
	}

	if updateElastic {
		req.Expression = 1
		req.ReactionID = react.ReactionID
		updatedReact, err := UpdateReaction(db, logger, req)

		notification := models.Notification[models.ReactionEvent]
		notification.SectionType = models.ThreadSection
		notification.Reactions = &updatedReact
		notification.ModifcationDetails = &models.ModifcationDetails{
			ThreadId:  req.ThreadID,
			ChannelId: req.ChannelsID,
		}

		if req.Type == "reply" {
			notification.SectionType = models.ReplySection
			notification.ModifcationDetails.MessageId = req.MessageID
		}

		err = centrifuge.PublishChannel(logger, req.ChannelsID, notification)
		if err != nil {
			logger.Error("Error Publishing pinned message event to with destination id: %s error: %v", req.ChannelsID, err.Error())
			return http.StatusInternalServerError, errors.New("failed to publish data: " + err.Error())
		}
		logger.Info("Published reactions to channel successfully")
	}

	return code, nil
}

func DeleteReaction(db *storage.Database, logger *utility.Logger, ids models.IDS) (int, error) {
	var (
		react models.Reaction
	)

	updateEntry := map[string]func() (models.ReactionRequest, error){
		"thread": func() (models.ReactionRequest, error) {

			var (
				checkThread models.ThreadDocument
				reactionReq models.ReactionRequest
			)

			checkThread.ID = ids.ThreadID

			exist, _, _ := checkThread.CheckExists()

			if !exist {
				return reactionReq, errors.New("thread does not exist")
			}

			exists := postgresql.CheckExists(db.Postgresql, &react, "type = ? AND reaction_id = ? AND thread_id = ? AND user_id = ?", ids.Type, ids.ReactionID, ids.ThreadID, ids.UserID)
			if !exists {
				return reactionReq, errors.New("reaction does not exists")
			}

			reactionReq = models.ReactionRequest{
				ReactionID: react.ReactionID,
				ThreadID:   react.ThreadID,
				Expression: -1,
				Type:       ids.Type,
			}

			if err := react.DeleteThreadReaction(db.Postgresql); err != nil {
				logger.Error("An error occurred while deleting reaction record: %v", err)
				return reactionReq, err
			}

			return reactionReq, nil
		},
		"reply": func() (models.ReactionRequest, error) {

			var (
				reactionReq  models.ReactionRequest
				react        models.Reaction
				checkMessage models.MessageDocument
			)

			checkMessage.ID = ids.MessageID

			exist, _, _ := checkMessage.CheckExists()

			if !exist {
				return reactionReq, errors.New("reply does not exist")
			}

			exists := postgresql.CheckExists(db.Postgresql, &react, "type = ? AND reaction_id = ? AND message_id = ? AND user_id = ?", ids.Type, ids.ReactionID, ids.MessageID, ids.UserID)
			if !exists {
				return reactionReq, errors.New("reaction does not exists")
			}

			reactionReq = models.ReactionRequest{
				ReactionID: react.ReactionID,
				MessageID:  *react.MessageID,
				ThreadID:   react.ThreadID,
				Expression: -1,
				Type:       ids.Type,
			}

			if err := react.DeleteReplyReaction(db.Postgresql); err != nil {
				logger.Error("An error occurred while deleting reaction record: %v", err)
				return reactionReq, err
			}

			return reactionReq, nil
		},
	}

	req, err := updateEntry[ids.Type]()
	if err != nil {
		return http.StatusBadRequest, err
	}

	updatedReact, err := UpdateReaction(db, logger, req)

	notification := models.Notification[models.ReactionEvent]
	notification.SectionType = models.ThreadSection
	notification.Reactions = &updatedReact
	notification.ModifcationDetails = &models.ModifcationDetails{
		ThreadId:  req.ThreadID,
		ChannelId: react.ChannelsID,
	}

	if req.Type == "reply" {
		notification.SectionType = models.ReplySection
		notification.ModifcationDetails.MessageId = req.MessageID
	}

	err = centrifuge.PublishChannel(logger, react.ChannelsID, notification)
	if err != nil {
		logger.Error("Error Publishing pinned message event to with destination id: %s error: %v", react.ChannelsID, err.Error())
		return http.StatusInternalServerError, errors.New("failed to publish data: " + err.Error())
	}

	return http.StatusOK, nil
}

func GetReactionUsernames(db *storage.Database, logger *utility.Logger, ids models.IDS) (int, []string, error) {
	var react models.Reaction

	react.ThreadID = ids.ThreadID
	react.MessageID = &ids.MessageID
	react.ReactionID = ids.ReactionID
	react.Type = ids.Type

	code, resp, err := react.GetReactionUsernameByID(db.Postgresql)

	if err != nil {
		logger.Error("An error occurred while fetching pinned messages: %v", err)
		return code, []string{}, err
	}

	return code, resp, nil
}

func UpdateReaction(db *storage.Database, logger *utility.Logger, req models.ReactionRequest) ([]models.ReactionDetails, error) {
	var reactionDetails []models.ReactionDetails

	updateEntry := map[string]func() ([]models.ReactionDetails, error){
		"thread": func() ([]models.ReactionDetails, error) {
			script := `
				if (ctx._source.reactions == null) {
					ctx._source.reactions = [];
				}
				boolean found = false;
				for (int i = 0; i < ctx._source.reactions.size(); i++) {
					if (ctx._source.reactions[i] != null && 
						ctx._source.reactions[i].reaction_id == params.reaction.reaction_id) {
						ctx._source.reactions[i].reaction_count = ctx._source.reactions[i].reaction_count + params.reaction.expression;
						if (ctx._source.reactions[i].reaction_count <= 0) {
							ctx._source.reactions.remove(i);
						}
						found = true;
						break;
					}
				}
				if (!found && params.reaction.expression > 0) {
					def newReaction = [
						'reaction': params.reaction.reaction,
						'reaction_id': params.reaction.reaction_id,
						'reaction_count': 1
					];
					ctx._source.reactions.add(newReaction);
				}`

			reqEntry := map[string]any{
				"script": map[string]any{
					"source": script,
					"params": map[string]any{
						"reaction": map[string]any{
							"reaction":    req.Reaction,
							"reaction_id": req.ReactionID,
							"expression":  req.Expression,
						},
					},
				},
			}

			err := elastic.UpdateDocWithScript(db.Elastic, models.ThreadIndexName, req.ThreadID, reqEntry)
			if err != nil {
				logger.Error(fmt.Sprintf("An error occurred while updating threads: %v", err))
				return reactionDetails, err
			}

			var threadEntry models.ThreadDocument

			err = threadEntry.GetThreadById(db.Postgresql, req.ThreadID)
			if err != nil {
				logger.Error(fmt.Sprintf("An error occurred while updating threads: %v", err))
				return reactionDetails, err
			}

			return threadEntry.Reactions, nil

		},

		"reply": func() ([]models.ReactionDetails, error) {
			script := `
				if (ctx._source.reactions == null) {
					ctx._source.reactions = [];
				}
				boolean found = false;
				for (int i = 0; i < ctx._source.reactions.size(); i++) {
					if (ctx._source.reactions[i] != null && 
						ctx._source.reactions[i].reaction_id == params.reaction.reaction_id) {
						ctx._source.reactions[i].reaction_count = ctx._source.reactions[i].reaction_count + params.reaction.expression;
						if (ctx._source.reactions[i].reaction_count <= 0) {
							ctx._source.reactions.remove(i);
						}
						found = true;
						break;
					}
				}
				if (!found && params.reaction.expression > 0) {
					def newReaction = [
						'reaction': params.reaction.reaction,
						'reaction_id': params.reaction.reaction_id,
						'reaction_count': 1
					];
					ctx._source.reactions.add(newReaction);
				}`

			reqEntry := map[string]any{
				"script": map[string]any{
					"source": script,
					"params": map[string]any{
						"reaction": map[string]any{
							"reaction":    req.Reaction,
							"reaction_id": req.ReactionID,
							"expression":  req.Expression,
						},
					},
				},
			}

			err := elastic.UpdateDocWithScript(db.Elastic, models.MessageIndexName, req.MessageID, reqEntry)
			if err != nil {
				logger.Error(fmt.Sprintf("An error occurred while updating reply message: %v", err))
				return reactionDetails, err
			}

			var msgDoc models.MessageDocument

			err = msgDoc.GetMessageById(db.Postgresql, req.MessageID)
			if err != nil {
				logger.Error(fmt.Sprintf("An error occurred while updating threads: %v", err))
				return reactionDetails, err
			}
			return msgDoc.Reactions, nil
		},
	}

	return updateEntry[req.Type]()
}
