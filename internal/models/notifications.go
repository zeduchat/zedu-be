package models

type NotificationType string
type SectionType string

var (
	Updated       NotificationType = "updated"
	Deleted       NotificationType = "deleted"
	NewMessage    NotificationType = "new_message"
	ThreadSection SectionType      = "thread_message"
	ReplySection  SectionType      = "reply_message"
)

type Content struct {
	NotificationType   NotificationType   `json:"notification_type"`
	SectionType        SectionType        `json:"section"`
	ModifcationDetails ModifcationDetails `json:"modification_ids,omitempty"`
	Content            interface{}        `json:"data,omitempty"`
}

type ModifcationDetails struct {
	ThreadId  string `json:"thread_id,omitempty"`
	MessageId string `json:"message_id,omitempty"`
	ChannelId string `json:"channel_id,omitempty"`
	UserId    string `json:"user_id,omitempty"`
	OrgId     string `json:"org_id,omitempty"`
}

var Notification = map[NotificationType]Content{

	Updated: Content{
		NotificationType: Updated,
		Content:          ModifcationDetails{},
	},

	Deleted: Content{
		NotificationType: Deleted,
		Content:          ModifcationDetails{},
	},

	NewMessage: Content{
		NotificationType: NewMessage,
	},
}
