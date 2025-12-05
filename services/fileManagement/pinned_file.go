package fileManagement

import (
	"encoding/json"
	"fmt"
	"time"

	go_redis "github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage/redis"
)

func PinFile(db *gorm.DB, rdb *go_redis.Client, pinnedFile *models.PinnedFile) error {
	if err := pinnedFile.PinFile(db); err != nil {
		return err
	}

	key := fmt.Sprintf("pinned_files:%s:%s", pinnedFile.OrganisationID, pinnedFile.UserID)
	redis.RedisDelete(rdb, key)
	return nil
}

func UnpinFile(db *gorm.DB, rdb *go_redis.Client, pinnedFile *models.PinnedFile) error {
	if err := pinnedFile.UnpinFile(db); err != nil {
		return err
	}

	key := fmt.Sprintf("pinned_files:%s:%s", pinnedFile.OrganisationID, pinnedFile.UserID)
	redis.RedisDelete(rdb, key)
	return nil
}

func GetPinnedFiles(db *gorm.DB, rdb *go_redis.Client, userID, orgID string) ([]models.File, error) {
	key := fmt.Sprintf("pinned_files:%s:%s", orgID, userID)

	if data, err := redis.RedisGet(rdb, key); err == nil && data != nil {
		var files []models.File
		if json.Unmarshal(data, &files) == nil {
			return files, nil
		}
	}

	var files []models.File
	err := db.Table("files").
		Joins("JOIN pinned_files ON pinned_files.file_id = files.id").
		Where("pinned_files.organisation_id = ? AND pinned_files.user_id = ?", orgID, userID).
		Find(&files).Error

	if err != nil {
		return nil, err
	}

	// set cache with an hour TTL
	redis.RedisSet(rdb, key, files, time.Hour)
	return files, nil
}
