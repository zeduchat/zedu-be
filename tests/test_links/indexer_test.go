package test_links

import (
	"testing"
	"time"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/links"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func newSnapshot(content string) links.MessageSnapshot {
	return links.MessageSnapshot{
		ChannelID:      utility.GenerateUUID(),
		ThreadID:       utility.GenerateUUID(),
		MessageID:      utility.GenerateUUID(),
		UserID:         utility.GenerateUUID(),
		OrganisationID: utility.GenerateUUID(),
		Content:        content,
		CreatedAt:      time.Now().UTC(),
	}
}

func cleanupChannelLinks(t *testing.T, db *storage.Database, channelID string) {
	t.Helper()
	t.Cleanup(func() {
		db.Postgresql.Where("channel_id = ?", channelID).Delete(&models.ChannelLink{})
	})
}

func countMessageLinks(t *testing.T, db *storage.Database, messageID string) int64 {
	t.Helper()
	var count int64
	if err := db.Postgresql.Model(&models.ChannelLink{}).Where("message_id = ?", messageID).Count(&count).Error; err != nil {
		t.Fatalf("count message links: %v", err)
	}
	return count
}

func TestSyncMessageLinksDedupesURLs(t *testing.T) {
	tst.Setup()
	db := storage.Connection()

	snapshot := newSnapshot("see https://example.com/docs and https://example.com/docs again")
	cleanupChannelLinks(t, db, snapshot.ChannelID)

	if err := links.SyncMessageLinks(db.Postgresql, snapshot); err != nil {
		t.Fatalf("sync message links: %v", err)
	}

	if count := countMessageLinks(t, db, snapshot.MessageID); count != 1 {
		t.Fatalf("expected 1 stored link, got %d", count)
	}
}

func TestSyncMessageLinksReplacesOnEdit(t *testing.T) {
	tst.Setup()
	db := storage.Connection()

	snapshot := newSnapshot("first https://a.example.com")
	cleanupChannelLinks(t, db, snapshot.ChannelID)

	if err := links.SyncMessageLinks(db.Postgresql, snapshot); err != nil {
		t.Fatalf("sync first version: %v", err)
	}

	snapshot.Content = "second https://b.example.com"
	if err := links.SyncMessageLinks(db.Postgresql, snapshot); err != nil {
		t.Fatalf("sync edited version: %v", err)
	}

	var stored []models.ChannelLink
	if err := db.Postgresql.Where("message_id = ?", snapshot.MessageID).Find(&stored).Error; err != nil {
		t.Fatalf("load stored links: %v", err)
	}
	if len(stored) != 1 {
		t.Fatalf("expected 1 stored link after edit, got %d", len(stored))
	}
	if stored[0].URL != "https://b.example.com" {
		t.Fatalf("expected edited url, got %q", stored[0].URL)
	}
}

func TestDeleteMessageLinksCleanup(t *testing.T) {
	tst.Setup()
	db := storage.Connection()

	snapshot := newSnapshot("https://a.example.com https://b.example.com")
	cleanupChannelLinks(t, db, snapshot.ChannelID)

	if err := links.SyncMessageLinks(db.Postgresql, snapshot); err != nil {
		t.Fatalf("sync message links: %v", err)
	}
	if count := countMessageLinks(t, db, snapshot.MessageID); count != 2 {
		t.Fatalf("expected 2 stored links, got %d", count)
	}

	if err := models.DeleteChannelLinksByMessage(db.Postgresql, snapshot.MessageID); err != nil {
		t.Fatalf("delete message links: %v", err)
	}
	if count := countMessageLinks(t, db, snapshot.MessageID); count != 0 {
		t.Fatalf("expected message links to be removed, got %d", count)
	}
}

func TestDeleteThreadLinksCleanup(t *testing.T) {
	tst.Setup()
	db := storage.Connection()

	first := newSnapshot("https://a.example.com")
	second := newSnapshot("https://b.example.com")
	second.ChannelID = first.ChannelID
	second.ThreadID = first.ThreadID
	cleanupChannelLinks(t, db, first.ChannelID)

	for _, snapshot := range []links.MessageSnapshot{first, second} {
		if err := links.SyncMessageLinks(db.Postgresql, snapshot); err != nil {
			t.Fatalf("sync thread links: %v", err)
		}
	}

	if err := models.DeleteChannelLinksByThread(db.Postgresql, first.ThreadID); err != nil {
		t.Fatalf("delete thread links: %v", err)
	}

	var count int64
	if err := db.Postgresql.Model(&models.ChannelLink{}).Where("thread_id = ?", first.ThreadID).Count(&count).Error; err != nil {
		t.Fatalf("count thread links: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected thread links to be removed, got %d", count)
	}
}

func TestDeleteChannelLinksCleanup(t *testing.T) {
	tst.Setup()
	db := storage.Connection()

	deletedChannel := newSnapshot("https://a.example.com")
	keptChannel := newSnapshot("https://b.example.com")
	cleanupChannelLinks(t, db, deletedChannel.ChannelID)
	cleanupChannelLinks(t, db, keptChannel.ChannelID)

	for _, snapshot := range []links.MessageSnapshot{deletedChannel, keptChannel} {
		if err := links.SyncMessageLinks(db.Postgresql, snapshot); err != nil {
			t.Fatalf("sync channel links: %v", err)
		}
	}

	if err := models.DeleteChannelLinksByChannel(db.Postgresql, deletedChannel.ChannelID); err != nil {
		t.Fatalf("delete channel links: %v", err)
	}

	var deletedCount, keptCount int64
	db.Postgresql.Model(&models.ChannelLink{}).Where("channel_id = ?", deletedChannel.ChannelID).Count(&deletedCount)
	db.Postgresql.Model(&models.ChannelLink{}).Where("channel_id = ?", keptChannel.ChannelID).Count(&keptCount)
	if deletedCount != 0 || keptCount != 1 {
		t.Fatalf("expected deleted channel 0 and kept channel 1, got %d and %d", deletedCount, keptCount)
	}
}

func TestIndexMessageDoesNothingWhenDisabled(t *testing.T) {
	tst.Setup()
	db := storage.Connection()

	previous := config.Config.App.LinksIndexing
	config.Config.App.LinksIndexing = false
	t.Cleanup(func() {
		config.Config.App.LinksIndexing = previous
	})

	snapshot := newSnapshot("https://example.com/disabled")
	links.IndexMessage(snapshot)

	if count := countMessageLinks(t, db, snapshot.MessageID); count != 0 {
		t.Fatalf("expected no stored links while indexing is disabled, got %d", count)
	}
}
