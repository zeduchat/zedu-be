package test_channel

import (
	"fmt"
	"testing"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/tests/Test_profile"
	"github.com/hngprojects/telex_be/utility"
)

func TestChannelMembersOrgProfileResolution(t *testing.T) {
	_, profileController := test_profile.SetupProfileTestRouter()
	db := profileController.Db.Postgresql

	user1 := utility.GenerateUUID()
	org1 := utility.GenerateUUID()

	u1 := models.User{ID: user1, Name: "Channel User 1", Email: fmt.Sprintf("chanuser%s@qa.team", utility.GenerateUUID())}
	db.Create(&u1)

	testOrg := models.Organisation{
		ID:                 org1,
		Name:               "Test Chan Org",
		OwnerID:            user1,
		SubscriptionPlanId: utility.GenerateUUID(),
	}
	db.Create(&testOrg)

	var profModel models.Profile
	prof1, _ := profModel.GetOrCreateProfileForOrg(db, user1, org1)
	db.Model(&models.Profile{}).Where("id = ?", prof1.ID).Update("avatar_url", "http://example.com/chan_avatar1.png")

	channelID := utility.GenerateUUID()
	channel := models.Channels{
		ID:             channelID,
		Name:           "general-org1",
		OrganisationID: org1,
		OwnerId:        user1,
	}
	db.Create(&channel)

	db.Create(&models.UserChannels{
		ChannelsID: channelID,
		UserID:     user1,
	})

	t.Run("Channel Member Profile Resolution Test", func(t *testing.T) {
		var avatars []string
		err := db.Table("user_channels").
			Select("profiles.avatar_url").
			Joins("JOIN channels ON channels.id = user_channels.channels_id").
			Joins("JOIN profiles ON profiles.userid = user_channels.user_id AND (profiles.organisation_id IS NULL OR profiles.organisation_id = channels.organisation_id)").
			Where("user_channels.channels_id = ? AND profiles.avatar_url != ''", channelID).
			Pluck("profiles.avatar_url", &avatars).Error

		if err != nil {
			t.Fatalf("Failed to fetch channel member avatars: %v", err)
		}
		if len(avatars) == 0 {
			t.Fatalf("Expected at least 1 avatar URL")
		}
		if avatars[0] != "http://example.com/chan_avatar1.png" {
			t.Errorf("Expected avatar_url 'http://example.com/chan_avatar1.png', got '%s'", avatars[0])
		}
	})
}
