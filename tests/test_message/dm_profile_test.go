package test_message

import (
	"fmt"
	"testing"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/tests/Test_profile"
	"github.com/hngprojects/telex_be/utility"
)

func TestDMChannelsOrgProfileResolution(t *testing.T) {
	_, profileController := test_profile.SetupProfileTestRouter()
	db := profileController.Db.Postgresql

	userA := utility.GenerateUUID()
	userB := utility.GenerateUUID()
	org1 := utility.GenerateUUID()

	uA := models.User{ID: userA, Name: "User A", Email: fmt.Sprintf("usera%s@qa.team", utility.GenerateUUID())}
	uB := models.User{ID: userB, Name: "User B", Email: fmt.Sprintf("userb%s@qa.team", utility.GenerateUUID())}
	db.Create(&uA)
	db.Create(&uB)

	var profModel models.Profile
	profA, _ := profModel.GetOrCreateProfileForOrg(db, userA, org1)
	profB, _ := profModel.GetOrCreateProfileForOrg(db, userB, org1)

	db.Model(&models.Profile{}).Where("id = ?", profA.ID).Updates(map[string]interface{}{"first_name": "User A Org1", "full_name": "User A Org1"})
	db.Model(&models.Profile{}).Where("id = ?", profB.ID).Updates(map[string]interface{}{"first_name": "User B Org1", "full_name": "User B Org1"})

	channelID := utility.GenerateUUID()
	dmChannel := models.DmChannels{
		ID:            utility.GenerateUUID(),
		ChannelId:     channelID,
		UserId:        userA,
		ParticipantId: &userB,
		OrgId:         org1,
		ChannelType:   "dm",
	}
	db.Create(&dmChannel)

	t.Run("DM Channel Participant Profile Resolution Test", func(t *testing.T) {
		dmResp, err := dmChannel.GetDmChannelResponse(db, nil)
		if err != nil {
			t.Fatalf("GetDmChannelResponse failed: %v", err)
		}

		if len(dmResp.Participants) == 0 {
			t.Fatalf("Expected non-empty participants slice")
		}

		p := dmResp.Participants[0]
		if p.UserId != userB {
			t.Errorf("Expected participant UserId %s, got %s", userB, p.UserId)
		}
		if p.FullName != "User B Org1" {
			t.Errorf("Expected DM participant FullName 'User B Org1', got '%s'", p.FullName)
		}
	})
}
