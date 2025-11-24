package huddle

import (
	"errors"
	"slices"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func SetActiveScreen(huddleId string) {

}

func Validate(db *storage.Database, logger *utility.Logger, huddleId string, userId string, activeViewId string) error {
	// Verify huddle exists
	var huddle models.Huddle

	err := db.Postgresql.First(&huddle, "id = ?", huddleId).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("huddle not found")
		}
		logger.Error("failed to get huddle: %v", err)
		return errors.New("failed to get huddle")
	}

	// Checks if user is a participant
	if !slices.Contains(huddle.ParticipantIDs, userId) {
		logger.Error("user is not a participant in this huddle")
		return errors.New("user is not a participant in this huddle")
	}

	// Checks if screen sharer is a participant
	if !slices.Contains(huddle.ParticipantIDs, activeViewId) {
		logger.Error("screen sharer is not a participant in this huddle")
		return errors.New("screen sharer is not a participant in this huddle")
	}

	// Find the record of the screen sharer
	var sharerParticipant models.HuddleParticipant
	err = db.Postgresql.Where("huddle_id = ? AND user_id = ?", huddleId, activeViewId).First(&sharerParticipant).Error
	if err != nil {
		return errors.New("failed to find screen sharer participant record")
	}
	// Check if screen sharer is currently sharing his/her screen
	if !sharerParticipant.IsSharingScreen {
		logger.Error("active view user is not currently sharing their screen")
		return errors.New("screen sharer is not actively sharing")
	}

	return nil
}

func UpdateState(db *storage.Database, logger *utility.Logger, huddleId string, userId string, activeViewId string) error {
	// Find the record of the screen viewer
	var viewerParticipant models.HuddleParticipant
	err := db.Postgresql.Where("huddle_id = ? AND user_id = ?", huddleId, userId).First(&viewerParticipant).Error
	if err != nil {
		return errors.New("failed to find screen sharer participant record")
	}
	err = db.Postgresql.Model(&viewerParticipant).Update("ActiveViewID", &activeViewId).Error
	if err != nil {
		logger.Error("failed to update viewer state: %v", err)
		return errors.New("database error updating viewer state")
	}

	return nil
}
