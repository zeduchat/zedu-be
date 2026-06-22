package buzz

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/gosimple/slug"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/permissions"
	"github.com/hngprojects/telex_be/pkg/repository/agora"
	"github.com/hngprojects/telex_be/pkg/repository/centrifuge"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func generateRecordingUID() string {
	return fmt.Sprintf("%d", 100000+rand.Intn(900000000))
}

func getSiteURL() string {
	conf := config.GetConfig()

	if conf.App.FRONTEND_URL != "" {
		return conf.App.FRONTEND_URL
	}

	return "http://localhost:3000"
}

func isRecordingBot(db *gorm.DB, userID string, buzzID string) bool {
	if len(userID) > 37 && userID[36] == '-' {
		extractedBuzzID := userID[:36]
		recordingUID := userID[37:]
		if extractedBuzzID == buzzID {
			var rec models.BuzzRecording
			if err := db.Where("buzz_id = ? AND recording_uid = ?", buzzID, recordingUID).First(&rec).Error; err == nil {
				return true
			}
		}
	}
	return false
}

func GenerateBotJWTToken(orgID string, buzzID string, recordingUID string, durationSecs uint32) (string, error) {
	cfg := config.GetConfig()
	botUserID := fmt.Sprintf("%s-%s", buzzID, recordingUID)
	userClaims := jwt.MapClaims{}
	userClaims["user_id"] = botUserID
	userClaims["access_uuid"] = utility.GenerateUUID()
	userClaims["role_id"] = ""
	userClaims["org_id"] = orgID
	userClaims["exp"] = time.Now().Add(time.Duration(durationSecs) * time.Second).Unix()
	userClaims["authorised"] = true

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, userClaims)
	return token.SignedString([]byte(cfg.Server.Secret))
}

func generateRecorderToken(buzzID string, recordingUID string, durationSecs uint32) (string, error) {
	svc := agora.Client.Service
	if svc == nil {
		return "", errors.New("agora service not initialized")
	}

	return svc.GenerateRTCToken(buzzID, recordingUID, recordingUID, durationSecs)
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

	remainingTime := buzz.GetRemainingTime(agora.DefaultTokenExpirationSeconds)
	if buzz.BuzzType == models.BuzzTypeOrganization {
		remainingTime = agora.DefaultTokenExpirationSeconds
	} else if remainingTime == 0 {
		return nil, http.StatusBadRequest, errors.New("buzz has expired")
	}

	orgID := getBuzzOrgID(buzz)
	recordingUID := generateRecordingUID()

	botToken, err := GenerateBotJWTToken(orgID, buzzID, recordingUID, remainingTime)
	if err != nil {
		logger.Error("failed to generate bot JWT token: %v", err)
		return nil, http.StatusInternalServerError, errors.New("failed to generate bot token")
	}

	resourceID, err := agora.AcquireRecording(logger, buzzID, recordingUID)
	if err != nil {
		logger.Error("[Agora] failed to acquire recording resource for buzz %s: %v", buzzID, err)
		return nil, http.StatusInternalServerError, errors.New("failed to acquire recording resource")
	}

	webpageURL := buildWebpageURL(db.Postgresql, orgID, buzz, botToken)

	maxIdleSecs := 300
	sid, err := agora.StartRecording(logger, resourceID, buzzID, webpageURL, recordingUID, maxIdleSecs)
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
		RecorderToken: botToken,
		RecordingUID:  recordingUID,
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

	files, stopErr := agora.StopRecording(rec.ResourceID, rec.Sid, buzzID, rec.RecordingUID)
	if stopErr != nil {
		logger.Error("failed to stop agora recording for buzz %s: %v", buzzID, stopErr)
	}

	now := time.Now().UTC()
	rec.Status = models.RecordingStatusStopped
	rec.EndedAt = &now
	rec.DurationSec = int(now.Sub(rec.StartedAt).Seconds())

	mp4File := resolveMP4File(files)
	if mp4File != "" {
		rec.FileURL = buildRecordingFileURL(mp4File)
	}

	if err := db.Postgresql.Save(rec).Error; err != nil {
		return nil, http.StatusInternalServerError, errors.New("failed to update recording status")
	}

	if mp4File != "" {
		_ = saveRecordingAsOrgFile(db.Postgresql, logger, rec, buzz, 0)
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

	isBot := isRecordingBot(db.Postgresql, userID, buzzID)
	isParticipant := isBot
	if !isBot {
		for _, pid := range buzz.ParticipantIDs {
			if pid == userID {
				isParticipant = true
				break
			}
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
	mp4File := resolveMP4File(files)
	if mp4File != "" && rec.FileURL == "" {
		rec.FileURL = buildRecordingFileURL(mp4File)
	}

	if err := db.Postgresql.Save(rec).Error; err != nil {
		logger.Error("failed to update recording status: %v", err)
	}

	if mp4File != "" {
		_ = saveRecordingAsOrgFile(db.Postgresql, logger, rec, &buzz, 0)
	}

	return rec, http.StatusOK, nil
}

func saveRecordingAsOrgFile(db *gorm.DB, logger *utility.Logger, rec *models.BuzzRecording, buzz *models.Buzz, fileSize int64) error {
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
		Size:           fileSize,
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
	return fmt.Sprintf("https://%s/%s/%s", cfg.Minio.MinioEndpoint, cfg.Minio.BucketName, filename)
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

	files, stopErr := agora.StopRecording(rec.ResourceID, rec.Sid, buzzID, rec.RecordingUID)
	if stopErr != nil {
		logger.Error("failed to stop agora recording on buzz end for buzz %s: %v", buzzID, stopErr)
	}

	logger.Info("[Agora] Stopped recording for buzz %s", buzzID)

	now := time.Now().UTC()
	rec.Status = models.RecordingStatusStopped
	rec.EndedAt = &now
	rec.DurationSec = int(now.Sub(rec.StartedAt).Seconds())

	mp4File := resolveMP4File(files)
	if mp4File != "" {
		rec.FileURL = buildRecordingFileURL(mp4File)
	}

	if err := db.Postgresql.Save(rec).Error; err != nil {
		logger.Error("failed to update recording status on buzz end: %v", err)
		return
	}

	if mp4File != "" {
		_ = saveRecordingAsOrgFile(db.Postgresql, logger, rec, buzz, 0)
	}

	logger.Info("[Agora] Saved recording as org file for buzz %s", buzzID)

	publishRecordingEvent(logger, buzz, rec, "recording_stopped")
	logger.Info("[Agora] Published recording_stopped event for buzz %s", buzzID)
}

func resolveMP4File(files []string) string {

	for _, f := range files {
		if strings.HasSuffix(f, ".mp4") {
			return f
		}
	}

	if len(files) > 0 {
		return files[0]
	}

	return ""
}

func buildWebpageURL(db *gorm.DB, orgID string, buzz *models.Buzz, botToken string) string {
	var orgSlug string

	if orgID != "" {
		var org models.Organisation
		if err := db.Where("id = ?", orgID).First(&org).Error; err == nil {
			orgSlug = slug.Make(org.Name)
		}
	}

	if orgSlug == "" {
		orgSlug = "org"
	}

	siteURL := getSiteURL()
	buzzCode := utility.ExtractBuzzCode(buzz.ID)

	return fmt.Sprintf("%s/%s/buzz-record/%s?token=%s&orgId=%s&mode=recorder",
		siteURL, orgSlug, buzzCode, botToken, orgID)
}
