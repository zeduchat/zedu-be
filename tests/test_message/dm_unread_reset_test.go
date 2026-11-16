package test_message

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

// A DM can end up with an unread count above 0 while its last read thread is
// already the latest one (the unread increment for a new message can land
// after the message was read). Opening the DM again must still reset it.
func TestUpdateLastReadResetsStuckDmUnreadCount(t *testing.T) {
	db, logger, user, org := setupUnreadReader(t)

	latestThreadID := utility.GenerateUUID()
	dm := models.DmChannels{
		ID:           utility.GenerateUUID(),
		UserId:       user.ID,
		ChannelId:    utility.GenerateUUID(),
		OrgId:        org.ID,
		ChatType:     "user",
		ChannelType:  "dm",
		LastThreadId: latestThreadID,
		ThreadCount:  2,
	}
	if err := db.Postgresql.Create(&dm).Error; err != nil {
		t.Fatalf("failed to create dm channel: %v", err)
	}

	read := models.UpdateLastRead{LastThreadId: latestThreadID, LastReadAt: time.Now()}
	reader := models.DmChannels{ChannelId: dm.ChannelId, UserId: user.ID, ChannelType: "dm"}

	if updated := reader.UpdateLastRead(db.Postgresql, read, &sync.Mutex{}, logger); !updated {
		t.Fatalf("expected the stuck unread count to be reset")
	}

	var after models.DmChannels
	if err := db.Postgresql.Where("id = ?", dm.ID).First(&after).Error; err != nil {
		t.Fatalf("failed to reload dm channel: %v", err)
	}
	if after.ThreadCount != 0 {
		t.Fatalf("expected thread_count 0 after reading, got %d", after.ThreadCount)
	}

	// Reading again with nothing new is still skipped.
	if updated := reader.UpdateLastRead(db.Postgresql, read, &sync.Mutex{}, logger); updated {
		t.Fatalf("expected no update when already read and count is 0")
	}
}

func setupUnreadReader(t *testing.T) (*storage.Database, *utility.Logger, models.User, models.Organisation) {
	t.Helper()
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	signUp := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("unread_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Unread",
		LastName:    "Reader",
		Password:    "password",
		UserName:    fmt.Sprintf("unread_%v", currUUID),
	}
	authController := auth.Controller{
		Db:        db,
		Validator: validator.New(),
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}
	tst.SignupUser(t, gin.Default(), authController, signUp, false)

	var user models.User
	if err := db.Postgresql.Where("email = ?", signUp.Email).First(&user).Error; err != nil {
		t.Fatalf("failed to get user: %v", err)
	}
	var org models.Organisation
	if err := db.Postgresql.Where("owner_id = ?", user.ID).First(&org).Error; err != nil {
		t.Fatalf("failed to get organisation: %v", err)
	}

	return db, logger, user, org
}

// Same as above for a group DM, which keeps its unread count on the
// channel participant row.
func TestUpdateLastReadResetsStuckGroupDmUnreadCount(t *testing.T) {
	db, logger, user, org := setupUnreadReader(t)

	latestThreadID := utility.GenerateUUID()
	participant := models.ChannelParticipant{
		ID:           utility.GenerateUUID(),
		ChannelId:    utility.GenerateUUID(),
		UserId:       user.ID,
		OrgId:        org.ID,
		LastThreadId: latestThreadID,
		ThreadCount:  2,
	}
	if err := db.Postgresql.Create(&participant).Error; err != nil {
		t.Fatalf("failed to create channel participant: %v", err)
	}

	read := models.UpdateLastRead{LastThreadId: latestThreadID, LastReadAt: time.Now()}
	reader := models.DmChannels{ChannelId: participant.ChannelId, UserId: user.ID, ChannelType: "group_dm"}

	if updated := reader.UpdateLastRead(db.Postgresql, read, &sync.Mutex{}, logger); !updated {
		t.Fatalf("expected the stuck group DM unread count to be reset")
	}

	var after models.ChannelParticipant
	if err := db.Postgresql.Where("id = ?", participant.ID).First(&after).Error; err != nil {
		t.Fatalf("failed to reload channel participant: %v", err)
	}
	if after.ThreadCount != 0 {
		t.Fatalf("expected thread_count 0 after reading, got %d", after.ThreadCount)
	}

	if updated := reader.UpdateLastRead(db.Postgresql, read, &sync.Mutex{}, logger); updated {
		t.Fatalf("expected no update when already read and count is 0")
	}
}
