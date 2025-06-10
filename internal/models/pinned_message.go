package models

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type PinnedMessage struct {
	ID             string                 `gorm:"type:uuid;primary_key" json:"id"`
	Content        string                 `gorm:"column:content; type:text; not null" json:"content"`
	ChannelsID     string                 `gorm:"type:uuid;not null;index" json:"channels_id"`
	OrganisationID string                 `gorm:"type:uuid;not null;index" json:"org_id"`
	UserID         string                 `gorm:"type:uuid;not null;index" json:"user_id"`
	Pinned         bool                   `gorm:"type:boolean;default:true" json:"pinned"`
	PinnedAt       time.Time              `gorm:"column:pinned_at; not null; autoCreateTime" json:"pinned_at"`
	ThreadID       uuid.UUID              `gorm:"type:uuid;null;index" json:"thread_id"`
	Media          []UploadedFileResponse `gorm:"type:jsonb;serializer:json" json:"media"`
}

type PinMessageRequest struct {
	Content    string                 `json:"content" validate:"required"`
	OrgId      string                 `json:"org_id" validate:"required"`
	ThreadId   string                 `json:"thread_id" validate:"required"`
	ChannelsId string                 `json:"channels_id"`
	UserId     string                 `json:"user_id"`
	Media      []UploadedFileResponse `json:"media,omitempty"`
}

func (m *PinnedMessage) CreatePinnedMessageRecord(db *gorm.DB) error {
	var (
		dmChannels   DmChannels
		userChannels UserChannels
		channels     Channels
		org          Organisation
	)

	userChanExist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", m.ChannelsID, m.UserID)
	dmChanExist := postgresql.CheckExists(db, &dmChannels, "channel_id = ?", m.ChannelsID)
	chanExist := postgresql.CheckExists(db, &channels, "id = ?", m.ChannelsID)

	if !chanExist {
		return errors.New("channel does not exist")
	}

	if !(dmChanExist || userChanExist) {
		return errors.New("user not in channel")
	}

	orgExist := postgresql.CheckExists(db, &org, "id = ?", m.OrganisationID)
	if !orgExist {
		return errors.New("organisation does not exist")
	}

	isMember, err := org.CheckUserIsMemberOfOrg(m.UserID, m.OrganisationID, db)
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

func (m *PinnedMessage) GetPinnedMessagesForChannel(db *gorm.DB, channelID, userID string) ([]PinnedMessage, error) {
	var (
		messages     []PinnedMessage
		channels     Channels
		userChannels UserChannels
		dmChannels   DmChannels
	)
	
	chanExist := postgresql.CheckExists(db, &channels, "id = ?", channelID)
	if !chanExist {
		return nil, errors.New("channel does not exist")
	}
	
	userChanExist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", channelID, userID)
	dmChanExist := postgresql.CheckExists(db, &dmChannels, "channel_id = ?", channelID)
	if !(dmChanExist || userChanExist) {
		return nil, errors.New("user not in channel")
	}

	err := db.Order("pinned_at DESC").Find(&messages).Where("pinned = ? AND channels_id = ?", true, channelID).Error
	return messages, err
}

func (m *PinnedMessage) GetPinnedMessageByID(db *gorm.DB, messageID, channelID, userID string) (*PinnedMessage, error) {
	var (
		userChannels UserChannels
		channels     Channels
		dmChannels   DmChannels
	)

	chanExist := postgresql.CheckExists(db, &channels, "id = ?", channelID)
	if !chanExist {
		return nil, errors.New("channel does not exist")
	}

	userChanExist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", channelID, userID)
	dmChanExist := postgresql.CheckExists(db, &dmChannels, "channel_id = ?", channelID)
	if !(dmChanExist || userChanExist) {
		return nil, errors.New("user not in channel")
	}

	query := db.Where("id = ?", messageID)

	err := query.First(&m).Error
	if err != nil {
		return nil, err
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
