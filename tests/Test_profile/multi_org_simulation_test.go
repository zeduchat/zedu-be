package test_profile

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hngprojects/telex_be/internal/avatar"
	"github.com/hngprojects/telex_be/internal/models"
	profileService "github.com/hngprojects/telex_be/services/profile"
	"github.com/hngprojects/telex_be/utility"
)

func TestMultiOrgSimulation_CurrentDBState_4Orgs(t *testing.T) {
	_, profileController := SetupProfileTestRouter()
	db := profileController.Db.Postgresql
	var profModel models.Profile

	// Setup: User with single base profile & 4 mapped orgs in user_organisations
	userID := utility.GenerateUUID()
	org1 := utility.GenerateUUID()
	org2 := utility.GenerateUUID()
	org3 := utility.GenerateUUID()
	org4 := utility.GenerateUUID()

	password, err := utility.HashPassword("password")
	assert.Nil(t, err)

	simUser := models.User{
		ID:       userID,
		Name:     "Simulated User",
		Email:    fmt.Sprintf("simuser_%s@qa.team", utility.GenerateUUID()),
		Password: password,
	}
	assert.Nil(t, db.Create(&simUser).Error)

	// Base profile with unassigned organisation_id (NULL)
	baseProf := models.Profile{
		ID:        utility.GenerateUUID(),
		Userid:    userID,
		FirstName: "Simulated",
		LastName:  "User",
		FullName:  "Simulated User",
		UserName:  "simuser",
		AvatarURL: avatar.GenerateDefaultAvatarURL(userID),
	}
	assert.Nil(t, db.Create(&baseProf).Error)

	// Create Organisation fixtures
	for i, oID := range []string{org1, org2, org3, org4} {
		orgObj := models.Organisation{ID: oID, Name: fmt.Sprintf("Sim Org %d", i+1), OwnerID: userID}
		assert.Nil(t, db.Create(&orgObj).Error)
		assert.Nil(t, db.Exec("INSERT INTO user_organisations (user_id, organisation_id) VALUES (?, ?)", userID, oID).Error)
	}

	// Point of Interaction 1: GetUserProfile (HTTP/Service layer) with Org3 context
	summary, status, err := profileService.GetUserProfile(db, userID, org3)
	assert.Nil(t, err)
	assert.Equal(t, 200, status)
	assert.NotNil(t, summary)
	assert.Equal(t, org3, summary.OrganisationID)

	// Point of Interaction 2: Direct GetOrCreateProfileForOrg call for Org4
	resolvedProf, err := profModel.GetOrCreateProfileForOrg(db, userID, org4, profileController.Logger)
	assert.Nil(t, err)
	assert.Equal(t, org4, resolvedProf.GetOrgID())

	// Validate that profiles now exist for all 4 mapped orgs
	var allUserProfiles []models.Profile
	db.Where("userid = ?", userID).Find(&allUserProfiles)
	assert.Equal(t, 4, len(allUserProfiles))

	orgSet := make(map[string]bool)
	for _, p := range allUserProfiles {
		if p.OrganisationID != nil {
			orgSet[*p.OrganisationID] = true
		}
	}
	assert.True(t, orgSet[org1], "Profile for Org1 must exist")
	assert.True(t, orgSet[org2], "Profile for Org2 must exist")
	assert.True(t, orgSet[org3], "Profile for Org3 must exist")
	assert.True(t, orgSet[org4], "Profile for Org4 must exist")
}

func TestMultiOrgSimulation_FirstTimeOrgJoin_ExpectedProfileNotCloned(t *testing.T) {
	_, profileController := SetupProfileTestRouter()
	db := profileController.Db.Postgresql
	var profModel models.Profile

	userID := utility.GenerateUUID()
	org1 := utility.GenerateUUID()
	org5 := utility.GenerateUUID()

	password, err := utility.HashPassword("password")
	assert.Nil(t, err)

	simUser := models.User{
		ID:       userID,
		Name:     "Custom Avatar User",
		Email:    fmt.Sprintf("custom_avatar_%s@qa.team", utility.GenerateUUID()),
		Password: password,
	}
	assert.Nil(t, db.Create(&simUser).Error)

	// Custom avatar URL set on Org1 profile
	customAvatarURL := "https://cdn.example.com/custom_avatar_image.jpg"
	profOrg1 := models.Profile{
		ID:             utility.GenerateUUID(),
		Userid:         userID,
		OrganisationID: &org1,
		FirstName:      "Custom",
		LastName:       "User",
		FullName:       "Custom User",
		UserName:       "customuser",
		AvatarURL:      customAvatarURL,
	}
	assert.Nil(t, db.Create(&profOrg1).Error)

	// Create Organisation fixtures for org1 and org5
	orgObj1 := models.Organisation{ID: org1, Name: "Org 1", OwnerID: userID}
	orgObj5 := models.Organisation{ID: org5, Name: "Org 5", OwnerID: userID}
	assert.Nil(t, db.Create(&orgObj1).Error)
	assert.Nil(t, db.Create(&orgObj5).Error)

	// Map user to Org1 and newly joined Org5
	assert.Nil(t, db.Exec("INSERT INTO user_organisations (user_id, organisation_id) VALUES (?, ?)", userID, org1).Error)
	assert.Nil(t, db.Exec("INSERT INTO user_organisations (user_id, organisation_id) VALUES (?, ?)", userID, org5).Error)

	// Point of Interaction: User joins/interacts in Org5 for the first time
	profOrg5, err := profModel.GetOrCreateProfileForOrg(db, userID, org5, profileController.Logger)
	assert.Nil(t, err)
	assert.Equal(t, org5, profOrg5.GetOrgID())

	// Validate base identity fields are inherited
	assert.Equal(t, "Custom", profOrg5.FirstName)
	assert.Equal(t, "User", profOrg5.LastName)
	assert.Equal(t, "Custom User", profOrg5.FullName)

	// Validate AvatarURL inherits base profile avatar URL when set
	assert.Equal(t, customAvatarURL, profOrg5.AvatarURL)
}
