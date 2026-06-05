package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/utility"
)

type NotificationType string
type SectionType string
type ChannelType string

var (
	Updated                     NotificationType = "updated"
	AgentUpdate                 NotificationType = "agent_update"
	AgentProcessingStarted      NotificationType = "agent_processing_started"
	AgentProcessingFinished     NotificationType = "agent_processing_finished"
	AgentErrorOccured           NotificationType = "agent_error_occured"
	AgentToolCallStarted        NotificationType = "agent_tool_call_started"
	AgentToolCallCompleted      NotificationType = "agent_tool_call_completed"
	Deleted                     NotificationType = "deleted"
	UpdatedMedia                NotificationType = "updated_media"
	NewMessage                  NotificationType = "new_message"
	StatusUpdate                NotificationType = "status_update"
	ProfileStatusUpdated        NotificationType = "profile_status_updated"
	UserPresenceChanged         NotificationType = "user_presence_changed"
	UnReadThreadChange          NotificationType = "unread_thread_change"
	ChannelMention              NotificationType = "channel_mention"
	ThreadReply                 NotificationType = "thread_reply"
	ReplyCountChange            NotificationType = "reply_count_change"
	PinnedMessageEvent          NotificationType = "pinned_message_event"
	UnPinnedMessageEvent        NotificationType = "unpinned_message_event"
	SavedMessageEvent           NotificationType = "saved_message_event"
	UnSavedMessageEvent         NotificationType = "unsaved_message_event"
	ReactionEvent               NotificationType = "reaction_event"
	SavedMessageRemainder       NotificationType = "saved_message_remainder"
	SavedMessageCompletionEvent NotificationType = "saved_message_completion_event"
	ArchiveSavedMessage         NotificationType = "archive_saved_message"
	UnArchiveSavedMessage       NotificationType = "unarchive_saved_message"
	BuzzStarted                 NotificationType = "buzz_started"
	UserJoinedBuzz              NotificationType = "user_joined_buzz"
	UserLeftBuzz                NotificationType = "user_left_buzz"
	BuzzEnded                   NotificationType = "buzz_ended"
	BuzzTimeWarning             NotificationType = "buzz_time_warning"
	CameraStatusChanged         NotificationType = "camera_status_changed"
	BuzzReactionEvent           NotificationType = "buzz_reaction_event"
	BuzzStickerEvent            NotificationType = "buzz_sticker_event"
	ThreadNotification          NotificationType = "thread_notification"
	MuteParticipants            NotificationType = "mute_participants"
	DirectCallInitiated         NotificationType = "direct_call_initiated"
	DirectCallResponseEvent     NotificationType = "direct_call_response"
	DirectCallCanceled          NotificationType = "direct_call_canceled"
	TriggerNotification         NotificationType = "trigger_notification"
	ThreadSection               SectionType      = "thread_message"
	ReplySection                SectionType      = "reply_message"
	ChannelsSection             SectionType      = "channels_section"
	DmChannelsSection           SectionType      = "dm_channels_section"
	AgentChannelsSection        SectionType      = "agent_channels_section"
	OrganisationUsersSection    SectionType      = "org_users_section"
	Channel                     ChannelType      = "channel"
	DMChannel                   ChannelType      = "dm_channel"
	GroupDMChannel              ChannelType      = "group_dm_channel"
	ThreadChannel               ChannelType      = "thread_channel"
)

type Content struct {
	NotificationType    NotificationType     `json:"notification_type"`
	SectionType         SectionType          `json:"section"`
	ModificationDetails *ModificationDetails `json:"modification_ids,omitempty"`
	Content             any                  `json:"data,omitempty"`
	UpdateChange        map[string]any       `json:"update_change,omitempty"`
	PinnedDetails       *PinnedDetails       `json:"pinned_details,omitempty"`
	Reactions           *[]ReactionDetails   `json:"reactions,omitempty"`
	NotificationId      string               `json:"notification_id,omitempty"`
}

type ModificationDetails struct {
	ThreadId  string `json:"thread_id,omitempty"`
	MessageId string `json:"message_id,omitempty"`
	ChannelId string `json:"channel_id,omitempty"`
	UserId    string `json:"user_id,omitempty"`
	OrgId     string `json:"org_id,omitempty"`
}

type ProfileStatusUpdatePayload struct {
	Text          string `json:"text"`
	Icon          string `json:"icon"`
	Username      string `json:"username"`
	Email         string `json:"email"`
	ProfilePic    string `json:"profile_pic"`
	StatusTimeout string `json:"status_timeout"`
	Online        bool   `json:"online"`
}

type UserPresenceChangedPayload struct {
	UserID string `json:"user_id"`
	Online bool   `json:"online"`
}

type ToolCallNotification struct {
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments"`
	Status    string                 `json:"status"`
	Result    interface{}            `json:"result"`
	Error     *string                `json:"error"`
}

type TriggerNotificationPayload struct {
	TriggerAction   string `json:"trigger_action"`
	TargetComponent string `json:"target_component,omitempty"`
	Data            any    `json:"data,omitempty"`
}

var Notification = map[NotificationType]Content{

	Updated: Content{
		NotificationType: Updated,
		Content:          ModificationDetails{},
	},

	AgentUpdate: Content{
		NotificationType: AgentUpdate,
		Content:          ModificationDetails{},
	},

	AgentProcessingStarted: Content{
		NotificationType: AgentProcessingStarted,
		Content:          ModificationDetails{},
	},

	AgentProcessingFinished: Content{
		NotificationType: AgentProcessingFinished,
		Content:          ModificationDetails{},
	},

	AgentErrorOccured: Content{
		NotificationType: AgentErrorOccured,
		Content:          ModificationDetails{},
	},

	AgentToolCallStarted: Content{
		NotificationType: AgentToolCallStarted,
		Content:          ModificationDetails{},
	},

	AgentToolCallCompleted: Content{
		NotificationType: AgentToolCallCompleted,
		Content:          ModificationDetails{},
	},

	Deleted: Content{
		NotificationType: Deleted,
		Content:          ModificationDetails{},
	},

	NewMessage: Content{
		NotificationType: NewMessage,
	},
	ProfileStatusUpdated: Content{
		NotificationType: ProfileStatusUpdated,
	},
	UserPresenceChanged: Content{
		NotificationType: UserPresenceChanged,
	},
	UnReadThreadChange: Content{
		NotificationType: UnReadThreadChange,
	},
	ReplyCountChange: Content{
		NotificationType: ReplyCountChange,
	},
	PinnedMessageEvent: Content{
		NotificationType: PinnedMessageEvent,
		Content:          ModificationDetails{},
	},
	UnPinnedMessageEvent: Content{
		NotificationType: UnPinnedMessageEvent,
		Content:          ModificationDetails{},
	},
	SavedMessageEvent: Content{
		NotificationType: SavedMessageEvent,
		Content:          ModificationDetails{},
	},
	UnSavedMessageEvent: Content{
		NotificationType: UnSavedMessageEvent,
		Content:          ModificationDetails{},
	},
	ReactionEvent: Content{
		NotificationType: ReactionEvent,
		Content:          ModificationDetails{},
	},
	ChannelMention: Content{
		NotificationType: ChannelMention,
	},
	ThreadReply: Content{
		NotificationType: ThreadReply,
	},
	SavedMessageRemainder: Content{
		NotificationType: SavedMessageRemainder,
	},
	SavedMessageCompletionEvent: Content{
		NotificationType: SavedMessageCompletionEvent,
	},
	ArchiveSavedMessage: Content{
		NotificationType: ArchiveSavedMessage,
	},
	UnArchiveSavedMessage: Content{
		NotificationType: UnArchiveSavedMessage,
	},
	BuzzStarted: Content{
		NotificationType: BuzzStarted,
	},
	UserJoinedBuzz: Content{
		NotificationType: UserJoinedBuzz,
	},
	BuzzEnded: Content{
		NotificationType: BuzzEnded,
	},
	BuzzTimeWarning: Content{
		NotificationType: BuzzTimeWarning,
	},
	UpdatedMedia: Content{
		NotificationType: UpdatedMedia,
		Content:          ModificationDetails{},
	},
	CameraStatusChanged: Content{
		NotificationType: CameraStatusChanged,
	},
	BuzzReactionEvent: Content{
		NotificationType: BuzzReactionEvent,
	},
	BuzzStickerEvent: Content{
		NotificationType: BuzzStickerEvent,
	},
	ThreadNotification: Content{
		NotificationType: ThreadNotification,
	},
	MuteParticipants: Content{
		NotificationType: MuteParticipants,
	},
	DirectCallInitiated: Content{
		NotificationType: DirectCallInitiated,
	},
	DirectCallResponseEvent: Content{
		NotificationType: DirectCallResponseEvent,
	},
	DirectCallCanceled: Content{
		NotificationType: DirectCallCanceled,
	},
	TriggerNotification: Content{
		NotificationType: TriggerNotification,
		Content:          ModificationDetails{},
	},
}

type DeviceNotificationSettings struct {
	DeviceNotification
	ChannelsID string `json:"channels_id"`
	UserID     string `json:"user_id"`
	DeviceType string `json:"device_type" validate:"required,oneof=web mobile desktop"`
	OrgID      string `json:"organisation_id"`
}

type NotificationOption string

const (
	AllMessages    NotificationOption = "all_new_messages"
	DirectMentions NotificationOption = "mentions"
	Nothing        NotificationOption = "nothing"
)

type DeviceNotification struct {
	Muted       bool               `json:"muted"`
	AtMentions  bool               `json:"at_mentions"`
	NotifyAbout NotificationOption `json:"notify_about,omitempty"`
	AtChannel   bool               `json:"at_channel"`
	SendMail    bool               `json:"send_mail"`
	TimeRange   string             `json:"time_range,omitempty"`
}

func GetDefaultChannelDeviceNotification() DeviceNotification {
	return DeviceNotification{
		Muted:       false,
		AtMentions:  true,
		AtChannel:   true,
		NotifyAbout: AllMessages,
	}
}

func GetDefaultOrgDeviceNotification() DeviceNotification {
	return DeviceNotification{
		NotifyAbout: AllMessages,
		TimeRange:   "12:00 AM - 11:59 PM",
	}
}

type ChannelNotificationInfo struct {
	ChannelsID  string `json:"channels_id"`
	ChannelName string `json:"channel_name"`
	Muted       bool   `json:"muted"`
}

type NotificationPreference map[string]DeviceNotification

func (n *NotificationPreference) Scan(value any) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, &n)
}

func (n NotificationPreference) Value() (driver.Value, error) {
	return json.Marshal(n)
}

func (n *DeviceNotificationSettings) UpdateDeviceNotification(db *gorm.DB) (DeviceNotification, int, error) {
	var pref UserChannels
	exist := postgresql.CheckExists(db, &pref, "channels_id = ? AND user_id = ?", n.ChannelsID, n.UserID)

	if !exist {
		return DeviceNotification{}, http.StatusBadRequest, fmt.Errorf("entry does not exist")
	}

	if pref.Preferences == nil {
		pref.Preferences = make(NotificationPreference)
	}

	deviceSettings := pref.Preferences[n.DeviceType]

	if deviceSettings.NotifyAbout == "" {
		deviceSettings = GetDefaultChannelDeviceNotification()
	}

	deviceSettings.AtChannel = n.AtChannel
	deviceSettings.AtMentions = n.AtMentions
	deviceSettings.Muted = n.Muted
	deviceSettings.NotifyAbout = n.NotifyAbout

	pref.Preferences[n.DeviceType] = deviceSettings

	if err := db.Save(&pref).Error; err != nil {
		return DeviceNotification{}, http.StatusBadRequest, fmt.Errorf("failed to save entry")
	}

	return deviceSettings, http.StatusOK, nil
}

func (n *DeviceNotificationSettings) GetOrCreateDeviceNotification(db *gorm.DB) (DeviceNotification, error) {
	var pref UserChannels

	exist := postgresql.CheckExists(db, &pref, "channels_id = ? AND user_id = ?", n.ChannelsID, n.UserID)

	if !exist {
		return DeviceNotification{}, fmt.Errorf("entry does not exist")
	}

	// Initialize if nil
	if pref.Preferences == nil {
		pref.Preferences = make(NotificationPreference)
	}

	deviceSettings, ok := pref.Preferences[n.DeviceType]
	if !ok {
		deviceSettings = GetDefaultChannelDeviceNotification()
		pref.Preferences[n.DeviceType] = deviceSettings

		if err := db.Save(&pref).Error; err != nil {
			return DeviceNotification{}, err
		}
	}

	return deviceSettings, nil
}

func (c *UserChannels) FetchChannelUsersNotificationPref(db *gorm.DB, logger *utility.Logger) ([]UserChannels, error) {
	var userChannel []UserChannels

	err := db.Preload("NotificationPref").Order("id").Where("channels_id = ?", c.ChannelsID).Find(userChannel).Error

	if err != nil {
		logger.Error("Error fetching:", err)
		return userChannel, err
	}

	return userChannel, nil
}

func (n *DeviceNotificationSettings) GetUserChannelsNotificationPrefs(db *gorm.DB, ids map[string]string) ([]ChannelNotificationInfo, error) {

	result := []ChannelNotificationInfo{}

	err := db.
		Table("user_channels AS uc").
		Select(`
			uc.channels_id,
			c.name AS channel_name,
			COALESCE((uc.preferences->?->>'muted')::boolean, false) AS muted
		`, ids["device_type"]).
		Joins("JOIN channels c ON c.id = uc.channels_id").
		Where("c.organisation_id = ? AND uc.user_id = ?", ids["org_id"], ids["user_id"]).
		Scan(&result).Error

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (n *DeviceNotificationSettings) UpdateDeviceOrgNotification(db *gorm.DB) (DeviceNotification, int, error) {
	var pref OrgUserManagement
	exist := postgresql.CheckExists(db, &pref, "organisation_id = ? AND user_id = ?", n.OrgID, n.UserID)
	if !exist {
		return DeviceNotification{}, http.StatusBadRequest, fmt.Errorf("entry does not exist")
	}

	if pref.Preferences == nil {
		pref.Preferences = make(NotificationPreference)
	}

	deviceSettings := pref.Preferences[n.DeviceType]

	if deviceSettings.NotifyAbout == "" {
		deviceSettings = GetDefaultOrgDeviceNotification()
	}

	deviceSettings.NotifyAbout = n.NotifyAbout
	deviceSettings.SendMail = n.SendMail
	deviceSettings.TimeRange = n.TimeRange

	pref.Preferences[n.DeviceType] = deviceSettings

	if err := db.Save(&pref).Error; err != nil {
		return DeviceNotification{}, http.StatusBadRequest, fmt.Errorf("failed to save entry")
	}

	return deviceSettings, http.StatusOK, nil
}

func (n *DeviceNotificationSettings) GetOrCreateDeviceOrgNotification(db *gorm.DB) (DeviceNotification, error) {

	var pref OrgUserManagement
	exist := postgresql.CheckExists(db, &pref, "organisation_id = ? AND user_id = ?", n.OrgID, n.UserID)

	if !exist {
		return DeviceNotification{}, fmt.Errorf("entry does not exist")
	}

	// Initialize if nil
	if pref.Preferences == nil {
		pref.Preferences = make(NotificationPreference)
	}

	deviceSettings, ok := pref.Preferences[n.DeviceType]
	if !ok {
		deviceSettings = GetDefaultOrgDeviceNotification()
		pref.Preferences[n.DeviceType] = deviceSettings

		if err := db.Save(&pref).Error; err != nil {
			return DeviceNotification{}, err
		}
	}

	return deviceSettings, nil
}
