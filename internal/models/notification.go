package models

import (
	"encoding/json"
	"fmt"

	"github.com/go-redis/redis/v8"

	dbRedis "github.com/hngprojects/telex_be/pkg/repository/storage/redis"
)

type NotificationRecord struct {
	Name string `json:"name"`
	Data string `json:"data"`
	Sent bool   `json:"sent"`
}

type SendOTP struct {
	Email    string `json:"email"  validate:"required"`
	OtpToken int    `json:"otp_token"  validate:"required"`
}

type SendWelcomeMail struct {
	Email string `json:"email"  validate:"required"`
}

type SendEmailVerificationMail struct {
	Email string `json:"email"`
	Code  uint   `json:"code" validate:"required"`
	Token string `json:"token"`
}

type SendResetPassword struct {
	Email string `json:"email"`
	Token int    `json:"token"  validate:"required"`
}

type SendInvitationLink struct {
	Email            string `json:"email" validate:"required"`
	InvitationLink   string `json:"invitation_link" validate:"required"`
	InviterName      string `json:"inviter_name" validate:"required"`
	OrganisationName string `json:"organisation_name" validate:"required"`
}

type SendMagicLink struct {
	Email     string `json:"email"  validate:"required"`
	MagicLink string `json:"magic_link"  validate:"required"`
}

type SendSqueeze struct {
	Email     string `json:"email"  validate:"required"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
}
type SendContactUsMail struct {
	Name        string `json:"name"  validate:"required"`
	Email       string `json:"email" validate:"required"`
	PhoneNumber string `json:"phone_number" validate:"required"`
	Message     string `json:"message" validate:"required"`
}

type SendNewsletterSubscriptionMail struct {
	Email string `json:"email"  validate:"required"`
}

type SendPasswordChangeConfirmationMail struct {
	Email string `json:"email"  validate:"required"`
}
type SendLoginAlertMail struct {
	Email string `json:"email"  validate:"required"`
}

type PushNotificationRecord struct {
	ChannelType         ChannelType         `json:"channel_type"`
	Section             SectionType         `json:"section"`
	Data                string              `json:"data"`
	ChannelId           string              `json:"channel_id"`
	Sent                bool                `json:"sent"`
	UpdateChange        map[string]any      `json:"update_change"`
	Type                NotificationType    `json:"type"`
	ModificationDetails ModificationDetails `json:"modificaion_details"`
}

type NotificationProcessPayload struct {
	Notification Content
	OrgId        string
	ChannelId    string
	UserId       string
	ChannelType  ChannelType
}

func (n *NotificationRecord) PushToQueue(rdb *redis.Client) error {
	err := dbRedis.PushToQueue(rdb, &n)

	if err != nil {
		return err
	}

	return nil
}

func (n *NotificationRecord) PopFromQueue(rdb *redis.Client) (NotificationRecord, error) {
	var rec NotificationRecord
	res, err := dbRedis.PopFromQueue(rdb)

	if err != nil {
		return rec, err
	}

	resJSON, err := json.Marshal(res)
	if err != nil {
		return rec, fmt.Errorf("error marshaling map: %v", err)
	}

	err = json.Unmarshal(resJSON, &rec)
	if err != nil {
		return rec, fmt.Errorf("error unmarshaling JSON: %v", err)
	}

	return rec, nil
}

func (n *PushNotificationRecord) PushToQueue(rdb *redis.Client) error {
	err := dbRedis.PushToNotificationQueue(rdb, &n)

	if err != nil {
		return err
	}

	return nil
}

func (n *PushNotificationRecord) PopFromQueue(rdb *redis.Client) (PushNotificationRecord, error) {
	var rec PushNotificationRecord
	res, err := dbRedis.PopFromNotificationQueue(rdb)

	if err != nil {
		return rec, err
	}

	resJSON, err := json.Marshal(res)
	if err != nil {
		return rec, fmt.Errorf("error marshaling map: %v", err)
	}

	err = json.Unmarshal(resJSON, &rec)
	if err != nil {
		return rec, fmt.Errorf("error unmarshaling JSON: %v", err)
	}

	return rec, nil
}
