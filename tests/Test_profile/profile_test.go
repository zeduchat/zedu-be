package test_profile

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	profileService "github.com/hngprojects/telex_be/services/profile"
	"github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestProfileFlow(t *testing.T) {
	_, profileController := SetupProfileTestRouter()
	db := profileController.Db.Postgresql
	currUUID := utility.GenerateUUID()
	password, err := utility.HashPassword("password")
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	regularUser := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "Regular User",
		Email:    fmt.Sprintf("user%v@qa.team", currUUID),
		Password: password,
	}

	db.Create(&regularUser)

	setup := func() (*gin.Engine, *auth.Controller) {
		router, profileController := SetupProfileTestRouter()
		authController := auth.Controller{
			Db:        profileController.Db,
			Validator: profileController.Validator,
			Logger:    profileController.Logger,
			ExtReq:    profileController.ExtReq,
		}
		return router, &authController
	}

	t.Run("Successfully Get User Profile", func(t *testing.T) {
		router, authController := setup()

		loginData := models.LoginRequestModel{
			Email:    regularUser.Email,
			Password: "password",
		}
		token := tests.GetLoginToken(t, router, *authController, loginData)

		req, err := http.NewRequest(http.MethodGet, "/api/v1/profile", nil)
		if err != nil {
			t.Fatalf("Failed to create new request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		tests.AssertStatusCode(t, resp.Code, http.StatusOK)
	})

	t.Run("Org Profile Binding and Hydration Test", func(t *testing.T) {
		org1 := utility.GenerateUUID()
		org2 := utility.GenerateUUID()
		uID := utility.GenerateUUID()

		testUser := models.User{
			ID:    uID,
			Name:  "Test User Org",
			Email: fmt.Sprintf("testorg%s@qa.team", utility.GenerateUUID()),
		}
		if err := db.Create(&testUser).Error; err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}

		var profModel models.Profile
		profOrg1, err := profModel.GetOrCreateProfileForOrg(db, uID, org1)
		if err != nil {
			t.Fatalf("Failed to create profile for org1: %v", err)
		}
		if profOrg1.GetOrgID() != org1 {
			t.Errorf("Expected org1 profile, got orgID %s", profOrg1.GetOrgID())
		}

		profOrg2, err := profModel.GetOrCreateProfileForOrg(db, uID, org2)
		if err != nil {
			t.Fatalf("Failed to create profile for org2: %v", err)
		}
		if profOrg2.GetOrgID() != org2 {
			t.Errorf("Expected org2 profile, got orgID %s", profOrg2.GetOrgID())
		}

		if profOrg1.ID == profOrg2.ID {
			t.Errorf("Expected different profile IDs for different orgs, got same: %s", profOrg1.ID)
		}

		msgDocLegacy := models.MessageDocument{
			ID:             utility.GenerateUUID(),
			UserID:         uID,
			OrganisationID: org1,
			Content:        "Legacy message test",
		}
		hydrated := models.HydrateMessageProfiles(db, []models.MessageDocument{msgDocLegacy})
		if len(hydrated) == 0 {
			t.Fatalf("Expected hydrated message slice")
		}
		if hydrated[0].ProfileID != profOrg1.ID {
			t.Errorf("Expected hydrated ProfileID to match org1 profile %s, got %s", profOrg1.ID, hydrated[0].ProfileID)
		}
	})

	t.Run("Profile Picture Upload and Org Binding Test", func(t *testing.T) {
		uID := utility.GenerateUUID()
		testUser := models.User{
			ID:    uID,
			Name:  "Test Avatar User",
			Email: fmt.Sprintf("avataruser%s@qa.team", utility.GenerateUUID()),
		}
		if err := db.Create(&testUser).Error; err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}

		userID := testUser.ID
		orgA := utility.GenerateUUID()
		orgB := utility.GenerateUUID()

		dummyFile := []byte("dummy image content")
		url1, err := profileService.UploadProfileImage(profileController.Logger, db, userID, dummyFile, "png", orgA)
		if err != nil {
			t.Fatalf("UploadProfileImage failed: %v", err)
		}
		if url1 == "" {
			t.Fatalf("Expected non-empty URL from UploadProfileImage")
		}

		url2, err := profileService.UploadProfileImage(profileController.Logger, db, userID, dummyFile, "png", orgA)
		if err != nil {
			t.Fatalf("Second UploadProfileImage failed: %v", err)
		}
		if url1 == url2 {
			t.Errorf("Expected cache-busting unique URLs, got identical URL: %s", url1)
		}

		var profModel models.Profile
		profA, err := profModel.GetOrCreateProfileForOrg(db, userID, orgA)
		if err != nil {
			t.Fatalf("Failed to fetch profile for orgA: %v", err)
		}
		db.Model(&models.Profile{}).Where("id = ?", profA.ID).Update("avatar_url", url2)

		profB, err := profModel.GetOrCreateProfileForOrg(db, userID, orgB)
		if err != nil {
			t.Fatalf("Failed to fetch profile for orgB: %v", err)
		}
		if profB.AvatarURL != url2 {
			t.Errorf("Expected cloned profile for orgB to maintain base profile avatarURL %s, got %s", url2, profB.AvatarURL)
		}

		// Upload distinct profile picture for Org B
		dummyFileB := []byte("dummy image content org B")
		url3, err := profileService.UploadProfileImage(profileController.Logger, db, userID, dummyFileB, "png", orgB)
		if err != nil {
			t.Fatalf("UploadProfileImage for orgB failed: %v", err)
		}
		db.Model(&models.Profile{}).Where("id = ?", profB.ID).Update("avatar_url", url3)

		fetchedURLA, err := profileService.GetUserProfileImageURL(db, userID, orgA)
		if err != nil || fetchedURLA != url2 {
			t.Errorf("Expected orgA avatar URL %s, got %s (err: %v)", url2, fetchedURLA, err)
		}

		fetchedURLB, err := profileService.GetUserProfileImageURL(db, userID, orgB)
		if err != nil || fetchedURLB != url3 {
			t.Errorf("Expected orgB avatar URL %s, got %s (err: %v)", url3, fetchedURLB, err)
		}

		if fetchedURLA == fetchedURLB {
			t.Errorf("Expected orgA and orgB avatar URLs to be distinct, but both matched: %s", fetchedURLA)
		}
	})

	t.Run("GetOrCreateProfileForOrg Zero UUID Fallback Test", func(t *testing.T) {
		uID := utility.GenerateUUID()
		testUser := models.User{
			ID:    uID,
			Name:  "Zero UUID User",
			Email: fmt.Sprintf("zerouuid%s@qa.team", utility.GenerateUUID()),
		}
		if err := db.Create(&testUser).Error; err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}

		var profModel models.Profile
		zeroUUID := "00000000-0000-0000-0000-000000000000"
		prof, err := profModel.GetOrCreateProfileForOrg(db, uID, zeroUUID)
		if err != nil {
			t.Fatalf("GetOrCreateProfileForOrg failed with zero UUID: %v", err)
		}
		if prof.Userid != uID {
			t.Errorf("Expected user ID %s, got %s", uID, prof.Userid)
		}
	})

	t.Run("GetUserByID Org Context Test", func(t *testing.T) {
		uID := utility.GenerateUUID()
		org1 := utility.GenerateUUID()
		org2 := utility.GenerateUUID()

		testUser := models.User{
			ID:    uID,
			Name:  "GetUserByID User",
			Email: fmt.Sprintf("getuserbyid%s@qa.team", utility.GenerateUUID()),
		}
		if err := db.Create(&testUser).Error; err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}

		var profModel models.Profile
		prof1, _ := profModel.GetOrCreateProfileForOrg(db, uID, org1)
		prof2, _ := profModel.GetOrCreateProfileForOrg(db, uID, org2)

		db.Model(&models.Profile{}).Where("id = ?", prof1.ID).Update("first_name", "FirstNameOrg1")
		db.Model(&models.Profile{}).Where("id = ?", prof2.ID).Update("first_name", "FirstNameOrg2")

		var userModel models.User
		uOrg1, err := userModel.GetUserByID(db, uID, org1)
		if err != nil || uOrg1.Profile.FirstName != "FirstNameOrg1" {
			t.Errorf("Expected profile first_name 'FirstNameOrg1' for org1, got '%s' (err: %v)", uOrg1.Profile.FirstName, err)
		}

		uOrg2, err := userModel.GetUserByID(db, uID, org2)
		if err != nil || uOrg2.Profile.FirstName != "FirstNameOrg2" {
			t.Errorf("Expected profile first_name 'FirstNameOrg2' for org2, got '%s' (err: %v)", uOrg2.Profile.FirstName, err)
		}
	})

	t.Run("SetProfileImageToEmpty Org Specific Test", func(t *testing.T) {
		uID := utility.GenerateUUID()
		org1 := utility.GenerateUUID()
		org2 := utility.GenerateUUID()

		testUser := models.User{
			ID:    uID,
			Name:  "SetProfileImage User",
			Email: fmt.Sprintf("setprofileimage%s@qa.team", utility.GenerateUUID()),
		}
		if err := db.Create(&testUser).Error; err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}

		var profModel models.Profile
		prof1, _ := profModel.GetOrCreateProfileForOrg(db, uID, org1)
		prof2, _ := profModel.GetOrCreateProfileForOrg(db, uID, org2)

		db.Model(&models.Profile{}).Where("id = ?", prof1.ID).Update("avatar_url", "http://example.com/avatar1.png")
		db.Model(&models.Profile{}).Where("id = ?", prof2.ID).Update("avatar_url", "http://example.com/avatar2.png")

		err := profModel.SetProfileImageToEmpty(db, uID, org1)
		if err != nil {
			t.Fatalf("SetProfileImageToEmpty failed: %v", err)
		}

		var checkProf1, checkProf2 models.Profile
		db.First(&checkProf1, "id = ?", prof1.ID)
		db.First(&checkProf2, "id = ?", prof2.ID)

		if checkProf1.AvatarURL != "" {
			t.Errorf("Expected org1 avatar_url to be empty, got '%s'", checkProf1.AvatarURL)
		}
		if checkProf2.AvatarURL != "http://example.com/avatar2.png" {
			t.Errorf("Expected org2 avatar_url to remain 'http://example.com/avatar2.png', got '%s'", checkProf2.AvatarURL)
		}
	})

	t.Run("DeleteUserProfileImage Targeted Org Test", func(t *testing.T) {
		uID := utility.GenerateUUID()
		org1 := utility.GenerateUUID()
		org2 := utility.GenerateUUID()

		testUser := models.User{
			ID:    uID,
			Name:  "DeleteAvatar User",
			Email: fmt.Sprintf("deleteavatar%s@qa.team", utility.GenerateUUID()),
		}
		if err := db.Create(&testUser).Error; err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}

		var profModel models.Profile
		prof1, _ := profModel.GetOrCreateProfileForOrg(db, uID, org1)
		prof2, _ := profModel.GetOrCreateProfileForOrg(db, uID, org2)

		db.Model(&models.Profile{}).Where("id = ?", prof1.ID).Update("avatar_url", "http://example.com/avatar1.png")
		db.Model(&models.Profile{}).Where("id = ?", prof2.ID).Update("avatar_url", "http://example.com/avatar2.png")

		_, err := profileService.DeleteUserProfileImage(db, profileController.Logger, uID, org1)
		if err != nil {
			t.Fatalf("DeleteUserProfileImage failed: %v", err)
		}

		var checkProf1, checkProf2 models.Profile
		db.First(&checkProf1, "id = ?", prof1.ID)
		db.First(&checkProf2, "id = ?", prof2.ID)

		if checkProf1.AvatarURL != "" {
			t.Errorf("Expected org1 avatar_url to be cleared, got '%s'", checkProf1.AvatarURL)
		}
		if checkProf2.AvatarURL != "http://example.com/avatar2.png" {
			t.Errorf("Expected org2 avatar_url to remain intact, got '%s'", checkProf2.AvatarURL)
		}
	})

	t.Run("GetOrCreateMultipleProfilesForOrg Bulk Resolution Test", func(t *testing.T) {
		orgID := utility.GenerateUUID()
		uID1 := utility.GenerateUUID()
		uID2 := utility.GenerateUUID()
		uID3 := utility.GenerateUUID()

		user1 := models.User{ID: uID1, Name: "User One", Email: fmt.Sprintf("u1_%s@qa.team", utility.GenerateUUID())}
		user2 := models.User{ID: uID2, Name: "User Two", Email: fmt.Sprintf("u2_%s@qa.team", utility.GenerateUUID())}
		user3 := models.User{ID: uID3, Name: "User Three", Email: fmt.Sprintf("u3_%s@qa.team", utility.GenerateUUID())}

		db.Create(&user1)
		db.Create(&user2)
		db.Create(&user3)

		var profModel models.Profile
		inputUserIDs := []string{uID1, uID2, uID3, "WEBHOOK", ""}
		profilesMap, err := profModel.GetOrCreateMultipleProfilesForOrg(db, inputUserIDs, orgID)
		if err != nil {
			t.Fatalf("GetOrCreateMultipleProfilesForOrg failed: %v", err)
		}

		if len(profilesMap) != 3 {
			t.Fatalf("Expected 3 profiles in map, got %d", len(profilesMap))
		}

		if p1, ok := profilesMap[uID1]; !ok || p1.FullName != "User One" {
			t.Errorf("Expected profile for user1 with FullName 'User One', got: %+v", p1)
		}
		if p2, ok := profilesMap[uID2]; !ok || p2.FullName != "User Two" {
			t.Errorf("Expected profile for user2 with FullName 'User Two', got: %+v", p2)
		}
		if p3, ok := profilesMap[uID3]; !ok || p3.FullName != "User Three" {
			t.Errorf("Expected profile for user3 with FullName 'User Three', got: %+v", p3)
		}

		threads := []models.ThreadDocument{
			{ID: utility.GenerateUUID(), UserId: uID1, OrganisationID: orgID},
			{ID: utility.GenerateUUID(), UserId: uID2, OrganisationID: orgID},
			{ID: utility.GenerateUUID(), UserId: uID3, OrganisationID: orgID},
		}

		hydratedThreads := models.HydrateThreadProfiles(db, threads)
		if len(hydratedThreads) != 3 {
			t.Fatalf("Expected 3 hydrated threads, got %d", len(hydratedThreads))
		}
		if hydratedThreads[0].ProfileID == "" || hydratedThreads[0].FullName != "User One" {
			t.Errorf("Expected hydrated thread[0] FullName 'User One', got %s", hydratedThreads[0].FullName)
		}
	})
}

