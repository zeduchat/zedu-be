package webhook

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
)

func CreateWebhook(req models.CreateWebhookRequest, db *gorm.DB) (gin.H, int, error) {

	var (
		resp    gin.H
		webhook models.Webhook
	)

	webhook = models.Webhook{
		ID:        utility.GenerateUUID(),
		EventName: "",
		ChannelId: req.ChannelID,
		OwnerId:   req.UserID,
	}

	err := webhook.CreateWebhook(db)

	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	return resp, http.StatusCreated, nil
}


func GetAllWebhook() (gin.H, int, error) {

	var (
		resp gin.H
	)

	return resp, http.StatusCreated, nil

}

func GetAWebhook() (gin.H, int, error) {

	var (
		resp gin.H
	)

	return resp, http.StatusCreated, nil

}

func GetWebhookHistory() (gin.H, int, error) {

	var (
		resp gin.H
	)

	return resp, http.StatusCreated, nil

}

func PostWebhook() (gin.H, int, error) {

	var (
		resp gin.H
	)

	return resp, http.StatusCreated, nil

}

func DeleteWebhook() (gin.H, int, error) {

	var (
		resp gin.H
	)

	return resp, http.StatusCreated, nil

}

func ChangeWebhookStatus() (gin.H, int, error) {

	var (
		resp gin.H
	)

	return resp, http.StatusCreated, nil

}
