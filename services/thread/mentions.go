package thread

import (
	"fmt"
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	tydb "github.com/hngprojects/telex_be/pkg/repository/storage/typesense"
	"github.com/hngprojects/telex_be/utility"
	"github.com/hngprojects/telex_be/utility/channels_utility"
	"github.com/typesense/typesense-go/v2/typesense"
	"gorm.io/gorm"
)

func CreateThreadDummy(req models.Threads, db *gorm.DB, typesenseDb *typesense.Client) (*models.Threads, error) {

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
		Status:       randomStatus,
	}

	if err := thread.CreateThread(db, typesenseDb); err != nil {
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
