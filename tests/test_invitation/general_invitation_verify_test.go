package test_invitation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	inviteCtrl "github.com/hngprojects/telex_be/pkg/controller/invitation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

// createTestUserHelper creates a new user and returns their login token and user ID.
func createTestUserHelper(t *testing.T, db *storage.Database, logger *utility.Logger, validatorRef *validator.Validate) (string, string) {
	t.Helper()
	uuid := utility.GenerateUUID()
	signUp := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("genuser_%v@qa.team", uuid),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "GenUser",
		LastName:    "Test",
		Password:    "password",
		UserName:    fmt.Sprintf("genuser_%v", uuid),
	}
	login := models.LoginRequestModel{
		Email:    signUp.Email,
		Password: signUp.Password,
	}

	authCtrl := auth.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	r := gin.Default()
	tst.SignupUser(t, r, authCtrl, signUp, false)
	token := tst.GetLoginToken(t, r, authCtrl, login)
	userID := tst.GetUserIDFromToken(t, token, db)

	return token, userID
}

func TestGeneralInvitationVerify_Success(t *testing.T) {
	orgID, ownerToken, _ := setupOrgWithGeneralInvite(t)

	db := storage.Connection()
	logger := tst.Setup()
	validatorRef := validator.New()

	// Get the generated shareable invite
	invite := getMostRecentGeneralInvite(t, db, orgID)

	// Create a new user who will verify/accept the general invitation link
	joiningToken, joiningUserID := createTestUserHelper(t, db, logger, validatorRef)

	invCtrl := inviteCtrl.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	r := gin.Default()
	invUrl := r.Group("/api/v1/invite", middleware.Authorize(db.Postgresql))
	invUrl.POST("/general/verify", invCtrl.GeneralInvitationVerify)

	body := models.VerifyShareableInvitationLink{
		Token: invite.Token,
	}
	var b bytes.Buffer
	json.NewEncoder(&b).Encode(body)

	req, err := http.NewRequest(http.MethodPost, "/api/v1/invite/general/verify", &b)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+joiningToken)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusOK)
	data := tst.ParseResponse(rr)
	tst.AssertResponseMessage(t, data["message"].(string), "User invited successfully")

	// Verify user is now a member of the organization in DB
	isMember := db.Postgresql.Where("user_id = ? AND organisation_id = ?", joiningUserID, orgID).
		First(&models.OrgUserManagement{}).Error == nil
	if !isMember {
		t.Errorf("expected user %s to be added to organisation %s, but wasn't", joiningUserID, orgID)
	}

	// Verify user's current_org is updated to the invited org ID
	var updatedUser models.User
	if err := db.Postgresql.Where("id = ?", joiningUserID).First(&updatedUser).Error; err != nil {
		t.Fatalf("failed to query user: %v", err)
	}
	if updatedUser.CurrentOrg.String() != orgID {
		t.Errorf("expected current_org=%s, got %s", orgID, updatedUser.CurrentOrg.String())
	}

	_ = ownerToken
}

func TestGeneralInvitationVerify_AlreadyMember(t *testing.T) {
	orgID, _, _ := setupOrgWithGeneralInvite(t)

	db := storage.Connection()
	logger := tst.Setup()
	validatorRef := validator.New()

	invite := getMostRecentGeneralInvite(t, db, orgID)
	joiningToken, _ := createTestUserHelper(t, db, logger, validatorRef)

	invCtrl := inviteCtrl.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	r := gin.Default()
	invUrl := r.Group("/api/v1/invite", middleware.Authorize(db.Postgresql))
	invUrl.POST("/general/verify", invCtrl.GeneralInvitationVerify)

	// First verification: user joins org
	body := models.VerifyShareableInvitationLink{Token: invite.Token}
	var b1 bytes.Buffer
	json.NewEncoder(&b1).Encode(body)
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/invite/general/verify", &b1)
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+joiningToken)

	rr1 := httptest.NewRecorder()
	r.ServeHTTP(rr1, req1)
	tst.AssertStatusCode(t, rr1.Code, http.StatusOK)

	// Second verification: user is ALREADY a member, should be idempotent and return 200 OK
	var b2 bytes.Buffer
	json.NewEncoder(&b2).Encode(body)
	req2, _ := http.NewRequest(http.MethodPost, "/api/v1/invite/general/verify", &b2)
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+joiningToken)

	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)
	tst.AssertStatusCode(t, rr2.Code, http.StatusOK)
}

func TestGeneralInvitationVerify_Deactivated(t *testing.T) {
	orgID, ownerToken, _ := setupOrgWithGeneralInvite(t)

	db := storage.Connection()
	logger := tst.Setup()
	validatorRef := validator.New()

	invCtrl := inviteCtrl.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	// Disable/deactivate the invite link
	rrStatus := callChangeGeneralInviteStatus(t, db, invCtrl, ownerToken, false)
	tst.AssertStatusCode(t, rrStatus.Code, http.StatusOK)

	invite := getMostRecentGeneralInvite(t, db, orgID)
	joiningToken, _ := createTestUserHelper(t, db, logger, validatorRef)

	r := gin.Default()
	invUrl := r.Group("/api/v1/invite", middleware.Authorize(db.Postgresql))
	invUrl.POST("/general/verify", invCtrl.GeneralInvitationVerify)

	body := models.VerifyShareableInvitationLink{Token: invite.Token}
	var b bytes.Buffer
	json.NewEncoder(&b).Encode(body)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/invite/general/verify", &b)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+joiningToken)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusNotFound)
}

func TestGeneralInvitationVerify_Expired(t *testing.T) {
	orgID, _, roleID := setupOrgWithGeneralInvite(t)

	db := storage.Connection()
	logger := tst.Setup()
	validatorRef := validator.New()

	// Insert an expired general invitation link manually
	expiredInvite := models.GeneralInvitation{
		ID:             utility.GenerateUUID(),
		Token:          "expired-general-token-" + utility.GenerateUUID(),
		ActiveStatus:   true,
		Role:           roleID,
		OrganisationID: orgID,
		InvitedBy:      utility.GenerateUUID(),
		CreatedAt:      time.Now().Add(-72 * time.Hour),
		ExpiresAt:      time.Now().Add(-1 * time.Hour), // expired 1 hour ago
	}
	if err := db.Postgresql.Create(&expiredInvite).Error; err != nil {
		t.Fatalf("failed to insert expired general invite: %v", err)
	}

	joiningToken, _ := createTestUserHelper(t, db, logger, validatorRef)

	invCtrl := inviteCtrl.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	r := gin.Default()
	invUrl := r.Group("/api/v1/invite", middleware.Authorize(db.Postgresql))
	invUrl.POST("/general/verify", invCtrl.GeneralInvitationVerify)

	body := models.VerifyShareableInvitationLink{Token: expiredInvite.Token}
	var b bytes.Buffer
	json.NewEncoder(&b).Encode(body)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/invite/general/verify", &b)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+joiningToken)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusNotFound)
}

func TestGeneralInvitationVerify_InvalidToken(t *testing.T) {
	db := storage.Connection()
	logger := tst.Setup()
	validatorRef := validator.New()

	joiningToken, _ := createTestUserHelper(t, db, logger, validatorRef)

	invCtrl := inviteCtrl.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	r := gin.Default()
	invUrl := r.Group("/api/v1/invite", middleware.Authorize(db.Postgresql))
	invUrl.POST("/general/verify", invCtrl.GeneralInvitationVerify)

	body := models.VerifyShareableInvitationLink{Token: "non-existent-token-xyz-123"}
	var b bytes.Buffer
	json.NewEncoder(&b).Encode(body)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/invite/general/verify", &b)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+joiningToken)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	tst.AssertStatusCode(t, rr.Code, http.StatusNotFound)
}

func TestGeneralInvitationVerify_Unauthorized(t *testing.T) {
	db := storage.Connection()
	logger := tst.Setup()
	validatorRef := validator.New()

	invCtrl := inviteCtrl.Controller{
		Db:        db,
		Validator: validatorRef,
		Logger:    logger,
		ExtReq:    request.ExternalRequest{Logger: logger, Test: true},
	}

	r := gin.Default()
	invUrl := r.Group("/api/v1/invite", middleware.Authorize(db.Postgresql))
	invUrl.POST("/general/verify", invCtrl.GeneralInvitationVerify)

	body := models.VerifyShareableInvitationLink{Token: "some-token"}
	var b bytes.Buffer
	json.NewEncoder(&b).Encode(body)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/invite/general/verify", &b)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Authorization middleware should reject unauthenticated request with 401
	tst.AssertStatusCode(t, rr.Code, http.StatusUnauthorized)
}
