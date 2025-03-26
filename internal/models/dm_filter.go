package models

import (
	"errors"

	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type Dms struct {
	MessageId string `json:"message_id"`
	Message   string `json:"message"`
}

type DmFilter struct {
	UserId    string `json:"user_id"`
	OrgId     string `json:"org_id,omitempty"`
	ChannelId string `json:"channel_id"`
	Avatar    string `json:"avatar,omitempty"`
	ThreadId  string `json:"thread_id,omitempty"`
	Message   Dms    `json:"message"`
}

func FilterDms(db *storage.Database, userId, orgId string) {

}

func GetUserDmChannels(db *gorm.DB, userId, orgId string) ([]string, error) {
	org := Organisation{}
	if exists := postgresql.CheckExists(db, &org, "id = ?", orgId); exists == false {
		return nil, errors.New("Organisation does not exist")
	}
	dm := DmChannels{}

	db.Model(&dm).Where("user_id = ? AND org_id = ?", userId, orgId).Find(&dm)
	return nil, nil
}
