package thread

import (
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
	"github.com/hngprojects/telex_be/utility/channels_utility"
	"gorm.io/gorm"
)

func CreateThreadDummy(req models.Threads, db *gorm.DB) (string, error) {

	threadID := utility.GenerateUUID()
	thread := models.Threads{
		ID:           threadID,
		Username:     req.Username,
		ActionType:   req.ActionType,
		EventName:    req.EventName,
		ChannelsID:   req.ChannelsID,
		UserID:       req.UserID,
		MessageCount: 0,
		ThreadStatus: req.ThreadStatus,
	}

	if err := thread.CreateThread(db); err != nil {
		return "", err
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
