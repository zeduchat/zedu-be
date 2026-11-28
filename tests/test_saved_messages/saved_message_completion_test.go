package test_saved_messages

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

// TestSavedMessageCompletionToggle checks that a saved message can move
// between complete and incomplete in both directions, and that repeating the
// current state is a safe no-op (#1717).
func TestSavedMessageCompletionToggle(t *testing.T) {
	logger := tst.Setup()
	db := storage.Connection()

	m := models.SavedMessage{
		ID:         utility.GenerateUUID(),
		UserID:     utility.GenerateUUID(),
		OrgId:      utility.GenerateUUID(),
		ChannelsID: utility.GenerateUUID(),
		ThreadID:   utility.GenerateUUID(),
		Type:       "thread",
	}
	require.NoError(t, db.Postgresql.Create(&m).Error)
	t.Cleanup(func() { db.Postgresql.Unscoped().Where("id = ?", m.ID).Delete(&models.SavedMessage{}) })

	for _, desired := range []bool{true, true, false, false, true} {
		var receiver models.SavedMessage
		want := desired
		req := models.MarkCompleteSavedMessageRequest{SavedMessageID: m.ID, UserId: m.UserID, OrgId: m.OrgId, Completed: &want}
		status, err := receiver.UpdateSavedMessageCompletion(db.Postgresql, logger, req, map[string]any{"completed": req.Completed})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)

		var stored models.SavedMessage
		require.NoError(t, db.Postgresql.First(&stored, "id = ?", m.ID).Error)
		assert.Equal(t, desired, stored.Completed, "requested completed=%t was not persisted", desired)
	}

	t.Run("other user's saved message is not found", func(t *testing.T) {
		var receiver models.SavedMessage
		want := false
		req := models.MarkCompleteSavedMessageRequest{SavedMessageID: m.ID, UserId: utility.GenerateUUID(), OrgId: m.OrgId, Completed: &want}
		status, err := receiver.UpdateSavedMessageCompletion(db.Postgresql, logger, req, map[string]any{"completed": req.Completed})
		assert.Error(t, err)
		assert.Equal(t, http.StatusNotFound, status)
	})
}
