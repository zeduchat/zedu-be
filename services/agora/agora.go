package agora

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"

	rtctokenbuilder "github.com/AgoraIO-Community/go-tokenbuilder/rtctokenbuilder"
	rtmtokenbuilder "github.com/AgoraIO-Community/go-tokenbuilder/rtmtokenbuilder"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
)

const (
	defaultExpirySeconds = 3600
)

func GenerateRTCToken(logger *utility.Logger, req models.AgoraTokenRequest, userID string) (models.AgoraTokenResponse, int, error) {
	appID, appCert, err := fetchAgoraCredentials()
	if err != nil {
		return models.AgoraTokenResponse{}, http.StatusInternalServerError, err
	}

	expiry := expirySecondsOrDefault(req.ExpirySeconds)
	role := parseRTCRole(req.Role)
	tokenType := req.TokenType
	if tokenType == "" {
		tokenType = "userAccount"
	}

	uid := req.UID
	if uid == "" {
		uid = userID
	}

	var rtcToken string
	switch tokenType {
	case "userAccount":
		rtcToken, err = rtctokenbuilder.BuildTokenWithAccount(appID, appCert, req.ChannelName, uid, role, expiry)
	case "uid":
		uidUint, convErr := strconv.ParseUint(uid, 10, 64)
		if convErr != nil {
			return models.AgoraTokenResponse{}, http.StatusBadRequest, errors.New("uid must be a numeric string when token_type=uid")
		}
		rtcToken, err = rtctokenbuilder.BuildTokenWithUid(appID, appCert, req.ChannelName, uint32(uidUint), role, expiry)
	default:
		return models.AgoraTokenResponse{}, http.StatusBadRequest, errors.New("invalid token_type; use uid or userAccount")
	}

	if err != nil {
		logger.Error("failed to build RTC token: %v", err)
		return models.AgoraTokenResponse{}, http.StatusInternalServerError, errors.New("unable to generate rtc token")
	}

	return models.AgoraTokenResponse{RTCToken: rtcToken}, http.StatusOK, nil
}

func GenerateRTMToken(logger *utility.Logger, req models.AgoraTokenRequest, userID string) (models.AgoraTokenResponse, int, error) {
	appID, appCert, err := fetchAgoraCredentials()
	if err != nil {
		return models.AgoraTokenResponse{}, http.StatusInternalServerError, err
	}

	expiry := expirySecondsOrDefault(req.ExpirySeconds)
	uid := req.UID
	if uid == "" {
		uid = userID
	}

	rtmToken, err := rtmtokenbuilder.BuildToken(appID, appCert, uid, expiry, "")
	if err != nil {
		logger.Error("failed to build RTM token: %v", err)
		return models.AgoraTokenResponse{}, http.StatusInternalServerError, errors.New("unable to generate rtm token")
	}

	return models.AgoraTokenResponse{RTMToken: rtmToken}, http.StatusOK, nil
}

func GenerateBothTokens(logger *utility.Logger, req models.AgoraTokenRequest, userID string) (models.AgoraTokenResponse, int, error) {
	rtcResp, code, err := GenerateRTCToken(logger, req, userID)
	if err != nil {
		return models.AgoraTokenResponse{}, code, err
	}

	rtmResp, code, err := GenerateRTMToken(logger, req, userID)
	if err != nil {
		return models.AgoraTokenResponse{}, code, err
	}

	return models.AgoraTokenResponse{
		RTCToken: rtcResp.RTCToken,
		RTMToken: rtmResp.RTMToken,
	}, http.StatusOK, nil
}

func fetchAgoraCredentials() (string, string, error) {
	appID := os.Getenv("AGORA_APP_ID")
	appCert := os.Getenv("AGORA_APP_CERTIFICATE")
	if appID == "" || appCert == "" {
		return "", "", fmt.Errorf("agora credentials missing: set AGORA_APP_ID and AGORA_APP_CERTIFICATE")
	}
	return appID, appCert, nil
}

func expirySecondsOrDefault(value int) uint32 {
	if value <= 0 {
		return defaultExpirySeconds
	}
	return uint32(value)
}

func parseRTCRole(role string) rtctokenbuilder.Role {
	if role == "publisher" {
		return rtctokenbuilder.RolePublisher
	}
	return rtctokenbuilder.RoleSubscriber
}
