package telexaudit

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/typesense/typesense-go/v2/typesense"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/webhook"
	"github.com/hngprojects/telex_be/utility"
)

func SingupAudit(db *gorm.DB, logger *utility.Logger, data gin.H, typDb *typesense.Client) error {

	var (
		req models.CreateWebhookHistoryRequest
	)

	req.ChannelID = config.Config.Channels.Signup
	req.UserName = "Telex Backend"
	req.EventName = "User Signup"
	req.Status = "success"
	req.Message = fmt.Sprintf("%s%s", data["email"], data["username"])
	req.AvatarURL = fmt.Sprintf("%s/TelexIcon.svg", config.Config.App.FRONTEND_URL)

	_, _, err := webhook.PostFeedWebhook(db, logger, req, typDb)

	if err != nil {
		return err
	}

	return nil
}

func LoginAudit(db *gorm.DB, logger *utility.Logger, data gin.H, typDb *typesense.Client) error {

	var (
		req models.CreateWebhookHistoryRequest
	)

	req.ChannelID = config.Config.Channels.Login
	req.UserName = "Telex Backend"
	req.EventName = "User Login"
	req.Status = "success"
	req.Message = fmt.Sprintf("%s%s", data["email"], data["username"])
	req.AvatarURL = fmt.Sprintf("%s/TelexIcon.svg", config.Config.App.FRONTEND_URL)

	_, _, err := webhook.PostFeedWebhook(db, logger, req, typDb)

	if err != nil {
		return err
	}

	return nil
}
