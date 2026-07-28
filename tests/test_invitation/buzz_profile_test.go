package test_invitation

import (
	"fmt"
	"testing"
	"time"

	"github.com/hngprojects/telex_be/internal/models"
	buzzService "github.com/hngprojects/telex_be/services/buzz"
	"github.com/hngprojects/telex_be/tests/Test_profile"
	"github.com/hngprojects/telex_be/utility"
)

func TestBuzzInvitationOrgProfileResolution(t *testing.T) {
	_, profileController := test_profile.SetupProfileTestRouter()
	db := profileController.Db.Postgresql
	logger := profileController.Logger

	inviterID := utility.GenerateUUID()
	inviteeID := utility.GenerateUUID()
	orgID := utility.GenerateUUID()

	inviter := models.User{
		ID:    inviterID,
		Name:  "Inviter User",
		Email: fmt.Sprintf("inviter%s@qa.team", utility.GenerateUUID()),
	}
	invitee := models.User{
		ID:    inviteeID,
		Name:  "Invitee User",
		Email: fmt.Sprintf("invitee%s@qa.team", utility.GenerateUUID()),
	}
	if err := db.Create(&inviter).Error; err != nil {
		t.Fatalf("Failed to create inviter: %v", err)
	}
	if err := db.Create(&invitee).Error; err != nil {
		t.Fatalf("Failed to create invitee: %v", err)
	}

	testOrg := models.Organisation{
		ID:                 orgID,
		Name:               "Test Buzz Org",
		OwnerID:            inviterID,
		SubscriptionPlanId: utility.GenerateUUID(),
	}
	if err := db.Create(&testOrg).Error; err != nil {
		t.Fatalf("Failed to create test org: %v", err)
	}

	var profModel models.Profile
	profInviter, err := profModel.GetOrCreateProfileForOrg(db, inviterID, orgID)
	if err != nil {
		t.Fatalf("Failed to create profile for inviter: %v", err)
	}
	profInvitee, err := profModel.GetOrCreateProfileForOrg(db, inviteeID, orgID)
	if err != nil {
		t.Fatalf("Failed to create profile for invitee: %v", err)
	}

	_ = db.Exec("INSERT INTO user_organisations (user_id, organisation_id) VALUES (?, ?)", inviterID, orgID)
	_ = db.Exec("INSERT INTO user_organisations (user_id, organisation_id) VALUES (?, ?)", inviteeID, orgID)

	_ = db.Create(&models.OrgUserManagement{
		OrganisationID: orgID,
		UserID:         inviterID,
		RoleID:         utility.GenerateUUID(),
	})
	_ = db.Create(&models.OrgUserManagement{
		OrganisationID: orgID,
		UserID:         inviteeID,
		RoleID:         utility.GenerateUUID(),
	})

	buzzID := utility.GenerateUUID()
	channelID := utility.GenerateUUID()
	buzz := models.Buzz{
		ID:             buzzID,
		ChannelID:      channelID,
		HostID:         inviterID,
		OriginalHostID: inviterID,
		OrgID:          &orgID,
		BuzzType:       models.BuzzTypeOrganization,
		ParticipantIDs: []string{inviterID},
		Status:         "active",
		IsLiveStatus:   true,
		BuzzStartTime:  time.Now(),
	}
	if err := db.Create(&buzz).Error; err != nil {
		t.Fatalf("Failed to create buzz: %v", err)
	}

	userChanInviter := models.UserChannels{
		ChannelsID: channelID,
		UserID:     inviterID,
	}
	_ = db.Create(&userChanInviter)

	buzzParticipant := models.BuzzParticipant{
		ID:       utility.GenerateUUID(),
		BuzzID:   buzzID,
		UserID:   inviterID,
		Status:   models.BuzzParticipantStatusActive,
		JoinedAt: time.Now(),
	}
	if err := db.Create(&buzzParticipant).Error; err != nil {
		t.Fatalf("Failed to create buzz participant: %v", err)
	}

	t.Run("Invite User To Buzz Org Profile Resolution", func(t *testing.T) {
		req := models.InviteUsersToBuzzRequest{
			BuzzID:     buzzID,
			InviteeIDs: []string{inviteeID},
		}

		baseStorageDB := profileController.Db
		resp, code, err := buzzService.InviteUsersToBuzz(baseStorageDB, logger, req, inviterID)
		if err != nil {
			t.Fatalf("InviteUsersToBuzz failed: %v (code: %d)", err, code)
		}
		if len(resp.InvitedUserIDs) == 0 {
			t.Fatalf("Expected invited user IDs slice")
		}

		var createdInvitation models.BuzzInvitation
		if err := db.Where("buzz_id = ? AND invitee_id = ?", buzzID, inviteeID).First(&createdInvitation).Error; err != nil {
			t.Fatalf("Failed to fetch created invitation: %v", err)
		}

		if createdInvitation.InviterID != inviterID {
			t.Errorf("Expected InviterID to match inviter ID %s, got %s", inviterID, createdInvitation.InviterID)
		}
		if createdInvitation.InviteeID != inviteeID {
			t.Errorf("Expected InviteeID to match invitee ID %s, got %s", inviteeID, createdInvitation.InviteeID)
		}
		if createdInvitation.OrgID != orgID {
			t.Errorf("Expected OrgID to match org ID %s, got %s", orgID, createdInvitation.OrgID)
		}

		// Verify that the inviter and invitee organization profiles resolve correctly
		resolvedInviterProf, err := profModel.GetOrCreateProfileForOrg(db, createdInvitation.InviterID, orgID)
		if err != nil {
			t.Fatalf("Failed to resolve inviter profile for org: %v", err)
		}
		if resolvedInviterProf.ID != profInviter.ID {
			t.Errorf("Expected resolved inviter profile ID %s, got %s", profInviter.ID, resolvedInviterProf.ID)
		}

		resolvedInviteeProf, err := profModel.GetOrCreateProfileForOrg(db, createdInvitation.InviteeID, orgID)
		if err != nil {
			t.Fatalf("Failed to resolve invitee profile for org: %v", err)
		}
		if resolvedInviteeProf.ID != profInvitee.ID {
			t.Errorf("Expected resolved invitee profile ID %s, got %s", profInvitee.ID, resolvedInviteeProf.ID)
		}

		invitations, _, err := buzzService.GetPendingInvitations(baseStorageDB, logger, inviteeID)
		if err != nil {
			t.Fatalf("GetPendingInvitations failed: %v", err)
		}
		if len(invitations) == 0 {
			t.Fatalf("Expected at least 1 pending invitation")
		}
		if invitations[0].InviterName != profInviter.UserName {
			t.Errorf("Expected pending invitation InviterName to match inviter profile user_name '%s', got '%s'", profInviter.UserName, invitations[0].InviterName)
		}
	})
}
