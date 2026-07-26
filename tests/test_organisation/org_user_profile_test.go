package test_organisation

import (
	"fmt"
	"testing"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/tests/Test_profile"
	"github.com/hngprojects/telex_be/utility"
)

func TestOrgUsersProfileResolution(t *testing.T) {
	_, profileController := test_profile.SetupProfileTestRouter()
	db := profileController.Db.Postgresql

	user1 := utility.GenerateUUID()
	org1 := utility.GenerateUUID()

	u1 := models.User{ID: user1, Name: "Org Member 1", Email: fmt.Sprintf("orgmember%s@qa.team", utility.GenerateUUID())}
	db.Create(&u1)

	testOrg := models.Organisation{
		ID:                 org1,
		Name:               "Test Org Users",
		OwnerID:            user1,
		SubscriptionPlanId: utility.GenerateUUID(),
	}
	db.Create(&testOrg)

	var profModel models.Profile
	prof1, _ := profModel.GetOrCreateProfileForOrg(db, user1, org1)
	db.Model(&models.Profile{}).Where("id = ?", prof1.ID).Update("full_name", "OrgMember FullName Org1")

	_ = db.Exec("INSERT INTO user_organisations (user_id, organisation_id) VALUES (?, ?)", user1, org1)

	t.Run("GetUsersAndBotsInOrganisation Profile Resolution Test", func(t *testing.T) {
		var users []models.UserInOrgResponse
		err := db.Table("users").
			Select("users.id, users.email, profiles.phone as phone, profiles.full_name as name, profiles.avatar_url as avatar_url, users.created_at, profiles.online").
			Joins("JOIN user_organisations ON user_organisations.user_id = users.id").
			Joins("JOIN profiles ON profiles.userid = users.id AND (profiles.organisation_id IS NULL OR profiles.organisation_id = user_organisations.organisation_id)").
			Where("user_organisations.organisation_id = ?", org1).
			Find(&users).Error

		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if len(users) == 0 {
			t.Fatalf("Expected non-empty users list")
		}

		found := false
		for _, u := range users {
			if u.ID == user1 {
				found = true
				if u.Name != "OrgMember FullName Org1" {
					t.Errorf("Expected u.Name 'OrgMember FullName Org1', got '%s'", u.Name)
				}
			}
		}
		if !found {
			t.Errorf("User 1 not found in organisation users response")
		}
	})
}
