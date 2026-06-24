package agora

import (
	"errors"
	"fmt"
	"net/http"

	rtctokenbuilder "github.com/AgoraIO-Community/go-tokenbuilder/rtctokenbuilder"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

const (
	// DefaultTokenExpirationSeconds is the default token expiration time (4 hours - matches max buzz duration)
	DefaultTokenExpirationSeconds = 5400 // 1 hour 30 minutes
)

// AgoraService handles Agora RTC token generation
type AgoraService struct {
	logger         *utility.Logger
	appId          string
	appCertificate string
}

func NewAgoraService(logger *utility.Logger, config config.Agora) *AgoraService {
	service := &AgoraService{
		logger:         logger,
		appId:          config.AppId,
		appCertificate: config.AppCertificate,
	}

	Client.Service = service

	utility.LogAndPrint(logger, fmt.Sprintf("Agora service initialized with App ID: %s", config.AppId))
	return service
}

// GetAppId returns the Agora App ID
func (s *AgoraService) GetAppId() string {
	return s.appId
}

// check Agora config
func (s *AgoraService) IsHealthy() bool {
	return s.appId != "" && s.appCertificate != ""
}

// GenerateRTCToken creates an Agora RTC token for a user to join a buzz channel
func (s *AgoraService) GenerateRTCToken(channelName, userID string, uid string, expireTimeInSeconds uint32) (string, error) {
	if !s.IsHealthy() {
		return "", errors.New("agora App ID or App Certificate not configured")
	}

	// Default to 2 hours if not specified
	if expireTimeInSeconds == 0 {
		expireTimeInSeconds = 7200 // 2 hours
	}

	token, err := rtctokenbuilder.BuildTokenWithAccount(
		s.appId,
		s.appCertificate,
		channelName,
		uid,
		rtctokenbuilder.RolePublisher,
		expireTimeInSeconds,
	)

	if err != nil {
		return "", fmt.Errorf("failed to generate Agora RTC token: %w", err)
	}

	s.logger.Info("Generated Agora RTC token for user %s (UID: %s) in channel %s (expires in %d seconds)", userID, uid, channelName, expireTimeInSeconds)
	return token, nil
}

// GetAgoraToken generates an Agora RTC token for a user to join a buzz
func GetAgoraToken(db *storage.Database, logger *utility.Logger, buzzID, userID string, uid string) (models.BuzzAgoraTokenResponse, int, error) {
	var resp models.BuzzAgoraTokenResponse

	service := Client.Service
	if service == nil {
		return resp, http.StatusInternalServerError, errors.New("agora service not initialized")
	}

	// Validate buzz exists and is active
	var buzz models.Buzz
	if err := db.Postgresql.Where("id = ? AND status = ? AND is_live_status = ?",
		buzzID, models.BuzzStatusActive, true).First(&buzz).Error; err != nil {
		return resp, http.StatusNotFound, errors.New("buzz not found or not active")
	}

	isBot, recUid := checkAndResolveBotUID(db, buzzID, userID, uid)

	if !isBot && !isUserParticipant(buzz.ParticipantIDs, userID) {
		return resp, http.StatusForbidden, errors.New("user is not a participant in this buzz")
	}

	// Use Agora default token expiration (4 hours)
	expireTimeInSeconds := uint32(DefaultTokenExpirationSeconds)

	// Generate token using buzz ID as channel name and provided UID
	token, err := service.GenerateRTCToken(buzzID, userID, recUid, expireTimeInSeconds)
	if err != nil {
		logger.Error("Failed to generate Agora token: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to generate access token")
	}

	resp = models.BuzzAgoraTokenResponse{
		Token:       token,
		AppId:       service.appId,
		ChannelName: buzzID,
		UID:         recUid,
	}

	logger.Info("Generated Agora token for user %s (UID: %s) in buzz %s", userID, recUid, buzzID)
	return resp, http.StatusOK, nil
}

// isUserParticipant checks if a user is in the participants list
func isUserParticipant(participants []string, userID string) bool {
	for _, p := range participants {
		if p == userID {
			return true
		}
	}
	return false
}

func checkAndResolveBotUID(db *storage.Database, buzzID, userID, defaultUID string) (bool, string) {

	if len(userID) > 36 && userID[:36] == buzzID {
		var rec models.BuzzRecording

		if err := db.Postgresql.Where("buzz_id = ?", buzzID).First(&rec).Error; err == nil {
			botUserID := fmt.Sprintf("%s-%s", rec.BuzzID, rec.RecordingUID)
			
			if userID == botUserID {
				return true, rec.RecordingUID
			}
		}
	}

	return false, defaultUID
}
