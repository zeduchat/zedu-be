package models

import (
	"errors"
	"sort"
	"time"

	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
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
	ThreadID   string    `gorm:"type:uuid;null;index" json:"thread_id"`
	MessageID  *string   `gorm:"type:uuid;null;index" json:"message_id,omitempty"`
}

type PinMessageRequest struct {
	ThreadId   string `json:"thread_id" validate:"required"`
	MessageID  string `json:"message_id"`
	OrgId      string `json:"org_id"`
	ChannelsId string `json:"channels_id"`
	UserId     string `json:"user_id"`
}

func (m *PinnedMessage) CreatePinnedThreadRecord(db *gorm.DB) error {
	var (
		dmChannels    DmChannels
		userChannels  UserChannels
		org           Organisation
		pinnedMessage PinnedMessage
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

	exists := postgresql.CheckExists(db, &pinnedMessage, "user_id = ? AND org_id = ? AND channels_id = ? AND thread_id = ?", m.UserID, m.OrgId, m.ChannelsID, m.ThreadID)
	if exists {
		return errors.New("message already pinned")
	}

	if err := postgresql.CreateOneRecord(db, &m); err != nil {
		return err
	}

	return nil
}

func (m *PinnedMessage) CreatePinnedMessageRecord(db *gorm.DB) error {
	var (
		dmChannels    DmChannels
		userChannels  UserChannels
		org           Organisation
		pinnedMessage PinnedMessage
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

	exists := postgresql.CheckExists(db, &pinnedMessage, "user_id = ? AND org_id = ? AND channels_id = ? AND thread_id = ? AND message_id = ?", m.UserID, m.OrgId, m.ChannelsID, m.ThreadID, m.MessageID)
	if exists {
		return errors.New("message already pinned")
	}

	if err := postgresql.CreateOneRecord(db, &m); err != nil {
		return err
	}

	return nil
}

func (m *PinnedMessage) GetPinnedMessagesForChannel(db *gorm.DB, ids IDS) ([]PinnedMessage, error) {
	var (
		org          Organisation
		messages     []PinnedMessage
		userChannels UserChannels
		dmChannels   DmChannels
	)

	isMember, err := org.CheckUserIsMemberOfOrg(ids.UserID, ids.OrganisationID, db)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("user is not a member of this organisation")
	}

	userChanExist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", ids.ChannelID, ids.UserID)
	dmChanExist := postgresql.CheckExists(db, &dmChannels, "channel_id = ?", ids.ChannelID)
	if !(dmChanExist || userChanExist) {
		return nil, errors.New("user not in channel")
	}

	findErr := db.Order("pinned_at DESC").Find(&messages).Where("pinned = ? AND org_id = ? AND channels_id = ?", true, ids.OrganisationID, ids.ChannelID).Error
	return messages, findErr
}

func (m *PinnedMessage) GetAllPinnedMessagesForChannel(db *storage.Database, ids IDS) ([]MessageDocument, error) {
	var (
		org          Organisation
		userChannels UserChannels
		dmChannels   DmChannels
	)

	isMember, err := org.CheckUserIsMemberOfOrg(ids.UserID, ids.OrganisationID, db.Postgresql)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("user is not a member of this organisation")
	}

	userChanExist := postgresql.CheckExists(db.Postgresql, &userChannels, "channels_id = ? AND user_id = ?", ids.ChannelID, ids.UserID)
	dmChanExist := postgresql.CheckExists(db.Postgresql, &dmChannels, "channel_id = ?", ids.ChannelID)
	if !(dmChanExist || userChanExist) {
		return nil, errors.New("user not in channel")
	}

	pinnedThreads, err := GetPinnedThreadMsgs(db, ids.ChannelID)
	if err != nil {
		return nil, err
	}

	pinnedReplies, err := GetPinnedReplyMsgs(db, ids.ChannelID)
	if err != nil {
		return nil, err
	}

	allPinned := append(pinnedThreads, pinnedReplies...)

	sort.SliceStable(allPinned, func(i, j int) bool {
		return allPinned[i].CreatedAt.After(allPinned[j].CreatedAt)
	})
	return allPinned, nil
}

func GetPinnedThreadMsgs(db *storage.Database, channelsId string) ([]MessageDocument, error) {
	var message []MessageDocument

	query := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{
						"term": map[string]any{
							"channels_id": channelsId,
						},
					},
					{
						"term": map[string]interface{}{
							"is_pinned": true,
						},
					},
				},
			},
		},
	}

	var results interface{}
	if err := elastic.SelectAll(db.Elastic, ThreadIndexName, query, &results); err != nil {
		return nil, err
	}

	message, err := UnmarshalMessageResponse(results)
	if err != nil {
		return nil, err
	}

	return message, nil
}

func GetPinnedReplyMsgs(db *storage.Database, channelsId string) ([]MessageDocument, error) {
	var message []MessageDocument

	query := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{
						"term": map[string]any{
							"channels_id": channelsId,
						},
					},
					{
						"term": map[string]interface{}{
							"is_pinned": true,
						},
					},
				},
			},
		},
	}

	var results interface{}
	if err := elastic.SelectAll(db.Elastic, MessageIndexName, query, &results); err != nil {
		return nil, err
	}

	message, err := UnmarshalMessageResponse(results)
	if err != nil {
		return nil, err
	}

	return message, nil
}

func (m *PinnedMessage) DeletePinnedThreadMessageRecord(db *gorm.DB, ids IDS) error {
	query := db.Where("user_id = ? AND org_id = ? AND channels_id = ? AND thread_id = ?", ids.UserID, ids.OrganisationID, ids.ChannelID, ids.ThreadID)

	if err := query.First(&m).Error; err != nil {
		return err
	}

	if err := query.Delete(&PinnedMessage{}).Error; err != nil {
		return err
	}

	return nil
}

func (m *PinnedMessage) DeletePinnedReplyMessageRecord(db *gorm.DB, ids IDS) error {
	query := db.Where("user_id = ? AND org_id = ? AND channels_id = ? AND message_id = ?", ids.UserID, ids.OrganisationID, ids.ChannelID, ids.MessageID)

	if err := query.First(&m).Error; err != nil {
		return err
	}

	if err := query.Delete(&PinnedMessage{}).Error; err != nil {
		return err
	}

	return nil
}
