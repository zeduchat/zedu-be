package buzz

import (
	"errors"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func CreateBuzzNote(db *storage.Database, logger *utility.Logger, buzzID string, req models.CreateBuzzNoteRequest, userID string) (models.BuzzNoteResponse, int, error) {
	var resp models.BuzzNoteResponse

	var buzz models.Buzz
	err := db.Postgresql.Where("id = ?", buzzID).First(&buzz).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp, http.StatusNotFound, errors.New("buzz not found")
		}
		logger.Error("failed to fetch buzz: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to fetch buzz")
	}

	var participant models.BuzzParticipant
	err = db.Postgresql.Where("buzz_id = ? AND user_id = ?", buzzID, userID).First(&participant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp, http.StatusForbidden, errors.New("only participants can create notes")
		}
		logger.Error("failed to verify participant: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to verify participant")
	}

	if participant.Status != models.BuzzParticipantStatusActive {
		return resp, http.StatusForbidden, errors.New("only active participants can create notes")
	}

	now := time.Now().UTC()
	note := models.BuzzNote{
		ID:        utility.GenerateUUID(),
		BuzzID:    buzzID,
		UserID:    userID,
		Note:      req.Note,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err = db.Postgresql.Create(&note).Error
	if err != nil {
		logger.Error("failed to create buzz note: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to create buzz note")
	}

	resp = models.BuzzNoteResponse(note)

	logger.Info("buzz note created successfully for buzz %s by user %s", buzzID, userID)
	return resp, http.StatusCreated, nil
}

func GetBuzzNotes(db *storage.Database, logger *utility.Logger, buzzID string, userID string) (models.BuzzNotesListResponse, int, error) {
	var resp models.BuzzNotesListResponse

	var buzz models.Buzz
	err := db.Postgresql.Where("id = ?", buzzID).First(&buzz).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp, http.StatusNotFound, errors.New("buzz not found")
		}
		logger.Error("failed to fetch buzz: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to fetch buzz")
	}

	var participant models.BuzzParticipant
	err = db.Postgresql.Where("buzz_id = ? AND user_id = ?", buzzID, userID).First(&participant).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp, http.StatusForbidden, errors.New("only participants can read notes")
		}
		logger.Error("failed to verify participant: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to verify participant")
	}

	var notes []models.BuzzNote
	err = db.Postgresql.Where("buzz_id = ? AND user_id = ?", buzzID, userID).Order("created_at ASC").Find(&notes).Error
	if err != nil {
		logger.Error("failed to fetce buzz notes: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to fetce buzz notes")
	}

	noteResponses := make([]models.BuzzNoteResponse, 0, len(notes))
	for _, note := range notes {
		noteResponses = append(noteResponses, models.BuzzNoteResponse(note))
	}

	resp.Notes = noteResponses
	logger.Info("retrieved %d notes for buzz %s", len(noteResponses), buzzID)
	return resp, http.StatusOK, nil
}

func UpdateBuzzNote(db *storage.Database, logger *utility.Logger, buzzID, noteID string, req models.UpdateBuzzNoteRequest, userID string) (models.BuzzNoteResponse, int, error) {
	var resp models.BuzzNoteResponse

	var buzz models.Buzz
	err := db.Postgresql.Where("id = ?", buzzID).First(&buzz).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resp, http.StatusNotFound, errors.New("buzz not found")
		}
		logger.Error("failed to fetch buzz: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to fetch buzz")
	}

	var note models.BuzzNote
	err = db.Postgresql.Where("id = ? AND buzz_id = ?", noteID, buzzID).First(&note).Error
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

	var participant models.BuzzParticipant
	err = db.Postgresql.Where("buzz_id = ? AND user_id = ?", buzzID, userID).First(&participant).Error
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
		logger.Error("failed to update buzz note: %v", err)
		return resp, http.StatusInternalServerError, errors.New("failed to update buzz note")
	}

	resp = models.BuzzNoteResponse{
		ID:        note.ID,
		BuzzID:    note.BuzzID,
		UserID:    note.UserID,
		Note:      req.Note,
		CreatedAt: note.CreatedAt,
		UpdatedAt: now,
	}

	logger.Info("buzz note %s updated successfully for buzz %s by user %s", noteID, buzzID, userID)
	return resp, http.StatusOK, nil
}
