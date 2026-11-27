package test_agents

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	agentsvc "github.com/hngprojects/telex_be/services/agents"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

// TestSlashCommandUsernameOrgScope checks that slash commands resolve
// @usernames only among members of the caller's organisation (#2502).
func TestSlashCommandUsernameOrgScope(t *testing.T) {
	tst.Setup()
	db := storage.Connection().Postgresql
	tag := utility.GenerateUUID()[:8]

	var users, orgs, channels []string
	t.Cleanup(func() {
		db.Exec("DELETE FROM user_channels WHERE channels_id IN ?", channels)
		db.Exec("DELETE FROM channels WHERE id IN ?", channels)
		db.Exec("DELETE FROM profiles WHERE userid IN ?", users)
		db.Exec("DELETE FROM user_organisations WHERE user_id IN ?", users)
		db.Exec("DELETE FROM organisations WHERE id IN ?", orgs)
		db.Exec("DELETE FROM users WHERE id IN ?", users)
	})

	newOrg := func() string {
		id := utility.GenerateUUID()
		require.NoError(t, db.Exec("INSERT INTO organisations (id, name, email, created_at) VALUES (?, ?, ?, now())", id, "slash-"+tag, fmt.Sprintf("slash_%s_%s@qa.team", tag, id[:4])).Error)
		orgs = append(orgs, id)
		return id
	}
	// newMember creates a user with a profile username in profileOrg and makes
	// them a member of memberOrg.
	newMember := func(username, profileOrg, memberOrg string) string {
		id := utility.GenerateUUID()
		require.NoError(t, db.Exec("INSERT INTO users (id, name, email, password, created_at) VALUES (?, ?, ?, '', now())", id, username, fmt.Sprintf("slash_%s@qa.team", id)).Error)
		users = append(users, id)
		require.NoError(t, db.Exec("INSERT INTO profiles (id, first_name, last_name, user_name, userid, organisation_id, created_at) VALUES (?, 'slash', 'user', ?, ?, ?, now())", utility.GenerateUUID(), username, id, profileOrg).Error)
		require.NoError(t, db.Exec("INSERT INTO user_organisations (user_id, organisation_id) VALUES (?, ?)", id, memberOrg).Error)
		return id
	}
	newChannel := func(orgID string, members ...string) string {
		id := utility.GenerateUUID()
		name := "slash-" + tag + "-" + id[:4]
		require.NoError(t, db.Exec(`INSERT INTO channels (id, name, description, organisation_id, created_at, " deleted_at") VALUES (?, ?, 'test', ?, now(), now())`, id, name, orgID).Error)
		channels = append(channels, id)
		for _, m := range members {
			require.NoError(t, db.Exec("INSERT INTO user_channels (channels_id, user_id, created_at) VALUES (?, ?, now())", id, m).Error)
		}
		return name
	}

	orgA, orgB := newOrg(), newOrg()
	dup := "dup_" + tag
	dupA := newMember(dup, orgA, orgA)
	_ = newMember(dup, orgB, orgB)
	outsider := "outsider_" + tag
	newMember(outsider, orgB, orgB)
	ambig := "ambig_" + tag
	newMember(ambig, orgA, orgA)
	newMember(ambig, orgA, orgA)

	// dupA is already in the channel, so resolving @dup to A's user stops at
	// "user already in channel" without touching the channel.
	chanA := newChannel(orgA, dupA)

	run := func(username string) models.AddToChannelResponse {
		resp, err := agentsvc.ProcessSlashCommand(db, models.ProcessSlashCommandRequest{
			Command: fmt.Sprintf("/add-to-channel #%s @%s", chanA, username),
			Context: map[string]interface{}{"org_id": orgA},
		})
		require.NoError(t, err)
		out, ok := resp.(models.AddToChannelResponse)
		require.True(t, ok)
		require.Len(t, out.FailedUsers, 1)
		return out
	}

	t.Run("duplicate username resolves to the caller org's member", func(t *testing.T) {
		assert.Equal(t, "user already in channel", run(dup).FailedUsers[0].Error)
	})
	t.Run("user from another org is refused", func(t *testing.T) {
		assert.Contains(t, run(outsider).FailedUsers[0].Error, "not found in this organisation")
		var n int64
		db.Table("user_channels").Where("channels_id IN ?", channels).Count(&n)
		assert.Equal(t, int64(1), n, "no one may be added")
	})
	t.Run("ambiguous username in the org is refused", func(t *testing.T) {
		assert.Contains(t, run(ambig).FailedUsers[0].Error, "ambiguous")
	})
	t.Run("missing org context is refused", func(t *testing.T) {
		_, err := agentsvc.ProcessSlashCommand(db, models.ProcessSlashCommandRequest{
			Command: fmt.Sprintf("/add-to-channel #%s @%s", chanA, dup),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "organization context is required")
	})
}
