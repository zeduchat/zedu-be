package fcmtokens

import (
	"errors"
	"net/http"
	"strings"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
)

func CreateFcmToken(req models.CreateFcmTokenRequest, db *gorm.DB) (int, error) {
	platform := strings.ToLower(strings.TrimSpace(req.Platform))
	if platform == "" {
		platform = "web"
	}
	if platform != "web" && platform != "android" && platform != "ios" {
		return http.StatusUnprocessableEntity, errors.New("platform must be web, android, or ios")
	}

	token := models.UserDeviceToken{
		UserId:   req.UserId,
		Token:    strings.TrimSpace(req.FcmToken),
		Platform: platform,
	}
	if err := token.Upsert(db); err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusCreated, nil
}

func DeleteFcmToken(userID, token string, db *gorm.DB) (int, error) {
	if err := models.DeleteDeviceToken(db, userID, token); err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
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
	var token models.UserDeviceToken
	err := db.Where("user_id = ? AND token <> ''", userId).Order("updated_at desc").First(&token).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", false, nil
		}
		return "", false, err
	}
	return token.Token, true, nil
}

func GetWebPushTokenByUserId(userId string, db *gorm.DB) (*models.SubsPayload, bool, error) {

	var ft models.FcmTokens

	ft.UserId = userId

	exists, err := ft.GetValidWebPushTokenByUserId(db)

	if err != nil {
		return &ft.SubsPayload, exists, err
	}

	return &ft.SubsPayload, exists, nil

}

func GetFcmTokenByUserIds(userIds []string, db *gorm.DB) (*[]string, error) {
	tokens, err := models.ListDeviceTokensByUserIDs(db, userIds)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(tokens))
	for _, t := range tokens {
		result = append(result, t.Token)
	}
	return &result, nil
}

func CreateWebPushToken(req models.CreateWebPushTokenRequest, db *gorm.DB) (int, error) {

	var ft models.FcmTokens

	ft.SubsPayload = models.SubsPayload{
		Endpoint: req.Endpoint,
		Keys: models.Keys{
			Auth:   req.Auth,
			P256dh: req.P256dh,
		},
	}
	ft.IsLive = true
	ft.ID = utility.GenerateUUID()
	ft.UserId = req.UserId

	err := ft.CreateWebPushConfig(db)

	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusCreated, nil

}

func GetWebPushTokenByUserIds(userIds []string, db *gorm.DB) (*[]models.SubsPayload, error) {

	var ft models.FcmTokens
	webptokens := []models.SubsPayload{}

	resp, err := ft.GetValidWebPushTokenByUserIds(db, userIds)

	if err != nil {
		return &webptokens, err
	}

	for _, ft := range *resp {
		webptokens = append(webptokens, ft.SubsPayload)
	}

	return &webptokens, nil
}

func CreateVoIPToken(req models.VoIPTokenRequest, userId string, db *gorm.DB) (int, error) {
	var ft models.FcmTokens

	ft.VoIPToken = req.VoIPToken
	ft.UserId = userId
	ft.IsLive = true
	ft.ID = utility.GenerateUUID()

	err := ft.CreateVoIPToken(db)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func GetVoIPTokenByUserIds(userIds []string, db *gorm.DB) (*[]string, error) {
	var ft models.FcmTokens
	voiptokens := []string{}

	resp, err := ft.GetFcmTokenByUserIds(db, userIds)
	if err != nil {
		return &voiptokens, err
	}

	for _, token := range *resp {
		if token.VoIPToken != "" {
			voiptokens = append(voiptokens, token.VoIPToken)
		}
	}

	return &voiptokens, nil
}

func ClearVoIPToken(deviceToken string, db *gorm.DB) error {
	var ft models.FcmTokens
	ft.VoIPToken = deviceToken
	return ft.ClearVoIPToken(db)
}
