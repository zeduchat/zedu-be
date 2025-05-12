package fcmtokens

import (
	"net/http"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
)

func CreateFcmToken(req models.CreateFcmTokenRequest, db *gorm.DB) (int, error) {

	var ft models.FcmTokens

	ft.FcmToken = req.FcmToken
	ft.UserId = req.UserId
	ft.IsLive = true
	ft.ID = utility.GenerateUUID()

	err := ft.CreateFcmToken(db)

	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusCreated, nil

}

func CreateWnsConfig(req models.CreateWnsTokenRequest, db *gorm.DB) (int, error) {

	var ft models.FcmTokens

	ft.WnsToken = req.WnsToken
	ft.ChannelUri = req.ChannelUri
	ft.UserId = req.UserId
	ft.IsLive = true
	ft.Type = "wns"
	ft.ID = utility.GenerateUUID()

	err := ft.CreateWnsConfig(db)

	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusCreated, nil

}

func GetFcmTokenByUserId(userId string, db *gorm.DB) (string, bool, error) {

	var ft models.FcmTokens

	ft.UserId = userId

	exists, err := ft.GetFcmTokenByUserId(db)

	if err != nil {
		return ft.FcmToken, exists, err
	}

	return ft.FcmToken, exists, nil

}

func GetFcmTokenByUserIds(userIds []string, db *gorm.DB) ([]string, error) {

	var ft models.FcmTokens
	fcmtokens := []string{}

	resp, err := ft.GetFcmTokenByUserIds(db, userIds)

	if err != nil {
		return fcmtokens, err
	}

	for _, ft := range resp {
		fcmtokens = append(fcmtokens, ft.FcmToken)
	}

	return fcmtokens, nil
}
