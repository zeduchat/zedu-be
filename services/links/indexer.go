package links

import (
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
)

const systemUserID = "00000000-0000-0000-0000-000000000000"

type MessageSnapshot struct {
	ChannelID      string
	ThreadID       string
	MessageID      string
	UserID         string
	OrganisationID string
	Content        string
	CreatedAt      time.Time
}

// IndexMessage stores the links found in a message. The snapshot is copied by value, so the
// goroutine never holds a gin context or a request transaction.
func IndexMessage(snapshot MessageSnapshot) {
	if !config.Config.App.LinksIndexing {
		return
	}
	run(func() error {
		return SyncMessageLinks(storage.Connection().Postgresql, snapshot)
	})
}

// SyncMessageLinks replaces the stored links for a message; tests use it without the async wrapper.
func SyncMessageLinks(db *gorm.DB, snapshot MessageSnapshot) error {
	urls := ExtractURLs(snapshot.Content)
	userID := snapshot.UserID
	if userID == "" {
		userID = systemUserID
	}

	records := make([]models.ChannelLink, 0, len(urls))
	for _, link := range urls {
		records = append(records, models.ChannelLink{
			ChannelID:      snapshot.ChannelID,
			ThreadID:       snapshot.ThreadID,
			MessageID:      snapshot.MessageID,
			UserID:         userID,
			OrganisationID: snapshot.OrganisationID,
			URL:            link,
			CreatedAt:      snapshot.CreatedAt,
		})
	}
	return models.ReplaceMessageLinks(db, snapshot.MessageID, records)
}

func DeleteMessageLinks(messageID string) {
	run(func() error {
		return models.DeleteChannelLinksByMessage(storage.Connection().Postgresql, messageID)
	})
}

func DeleteThreadLinks(threadID string) {
	run(func() error {
		return models.DeleteChannelLinksByThread(storage.Connection().Postgresql, threadID)
	})
}

func DeleteChannelLinks(channelID string) {
	run(func() error {
		return models.DeleteChannelLinksByChannel(storage.Connection().Postgresql, channelID)
	})
}

func run(task func() error) {
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("links: background task panicked: %v", recovered)
			}
		}()
		if err := task(); err != nil {
			log.Printf("links: background task failed: %v", err)
		}
	}()
}
