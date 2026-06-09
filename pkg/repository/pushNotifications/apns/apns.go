package apns

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/token"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
)

type Client struct {
	client *apns2.Client
	topic  string
}

type DirectCallInvitePayload struct {
	BuzzID           string           `json:"buzz_id"`
	HostID           string           `json:"host_id"`
	CallerID         string           `json:"caller_id"`
	CallerName       string           `json:"caller_name"`
	ChannelID        string           `json:"channel_id"`
	BuzzCode         string           `json:"buzz_code"`
	AvatarURL        string           `json:"avatar_url,omitempty"`
	DefaultAvatarURL string           `json:"default_avatar_url,omitempty"`
	Participants     []map[string]any `json:"participants,omitempty"`
}

type DirectCallCancelPayload struct {
	BuzzID     string `json:"buzz_id"`
	CallerID   string `json:"caller_id,omitempty"`
	CallerName string `json:"caller_name,omitempty"`
	JoinStatus string `json:"join_status,omitempty"`
}

type voipEnvelope struct {
	APS   map[string]any `json:"aps"`
	Event string         `json:"event"`
}

type SendResult struct {
	Success      bool
	StatusCode   int
	Reason       string
	Unregistered bool
}

var APNsClient *Client

func ConnectAPNs(logger *utility.Logger, cfg config.Apple) {
	if cfg.PRIVATE_KEY == "" || cfg.KEY_ID == "" || cfg.TEAM_ID == "" {
		logger.Warning("ConnectAPNs: APNs credentials are not fully configured. APNs client will not be initialized.")
		return
	}

	keyStr := strings.ReplaceAll(cfg.PRIVATE_KEY, "\\n", "\n")
	authKey, err := token.AuthKeyFromBytes([]byte(keyStr))
	if err != nil {
		logger.Error("ConnectAPNs: failed to parse private key: %v", err)
		return
	}

	apnsToken := &token.Token{
		AuthKey: authKey,
		KeyID:   cfg.KEY_ID,
		TeamID:  cfg.TEAM_ID,
	}

	apnsClient := apns2.NewTokenClient(apnsToken).Development()
	topic := cfg.CLIENT_ID + ".voip"

	APNsClient = &Client{
		client: apnsClient,
		topic:  topic,
	}
}

func (c *Client) SendDirectCallInvite(ctx context.Context, logger *utility.Logger, deviceToken string, payload DirectCallInvitePayload) (*SendResult, error) {
	body := map[string]any{
		"aps": map[string]any{
			"content-available": 1,
		},
		"event":              "direct_call_initiated",
		"buzz_id":            payload.BuzzID,
		"host_id":            payload.HostID,
		"caller_id":          payload.CallerID,
		"caller_name":        payload.CallerName,
		"channel_id":         payload.ChannelID,
		"buzz_code":          payload.BuzzCode,
		"avatar_url":         payload.AvatarURL,
		"default_avatar_url": payload.DefaultAvatarURL,
		"participants":       payload.Participants,
	}

	return c.send(ctx, logger, deviceToken, body)
}

func (c *Client) SendDirectCallCancel(ctx context.Context, logger *utility.Logger, deviceToken string, payload DirectCallCancelPayload) (*SendResult, error) {
	body := map[string]any{
		"aps": map[string]any{
			"content-available": 1,
		},
		"event":       "direct_call_canceled",
		"buzz_id":     payload.BuzzID,
		"caller_id":   payload.CallerID,
		"caller_name": payload.CallerName,
		"join_status": payload.JoinStatus,
	}

	return c.send(ctx, logger, deviceToken, body)
}

func (c *Client) send(ctx context.Context, logger *utility.Logger, deviceToken string, body map[string]any) (*SendResult, error) {
	if deviceToken == "" {
		return nil, fmt.Errorf("apns: empty device token")
	}

	raw, err := json.Marshal(body)
	if err != nil {
		logger.Error("apns: marshal payload: %v", err)
		return nil, fmt.Errorf("apns: marshal payload: %w", err)
	}

	notification := &apns2.Notification{
		DeviceToken: deviceToken,
		Topic:       c.topic,
		Payload:     raw,
		PushType:    apns2.PushTypeVOIP,
		Priority:    apns2.PriorityHigh,
	}

	resp, err := c.client.PushWithContext(ctx, notification)
	if err != nil {
		logger.Error("apns: push failed: %v", err)
		return nil, fmt.Errorf("apns: push failed: %w", err)
	}

	result := &SendResult{
		Success:    resp.Sent(),
		StatusCode: resp.StatusCode,
		Reason:     resp.Reason,
	}

	if resp.StatusCode == 410 || resp.Reason == apns2.ReasonUnregistered || resp.Reason == apns2.ReasonBadDeviceToken {
		result.Unregistered = true
	}

	if !result.Success {
		logger.Error("apns voip push rejected",
			"status", resp.StatusCode,
			"reason", resp.Reason,
			"apns_id", resp.ApnsID,
		)
	}

	logger.Info("apns: direct call voip notification successfully sent to %s: %v", deviceToken, resp)

	return result, nil
}

func SendDirectCallVoIPNotification(logger *utility.Logger, deviceTokens []string, req models.PushRequest, callData map[string]interface{}, db *gorm.DB, userIDs []string) error {
	if APNsClient == nil {
		return fmt.Errorf("apns: client not initialized")
	}

	event, _ := callData["event"].(string)
	ctx := context.Background()

	for _, token := range deviceTokens {
		go func(deviceToken string) {
			var result *SendResult
			var err error

			if event == "direct_call_initiated" {
				var invitePayload DirectCallInvitePayload
				payloadBytes, _ := json.Marshal(callData)
				json.Unmarshal(payloadBytes, &invitePayload)

				result, err = APNsClient.SendDirectCallInvite(ctx, logger, deviceToken, invitePayload)
			} else if event == "direct_call_canceled" {
				var cancelPayload DirectCallCancelPayload
				payloadBytes, _ := json.Marshal(callData)
				json.Unmarshal(payloadBytes, &cancelPayload)

				result, err = APNsClient.SendDirectCallCancel(ctx, logger, deviceToken, cancelPayload)
			} else {
				return
			}

			if err != nil {
				return
			}

			if result != nil && result.Unregistered {
				if db != nil {
					db.Model(&models.FcmTokens{}).
						Where("voip_token = ?", deviceToken).
						Update("voip_token", "")
				}
			}
		}(token)
	}

	return nil
}
