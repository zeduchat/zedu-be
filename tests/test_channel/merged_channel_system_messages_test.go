package test_channel

import (
	"html"
	"strings"
	"testing"

	"github.com/hngprojects/telex_be/internal/models"
)

func TestNoticeLabelsAreEscaped(t *testing.T) {
	label := `"><img src=x onerror=alert(1)>&'`
	users := []models.UserMention{{UserID: "u1", Username: label}, {UserID: "u2", Username: label}}
	actor := &models.UserMention{UserID: "actor", Username: label}
	for _, message := range []string{
		models.FormatMergedChannelJoinMessage(users, actor, "general"),
		models.FormatMergedChannelLeaveMessage(users, actor, "general"),
	} {
		if strings.Contains(message, label) || strings.Contains(message, "<img") {
			t.Fatalf("unsafe profile HTML reached a notice: %s", message)
		}
		if strings.Count(message, html.EscapeString(label)) != 6 {
			t.Fatalf("all participant and actor attribute/text labels must be escaped: %s", message)
		}
	}
}

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

func TestMergedJoinNoticePreservesResolvedDisplayNames(t *testing.T) {
	users := []models.UserMention{
		{UserID: "u1", Username: "Lynn B"},
		{UserID: "u2", Username: "Godwin"},
	}

	t.Run("merged notice keeps each participant display name and id", func(t *testing.T) {
		got := models.FormatMergedChannelJoinMessage(users, nil, "general")
		if !strings.Contains(got, `data-id="u1"`) || !strings.Contains(got, "@Lynn B") {
			t.Errorf("expected first participant display name and id preserved, got %q", got)
		}
		if !strings.Contains(got, `data-id="u2"`) || !strings.Contains(got, "@Godwin") {
			t.Errorf("expected second participant display name and id preserved, got %q", got)
		}
		if strings.Contains(got, "Pauline Banye") {
			t.Errorf("expected no full name in merged notice, got %q", got)
		}
	})
}
