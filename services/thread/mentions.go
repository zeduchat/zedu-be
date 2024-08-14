package thread

import (
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
	"github.com/hngprojects/telex_be/utility/channels_utility"
	"gorm.io/gorm"
)

func CreateThreadIfNeeded(req models.CreateMessageRequest, db *gorm.DB) (string, error) {
	var threadID string
	var threadStatus = "pending"

	if req.ThreadId == "" {
		threadID = utility.GenerateUUID()
		thread := models.Threads{
			ID:           threadID,
			ChannelsID:   req.ChannelsId,
			UserID:       req.UserId,
			MessageCount: 1,
			ThreadStatus: threadStatus,
		}

		if err := thread.CreateThread(db); err != nil {
			return "", err
		}
	} else {
		threadID = req.ThreadId
		var thread models.Threads
		threadData, err := thread.GetThreadById(db, threadID)
		if err != nil {
			return "", err
		}
		threadData.MessageCount++
		if _, err := threadData.UpdateThread(db); err != nil {
			return "", err
		}
	}

	return threadID, nil
}

func DetectAndAddMentions(messageID string, content string, db *gorm.DB) error {
	mentions := channels_utility.DetectMentions(content)

	for _, username := range mentions {
		var user models.Profile
		user, err := user.GetUserByUsername(db, username)
		if err != nil {
			continue
		}

		mention := models.Mentions{
			ID:        utility.GenerateUUID(),
			MessageID: messageID,
			UserID:    user.ID,
		}

		if err := mention.CreateMention(db); err != nil {
			return err
		}
	}

	return nil
}
