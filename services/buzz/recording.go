package buzz

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/permissions"
	"github.com/hngprojects/telex_be/pkg/repository/agora"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

const recordingUID = "1000"

func generateRecorderToken(buzzID string) (string, error) {
	svc := agora.Client.Service
	if svc == nil {
		return "", errors.New("agora service not initialized")
	}
	return svc.GenerateRTCToken(buzzID, recordingUID, recordingUID, agora.DefaultTokenExpirationSeconds)
}

func getActiveBuzzRecording(db *gorm.DB, buzzID string) (*models.BuzzRecording, error) {
	var rec models.BuzzRecording
	err := db.Where("buzz_id = ? AND status NOT IN (?, ?)", buzzID,
		models.RecordingStatusStopped, models.RecordingStatusFailed).
		First(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &rec, err
}

func getBuzzOrgID(buzz *models.Buzz) string {
	if buzz.OrgID != nil {
		return *buzz.OrgID
	}
	return ""
}

func StartBuzzRecording(db *storage.Database, logger *utility.Logger, buzzID, hostID string) (*models.BuzzRecording, int, error) {
	buzz, err := permissions.CanPerformHostAction(db.Postgresql, buzzID, hostID)
	if err != nil {
		logger.Error("[Agora] failed to perform host action for buzz %s: %v", buzzID, err)
		statusCode, errMsg := mapPermissionError(err, "start recording")
		return nil, statusCode, errors.New(errMsg)
	}

	existing, err := getActiveBuzzRecording(db.Postgresql, buzzID)
	if err != nil {
		logger.Error("[Agora] failed to check recording status for buzz %s: %v", buzzID, err)
		return nil, http.StatusInternalServerError, errors.New("failed to check recording status")
	}
	if existing != nil {
		return nil, http.StatusConflict, errors.New("recording already in progress")
	}

	orgID := getBuzzOrgID(buzz)
	resourceID, err := agora.AcquireRecording(logger, buzzID, recordingUID)
	if err != nil {
		logger.Error("[Agora] failed to acquire recording resource for buzz %s: %v", buzzID, err)
		return nil, http.StatusInternalServerError, errors.New("failed to acquire recording resource")
	}

	recorderToken, err := generateRecorderToken(buzzID)
	if err != nil {
		logger.Error("failed to generate recorder token for buzz %s: %v", buzzID, err)
		return nil, http.StatusInternalServerError, errors.New("failed to generate recorder token")
	}

	durationSecs := 300

	sid, err := agora.StartRecording(logger, resourceID, buzzID, recordingUID, recorderToken, durationSecs)
	if err != nil {
		logger.Error("[Agora] failed to start recording for buzz %s: %v", buzzID, err)
		return nil, http.StatusInternalServerError, errors.New("failed to start recording")
	}

	rec := &models.BuzzRecording{
		ID:            utility.GenerateUUID(),
		BuzzID:        buzzID,
		OrgID:         orgID,
		ResourceID:    resourceID,
		Sid:           sid,
		RecorderToken: recorderToken,
		Status:        models.RecordingStatusStarting,
		StartedAt:     time.Now().UTC(),
	}

	if err := db.Postgresql.Create(rec).Error; err != nil {
		logger.Error("failed to save recording record: %v", err)
		return nil, http.StatusInternalServerError, errors.New("failed to save recording")
	}

	publishRecordingEvent(logger, buzz, rec, "recording_started")
	logger.Info("[Agora] Recording started for buzz %s, sid: %s", buzzID, sid)
	return rec, http.StatusOK, nil
}

func StopBuzzRecording(db *storage.Database, logger *utility.Logger, buzzID, hostID string) (*models.BuzzRecording, int, error) {
	buzz, err := permissions.CanPerformHostAction(db.Postgresql, buzzID, hostID)
	if err != nil {
		statusCode, errMsg := mapPermissionError(err, "stop recording")
		return nil, statusCode, errors.New(errMsg)
	}

	rec, err := getActiveBuzzRecording(db.Postgresql, buzzID)
	if err != nil {
		return nil, http.StatusInternalServerError, errors.New("failed to check recording status")
	}
	if rec == nil {
		return nil, http.StatusNotFound, errors.New("no active recording found for this buzz")
	}

	if rec.Status == models.RecordingStatusStopped || rec.Status == models.RecordingStatusFailed {
		logger.Info("recording for buzz %s already stopped (status: %s), skipping agora stop call", buzzID, rec.Status)
		return rec, http.StatusOK, nil
	}

	m3u8Key, stopErr := agora.StopRecording(rec.ResourceID, rec.Sid, buzzID, recordingUID, rec.RecorderToken)
	if stopErr != nil {
		logger.Error("failed to stop agora recording for buzz %s: %v", buzzID, stopErr)
	}

	now := time.Now().UTC()
	rec.Status = models.RecordingStatusStopped
	rec.EndedAt = &now
	rec.DurationSec = int(now.Sub(rec.StartedAt).Seconds())

	if err := db.Postgresql.Save(rec).Error; err != nil {
		return nil, http.StatusInternalServerError, errors.New("failed to update recording status")
	}

	if m3u8Key != "" {
		go func(key string) {
			mp4Key, mergeErr := agora.MergeAndUploadRecording(context.Background(), logger, key)
			if mergeErr != nil {
				logger.Error("failed to merge recording segments for buzz %s: %v", buzzID, mergeErr)
				return
			}
			rec.FileURL = buildRecordingFileURL(mp4Key)
			if err := db.Postgresql.Save(rec).Error; err != nil {
				logger.Error("failed to persist mp4 url for buzz %s: %v", buzzID, err)
				return
			}
			saveRecordingAsOrgFile(db.Postgresql, logger, rec, buzz)
		}(m3u8Key)
	}

	publishRecordingEvent(logger, buzz, rec, "recording_stopped")
	logger.Info("recording stopped for buzz %s, duration: %ds", buzzID, rec.DurationSec)
	return rec, http.StatusOK, nil
}

func CheckRecordingStatus(db *storage.Database, logger *utility.Logger, buzzID, userID string) (*models.BuzzRecording, int, error) {
	var buzz models.Buzz
	if err := db.Postgresql.Where("id = ?", buzzID).First(&buzz).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, http.StatusNotFound, errors.New("buzz not found")
		}
		return nil, http.StatusInternalServerError, errors.New("failed to fetch buzz")
	}

	isParticipant := false
	for _, pid := range buzz.ParticipantIDs {
		if pid == userID {
			isParticipant = true
			break
		}
	}
	if !isParticipant {
		return nil, http.StatusForbidden, errors.New("user is not a participant in this buzz")
	}

	rec, err := getActiveBuzzRecording(db.Postgresql, buzzID)
	if err != nil {
		return nil, http.StatusInternalServerError, errors.New("failed to fetch recording status")
	}
	if rec == nil {
		return nil, http.StatusNotFound, errors.New("no active recording found for this buzz")
	}

	statusStr, files, err := agora.QueryRecordingStatus(logger, rec.ResourceID, rec.Sid, buzzID)
	if err != nil {
		logger.Error("failed to query agora recording status: %v", err)
		return rec, http.StatusOK, nil
	}

	rec.Status = statusStr
	if len(files) > 0 && rec.FileURL == "" {
		rec.FileURL = buildRecordingFileURL(files[0])
	}

	if err := db.Postgresql.Save(rec).Error; err != nil {
		logger.Error("failed to update recording status: %v", err)
	}

	return rec, http.StatusOK, nil
}

func saveRecordingAsOrgFile(db *gorm.DB, logger *utility.Logger, rec *models.BuzzRecording, buzz *models.Buzz) error {
	if rec.FileURL == "" || rec.FileID != nil {
		return nil
	}

	logger.Info("[Agora-Recording] Saving recording file for buzz %s ...", rec.BuzzID)

	buzzCode := utility.ExtractBuzzCode(buzz.ID)
	file := &models.File{
		ID:             utility.GenerateUUID(),
		FileName:       fmt.Sprintf("buzz-recording-%s.mp4", buzzCode),
		FileType:       "video",
		MimeType:       "video/mp4",
		FileLink:       rec.FileURL,
		OrganisationID: rec.OrgID,
		UserID:         buzz.HostID,
	}

	logger.Info("[Agora-Recording] Started saving recording file for buzz %s", rec.BuzzID)

	if err := db.Create(file).Error; err != nil {
		logger.Error("[Agora-Recording] Failed to create file record for buzz %s: %v", rec.BuzzID, err)
		return fmt.Errorf("failed to create file record: %w", err)
	}

	logger.Info("[Agora] Call recording created for buzz %s, file id: %s", buzzCode, file.ID)

	rec.FileID = &file.ID
	if err := db.Model(rec).Update("file_id", file.ID).Error; err != nil {
		logger.Error("[Agora-Recording] Failed to link file record for buzz %s: %v", rec.BuzzID, err)
		return err
	}

	logger.Info("[Agora-Recording] Saving complete for buzz %s — file: %s", rec.BuzzID, file.FileName)
	return nil
}

func buildRecordingFileURL(filename string) string {
	cfg := config.GetConfig()
	return fmt.Sprintf("%s/%s/%s", cfg.Minio.MinioEndpoint, cfg.Minio.BucketName, filename)
}

func publishRecordingEvent(logger *utility.Logger, buzz *models.Buzz, rec *models.BuzzRecording, eventName string) {
	payload := map[string]interface{}{
		"event":            eventName,
		"buzz_id":          buzz.ID,
		"channel_id":       buzz.ChannelID,
		"host_id":          buzz.HostID,
		"recording_id":     rec.ID,
		"resource_id":      rec.ResourceID,
		"sid":              rec.Sid,
		"recording_status": rec.Status,
		"is_recording":     rec.Status == models.RecordingStatusRecording || rec.Status == models.RecordingStatusStarting,
		"started_at":       rec.StartedAt,
	}

	publishChannel := getPublishChannel(buzz)
	if err := centrifuge.PublishChannel(logger, publishChannel, payload); err != nil {
		logger.Error("[Agora] Failed to publish %s event for buzz %s: %v", eventName, buzz.ID, err)
	}
	logger.Info("[Agora] %s event for buzz %s, payload: %v", eventName, buzz.ID, payload)
}

func fetchBuzzRecordingStatus(db *gorm.DB, buzzID string) (string, bool) {
	rec, err := getActiveBuzzRecording(db, buzzID)
	if err != nil || rec == nil {
		return models.RecordingStatusIdle, false
	}
	isRecording := rec.Status == models.RecordingStatusRecording || rec.Status == models.RecordingStatusStarting

	return rec.Status, isRecording
}

func StopActiveRecordingForBuzz(db *storage.Database, logger *utility.Logger, buzzID string, buzz *models.Buzz) {
	rec, err := getActiveBuzzRecording(db.Postgresql, buzzID)
	if err != nil || rec == nil {
		return
	}

	if rec.Status == models.RecordingStatusStopped || rec.Status == models.RecordingStatusFailed {
		logger.Info("recording for buzz %s already stopped (status: %s), skipping agora stop call", buzzID, rec.Status)
		return
	}

	m3u8Key, stopErr := agora.StopRecording(rec.ResourceID, rec.Sid, buzzID, recordingUID, rec.RecorderToken)
	if stopErr != nil {
		logger.Error("failed to stop agora recording on buzz end for buzz %s: %v", buzzID, stopErr)
	}

	logger.Info("[Agora] Stopped recording for buzz %s", buzzID)

	now := time.Now().UTC()
	rec.Status = models.RecordingStatusStopped
	rec.EndedAt = &now
	rec.DurationSec = int(now.Sub(rec.StartedAt).Seconds())

	if err := db.Postgresql.Save(rec).Error; err != nil {
		logger.Error("failed to update recording status on buzz end: %v", err)
		return
	}

	if m3u8Key != "" {
		go func(key string) {
			mp4Key, mergeErr := agora.MergeAndUploadRecording(context.Background(), logger, key)
			if mergeErr != nil {
				logger.Error("failed to merge recording segments on buzz end for buzz %s: %v", buzzID, mergeErr)
				return
			}
			rec.FileURL = buildRecordingFileURL(mp4Key)
			if err := db.Postgresql.Save(rec).Error; err != nil {
				logger.Error("failed to persist mp4 url on buzz end for buzz %s: %v", buzzID, err)
				return
			}
			saveRecordingAsOrgFile(db.Postgresql, logger, rec, buzz)
		}(m3u8Key)
	}

	logger.Info("[Agora] Saved recording as org file for buzz %s", buzzID)

	publishRecordingEvent(logger, buzz, rec, "recording_stopped")
	logger.Info("[Agora] Published recording_stopped event for buzz %s", buzzID)
}
