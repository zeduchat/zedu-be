package test_onesignal

import (
	"testing"

	"github.com/hngprojects/telex_be/services/notification_pref"
	pushNotifications "github.com/hngprojects/telex_be/services/pushNotifications"
)

func TestResolveNotifType(t *testing.T) {
	tests := []struct {
		name    string
		payload any
		want    notificationpref.NotificationType
	}{
		{
			name: "Payload with notification_type = dm",
			payload: map[string]interface{}{
				"notification_type": "dm",
				"message":           "Hello DM",
			},
			want: notificationpref.NotificationTypeDirectMessage,
		},
		{
			name: "Payload with notification_type = channel",
			payload: map[string]interface{}{
				"notification_type": "channel",
				"message":           "Hello Channel",
			},
			want: notificationpref.NotificationTypeAllMessages,
		},
		{
			name: "Payload without notification_type defaults to AllMessages",
			payload: map[string]interface{}{
				"message": "Hello Generic",
			},
			want: notificationpref.NotificationTypeAllMessages,
		},
		{
			name:    "Non-map payload defaults to AllMessages",
			payload: "invalid_payload_string",
			want:    notificationpref.NotificationTypeAllMessages,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pushNotifications.ResolveNotifType(tt.payload)
			if got != tt.want {
				t.Errorf("ResolveNotifType() = %v; want %v", got, tt.want)
			}
		})
	}
}
