package test_onesignal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/services/notification_processor"
)

func TestResolveChannelPushTargetUserIDs(t *testing.T) {
	channelUsers := []string{"user-1", "user-2", "user-3"}

	t.Run("No Mentions - Returns Empty Targets", func(t *testing.T) {
		feed := models.FeedMessageRequest{
			Content: "Hello world without any mentions",
		}

		targets, err := notification_processor.ResolveChannelPushTargetUserIDs(nil, feed, channelUsers, "chan-1", "org-1", "sender-1")
		require.NoError(t, err)
		assert.Empty(t, targets, "expected 0 push targets for message with no mentions")
	})

	t.Run("Channel Mention via Mentions Slice", func(t *testing.T) {
		feed := models.FeedMessageRequest{
			Content: "Hello everyone @channel",
			Mentions: []models.Mention{
				{ID: "00000000-0000-0000-0000-000000000000", Type: "user"},
			},
		}

		targets, err := notification_processor.ResolveChannelPushTargetUserIDs(nil, feed, channelUsers, "chan-1", "org-1", "sender-1")
		require.NoError(t, err)
		assert.Len(t, targets, len(channelUsers), "expected all channel users for @channel mention")
	})

	t.Run("Specific Tagged User via Mentions Slice", func(t *testing.T) {
		feed := models.FeedMessageRequest{
			Content: "Hey check this out",
			Mentions: []models.Mention{
				{ID: "user-2", Type: "user"},
			},
		}

		targets, err := notification_processor.ResolveChannelPushTargetUserIDs(nil, feed, channelUsers, "chan-1", "org-1", "sender-1")
		require.NoError(t, err)
		assert.Equal(t, []string{"user-2"}, targets)
	})

	t.Run("Specific Tagged User via Mentions Slice", func(t *testing.T) {
		feed := models.FeedMessageRequest{
			Content: `<p>@alice please check</p>`,
			Mentions: []models.Mention{
				{ID: "user-3", Type: "user"},
			},
		}

		targets, err := notification_processor.ResolveChannelPushTargetUserIDs(nil, feed, channelUsers, "chan-1", "org-1", "sender-1")
		require.NoError(t, err)
		assert.Equal(t, []string{"user-3"}, targets)
	})
}
