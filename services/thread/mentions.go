package thread

import (
	"fmt"
	"math/rand"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
	"github.com/hngprojects/telex_be/utility/channels_utility"
	"gorm.io/gorm"
)

func CreateThreadDummy(req models.Threads, db *gorm.DB) (*models.Threads, error) {

	threadID := utility.GenerateUUID()

	statuses := []string{"failed", "pending", "completed"}
	randomStatus := statuses[rand.Intn(len(statuses))]

	thread := models.Threads{
		ID:           threadID,
		Username:     fmt.Sprintf("User_%s", utility.RandomString(7)),
		ActionType:   fmt.Sprintf("Action_%s", utility.RandomString(7)),
		EventName:    fmt.Sprintf("Event_%s", utility.RandomString(7)),
		ChannelsID:   req.ChannelsID,
		MessageCount: 0,
		ThreadStatus: randomStatus,
	}

	if err := thread.CreateThread(db); err != nil {
		return nil, err
	}

	return &thread, nil
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
