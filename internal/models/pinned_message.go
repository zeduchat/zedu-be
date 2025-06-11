package models

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type PinnedMessage struct {
	ID         string    `gorm:"type:uuid;primary_key" json:"id"`
	ChannelsID string    `gorm:"type:uuid;not null;index" json:"channels_id"`
	OrgId      string    `gorm:"type:uuid;not null;index" json:"org_id"`
	UserID     string    `gorm:"type:uuid;not null;index" json:"user_id"`
	Pinned     bool      `gorm:"type:boolean;default:true" json:"pinned"`
	PinnedAt   time.Time `gorm:"column:pinned_at; not null; autoCreateTime" json:"pinned_at"`
	ThreadID   uuid.UUID `gorm:"type:uuid;null;index" json:"thread_id"`
}

type PinMessageRequest struct {
	ThreadId   string `json:"thread_id" validate:"required"`
	OrgId      string `json:"org_id"`
	ChannelsId string `json:"channels_id"`
	UserId     string `json:"user_id"`
}

func (m *PinnedMessage) CreatePinnedMessageRecord(db *gorm.DB) error {
	var (
		dmChannels   DmChannels
		userChannels UserChannels
		org          Organisation
	)

	userChanExist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", m.ChannelsID, m.UserID)
	dmChanExist := postgresql.CheckExists(db, &dmChannels, "channel_id = ?", m.ChannelsID)

	if !(dmChanExist || userChanExist) {
		return errors.New("user not in channel")
	}

	isMember, err := org.CheckUserIsMemberOfOrg(m.UserID, m.OrgId, db)
	if err != nil {
		return err
	}
	if !isMember {
		return errors.New("user is not a member of this organisation")
	}

	if err := postgresql.CreateOneRecord(db, &m); err != nil {
		return err
	}

	return nil
}

func (m *PinnedMessage) GetPinnedMessagesForChannel(db *gorm.DB, orgID, channelID, userID string) ([]PinnedMessage, error) {
	var (
		org          Organisation
		messages     []PinnedMessage
		userChannels UserChannels
		dmChannels   DmChannels
	)

	isMember, err := org.CheckUserIsMemberOfOrg(userID, orgID, db)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("user is not a member of this organisation")
	}

	userChanExist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", channelID, userID)
	dmChanExist := postgresql.CheckExists(db, &dmChannels, "channel_id = ?", channelID)
	if !(dmChanExist || userChanExist) {
		return nil, errors.New("user not in channel")
	}

	findErr := db.Order("pinned_at DESC").Find(&messages).Where("pinned = ? AND org_id = ? AND channels_id = ?", true, orgID, channelID).Error
	return messages, findErr
}

func (m *PinnedMessage) GetPinnedMessageByID(db *gorm.DB, messageID, orgID, channelID, userID string) (*PinnedMessage, error) {
	var (
		org          Organisation
		userChannels UserChannels
		dmChannels   DmChannels
	)

	isMember, err := org.CheckUserIsMemberOfOrg(userID, orgID, db)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("user is not a member of this organisation")
	}

	userChanExist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", channelID, userID)
	dmChanExist := postgresql.CheckExists(db, &dmChannels, "channel_id = ?", channelID)
	if !(dmChanExist || userChanExist) {
		return nil, errors.New("user not in channel")
	}

	query := db.Where("id = ? AND org_id = ? AND channels_id = ?", messageID, orgID, channelID)

	findErr := query.First(&m).Error
	if findErr != nil {
		return nil, findErr
	}

	return m, nil
}

func (m *PinnedMessage) DeletePinnedMessageRecord(db *gorm.DB, pinnedID string) error {
	query := db.Where("id = ?", pinnedID)

	err := query.Delete(&PinnedMessage{}).Error
	if err != nil {
		return err
	}

	return nil
}
