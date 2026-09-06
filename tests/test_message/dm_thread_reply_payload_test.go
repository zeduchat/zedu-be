package test_message

import (
	"fmt"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/gofrs/uuid"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
	dm "github.com/hngprojects/telex_be/services/directMessage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestReplyChannelDMMessageProfileAndPayloadConsistency(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	defer tst.Cleanup(db)

	user1SignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("replyuser1_%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "ReplyUser",
		LastName:    "One",
		Password:    "password",
		UserName:    fmt.Sprintf("replyuser1_%v", currUUID),
	}

	loginData1 := models.LoginRequestModel{
		Email:    user1SignUpData.Email,
		Password: user1SignUpData.Password,
	}

	authController := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	r := gin.Default()
	tst.SignupUser(t, r, authController, user1SignUpData, false)
	_ = tst.GetLoginToken(t, r, authController, loginData1)

	var user1 models.User
	if err := db.Postgresql.Where("email = ?", user1SignUpData.Email).First(&user1).Error; err != nil {
		t.Fatalf("Failed to get user1: %v", err)
	}

	var org models.Organisation
	if err := db.Postgresql.Where("owner_id = ?", user1.ID).First(&org).Error; err != nil {
		t.Fatalf("Failed to get organization: %v", err)
	}

	// Update org-specific profile with explicit org-specific username and avatar
	orgUsername := fmt.Sprintf("org_custom_name_%v", currUUID)
	orgAvatar := "https://example.com/org_avatar.png"

	var prof models.Profile
	if err := db.Postgresql.Where("userid = ? AND organisation_id = ?", user1.ID, org.ID).First(&prof).Error; err == nil {
		db.Postgresql.Model(&prof).Updates(map[string]any{
			"user_name":  orgUsername,
			"avatar_url": orgAvatar,
		})
	}

	dmChannelID := utility.GenerateUUID()
	dmChannel := models.DmChannels{
		ID:          utility.GenerateUUID(),
		UserId:      user1.ID,
		ChannelId:   dmChannelID,
		OrgId:       org.ID,
		ChatType:    "user",
		ChannelType: "dm",
	}
	if err := db.Postgresql.Create(&dmChannel).Error; err != nil {
		t.Fatalf("Failed to create DM channel: %v", err)
	}

	threadID := utility.GenerateUUID()
	threadUUID, _ := uuid.FromString(threadID)
	threadDoc := models.ThreadDocument{
		ID:             threadID,
		ChannelsID:     dmChannelID,
		OrganisationID: org.ID,
		UserId:         user1.ID,
		Username:       orgUsername,
		Content:        "Parent thread message",
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	if err := elastic.AddDocument(db.Elastic, models.ThreadIndexName, threadID, threadDoc, logger); err != nil {
		t.Fatalf("Failed to create parent thread in ES: %v", err)
	}

	t.Run("Verify Profile Resolution Uses Channel OrgId and Populates Required Fields", func(t *testing.T) {
		req := models.CreateMessageRequest{
			Content:    "DM thread reply test message",
			UserId:     user1.ID,
			ChannelsId: dmChannelID,
			ThreadId:   threadID,
			OrgId:      "", // Intentionally leave empty to verify resolution via channel.OrgId
		}

		msgDoc, code, err := dm.ReplyChannelDMMessage(req, db, logger)
		if err != nil {
			t.Fatalf("ReplyChannelDMMessage failed with code %d: %v", code, err)
		}

		if msgDoc == nil {
			t.Fatal("Expected non-nil MessageDocument response")
		}

		if msgDoc.Username != orgUsername {
			t.Errorf("Expected username '%s', got '%s'", orgUsername, msgDoc.Username)
		}

		if msgDoc.AvatarURL != orgAvatar {
			t.Errorf("Expected avatar_url '%s', got '%s'", orgAvatar, msgDoc.AvatarURL)
		}

		if msgDoc.ProfileID == "" {
			t.Error("Expected profile_id to be populated, got empty string")
		}

		if msgDoc.DefaultAvatarURL == "" {
			t.Error("Expected default_avatar_url to be populated, got empty string")
		}

		if msgDoc.OrganisationID != org.ID {
			t.Errorf("Expected organisation_id '%s', got '%s'", org.ID, msgDoc.OrganisationID)
		}

		if msgDoc.ThreadID != threadUUID {
			t.Errorf("Expected thread_id '%s', got '%s'", threadUUID.String(), msgDoc.ThreadID.String())
		}
	})
}
