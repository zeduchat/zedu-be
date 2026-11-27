package test_webhook

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	webhooksvc "github.com/hngprojects/telex_be/services/webhook"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

// TestPostWebhookMissingChannel checks that a webhook whose channel is gone
// gets a controlled 404, and a database failure while loading the channel a
// controlled 500, instead of a panic or a generic 500 (#2501).
func TestPostWebhookMissingChannel(t *testing.T) {
	logger := tst.Setup()
	db := storage.Connection()

	slug := utility.GenerateUUID()[:12]
	hook := models.Webhook{
		ID:          utility.GenerateUUID(),
		WebhookName: "missing-channel",
		Status:      "active",
		OwnerId:     utility.GenerateUUID(),
		WebhookUrl:  "https://example.com/" + slug,
		WebhookSlug: slug,
		ChannelId:   utility.GenerateUUID(), // no such channel
	}
	require.NoError(t, db.Postgresql.Create(&hook).Error)
	t.Cleanup(func() { db.Postgresql.Unscoped().Where("id = ?", hook.ID).Delete(&models.Webhook{}) })

	req := models.CreateWebhookHistoryRequest{
		WebhookSlug: slug,
		EventName:   "deploy",
		UserName:    "ci",
		Status:      "success",
		Message:     "done",
	}

	t.Run("missing channel returns 404", func(t *testing.T) {
		var (
			code int
			err  error
		)
		require.NotPanics(t, func() {
			_, code, err = webhooksvc.PostWebhook(db, logger, req)
		})
		require.Error(t, err)
		assert.Equal(t, http.StatusNotFound, code)
		assert.Equal(t, "channel does not exist", err.Error())
	})

	t.Run("database failure returns 500", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		broken := *db
		broken.Postgresql = db.Postgresql.WithContext(ctx)

		var (
			code int
			err  error
		)
		require.NotPanics(t, func() {
			_, code, err = webhooksvc.PostWebhook(&broken, logger, req)
		})
		require.Error(t, err)
		assert.Equal(t, http.StatusInternalServerError, code)
	})
}
