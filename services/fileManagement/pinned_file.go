package fileManagement

import (
	"encoding/json"
	"fmt"
	"time"

	go_redis "github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/redis"
	"github.com/hngprojects/telex_be/utility"
)

func PinFile(db *gorm.DB, logger *utility.Logger, rdb *go_redis.Client, pinnedFile *models.PinnedFile) (*models.PinnedFileResponse, error) {
	if err := pinnedFile.PinFile(db); err != nil {
		return nil, err
	}
	key := getPinnedFilesKey(pinnedFile.OrganisationID, pinnedFile.UserID)
	redis.RedisDelete(rdb, key)

	var file models.File
	if err := db.Where("id = ?", pinnedFile.FileID).First(&file).Error; err != nil {
		return nil, err
	}

	return &models.PinnedFileResponse{
		PinnedFileID: pinnedFile.ID,
		File:         file,
	}, nil
}

func UnpinFile(db *gorm.DB, logger *utility.Logger, rdb *go_redis.Client, pinnedFile *models.PinnedFile) error {

	key := getPinnedFilesKey(pinnedFile.OrganisationID, pinnedFile.UserID)
	logger.Info(fmt.Sprintf("UnpinFile: Invalidating Key: %s", key))
	redis.RedisDelete(rdb, key)

	if err := pinnedFile.UnpinFile(db); err != nil {
		logger.Error(fmt.Sprintf("UnpinFile: DB Delete Error: %v", err))
		return err
	}
	return nil
}

func GetPinnedFiles(db *gorm.DB, logger *utility.Logger, rdb *go_redis.Client, userID, orgID string) ([]models.PinnedFileResponse, error) {
	key := getPinnedFilesKey(orgID, userID)

	if data, err := redis.RedisGet(rdb, key); err == nil && data != nil {
		var files []models.PinnedFileResponse
		if json.Unmarshal(data, &files) == nil {
			logger.Info("GetPinnedFiles: Cache HIT")
			return files, nil
		}
	}
	logger.Info("GetPinnedFiles: Cache MISS")

	response := make([]models.PinnedFileResponse, 0)

	err := db.Table("files").
		Select("files.*, pinned_files.id as pinned_file_id").
		Joins("JOIN pinned_files ON pinned_files.file_id = files.id").
		Where("pinned_files.organisation_id = ? AND pinned_files.user_id = ?", orgID, userID).
		Scan(&response).Error

	if err != nil {
		return nil, err
	}

	// set cache with an hour TTL
	redis.RedisSet(rdb, key, response, time.Hour)
	return response, nil
}

func getPinnedFilesKey(orgID, userID string) string {
	return fmt.Sprintf("pinned_files:%s:%s", orgID, userID)
}
