package test_channel

import (
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestJoinNoticeProfileResolution(t *testing.T) {
	tst.Setup()
	tx := storage.Connection().Postgresql.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	userID := utility.GenerateUUID()
	personalOrg, targetOrg := utility.GenerateUUID(), utility.GenerateUUID()
	personalProfile := models.Profile{
		ID: utility.GenerateUUID(), Userid: userID, OrganisationID: &personalOrg,
		UserName: "original", DisplayName: "Original Name", CreatedAt: time.Now().Add(-time.Hour),
	}
	if err := tx.Create(&personalProfile).Error; err != nil {
		t.Fatal(err)
	}
	profile, err := models.LoadJoinNoticeProfile(tx, userID, targetOrg)
	if err != nil || profile.ID != personalProfile.ID {
		t.Fatalf("personal-org base profile: profile=%s err=%v", profile.ID, err)
	}
	base := models.Profile{ID: utility.GenerateUUID(), Userid: userID, UserName: "base", DisplayName: "Base Name"}
	if err := tx.Create(&base).Error; err != nil {
		t.Fatal(err)
	}
	profile, err = models.LoadJoinNoticeProfile(tx, userID, targetOrg)
	if err != nil || profile.ID != base.ID {
		t.Fatalf("unscoped profile must take precedence: profile=%s err=%v", profile.ID, err)
	}
	scoped := models.Profile{ID: utility.GenerateUUID(), Userid: userID, OrganisationID: &targetOrg, DisplayName: "Org Name"}
	if err := tx.Create(&scoped).Error; err != nil {
		t.Fatal(err)
	}
	mention, err := models.NewChannelJoinMention(tx, userID, targetOrg)
	if err != nil || mention.Username != "Org Name" || mention.UserID != userID {
		t.Fatalf("organisation profile must take precedence: mention=%+v err=%v", mention, err)
	}
	var count int64
	if err := tx.Model(&models.Profile{}).Where("userid = ?", userID).Count(&count).Error; err != nil || count != 3 {
		t.Fatalf("notice lookup changed profiles: count=%d err=%v", count, err)
	}
	var stored models.Profile
	if err := tx.First(&stored, "id = ?", personalProfile.ID).Error; err != nil || stored.OrganisationID == nil || *stored.OrganisationID != personalOrg {
		t.Fatalf("notice lookup rebound the personal profile: profile=%+v err=%v", stored, err)
	}
	if _, err := models.LoadJoinNoticeProfile(tx, utility.GenerateUUID(), targetOrg); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("missing profiles must remain missing: %v", err)
	}
}
