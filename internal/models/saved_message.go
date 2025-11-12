package models

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

type SavedMessage struct {
	ID                   string         `gorm:"type:uuid;primary_key" json:"id"`
	ChannelsID           string         `gorm:"type:uuid;not null;index" json:"channels_id"`
	OrgId                string         `gorm:"type:uuid;not null;index" json:"org_id"`
	UserID               string         `gorm:"type:uuid;not null;index" json:"user_id"`
	Type                 string         `gorm:"type:text;not null;index" json:"type,omitempty"`
	MessageID            *string        `gorm:"type:uuid;null;index" json:"message_id,omitempty"`
	ThreadID             string         `gorm:"type:uuid;null;index" json:"thread_id"`
	CreatedAt            time.Time      `gorm:"column:created_at; not null; autoCreateTime" json:"created_at"`
	RemainderMessage     bool           `gorm:"column:remainder; default:false" json:"remainder_message,omitempty"`
	RemainderAt          *time.Time     `gorm:"column:remainder_at; null" json:"remainder_at,omitempty"`
	RemainderDescription *string        `gorm:"column:remainder_description; type:text; null" json:"remainder_description,omitempty"`
	RiverJobID           *int64         `gorm:"type:bigint;index" json:"river_job_id,omitempty"`
	Archived             bool           `gorm:"column:archived; default:false" json:"archived,omitempty"`
	Completed            bool           `gorm:"column:completed; default:false" json:"completed,omitempty"`
	DeletedAt            gorm.DeletedAt `gorm:"index" json:"-"`
}

type SavedMessagesResp struct {
	ID          string     `json:"id,omitempty"`
	ThreadID    string     `json:"thread_id,omitempty"`
	MessageID   *string    `json:"message_id,omitempty"`
	AvatarURL   string     `json:"avatar_url,omitempty"`
	Username    string     `json:"username,omitempty"`
	Content     string     `json:"content,omitempty"`
	UserID      string     `json:"user_id,omitempty"`
	ChannelID   string     `json:"channel_id,omitempty"`
	ChannelName string     `json:"channel_name,omitempty"`
	Type        string     `json:"type,omitempty"`         // thread or message(thread-reply)
	ChannelType string     `json:"channel_type,omitempty"` // dm, groupDM, public, private
	SavedAt     time.Time  `json:"saved_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at,omitempty"`
	Overdue     bool       `json:"overdue,omitempty"`
	OverDueTime *time.Time `json:"overdue_time,omitempty"`
}

type SaveThreadRequest struct {
	ChannelsId string `json:"channels_id" validate:"required"`
	ThreadId   string `json:"thread_id" validate:"required"`
	OrgId      string `json:"org_id"`
	UserId     string `json:"user_id"`
}

type SaveMessageRequest struct {
	ChannelsId string `json:"channels_id" validate:"required"`
	ThreadId   string `json:"thread_id" validate:"required"`
	MessageId  string `json:"message_id" validate:"required"`
	OrgId      string `json:"org_id"`
	UserId     string `json:"user_id"`
}

type SavedMessageIds struct {
	MessageID      string
	ThreadID       string
	OrgID          string
	UserID         string
	SavedMessageID string
	ChannelID      string
}

type SetRemainderRequest struct {
	Type        string    `json:"type" validate:"required"` // thread or message
	ChannelsId  string    `json:"channels_id" validate:"required"`
	ThreadId    string    `json:"thread_id" validate:"required"`
	MessageId   *string   `json:"message_id,omitempty"`
	OrgId       string    `json:"org_id"`
	UserId      string    `json:"user_id"`
	RemainderAt time.Time `json:"remainder_at" validate:"required"`
}

func (m *SavedMessage) CreateSavedThreadRecord(db *gorm.DB) (bool, error) {
	var (
		org          Organisation
		dmChannels   DmChannels
		userChannels UserChannels
		savedMessage SavedMessage
	)

	exists := postgresql.CheckExists(db, &org, "id = ?", m.OrgId)
	if !exists {
		return false, errors.New("organisation not found")
	}

	isMember, err := new(Organisation).CheckUserIsMemberOfOrg(m.UserID, m.OrgId, db)
	if err != nil {
		return false, err
	}

	if !isMember {
		return false, errors.New("user is not a member of organisation")
	}

	chanExist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", m.ChannelsID, m.UserID)
	dmChanExist := postgresql.CheckExists(db, &dmChannels, "channel_id = ?", m.ChannelsID)

	if !(dmChanExist || chanExist) {
		return false, errors.New("user not in channel")
	}

	if chanExist {
		var channels Channels
		exists := postgresql.CheckExists(db, &channels, "id = ? AND organisation_id = ?", m.ChannelsID, m.OrgId)
		if !exists {
			return false, errors.New("channel not found in organisation")
		}
	}

	if dmChanExist {
		var dmChannel DmChannels
		exists := postgresql.CheckExists(db, &dmChannel, "channel_id = ? AND organisation_id = ?", m.ChannelsID, m.OrgId)
		if !exists {
			return false, errors.New("direct message channel not found in organisation")
		}
	}

	threadExists := postgresql.CheckExists(db, &savedMessage, "org_id = ? AND user_id = ? AND thread_id = ?", m.OrgId, m.UserID, m.ThreadID)
	if threadExists {
		err := savedMessage.DeleteSavedMessageByID(db)
		if err != nil {
			return false, err
		}
		return false, nil // unsaved
	}

	m.ID = utility.GenerateUUID()
	createErr := postgresql.CreateOneRecord(db, &m)
	if createErr != nil {
		return false, createErr
	}

	return true, nil // saved
}

func (m *SavedMessage) CreateReplyMessageRecord(db *gorm.DB) (bool, error) {
	var (
		org          Organisation
		dmChannels   DmChannels
		userChannels UserChannels
		savedMessage SavedMessage
	)

	exists := postgresql.CheckExists(db, &org, "id = ?", m.OrgId)
	if !exists {
		return false, errors.New("organisation not found")
	}

	isMember, err := new(Organisation).CheckUserIsMemberOfOrg(m.UserID, m.OrgId, db)
	if err != nil {
		return false, err
	}
	if !isMember {
		return false, errors.New("user is not a member of organisation")
	}

	chanExist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", m.ChannelsID, m.UserID)
	dmChanExist := postgresql.CheckExists(db, &dmChannels, "channel_id = ?", m.ChannelsID)

	if !(dmChanExist || chanExist) {
		return false, errors.New("user not in channel")
	}

	if chanExist {
		var channels Channels
		exists := postgresql.CheckExists(db, &channels, "id = ? AND organisation_id = ?", m.ChannelsID, m.OrgId)
		if !exists {
			return false, errors.New("channel not found in organisation")
		}
	}

	if dmChanExist {
		var dmChannel DmChannels
		exists := postgresql.CheckExists(db, &dmChannel, "channel_id = ? AND organisation_id = ?", m.ChannelsID, m.OrgId)
		if !exists {
			return false, errors.New("direct message channel not found in organisation")
		}
	}

	msgExists := postgresql.CheckExists(db, &savedMessage, "org_id = ? AND user_id = ? AND thread_id = ? AND message_id = ?", m.OrgId, m.UserID, m.ThreadID, m.MessageID)
	if msgExists {
		err := savedMessage.DeleteSavedMessageByID(db)
		if err != nil {
			return false, err
		}
		return false, nil // unsaved
	}

	m.ID = utility.GenerateUUID()
	createErr := postgresql.CreateOneRecord(db, &m)
	if createErr != nil {
		return false, createErr
	}

	return true, nil // saved
}

func (m *SavedMessage) GetSavedMessages(db *gorm.DB, ids SavedMessageIds) ([]SavedMessagesResp, error) {
	var (
		org                     Organisation
		organisation            *Organisation
		messages                []SavedMessage
		messagesResp            = make([]SavedMessagesResp, 0)
		foundMsgs, notFoundMsgs []SavedMessagesResp
	)

	exists := postgresql.CheckExists(db, &org, "id = ?", ids.OrgID)
	if !exists {
		return nil, errors.New("organisation not found")
	}

	isMember, err := organisation.CheckUserIsMemberOfOrg(ids.UserID, ids.OrgID, db)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("user is not a member of organisation")
	}

	err = postgresql.SelectAllFromDb(db, "", &messages, "org_id = ? AND user_id = ?", ids.OrgID, ids.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve saved messages: %w", err)
	}

	resolveChannelName := func(db *gorm.DB, channelID string) string {
		var dmchan DmChannels
		if postgresql.CheckExists(db, &dmchan, "channel_id = ?", channelID) {
			return "Direct Message"
		}
		var ch Channels
		if postgresql.CheckExists(db, &ch, "id = ?", channelID) {
			return ch.Name
		}

		return "unknown"
	}

	resolveChannelType := func(db *gorm.DB, channelID, user_id, org_id string) string {
		var (
			dmChannels   DmChannels
			userChannels UserChannels
		)
		chanExist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", channelID, user_id)
		dmChanExist := postgresql.CheckExists(db, &dmChannels, "channel_id = ?", channelID)

		if !(dmChanExist || chanExist) {
			return "unknown"
		}

		if dmChanExist {
			var dmChannel DmChannels
			exists := postgresql.CheckExists(db, &dmChannel, "channel_id = ?", channelID)
			if !exists {
				return "unknown"
			}

			if dmChannel.ChatType == "bot" {
				return "Bot"
			} else {
				if dmChannel.ChannelType == "dm" {
					return "Direct Message"
				} else {
					return "Group Direct Message"
				}
			}
		}

		if chanExist {
			var channels Channels
			exists := postgresql.CheckExists(db, &channels, "id = ? AND organisation_id = ?", channelID, org_id)
			if !exists {
				return "unknown"
			}

			if channels.IsPrivate {
				return "Private Channel"
			} else {
				return "Public"
			}
		}
		return "unknown"
	}

	for _, msg := range messages {
		var (
			t  ThreadDocument
			m  MessageDocument
			mr SavedMessagesResp
		)

		populateNotFoundDocument := func() SavedMessagesResp {
			resp := SavedMessagesResp{
				Content:   "A message you saved was not found.",
				AvatarURL: "",
			}
			return resp
		}

		if msg.RemainderAt.Before(time.Now().UTC()) {
			mr.Overdue = true
		}

		mr.OverDueTime = msg.RemainderAt

		if msg.MessageID != nil {
			if err := m.GetMessageById(db, *msg.MessageID); err != nil {
				resp := populateNotFoundDocument()
				notFoundMsgs = append(notFoundMsgs, resp)
				continue
			}

			mr.ID = msg.ID
			mr.ThreadID = msg.ThreadID
			mr.MessageID = msg.MessageID
			mr.AvatarURL = m.AvatarURL
			mr.Username = m.Username
			mr.Content = m.Content
			mr.SavedAt = msg.CreatedAt
			mr.UserID = m.UserID
			mr.Type = "message"
			mr.ChannelID = m.ChannelsID
			mr.ChannelName = resolveChannelName(db, m.ChannelsID)
			mr.ChannelType = resolveChannelType(db, m.ChannelsID, msg.UserID, msg.OrgId)
		} else {
			if err := t.GetThreadById(msg.ThreadID); err != nil {
				resp := populateNotFoundDocument()
				notFoundMsgs = append(notFoundMsgs, resp)
				continue
			}

			mr.ID = msg.ID
			mr.ThreadID = msg.ThreadID
			mr.MessageID = nil
			mr.AvatarURL = t.AvatarURL
			mr.Username = t.Username
			mr.Content = t.Content
			mr.SavedAt = msg.CreatedAt
			mr.UserID = t.UserId
			mr.Type = "thread"
			mr.ChannelID = t.ChannelsID
			mr.ChannelName = resolveChannelName(db, t.ChannelsID)
			mr.ChannelType = resolveChannelType(db, t.ChannelsID, msg.UserID, msg.OrgId)
		}

		foundMsgs = append(foundMsgs, mr)
	}

	messagesResp = append(foundMsgs, notFoundMsgs...)
	return messagesResp, nil
}

func (m *SavedMessage) GetSavedMessageByID(db *gorm.DB, ids SavedMessageIds) error {
	var org Organisation

	exists := postgresql.CheckExists(db, &org, "id = ?", ids.OrgID)
	if !exists {
		return errors.New("organisation not found")
	}

	isMember, err := new(Organisation).CheckUserIsMemberOfOrg(ids.UserID, ids.OrgID, db)
	if err != nil {
		return err
	}
	if !isMember {
		return errors.New("user is not a member of organisation")
	}

	fmt.Println(ids.SavedMessageID)

	err, _ = postgresql.SelectOneFromDb(db, &m, "id = ?", ids.SavedMessageID)
	if err != nil {
		return err
	}

	return nil
}

func (m *SavedMessage) DeleteSavedMessageByID(db *gorm.DB) error {

	err := postgresql.HardDeleteRecordFromDb(db, m)
	if err != nil {
		return fmt.Errorf("failed to delete saved message: %w", err)
	}

	return nil
}

func (m *SavedMessage) SavedThreadMsgExists(db *gorm.DB, ids SavedMessageIds) bool {
	var (
		savedMessage SavedMessage
	)

	return postgresql.CheckExists(db, &savedMessage, "thread_id = ? AND org_id = ?", ids.ThreadID, ids.OrgID)
}

func (m *SavedMessage) SavedReplyMsgExists(db *gorm.DB, ids SavedMessageIds) bool {
	var (
		savedMessage SavedMessage
	)

	return postgresql.CheckExists(db, &savedMessage, "message_id = ? AND thread_id = ? AND org_id = ?", ids.MessageID, ids.ThreadID, ids.OrgID)
}

func (m *SavedMessage) DeleteSavedThreadMsgByMessageID(db *gorm.DB, ids SavedMessageIds) error {
	var (
		savedMessage SavedMessage
	)

	idExists := postgresql.CheckExists(db, &savedMessage, "thread_id = ? AND org_id = ?", ids.ThreadID, ids.OrgID)
	if !idExists {
		return errors.New("invalid message ID")
	}

	query := db.Where("thread_id = ? AND org_id = ? AND user_id = ?", ids.ThreadID, ids.OrgID, ids.UserID)
	err := query.Delete(&SavedMessage{}).Error
	if err != nil {
		return err
	}

	return nil
}

func (m *SavedMessage) DeleteSavedMessageByMessageID(db *gorm.DB, ids SavedMessageIds) error {
	var (
		savedMessage SavedMessage
	)

	idExists := postgresql.CheckExists(db, &savedMessage, "message_id = ? AND thread_id = ? AND org_id = ?", ids.MessageID, ids.ThreadID, ids.OrgID)
	if !idExists {
		return errors.New("invalid message ID")
	}

	query := db.Where("message_id = ? AND thread_id = ? AND org_id = ? AND user_id = ?", ids.MessageID, ids.ThreadID, ids.OrgID, ids.UserID)
	err := query.Delete(&SavedMessage{}).Error
	if err != nil {
		return err
	}

	return nil
}

func (m *SavedMessage) DeleteSavedMessagesByChannelID(db *gorm.DB, channelID, userID string) error {
	var (
		savedMessage SavedMessage
	)

	idExists := postgresql.CheckExists(db, &savedMessage, "channels_id = ? AND user_id", channelID, userID)
	if !idExists {
		return errors.New("invalid message ID")
	}

	query := db.Where("channels_id = ? AND user_id", channelID, userID)
	err := query.Delete(&SavedMessage{}).Error
	if err != nil {
		return fmt.Errorf("failed to delete saved messages : %v", err)
	}

	return nil
}

func (m *SavedMessage) UpdateSavedMessageRemainder(db *gorm.DB, req SetRemainderRequest, update map[string]any) (int, error) {
	var (
		savedMessage SavedMessage
		query        string
		args         []any
	)

	if req.MessageId == nil {
		query = "thread_id = ? AND type = ? AND user_id = ?"
		args = []any{req.ThreadId, req.Type, req.UserId}
	} else {
		query = "message_id = ? AND thread_id = ? AND type = ? AND user_id = ?"
		args = []any{*req.MessageId, req.ThreadId, req.Type, req.UserId}
	}

	exists := postgresql.CheckExists(db, &savedMessage, query, args...)
	if !exists {
		return http.StatusNotFound, errors.New("saved message or thread not found")
	}

	res, err := postgresql.UpdateFields(db, &savedMessage, update, query, args...)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to update saved message reminder: %w", err)
	}

	if res.RowsAffected == 0 {
		return http.StatusConflict, errors.New("no rows were updated")
	}

	return http.StatusOK, nil
}

func (m *SavedMessage) GetByUserAndThread(db *gorm.DB, userID, orgID, channelID, threadID string) (*SavedMessage, error) {
	var savedMessage SavedMessage

	err, _ := postgresql.SelectOneFromDb(db, &savedMessage, "user_id = ? AND org_id = ? AND channels_id = ? AND thread_id = ?", userID, orgID, channelID, threadID)
	if err != nil {
		return nil, err
	}

	return &savedMessage, nil
}
