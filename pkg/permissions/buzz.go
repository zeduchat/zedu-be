package permissions

import (
	"errors"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
)

var (
	ErrNotHost              = errors.New("only the buzz host can perform this action")
	ErrNotParticipant       = errors.New("you must be a buzz participant")
	ErrBuzzEnded            = errors.New("this buzz has ended")
	ErrBuzzNotFound         = errors.New("buzz not found")
	ErrNotChannelMember     = errors.New("participant is not a channel member")
	ErrBuzzAlreadyActive    = errors.New("channel already has an active buzz")
	ErrNotActiveParticipant = errors.New("you must be an active participant")
	ErrChannelNotFound      = errors.New("channel not found")
	ErrAlreadyInBuzz        = errors.New("user is already in the buzz")
)

// GetChannelType determines the type of channel (regular, DM, or group DM)
func GetChannelType(db *gorm.DB, channelID string) (string, error) {
	// Check regular channels first
	var channel models.Channels
	err := db.Where("id = ?", channelID).First(&channel).Error
	if err == nil {
		return models.ChannelTypeRegular, nil
	}
	// If it's not a "not found" error, it's a real database error
	if err != gorm.ErrRecordNotFound {
		return "", err
	}

	// Check DM channels table
	var dmChannel models.DmChannels
	err = db.Where("channel_id = ?", channelID).First(&dmChannel).Error
	if err == nil {
		if dmChannel.ChannelType == "group_dm" {
			return models.ChannelTypeGroupDM, nil
		}
		return models.ChannelTypeDM, nil
	}
	// If it's not a "not found" error, it's a real database error
	if err != gorm.ErrRecordNotFound {
		return "", err
	}

	return "", ErrChannelNotFound
}

// IsHost checks if the user is the buzz host
func IsHost(db *gorm.DB, buzzID, userID string) (bool, error) {
	var buzz models.Buzz
	if err := db.Where("id = ?", buzzID).First(&buzz).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, ErrBuzzNotFound
		}
		return false, err
	}

	if buzz.HostID != userID {
		return false, ErrNotHost
	}

	return true, nil
}

// IsBuzzActive checks if a buzz is currently active and live
func IsBuzzActive(db *gorm.DB, buzzID string) (*models.Buzz, error) {
	var buzz models.Buzz
	if err := db.Where("id = ?", buzzID).First(&buzz).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrBuzzNotFound
		}
		return nil, err
	}

	if buzz.Status != models.BuzzStatusActive || !buzz.IsLiveStatus {
		return nil, ErrBuzzEnded
	}

	return &buzz, nil
}

// IsActiveParticipant checks if user is an active participant in the buzz
func IsActiveParticipant(db *gorm.DB, buzzID, userID string) (bool, error) {
	var participant models.BuzzParticipant
	err := db.Where("buzz_id = ? AND user_id = ? AND status = ?",
		buzzID, userID, models.BuzzParticipantStatusActive).First(&participant).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, ErrNotActiveParticipant
		}
		return false, err
	}

	return true, nil
}

// HasActiveBuzzInChannel checks if a channel already has an active buzz
func HasActiveBuzzInChannel(db *gorm.DB, channelID string) (bool, error) {
	var buzz models.Buzz
	err := db.Where("channel_id = ? AND status = ? AND is_live_status = ?",
		channelID, models.BuzzStatusActive, true).First(&buzz).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil // No active buzz found - this is good
		}
		return false, err
	}

	return true, nil // Active buzz exists - this is an error
}

// ValidateParticipantsAreChannelMembers validates that all participant IDs are channel members
func ValidateParticipantsAreChannelMembers(db *gorm.DB, channelID string, participantIDs []string) error {
	for _, participantID := range participantIDs {
		if participantID == "" {
			continue
		}

		if !models.IsUserInChannel(db, channelID, participantID) {
			return ErrNotChannelMember
		}
	}

	return nil
}

// CanPerformBuzzAction checks if user can perform an action on a buzz (must be active participant)
func CanPerformBuzzAction(db *gorm.DB, buzzID, userID string) (*models.Buzz, error) {
	// Check if buzz is active
	buzz, err := IsBuzzActive(db, buzzID)
	if err != nil {
		return nil, err
	}

	// Check if user is an active participant
	isParticipant, err := IsActiveParticipant(db, buzzID, userID)
	if err != nil {
		return nil, err
	}

	if !isParticipant {
		return nil, ErrNotActiveParticipant
	}

	return buzz, nil
}

// CanPerformHostAction checks if user can perform a host-only action
func CanPerformHostAction(db *gorm.DB, buzzID, userID string) (*models.Buzz, error) {
	// Check if buzz exists and is active
	buzz, err := IsBuzzActive(db, buzzID)
	if err != nil {
		return nil, err
	}

	// Check if user is the host
	if buzz.HostID != userID {
		return nil, ErrNotHost
	}

	return buzz, nil
}

// CanCreateBuzz validates if a user can create a buzz in a channel (regular, DM, or group DM)
func CanCreateBuzz(db *gorm.DB, channelID, hostID string) error {
	// Check if channel exists in any table (channels, dm_channels, or channel_participants)
	channelExists := false

	// Check regular channels
	chModel := models.Channels{}
	exists, _ := chModel.CheckChannelExists(db, channelID)
	if exists {
		channelExists = true
	}

	// Check DM channels
	if !channelExists {
		var dmChannel models.DmChannels
		if err := db.Where("channel_id = ?", channelID).First(&dmChannel).Error; err == nil {
			channelExists = true
		}
	}

	// Check group DM channels (via channel_participants)
	if !channelExists {
		var participant models.ChannelParticipant
		if err := db.Where("channel_id = ?", channelID).First(&participant).Error; err == nil {
			channelExists = true
		}
	}

	if !channelExists {
		return ErrChannelNotFound
	}

	// Check if user is a channel member
	if !models.IsUserInChannel(db, channelID, hostID) {
		return ErrNotChannelMember
	}

	// Check if channel already has an active buzz
	hasActiveBuzz, err := HasActiveBuzzInChannel(db, channelID)
	if err != nil {
		// Log the database error but return a user-friendly error
		// to prevent 500 errors from propagating
		return ErrChannelNotFound
	}
	if hasActiveBuzz {
		return ErrBuzzAlreadyActive
	}

	return nil
}

func CanJoinBuzz(db *gorm.DB, buzzID, userID string) (*models.Buzz, error) {
	buzz, err := IsBuzzActive(db, buzzID)
	if err != nil {
		return nil, err
	}

	if !models.IsUserInChannel(db, buzz.ChannelID, userID) && buzz.BuzzType != models.BuzzTypeOrganization {
		return nil, ErrNotChannelMember
	}

	return buzz, nil
}
