package test_links

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/channel"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/postgresql"
	"github.com/hngprojects/telex_be/services/links"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func newLinksTestRouter(db *storage.Database) *gin.Engine {
	gin.SetMode(gin.TestMode)
	controller := channel.Controller{Db: db}
	r := gin.New()
	r.GET("/api/v1/channels/:channelId/links", controller.GetChannelLinks)
	return r
}

func TestGetChannelLinksService(t *testing.T) {
	tst.Setup()
	db := storage.Connection()

	channelID := utility.GenerateUUID()
	threadID := utility.GenerateUUID()
	orgID := utility.GenerateUUID()
	enrichedUserID := utility.GenerateUUID()
	otherUserID := utility.GenerateUUID()

	profile := models.Profile{
		ID:             utility.GenerateUUID(),
		FirstName:      "Ada",
		LastName:       "Lovelace",
		FullName:       "Ada Lovelace",
		UserName:       "ada",
		AvatarURL:      "https://cdn.example.com/ada.png",
		Userid:         enrichedUserID,
		OrganisationID: &orgID,
	}
	if err := db.Postgresql.Create(&profile).Error; err != nil {
		t.Fatalf("create profile: %v", err)
	}
	t.Cleanup(func() {
		db.Postgresql.Unscoped().Where("userid = ?", enrichedUserID).Delete(&models.Profile{})
		db.Postgresql.Where("channel_id = ?", channelID).Delete(&models.ChannelLink{})
	})

	now := time.Now().UTC()
	rows := []models.ChannelLink{
		{
			ChannelID:      channelID,
			ThreadID:       threadID,
			MessageID:      utility.GenerateUUID(),
			UserID:         otherUserID,
			OrganisationID: orgID,
			URL:            "https://example.com/oldest",
			CreatedAt:      now.Add(-3 * time.Hour),
		},
		{
			ChannelID:      channelID,
			ThreadID:       threadID,
			MessageID:      utility.GenerateUUID(),
			UserID:         enrichedUserID,
			OrganisationID: orgID,
			URL:            "https://example.com/middle",
			CreatedAt:      now.Add(-2 * time.Hour),
		},
		{
			ChannelID:      channelID,
			ThreadID:       threadID,
			MessageID:      utility.GenerateUUID(),
			UserID:         enrichedUserID,
			OrganisationID: orgID,
			URL:            "https://example.com/newest",
			CreatedAt:      now.Add(-1 * time.Hour),
		},
	}
	if err := db.Postgresql.Create(&rows).Error; err != nil {
		t.Fatalf("create channel links: %v", err)
	}

	response, pagination, err := links.GetChannelLinks(db.Postgresql, channelID, postgresql.Pagination{Page: 1, Limit: 2})
	if err != nil {
		t.Fatalf("get channel links: %v", err)
	}

	if len(response) != 2 {
		t.Fatalf("expected 2 links on page 1, got %d", len(response))
	}
	if response[0].URL != "https://example.com/newest" {
		t.Fatalf("expected newest link first, got %q", response[0].URL)
	}
	if response[1].URL != "https://example.com/middle" {
		t.Fatalf("expected middle link second, got %q", response[1].URL)
	}
	if response[0].Username != "ada" || response[0].AvatarURL != "https://cdn.example.com/ada.png" {
		t.Fatalf("expected enriched profile, got username %q and avatar %q", response[0].Username, response[0].AvatarURL)
	}

	if pagination.CurrentPage != 1 || pagination.PageCount != 2 {
		t.Fatalf("unexpected pagination window: %+v", pagination)
	}
	if pagination.TotalItems != 3 || pagination.TotalPagesCount != 2 {
		t.Fatalf("unexpected pagination totals: %+v", pagination)
	}
}

func TestGetChannelLinksRejectsInvalidChannelID(t *testing.T) {
	tst.Setup()
	r := newLinksTestRouter(storage.Connection())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/channels/not-a-uuid/links", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestGetChannelLinksRequiresClaims(t *testing.T) {
	tst.Setup()
	r := newLinksTestRouter(storage.Connection())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/channels/"+utility.GenerateUUID()+"/links", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}
