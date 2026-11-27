package token

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
)

func GetConnToken(userId string, db *gorm.DB) (gin.H, int, error) {

	userClaims := jwt.MapClaims{}

	userClaims["sub"] = userId
	userClaims["exp"] = time.Now().Unix() + int64(3600)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, userClaims)

	connToken, err := token.SignedString([]byte(config.Config.Centrifuge.Secret))
	if err != nil {
		return gin.H{}, http.StatusInternalServerError, err
	}

	res := gin.H{
		"token": connToken,
	}

	return res, http.StatusOK, nil

}

func GetSubToken(userId string, req models.ChannelSubTokenReq, db *gorm.DB) (gin.H, int, error) {

	var (
		channelName = req.Channel
		org         = models.Organisation{}
		orgId       string
		cUserId     string
	)

	parts := strings.Split(req.Channel, "/")
	if len(parts) == 2 {
		orgId = parts[0]
		cUserId = parts[1]

		if _, err := uuid.Parse(orgId); err != nil {
			return gin.H{}, http.StatusBadRequest, errors.New("invalid channel id format")
		}

		exist, _ := org.CheckUserIsMemberOfOrg(userId, orgId, db)

		if !exist || cUserId != userId {
			return gin.H{}, http.StatusBadRequest, errors.New("invalid channel subscription")
		}

	} else {
		if !utility.IsValidBuzzCodeOrUUID(channelName) {
			return gin.H{}, http.StatusBadRequest, errors.New("invalid channel id supplied")
		}

		allowed, err := canSubscribe(db, userId, channelName)
		if err != nil {
			return gin.H{}, http.StatusInternalServerError, errors.New("could not verify channel access")
		}
		if !allowed {
			return gin.H{}, http.StatusForbidden, errors.New("you do not have access to this channel")
		}
	}

	return SignSubToken(userId, channelName)
}

// SignSubToken signs a one-hour Centrifugo subscription token. Callers must have
// authorised userId for channel first.
func SignSubToken(userId, channel string) (gin.H, int, error) {
	userClaims := jwt.MapClaims{}
	userClaims["sub"] = userId
	userClaims["channel"] = channel
	userClaims["exp"] = time.Now().Unix() + int64(3600)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, userClaims)

	subToken, err := token.SignedString([]byte(config.Config.Centrifuge.Secret))
	if err != nil {
		return gin.H{}, http.StatusInternalServerError, err
	}

	res := gin.H{
		"token": subToken,
	}

	return res, http.StatusOK, nil
}

// canSubscribe reports whether userId may follow the Centrifugo channel: an
// organisation feed (org id), a channel, DM or group DM, a buzz (id or code), or a
// thread (via its parent channel). Unknown ids are refused.
func canSubscribe(db *gorm.DB, userId, channel string) (bool, error) {
	// Recording bot: "<buzz_id>-<recording_uid>", may follow only its own buzz.
	if len(userId) > 37 && userId[36] == '-' {
		var rec models.BuzzRecording
		if err := db.Where("buzz_id = ? AND recording_uid = ?", userId[:36], userId[37:]).First(&rec).Error; err != nil {
			return false, nil
		}
		var buzz models.Buzz
		if err := db.Where("id = ?", userId[:36]).First(&buzz).Error; err != nil {
			return false, nil
		}
		return channel == buzz.ID || channel == buzz.ChannelID ||
			strings.EqualFold(channel, utility.ExtractBuzzCode(buzz.ID)), nil
	}
	if !utility.IsValidUUID(userId) || userId == "00000000-0000-0000-0000-000000000000" {
		return false, nil
	}

	if utility.IsValidBuzzCode(channel) {
		buzzID, err := utility.ResolveBuzzCode(db, channel)
		if err != nil {
			return false, nil
		}
		channel = buzzID
	}

	var org models.Organisation
	if ok, _ := org.CheckUserIsMemberOfOrg(userId, channel, db); ok {
		return true, nil // organisation-wide feed
	}
	if ok, err := models.UserCanAccessChannel(db, channel, userId); err != nil || ok {
		return ok, err // channel, DM or group DM
	}

	var buzz models.Buzz
	if err := db.Where("id = ?", channel).First(&buzz).Error; err == nil {
		var p models.BuzzParticipant
		if db.Where("buzz_id = ? AND user_id = ?", buzz.ID, userId).First(&p).Error == nil {
			return true, nil
		}
		if buzz.OrgID != nil {
			if ok, _ := org.CheckUserIsMemberOfOrg(userId, *buzz.OrgID, db); ok {
				return true, nil
			}
		}
		return models.UserCanAccessChannel(db, buzz.ChannelID, userId)
	}

	var thread models.ThreadDocument
	if err := thread.GetThreadById(channel); err == nil && thread.ChannelsID != "" {
		return models.UserCanAccessChannel(db, thread.ChannelsID, userId)
	}
	return false, nil
}
