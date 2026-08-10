package models

import (
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/utility"
)

type UserThreadRead struct {
	ID         string    `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID     string    `gorm:"type:uuid;index;not null" json:"user_id"`
	ThreadID   string    `gorm:"type:uuid;index;not null" json:"thread_id"`
	OrgID      string    `gorm:"type:uuid;index;not null" json:"org_id"`
	HasUnseen  bool      `gorm:"default:true;not null" json:"has_unseen"`
	LastReadAt time.Time `gorm:"autoCreateTime" json:"last_read_at"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (u *UserThreadRead) TableName() string {
	return "user_thread_reads"
}

func MarkThreadUnseenForParticipants(db *gorm.DB, orgID, threadID string, participantIDs []string) ([]string, error) {
	if len(participantIDs) == 0 {
		return nil, nil
	}

	var newlyUnseenUserIDs []string

	for _, pID := range participantIDs {
		if pID == "" || !utility.IsValidUUID(pID) || pID == "00000000-0000-0000-0000-000000000000" {
			continue
		}

		var record UserThreadRead
		err := db.Where("user_id = ? AND thread_id = ?", pID, threadID).First(&record).Error

		if err != nil && err == gorm.ErrRecordNotFound {
			newRecord := UserThreadRead{
				ID:         utility.GenerateUUID(),
				UserID:     pID,
				ThreadID:   threadID,
				OrgID:      orgID,
				HasUnseen:  true,
				LastReadAt: time.Now().UTC(),
				UpdatedAt:  time.Now().UTC(),
			}
			if createErr := db.Create(&newRecord).Error; createErr == nil {
				newlyUnseenUserIDs = append(newlyUnseenUserIDs, pID)
			}
		} else if err == nil {
			if !record.HasUnseen {
				db.Model(&UserThreadRead{}).Where("id = ?", record.ID).Updates(map[string]interface{}{
					"has_unseen": true,
					"updated_at": time.Now().UTC(),
				})
				newlyUnseenUserIDs = append(newlyUnseenUserIDs, pID)
			}
		}
	}

	return newlyUnseenUserIDs, nil
}

func MarkThreadSeenForUser(db *gorm.DB, userID, threadID string) (bool, error) {
	if userID == "" || threadID == "" || !utility.IsValidUUID(userID) || !utility.IsValidUUID(threadID) {
		return false, nil
	}

	var record UserThreadRead
	err := db.Where("user_id = ? AND thread_id = ?", userID, threadID).First(&record).Error

	if err == nil && record.HasUnseen {
		errUpdate := db.Model(&UserThreadRead{}).Where("id = ?", record.ID).Updates(map[string]interface{}{
			"has_unseen":   false,
			"last_read_at": time.Now().UTC(),
			"updated_at":   time.Now().UTC(),
		}).Error
		if errUpdate == nil {
			return true, nil
		}
	}

	return false, nil
}

func GetUnseenThreadCountForUser(db *gorm.DB, userID, orgID string) (int64, error) {
	if !utility.IsValidUUID(userID) || !utility.IsValidUUID(orgID) {
		return 0, nil
	}
	var count int64
	err := db.Model(&UserThreadRead{}).
		Where("user_id = ? AND org_id = ? AND has_unseen = ?", userID, orgID, true).
		Count(&count).Error
	return count, err
}
