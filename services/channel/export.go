package channel

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

func InitiateChannelExport(db *storage.Database, channelID, userID, orgID string, logger *utility.Logger) (*models.ChannelExport, int, error) {
	var channelModel models.Channels
	if !channelModel.CheckChannelExistsInOrg(db.Postgresql, channelID, orgID) {
		return nil, http.StatusNotFound, errors.New("channel does not exist in organisation")
	}

	var userChannel models.UserChannels
	if err := userChannel.UserInChannels(db.Postgresql, channelID, userID); err != nil {
		return nil, http.StatusForbidden, errors.New("user is not a member of this channel")
	}

	var exportModel models.ChannelExport

	// Deduplication: check if export is currently pending or in progress
	activeExport, err := exportModel.GetActiveExport(db.Postgresql, channelID, userID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, http.StatusInternalServerError, fmt.Errorf("error checking active export: %w", err)
	}

	if activeExport != nil {
		// Active export already exists and processing, return existing record
		return activeExport, http.StatusOK, nil
	}

	// Create new pending export entry
	newExport := models.ChannelExport{
		ID:             utility.GenerateUUID(),
		ChannelID:      channelID,
		UserID:         userID,
		OrganisationID: orgID,
		Status:         models.ExportStatusPending,
	}

	if err := newExport.CreateExport(db.Postgresql); err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to create export record: %w", err)
	}

	// Enqueue River Queue job
	if storage.DB != nil && storage.DB.River != nil {
		_, err := storage.DB.River.Insert(context.Background(), models.ChannelExportJobArgs{
			ExportID:       newExport.ID,
			ChannelID:      channelID,
			UserID:         userID,
			OrganisationID: orgID,
		}, nil)

		if err != nil {
			logger.Error("Failed to enqueue channel export job: %v", err)
			errMsg := fmt.Sprintf("Failed to enqueue export job: %v", err)
			_ = newExport.UpdateStatus(db.Postgresql, models.ExportStatusFailed, nil, nil, &errMsg)
			return nil, http.StatusInternalServerError, fmt.Errorf("failed to enqueue export job: %w", err)
		}
	} else {
		logger.Warning("River queue client not available, export job created but not queued")
	}

	return &newExport, http.StatusAccepted, nil
}

func GetLatestChannelExportStatus(db *storage.Database, channelID, userID string) (*models.ChannelExport, int, error) {
	var exportModel models.ChannelExport
	export, err := exportModel.GetLatestExport(db.Postgresql, channelID, userID)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	if export == nil {
		return nil, http.StatusNotFound, errors.New("no export history found for this channel")
	}
	return export, http.StatusOK, nil
}

func GetChannelExportHistory(db *storage.Database, channelID, userID string, page, limit int) ([]models.ChannelExport, postgresql.PaginationResponse, int, error) {
	var exportModel models.ChannelExport
	exports, pagination, err := exportModel.GetExportHistory(db.Postgresql, channelID, userID, page, limit)
	if err != nil {
		return nil, postgresql.PaginationResponse{}, http.StatusInternalServerError, err
	}
	return exports, pagination, http.StatusOK, nil
}
