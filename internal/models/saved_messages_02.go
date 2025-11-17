package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"gorm.io/gorm"
)

type SavedMessageFilter struct {
	OrgID     string
	UserID    string
	Completed *bool
	Archived  *bool
}

func (m *SavedMessage) GetSavedMessages(db *gorm.DB, ids SavedMessageIds) ([]SavedMessagesResp, error) {
	filter := SavedMessageFilter{
		OrgID:  ids.OrgID,
		UserID: ids.UserID,
	}
	return m.getSavedMessagesWithFilter(db, filter)
}

func (m *SavedMessage) GetCompletedSavedMessages(db *gorm.DB, orgID, userID string) ([]SavedMessagesResp, error) {
	completed := true
	filter := SavedMessageFilter{
		OrgID:     orgID,
		UserID:    userID,
		Completed: &completed,
	}
	return m.getSavedMessagesWithFilter(db, filter)
}

func (m *SavedMessage) GetArchivedSavedMessages(db *gorm.DB, orgID, userID string) ([]SavedMessagesResp, error) {
	archived := true
	filter := SavedMessageFilter{
		OrgID:    orgID,
		UserID:   userID,
		Archived: &archived,
	}
	return m.getSavedMessagesWithFilter(db, filter)
}

func (m *SavedMessage) getSavedMessagesWithFilter(db *gorm.DB, filter SavedMessageFilter) ([]SavedMessagesResp, error) {
	var org Organisation
	var organisation *Organisation

	if !postgresql.CheckExists(db, &org, "id = ?", filter.OrgID) {
		return nil, errors.New("organisation not found")
	}

	isMember, err := organisation.CheckUserIsMemberOfOrg(filter.UserID, filter.OrgID, db)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, errors.New("user is not a member of organisation")
	}

	messages, err := m.queryMessages(db, filter)
	if err != nil {
		return nil, err
	}

	return m.processMessages(db, messages)
}

func (m *SavedMessage) queryMessages(db *gorm.DB, filter SavedMessageFilter) ([]SavedMessage, error) {
	var messages []SavedMessage
	query := "org_id = ? AND user_id = ?"
	args := []any{filter.OrgID, filter.UserID}

	if filter.Completed != nil {
		query += " AND completed = ?"
		args = append(args, *filter.Completed)
	}

	if filter.Archived != nil {
		query += " AND archived = ?"
		args = append(args, *filter.Archived)
	}

	err := postgresql.SelectAllFromDb(db, "", &messages, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve saved messages: %w", err)
	}

	return messages, nil
}

func (m *SavedMessage) processMessages(db *gorm.DB, messages []SavedMessage) ([]SavedMessagesResp, error) {
	foundMsgs, notFoundMsgs := []SavedMessagesResp{}, []SavedMessagesResp{}

	for _, msg := range messages {
		resp, found := m.buildMessageResponse(db, msg)
		if found {
			foundMsgs = append(foundMsgs, resp)
		} else {
			notFoundMsgs = append(notFoundMsgs, resp)
		}
	}

	return append(foundMsgs, notFoundMsgs...), nil
}

func (m *SavedMessage) buildMessageResponse(db *gorm.DB, msg SavedMessage) (SavedMessagesResp, bool) {
	mr := SavedMessagesResp{
		ID:          msg.ID,
		ThreadID:    msg.ThreadID,
		SavedAt:     msg.CreatedAt,
		Overdue:     msg.RemainderAt != nil && msg.RemainderAt.Before(time.Now().UTC()),
		OverDueTime: msg.RemainderAt,
	}

	if msg.MessageID != nil {
		if !m.populateFromMessage(db, &mr, msg) {
			return m.createNotFoundResponse(), false
		}
	} else {
		if !m.populateFromThread(db, &mr, msg) {
			return m.createNotFoundResponse(), false
		}
	}

	return mr, true
}

func (m *SavedMessage) populateFromMessage(db *gorm.DB, mr *SavedMessagesResp, msg SavedMessage) bool {
	var msgDoc MessageDocument
	if err := msgDoc.GetMessageById(db, *msg.MessageID); err != nil {
		return false
	}

	mr.MessageID = msg.MessageID
	mr.AvatarURL = msgDoc.AvatarURL
	mr.Username = msgDoc.Username
	mr.Content = msgDoc.Content
	mr.UserID = msgDoc.UserID
	mr.Type = "message"
	mr.ChannelID = msgDoc.ChannelsID
	mr.ChannelName = resolveChannelName(db, msgDoc.ChannelsID)
	mr.ChannelType = resolveChannelType(db, msgDoc.ChannelsID, msg.UserID, msg.OrgId)

	return true
}

func (m *SavedMessage) populateFromThread(db *gorm.DB, mr *SavedMessagesResp, msg SavedMessage) bool {
	var thread ThreadDocument
	if err := thread.GetThreadById(msg.ThreadID); err != nil {
		return false
	}

	mr.MessageID = nil
	mr.AvatarURL = thread.AvatarURL
	mr.Username = thread.Username
	mr.Content = thread.Content
	mr.UserID = thread.UserId
	mr.Type = "thread"
	mr.ChannelID = thread.ChannelsID
	mr.ChannelName = resolveChannelName(db, thread.ChannelsID)
	mr.ChannelType = resolveChannelType(db, thread.ChannelsID, msg.UserID, msg.OrgId)

	return true
}

func (m *SavedMessage) createNotFoundResponse() SavedMessagesResp {
	return SavedMessagesResp{
		Content:   "A message you saved was not found.",
		AvatarURL: "",
	}
}

func resolveChannelName(db *gorm.DB, channelID string) string {
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

func resolveChannelType(db *gorm.DB, channelID, userID, orgID string) string {
	var (
		dmChannels   DmChannels
		userChannels UserChannels
	)

	chanExist := postgresql.CheckExists(db, &userChannels, "channels_id = ? AND user_id = ?", channelID, userID)
	dmChanExist := postgresql.CheckExists(db, &dmChannels, "channel_id = ?", channelID)

	if !dmChanExist && !chanExist {
		return "unknown"
	}

	if dmChanExist {
		return resolveDMChannelType(db, channelID)
	}

	if chanExist {
		return resolveRegularChannelType(db, channelID, orgID)
	}

	return "unknown"
}

func resolveDMChannelType(db *gorm.DB, channelID string) string {
	var dmChannel DmChannels
	if !postgresql.CheckExists(db, &dmChannel, "channel_id = ?", channelID) {
		return "unknown"
	}

	if dmChannel.ChatType == "bot" {
		return "Bot"
	}

	if dmChannel.ChannelType == "dm" {
		return "Direct Message"
	}

	return "Group Direct Message"
}

func resolveRegularChannelType(db *gorm.DB, channelID, orgID string) string {
	var channels Channels
	if !postgresql.CheckExists(db, &channels, "id = ? AND organisation_id = ?", channelID, orgID) {
		return "unknown"
	}

	if channels.IsPrivate {
		return "Private Channel"
	}

	return channels.Name
}
