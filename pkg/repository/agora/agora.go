package agora

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"time"

	rtctokenbuilder "github.com/AgoraIO-Community/go-tokenbuilder/rtctokenbuilder"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
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

// GenerateRTCToken creates an Agora RTC token for a user to join a buzz channel
// expireTimeInSeconds: token expiration time in seconds (0 = use default 2 hours)
func (s *AgoraService) GenerateRTCToken(channelName, userID string, expireTimeInSeconds uint32) (string, error) {
	if s.appId == "" || s.appCertificate == "" {
		return "", errors.New("Agora App ID or App Certificate not configured")
	}

	// Default to 2 hours if not specified
	if expireTimeInSeconds == 0 {
		expireTimeInSeconds = 7200 // 2 hours
	}

	// Build token with UID (user can publish and subscribe)
	// Role: 1 = publisher (can publish and subscribe)
	token, err := rtctokenbuilder.BuildTokenWithUid(
		s.appId,
		s.appCertificate,
		channelName,
		0, // Use 0 for string-based user accounts
		rtctokenbuilder.RolePublisher, // RolePublisher allows both publish and subscribe
		expireTimeInSeconds,
	)

	if err != nil {
		return "", fmt.Errorf("failed to generate Agora RTC token: %w", err)
	}

	s.logger.Info("Generated Agora RTC token for user %s in channel %s (expires in %d seconds)", userID, channelName, expireTimeInSeconds)
	return token, nil
}

// GetAgoraToken generates an Agora RTC token for a user to join a buzz
func GetAgoraToken(db *storage.Database, logger *utility.Logger, buzzID, userID string) (models.AgoraTokenResponse, int, error) {
	var resp models.AgoraTokenResponse

	service := Client.Service
	if service == nil {
		return resp, http.StatusInternalServerError, errors.New("Agora service not initialized")
	}

	// Validate buzz exists and is active
	var buzz models.Buzz
	if err := db.Postgresql.Where("id = ? AND status = ? AND is_live_status = ?",
		buzzID, models.BuzzStatusActive, true).First(&buzz).Error; err != nil {
		return resp, http.StatusNotFound, errors.New("buzz not found or not active")
	}

	// Validate user is a participant
	if !isUserParticipant(buzz.ParticipantIDs, userID) {
		return resp, http.StatusForbidden, errors.New("user is not a participant in this buzz")
	}

	// Calculate dynamic token expiration based on buzz duration
	expireTimeInSeconds := calculateTokenExpiration(buzz.BuzzStartTime, logger)

	// Generate token using buzz ID as channel name
	token, err := service.GenerateRTCToken(buzzID, userID, expireTimeInSeconds)
	if err != nil {
		logger.Error("Failed to generate Agora token: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to generate access token")
	}

	resp = models.AgoraTokenResponse{
		Token:       token,
		AppId:       service.appId,
		ChannelName: buzzID,
		UID:         0, // Using 0 for string-based user accounts
	}

	logger.Info("Generated Agora token for user %s in buzz %s", userID, buzzID)
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

// calculateTokenExpiration calculates dynamic token expiration based on buzz duration
func calculateTokenExpiration(buzzStartTime time.Time, logger *utility.Logger) uint32 {
	const (
		maxBuzzDurationHours    = 4  // Maximum allowed buzz duration (4 hours)
		minTokenValidityMinutes = 15 // Minimum token validity (15 minutes safety buffer)
	)

	// Calculate elapsed time since buzz started
	elapsedTime := time.Since(buzzStartTime)
	elapsedSeconds := int64(elapsedTime.Seconds())

	// Calculate maximum buzz duration in seconds
	maxDurationSeconds := int64(maxBuzzDurationHours * 3600)

	// Calculate remaining time until max duration
	remainingSeconds := maxDurationSeconds - elapsedSeconds

	// Ensure minimum token validity (15 minutes)
	minValiditySeconds := int64(minTokenValidityMinutes * 60)

	// Use the greater of remaining time or minimum validity
	tokenExpiry := math.Max(float64(remainingSeconds), float64(minValiditySeconds))

	// Cap at max duration if buzz just started
	if tokenExpiry > float64(maxDurationSeconds) {
		tokenExpiry = float64(maxDurationSeconds)
	}

	expirySeconds := uint32(tokenExpiry)

	logger.Info("Token expiration calculated: buzz running for %.2f minutes, token valid for %.2f minutes",
		elapsedTime.Minutes(), float64(expirySeconds)/60)

	return expirySeconds
}
