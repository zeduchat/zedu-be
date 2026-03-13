package telexaudit

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/webhook"
	"github.com/hngprojects/telex_be/utility"
)

func SignupAudit(db *storage.Database, logger *utility.Logger, data gin.H) error {

	var (
		req models.CreateWebhookHistoryRequest
	)

	email := data["user"].(map[string]any)["email"]

	channelID := config.Config.Channels.Signup

	if _, err := uuid.Parse(channelID); err != nil {
		logger.Info("error parsing channel id")
		return nil
	}
	req.ChannelID = channelID
	req.UserName = "Telex Backend"
	req.EventName = "User Signup"
	req.Status = "success"
	req.Message = fmt.Sprintf("User with email %s signed up", email)
	req.AvatarURL = fmt.Sprintf("%s/TelexIcon.svg", config.Config.App.FRONTEND_URL)

	_, _, err := webhook.PostFeedWebhook(db, logger, req)

	if err != nil {
		return err
	}

	return nil
}

func LoginAudit(db *storage.Database, logger *utility.Logger, data gin.H) error {

	var (
		req models.CreateWebhookHistoryRequest
	)

	email := data["user"].(map[string]any)["email"]

	channelID := config.Config.Channels.Login

	if _, err := uuid.Parse(channelID); err != nil {
		logger.Info("error parsing channel id")
		return nil
	}

	req.ChannelID = channelID
	req.UserName = "Telex Backend"
	req.EventName = "User Login"
	req.Status = "success"
	req.Message = fmt.Sprintf("User with email %s loggedin ", email)
	req.AvatarURL = fmt.Sprintf("%s/TelexIcon.svg", config.Config.App.FRONTEND_URL)

	_, _, err := webhook.PostFeedWebhook(db, logger, req)

	if err != nil {
		return err
	}

	return nil
}

func RoleChangeAudit(db *storage.Database, logger *utility.Logger, requesterEmail, targetEmail, oldRole, newRole string) error {
	var req models.CreateWebhookHistoryRequest

	channelID := config.Config.Channels.Audit

	if _, err := uuid.Parse(channelID); err != nil {
		return fmt.Errorf("invalid audit channel id")
	}

	req.ChannelID = channelID
	req.UserName = "Security Monitor"
	req.EventName = "Role Change"
	req.Status = "success"
	req.Message = fmt.Sprintf("Superadmin (%s) changed role of %s from %s to %s",
		requesterEmail, targetEmail, oldRole, newRole)
	req.AvatarURL = fmt.Sprintf("%s/TelexIcon.svg", config.Config.App.FRONTEND_URL)

	_, _, err := webhook.PostFeedWebhook(db, logger, req)
	return err
}

func CreateAdminAudit(db *storage.Database, logger *utility.Logger, requesterEmail, targetEmail string) error {
	var req models.CreateWebhookHistoryRequest

	channelID := config.Config.Channels.Audit

	if _, err := uuid.Parse(channelID); err != nil {
		if logger != nil {
			logger.Info("error parsing channel id")
		}
		return nil
	}

	req.ChannelID = channelID
	req.UserName = "Security Monitor"
	req.EventName = "Admin Created"
	req.Status = "success"
	req.Message = fmt.Sprintf("Superadmin (%s) created new admin account for %s",
		requesterEmail, targetEmail)
	req.AvatarURL = fmt.Sprintf("%s/TelexIcon.svg", config.Config.App.FRONTEND_URL)

	_, _, err := webhook.PostFeedWebhook(db, logger, req)
	return err
}
