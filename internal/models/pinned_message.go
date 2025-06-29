package models

import (
	"errors"
	"net/http"
	"sort"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
)

type PinnedMessage struct {
	ID         string    `gorm:"type:uuid;primary_key" json:"-"`
	ChannelsID string    `gorm:"type:uuid;not null;index" json:"channels_id"`
	UserID     string    `gorm:"type:uuid;not null;index" json:"user_id"`
	Pinned     bool      `gorm:"type:boolean;default:true" json:"pinned"`
	PinnedAt   time.Time `gorm:"column:pinned_at; not null; autoCreateTime" json:"pinned_at"`
	ThreadID   string    `gorm:"type:uuid;null;index" json:"thread_id"`
	Type       string    `gorm:"type:text;default:thread" json:"type"`
	MessageID  *string   `gorm:"type:uuid;null;index" json:"message_id,omitempty"`
}

type PinnedMessageResponse struct {
	MessageID  string    `json:"message_id,omitempty"`
	Content    string    `json:"message"`
	ChannelsID string    `json:"channels_id"`
	UserID     string    `json:"user_id"`
	Username   string    `json:"username"`
	CreatedAt  time.Time `json:"created_at"`
	ThreadID   string    `json:"thread_id"`
	FullName   string    `json:"full_name"`
	Email      string    `json:"email"`
	Mentions   []Mention `json:"mentions,omitempty"`
	Type       string    `json:"type"`
}

type PinMessageRequest struct {
	ThreadId   string `json:"thread_id" validate:"required"`
	MessageID  string `json:"message_id"`
	ChannelsId string `json:"channels_id"`
	UserId     string `json:"user_id"`
}

func (m *PinnedMessage) CreatePinnedThreadRecord(db *gorm.DB) (int, error) {
	var (
		dmChannels    DmChannels
		userChannels  UserChannels
		pinnedMessage PinnedMessage
		checkThread   ThreadDocument
	)

	checkThread.ID = m.ThreadID
	checkThread.ChannelsID = m.ChannelsID

	exist, _, _ := checkThread.CheckExists()

	if !exist {
		return http.StatusBadRequest, errors.New("thread does not exist")
	}

	userChanExist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", m.ChannelsID, m.UserID)
	dmChanExist := postgresql.CheckExists(db, &dmChannels, "channel_id = ?", m.ChannelsID)

	if !(dmChanExist || userChanExist) {
		return http.StatusBadRequest, errors.New("user not in channel")
	}

	exists := postgresql.CheckExists(db, &pinnedMessage, "type = ? AND channels_id = ? AND thread_id = ?", "thread", m.ChannelsID, m.ThreadID)
	if !exists {
		if err := postgresql.CreateOneRecord(db, &m); err != nil {
			return http.StatusInternalServerError, err
		}
	}

	return http.StatusCreated, nil
}

func (m *PinnedMessage) CreatePinnedMessageRecord(db *gorm.DB) (int, error) {
	var (
		dmChannels        DmChannels
		userChannels      UserChannels
		pinnedMessage     PinnedMessage
		checkReplyMessage MessageDocument
	)

	checkReplyMessage.ID = *m.MessageID
	checkReplyMessage.ChannelsID = m.ChannelsID

	exist, _, _ := checkReplyMessage.CheckExists()
	if !exist {
		return http.StatusBadRequest, errors.New("reply message does not exist")
	}

	userChanExist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", m.ChannelsID, m.UserID)
	dmChanExist := postgresql.CheckExists(db, &dmChannels, "channel_id = ?", m.ChannelsID)

	if !(dmChanExist || userChanExist) {
		return http.StatusBadRequest, errors.New("user not in channel")
	}

	exists := postgresql.CheckExists(db, &pinnedMessage, "type = ? AND thread_id = ? AND message_id = ?", "reply", m.ThreadID, m.MessageID)

	if !exists {

		if err := postgresql.CreateOneRecord(db, &m); err != nil {
			return http.StatusInternalServerError, err
		}
	}

	return http.StatusCreated, nil
}

func (m *PinnedMessage) GetPinnedMessagesForChannel(db *gorm.DB, ids IDS) ([]PinnedMessage, error) {
	var (
		messages     []PinnedMessage
		userChannels UserChannels
		dmChannels   DmChannels
	)

	userChanExist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", ids.ChannelID, ids.UserID)
	dmChanExist := postgresql.CheckExists(db, &dmChannels, "channel_id = ?", ids.ChannelID)
	if !(dmChanExist || userChanExist) {
		return nil, errors.New("user not in channel")
	}

	findErr := db.Order("pinned_at DESC").Find(&messages).Where("pinned = ? AND org_id = ? AND channels_id = ?", true, ids.OrganisationID, ids.ChannelID).Error
	return messages, findErr
}

func (m *PinnedMessage) GetAllPinnedMessagesForChannel(db *storage.Database, ids IDS) ([]PinnedMessageResponse, error) {
	var (
		userChannels UserChannels
		dmChannels   DmChannels
	)

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

	allPinned := []PinnedMessageResponse{}

	for _, con := range pinnedThreads {
		allPinned = append(allPinned, PinnedMessageResponse{
			ThreadID:   con.ID,
			Content:    con.Content,
			ChannelsID: con.ChannelsID,
			UserID:     con.UserId,
			Username:   con.Username,
			CreatedAt:  con.CreatedAt,
			FullName:   con.FullName,
			Email:      con.Email,
			Type:       "thread",
			Mentions:   con.Mentions,
		})
	}

	for _, con := range pinnedReplies {
		allPinned = append(allPinned, PinnedMessageResponse{
			ThreadID:   con.ThreadID.String(),
			MessageID:  con.ID,
			Content:    con.Content,
			ChannelsID: con.ChannelsID,
			UserID:     con.UserID,
			Username:   con.Username,
			CreatedAt:  con.CreatedAt,
			FullName:   con.FullName,
			Email:      con.Email,
			Type:       "reply",
			Mentions:   con.Mentions,
		})
	}

	sort.SliceStable(allPinned, func(i, j int) bool {
		return allPinned[i].CreatedAt.After(allPinned[j].CreatedAt)
	})
	return allPinned, nil
}

func GetPinnedThreadMsgs(db *storage.Database, channelsId string) ([]Threads, error) {
	var message []Threads

	query := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{
						"term": map[string]any{
							"channels_id.keyword": channelsId,
						},
					},
					{
						"term": map[string]any{
							"is_pinned": true,
						},
					},
				},
			},
		},
	}

	var results any
	if err := elastic.SelectAll(db.Elastic, ThreadIndexName, query, &results); err != nil {
		return nil, err
	}

	message, err := UnmarshalThreadResponse(results)
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
							"channels_id.keyword": channelsId,
						},
					},
					{
						"term": map[string]any{
							"is_pinned": true,
						},
					},
				},
			},
		},
	}

	var results any
	if err := elastic.SelectAll(db.Elastic, MessageIndexName, query, &results); err != nil {
		return nil, err
	}

	message, err := UnmarshalMessageResponse(results)
	if err != nil {
		return nil, err
	}

	return message, nil
}

func (m *PinnedMessage) DeletePinnedThreadMessageRecord(db *gorm.DB) error {
	query := db.Where("type = ? AND channels_id = ? AND thread_id = ?", "thread", m.ChannelsID, m.ThreadID)

	if err := query.First(&m).Error; err != nil {
		return err
	}

	if err := query.Delete(&PinnedMessage{}).Error; err != nil {
		return err
	}

	return nil
}

func (m *PinnedMessage) DeletePinnedReplyMessageRecord(db *gorm.DB) error {
	query := db.Where("type = ? AND channels_id = ? AND message_id = ?", "reply", m.ChannelsID, m.MessageID)

	if err := query.First(&m).Error; err != nil {
		return err
	}

	if err := query.Delete(&PinnedMessage{}).Error; err != nil {
		return err
	}

	return nil
}

func (m *PinnedMessage) CheckPinnedThreadExists(db *gorm.DB) bool {
	var pinnedMessage PinnedMessage

	return postgresql.CheckExists(db, &pinnedMessage, "type = ? AND channels_id = ? AND thread_id = ?", "thread", m.ChannelsID, m.ThreadID)

}

func (m *PinnedMessage) CheckPinnedReplyExists(db *gorm.DB) bool {
	var pinnedMessage PinnedMessage

	return postgresql.CheckExists(db, &pinnedMessage, "type = ? AND thread_id = ? AND message_id = ?", "reply", m.ThreadID, m.MessageID)

}
