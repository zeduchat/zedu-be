package test_channel

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	channelService "github.com/hngprojects/telex_be/services/channel"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestChannelStarReviewRegressions(t *testing.T) {
	_, db, controller, ownerToken, joinerToken, channelID, _ := setupJoinNoticeTestEnv(t, "Joiner", "joiner", "Joiner")
	var ch models.Channels
	if err := db.Postgresql.First(&ch, "id = ?", channelID).Error; err != nil {
		t.Fatal(err)
	}
	ids := models.IDS{UserID: ch.OwnerId, OrganisationID: ch.OrganisationID}
	router := gin.New()
	router.POST("/organisations/:org_id/channels/:channelId/star", middleware.Authorize(db.Postgresql), controller.ToggleStarChannel)

	t.Run("path organisation is ignored in favour of the token", func(t *testing.T) {
		for _, pathOrg := range []string{strings.ToUpper(ch.OrganisationID), utility.GenerateUUID(), "not-a-uuid"} {
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/organisations/%s/channels/%s/star", pathOrg, channelID), nil)
			req.Header.Set("Authorization", "Bearer "+ownerToken)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("path organisation %q: got %d, want %d: %s", pathOrg, rr.Code, http.StatusOK, rr.Body.String())
			}
			db.Postgresql.Where("channel_id = ?", channelID).Unscoped().Delete(&models.ChannelStar{})
		}
	})

	t.Run("channel outside the token organisation is hidden", func(t *testing.T) {
		var otherOrg models.Organisation
		if err := db.Postgresql.Where("owner_id = ? AND id <> ?", ch.OwnerId, ch.OrganisationID).First(&otherOrg).Error; err != nil {
			t.Fatalf("failed to load a second organisation for the owner: %v", err)
		}
		foreignChannel := models.Channels{
			ID:             utility.GenerateUUID(),
			Name:           "foreign-org-channel",
			Description:    "channel owned by another organisation",
			OwnerId:        ch.OwnerId,
			OrganisationID: otherOrg.ID,
		}
		if err := db.Postgresql.Create(&foreignChannel).Error; err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.Postgresql.Unscoped().Delete(&foreignChannel) })

		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/organisations/%s/channels/%s/star", ch.OrganisationID, foreignChannel.ID), nil)
		req.Header.Set("Authorization", "Bearer "+ownerToken)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("foreign channel: got %d, want %d: %s", rr.Code, http.StatusNotFound, rr.Body.String())
		}
	})

	t.Run("empty and nonexistent organisations", func(t *testing.T) {
		list, code, err := channelService.GetStarredChannels(db, ids)
		if err != nil || code != http.StatusOK || list == nil || len(list) != 0 {
			t.Fatalf("empty list: list=%v code=%d err=%v", list, code, err)
		}
		missing := ids
		missing.OrganisationID = utility.GenerateUUID()
		_, code, err = channelService.GetStarredChannels(db, missing)
		if code != http.StatusNotFound || err == nil {
			t.Fatalf("missing organisation: code=%d err=%v", code, err)
		}
	})

	t.Run("nonmembers cannot star", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/organisations/%s/channels/%s/star", ch.OrganisationID, channelID), nil)
		req.Header.Set("Authorization", "Bearer "+joinerToken)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("nonmember: got %d, want 403: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("only starred channels are enriched", func(t *testing.T) {
		star := models.ChannelStar{ID: utility.GenerateUUID(), UserId: ids.UserID, ChannelId: channelID, OrgId: ids.OrganisationID}
		if err := db.Postgresql.Create(&star).Error; err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.Postgresql.Unscoped().Delete(&star) })
		var enriched []string
		callbackName := "test:starred_channel_enrichment"
		if err := db.Postgresql.Callback().Query().After("gorm:query").Register(callbackName, func(tx *gorm.DB) {
			if tx.Statement.Table == "channels" && tx.Statement.Dest != nil {
				enriched = append(enriched, tx.Statement.SQL.String())
			}
		}); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.Postgresql.Callback().Query().Remove(callbackName) })
		list, code, err := channelService.GetStarredChannels(db, ids)
		if err != nil || code != http.StatusOK || len(list) != 1 || list[0].ID != channelID || !list[0].IsStarred {
			t.Fatalf("starred list: list=%v code=%d err=%v", list, code, err)
		}
		if len(enriched) > 1 {
			t.Fatalf("enriched unrelated channels: %d channel lookups", len(enriched))
		}
	})

	t.Run("star lookup errors reach the caller", func(t *testing.T) {
		tx := db.Postgresql.Begin()
		if tx.Error != nil {
			t.Fatal(tx.Error)
		}
		defer tx.Rollback()
		if err := tx.Exec("DROP TABLE channel_stars").Error; err != nil {
			t.Fatal(err)
		}
		noChannels := ids
		noChannels.ChannelIDs = []string{}
		var uc models.UserChannels
		_, err := uc.GetUserChannels(&storage.Database{Postgresql: tx}, noChannels)
		if err == nil || !strings.Contains(err.Error(), "error fetching starred channels") {
			t.Fatalf("expected star lookup failure, got %v", err)
		}
	})
}

func TestChannelStarForeignKeys(t *testing.T) {
	tst.Setup()
	db := storage.Connection().Postgresql
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	if err := tx.Exec("CREATE SCHEMA channel_star_fk_review").Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec("SET LOCAL search_path TO channel_star_fk_review").Error; err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{"CREATE TABLE users (id UUID PRIMARY KEY)", "CREATE TABLE channels (id UUID PRIMARY KEY)", "CREATE TABLE organisations (id UUID PRIMARY KEY)"} {
		if err := tx.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"000003_add_channel_stars.up.sql", "000004_add_channel_star_foreign_keys.up.sql", "000004_add_channel_star_foreign_keys.up.sql"} {
		migration, err := os.ReadFile("../../db/migrations/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if err := tx.Exec(string(migration)).Error; err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if name == "000003_add_channel_stars.up.sql" {
			orphan := models.ChannelStar{ID: utility.GenerateUUID(), UserId: utility.GenerateUUID(), ChannelId: utility.GenerateUUID(), OrgId: utility.GenerateUUID()}
			if err := tx.Omit("User", "Channel", "Organisation").Create(&orphan).Error; err != nil {
				t.Fatal(err)
			}
		}
	}
	var orphanCount int64
	if err := tx.Model(&models.ChannelStar{}).Count(&orphanCount).Error; err != nil || orphanCount != 0 {
		t.Fatalf("migration must remove orphaned stars before adding constraints: count=%d err=%v", orphanCount, err)
	}
	for _, table := range []string{"users", "channels", "organisations"} {
		t.Run("cascade deleting "+table, func(t *testing.T) {
			userID, channelID, orgID := utility.GenerateUUID(), utility.GenerateUUID(), utility.GenerateUUID()
			for _, parent := range []struct{ table, id string }{{"users", userID}, {"channels", channelID}, {"organisations", orgID}} {
				if err := tx.Table(parent.table).Create(map[string]any{"id": parent.id}).Error; err != nil {
					t.Fatal(err)
				}
			}
			star := models.ChannelStar{ID: utility.GenerateUUID(), UserId: userID, ChannelId: channelID, OrgId: orgID}
			if err := tx.Omit("User", "Channel", "Organisation").Create(&star).Error; err != nil {
				t.Fatal(err)
			}
			parentID := map[string]string{"users": userID, "channels": channelID, "organisations": orgID}[table]
			if err := tx.Table(table).Where("id = ?", parentID).Delete(map[string]any{}).Error; err != nil {
				t.Fatal(err)
			}
			var count int64
			if err := tx.Unscoped().Model(&models.ChannelStar{}).Where("id = ?", star.ID).Count(&count).Error; err != nil || count != 0 {
				t.Fatalf("orphaned star: count=%d err=%v", count, err)
			}
		})
	}
	modelSchema, err := schema.Parse(&models.ChannelStar{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"User", "Channel", "Organisation"} {
		constraint := modelSchema.Relationships.Relations[name].ParseConstraint()
		if constraint == nil || constraint.OnDelete != "CASCADE" {
			t.Fatalf("GORM %s deletion policy differs from SQL migration", name)
		}
	}
}
