package test_channel

import (
	"testing"

	"github.com/hngprojects/telex_be/internal/models"
)

func TestMergedChannelJoinAndLeaveMessages(t *testing.T) {
	users := []models.UserMention{
		{UserID: "u1", Username: "elizabeth"},
		{UserID: "u2", Username: "Godwin"},
		{UserID: "u3", Username: "john"},
		{UserID: "u4", Username: "sarah"},
	}

	adder := &models.UserMention{UserID: "admin1", Username: "edith"}

	t.Run("1 user self-joined", func(t *testing.T) {
		got := models.FormatMergedChannelJoinMessage(users[:1], nil, "general")
		expected := `<p><span class="mention" data-type="mention" data-id="u1" data-label="elizabeth" data-mention-suggestion-char="@">@elizabeth</span> joined this channel</p><p></p>`
		if got != expected {
			t.Errorf("got %q, want %q", got, expected)
		}
	})

	t.Run("1 user added by inviter", func(t *testing.T) {
		got := models.FormatMergedChannelJoinMessage(users[:1], adder, "general")
		expected := `<p><span class="mention" data-type="mention" data-id="u1" data-label="elizabeth" data-mention-suggestion-char="@">@elizabeth</span> has been added to this channel by <span class="mention" data-type="mention" data-id="admin1" data-label="edith" data-mention-suggestion-char="@">@edith</span></p><p></p>`
		if got != expected {
			t.Errorf("got %q, want %q", got, expected)
		}
	})

	t.Run("2 users added by inviter", func(t *testing.T) {
		got := models.FormatMergedChannelJoinMessage(users[:2], adder, "general")
		expected := `<p><span class="mention" data-type="mention" data-id="u1" data-label="elizabeth" data-mention-suggestion-char="@">@elizabeth</span> has been added to this channel by <span class="mention" data-type="mention" data-id="admin1" data-label="edith" data-mention-suggestion-char="@">@edith</span>. <span class="mention" data-type="mention" data-id="u2" data-label="Godwin" data-mention-suggestion-char="@">@Godwin</span> also joined.</p><p></p>`
		if got != expected {
			t.Errorf("got %q, want %q", got, expected)
		}
	})

	t.Run("3 users added by inviter", func(t *testing.T) {
		got := models.FormatMergedChannelJoinMessage(users[:3], adder, "general")
		expected := `<p><span class="mention" data-type="mention" data-id="u1" data-label="elizabeth" data-mention-suggestion-char="@">@elizabeth</span> has been added to this channel by <span class="mention" data-type="mention" data-id="admin1" data-label="edith" data-mention-suggestion-char="@">@edith</span>. <span class="mention" data-type="mention" data-id="u2" data-label="Godwin" data-mention-suggestion-char="@">@Godwin</span> and 1 other also joined.</p><p></p>`
		if got != expected {
			t.Errorf("got %q, want %q", got, expected)
		}
	})

	t.Run("4 users added by inviter", func(t *testing.T) {
		got := models.FormatMergedChannelJoinMessage(users[:4], adder, "general")
		expected := `<p><span class="mention" data-type="mention" data-id="u1" data-label="elizabeth" data-mention-suggestion-char="@">@elizabeth</span> has been added to this channel by <span class="mention" data-type="mention" data-id="admin1" data-label="edith" data-mention-suggestion-char="@">@edith</span>. <span class="mention" data-type="mention" data-id="u2" data-label="Godwin" data-mention-suggestion-char="@">@Godwin</span> and 2 others also joined.</p><p></p>`
		if got != expected {
			t.Errorf("got %q, want %q", got, expected)
		}
	})

	t.Run("4 users self-joined", func(t *testing.T) {
		got := models.FormatMergedChannelJoinMessage(users[:4], nil, "general")
		expected := `<p><span class="mention" data-type="mention" data-id="u1" data-label="elizabeth" data-mention-suggestion-char="@">@elizabeth</span> joined this channel. <span class="mention" data-type="mention" data-id="u2" data-label="Godwin" data-mention-suggestion-char="@">@Godwin</span> and 2 others also joined.</p><p></p>`
		if got != expected {
			t.Errorf("got %q, want %q", got, expected)
		}
	})

	t.Run("1 user self-left", func(t *testing.T) {
		got := models.FormatMergedChannelLeaveMessage(users[:1], nil, "general")
		expected := `<p><span class="mention" data-type="mention" data-id="u1" data-label="elizabeth" data-mention-suggestion-char="@">@elizabeth</span> left this channel</p><p></p>`
		if got != expected {
			t.Errorf("got %q, want %q", got, expected)
		}
	})

	t.Run("1 user removed by admin", func(t *testing.T) {
		got := models.FormatMergedChannelLeaveMessage(users[:1], adder, "general")
		expected := `<p><span class="mention" data-type="mention" data-id="u1" data-label="elizabeth" data-mention-suggestion-char="@">@elizabeth</span> was removed from this channel by <span class="mention" data-type="mention" data-id="admin1" data-label="edith" data-mention-suggestion-char="@">@edith</span></p><p></p>`
		if got != expected {
			t.Errorf("got %q, want %q", got, expected)
		}
	})

	t.Run("4 users self-left", func(t *testing.T) {
		got := models.FormatMergedChannelLeaveMessage(users[:4], nil, "general")
		expected := `<p><span class="mention" data-type="mention" data-id="u1" data-label="elizabeth" data-mention-suggestion-char="@">@elizabeth</span> left this channel. <span class="mention" data-type="mention" data-id="u2" data-label="Godwin" data-mention-suggestion-char="@">@Godwin</span> and 2 others also left.</p><p></p>`
		if got != expected {
			t.Errorf("got %q, want %q", got, expected)
		}
	})
}
