package test_webhook

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/thread"
)

func TestBuildWebhookNotificationRecord(t *testing.T) {
	feed := models.FeedMessageRequest{
		ChannelID:   "11111111-1111-1111-1111-111111111111",
		ChannelName: "alerts",
		OrgId:       "22222222-2222-2222-2222-222222222222",
		UserId:      "WEBHOOK",
		Content:     "deploy finished",
	}

	rec := thread.BuildWebhookNotificationRecord(feed)

	assert.Equal(t, models.Channel, rec.ChannelType)
	assert.Equal(t, models.NewMessage, rec.Type)
	assert.Equal(t, feed.ChannelID, rec.ChannelId)
	assert.Empty(t, rec.UserIds)

	var data models.FeedMessageRequest
	require.NoError(t, json.Unmarshal([]byte(rec.Data), &data))
	assert.Equal(t, "00000000-0000-0000-0000-000000000000", data.UserId)
	assert.Equal(t, "Webhook", data.UserName)
	assert.Equal(t, "message", data.Type)
	assert.Equal(t, feed.OrgId, data.OrgId)
	assert.Equal(t, feed.Content, data.Content)
}
