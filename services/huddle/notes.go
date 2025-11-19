package huddle

import (
	"errors"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func CreateHuddleNote(db *storage.Database, logger *utility.Logger, huddleID string, req models.CreateHuddleNoteRequest, userID string) (models.HuddleNoteResponse, int, error) {
	var resp models.HuddleNoteResponse

	var huddle models.Huddle
	err := db.Postgresql.Where("id = ?", huddleID).First(&huddle).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp, http.StatusNotFound, errors.New("huddle not found")
		}
		logger.Error("failed to fetch huddle: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to fetch huddle")
	}

	var participant models.HuddleParticipant
	err = db.Postgresql.Where("huddle_id = ? AND user_id = ?", huddleID, userID).First(&participant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp, http.StatusForbidden, errors.New("only participants can create notes")
		}
		logger.Error("failed to verify participant: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to verify participant")
	}

	if participant.Status != models.HuddleParticipantStatusActive {
		return resp, http.StatusForbidden, errors.New("only active participants can create notes")
	}

	now := time.Now().UTC()
	note := models.HuddleNote{
		ID:        utility.GenerateUUID(),
		HuddleID:  huddleID,
		UserID:    userID,
		Note:      req.Note,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err = db.Postgresql.Create(&note).Error
	if err != nil {
		logger.Error("failed to create huddle note: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to create huddle note")
	}

	resp = models.HuddleNoteResponse(note)

	logger.Info("huddle note created successfully for huddle %s by user %s", huddleID, userID)
	return resp, http.StatusCreated, nil
}

func GetHuddleNotes(db *storage.Database, logger *utility.Logger, huddleID string, userID string) (models.HuddleNotesListResponse, int, error) {
	var resp models.HuddleNotesListResponse

	var huddle models.Huddle
	err := db.Postgresql.Where("id = ?", huddleID).First(&huddle).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp, http.StatusNotFound, errors.New("huddle not found")
		}
		logger.Error("failed to fetch huddle: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to fetch huddle")
	}

	var participant models.HuddleParticipant
	err = db.Postgresql.Where("huddle_id = ? AND user_id = ?", huddleID, userID).First(&participant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp, http.StatusForbidden, errors.New("only participants can read notes")
		}
		logger.Error("failed to verify participant: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to verify participant")
	}

	var notes []models.HuddleNote
	err = db.Postgresql.Where("huddle_id = ?", huddleID).Order("created_at ASC").Find(&notes).Error
	if err != nil {
		logger.Error("failed to fetch huddle notes: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to fetch huddle notes")
	}

	noteResponses := make([]models.HuddleNoteResponse, 0, len(notes))
	for _, note := range notes {
		noteResponses = append(noteResponses, models.HuddleNoteResponse(note))
	}

	resp.Notes = noteResponses
	logger.Info("retrieved %d notes for huddle %s", len(noteResponses), huddleID)
	return resp, http.StatusOK, nil
}

func UpdateHuddleNote(db *storage.Database, logger *utility.Logger, huddleID, noteID string, req models.UpdateHuddleNoteRequest, userID string) (models.HuddleNoteResponse, int, error) {
	var resp models.HuddleNoteResponse

	var huddle models.Huddle
	err := db.Postgresql.Where("id = ?", huddleID).First(&huddle).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp, http.StatusNotFound, errors.New("huddle not found")
		}
		logger.Error("failed to fetch huddle: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to fetch huddle")
	}

	var note models.HuddleNote
	err = db.Postgresql.Where("id = ? AND huddle_id = ?", noteID, huddleID).First(&note).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp, http.StatusNotFound, errors.New("note not found")
		}
		logger.Error("failed to fetch note: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to fetch note")
	}

	if note.UserID != userID {
		return resp, http.StatusForbidden, errors.New("you can only edit your own notes")
	}

	var participant models.HuddleParticipant
	err = db.Postgresql.Where("huddle_id = ? AND user_id = ?", huddleID, userID).First(&participant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp, http.StatusForbidden, errors.New("only participants can edit notes")
		}
		logger.Error("failed to verify participant: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to verify participant")
	}

	now := time.Now().UTC()
	err = db.Postgresql.Model(&note).Updates(map[string]interface{}{
		"note":       req.Note,
		"updated_at": now,
	}).Error
	if err != nil {
		logger.Error("failed to update huddle note: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to update huddle note")
	}

	resp = models.HuddleNoteResponse{
		ID:        note.ID,
		HuddleID:  note.HuddleID,
		UserID:    note.UserID,
		Note:      req.Note,
		CreatedAt: note.CreatedAt,
		UpdatedAt: now,
	}

	logger.Info("huddle note %s updated successfully for huddle %s by user %s", noteID, huddleID, userID)
	return resp, http.StatusOK, nil
}
