package thread

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/typesense/typesense-go/v2/typesense"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	tydb "github.com/hngprojects/telex_be/pkg/repository/storage/typesense"
	"github.com/hngprojects/telex_be/utility"
	"github.com/hngprojects/telex_be/utility/channels_utility"
)

func CreateThreadMessage(req models.CreateThreadMsgReq, db *gorm.DB, typesenseDb *typesense.Client) (*models.Threads, error) {

	var (
		profile  models.Profile
		username string
	)

	err := profile.GetProfileByUserId(db, req.UserId)

	if err != nil {
		return nil, errors.New("failed to get user profile")
	}

	username = profile.UserName

	if username == "" {
		username = profile.FirstName
	}

	threadID := utility.GenerateUUID()

	thread := models.Threads{
		ID:           threadID,
		Username:     username,
		Content:      req.Content,
		ChannelsID:   req.ChannelsID,
		Type:         "message",
		MessageCount: 0,
		AvatarURL:    profile.AvatarURL,
	}

	if err = thread.CreateThread(db, typesenseDb); err != nil {
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

func SearchChannel(channelID, searchWords string, db *gorm.DB, c *gin.Context, typesenseDb *typesense.Client) (*[]map[string]interface{}, int, error) {
	searchField := "username,content,event_name,action_type"
	documents, err := tydb.SearchDocuments(typesenseDb, channelID, searchWords, searchField)
	if err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("error searching documents: %v", err)
	}

	return &documents, http.StatusOK, nil
}
