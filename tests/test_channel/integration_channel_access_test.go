package test_channel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/channel"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func performDeleteRequest(t *testing.T, r *gin.Engine, path, token string, body any, expectedCode int) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req, err := http.NewRequest(http.MethodDelete, path, &buf)
	if err != nil {
		t.Fatalf("failed to create DELETE request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	tst.AssertStatusCode(t, rr.Code, expectedCode)
	return rr
}

func signupAndLogin(t *testing.T, r *gin.Engine, authCtrl auth.Controller, db *storage.Database, prefix string) (string, string) {
	t.Helper()
	id := utility.GenerateUUID()
	signUp := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("%s_%v@qa.team", prefix, id),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   prefix,
		LastName:    "Test",
		Password:    "password",
		UserName:    fmt.Sprintf("%s_%v", prefix, id),
	}
	tst.SignupUser(t, r, authCtrl, signUp, false)
	token := tst.GetLoginToken(t, r, authCtrl, models.LoginRequestModel{Email: signUp.Email, Password: signUp.Password})
	var u models.User
	if err := db.Postgresql.Where("email = ?", signUp.Email).First(&u).Error; err != nil {
		t.Fatalf("failed to load %s: %v", prefix, err)
	}
	return u.ID, token
}

// TestIntegrationChannelRoutes_RequireChannelAccess covers #1550: the
// integration-channel routes must check that the caller is in the channel and,
// for add and delete, may manage integrations in the channel's organisation.
func TestIntegrationChannelRoutes_RequireChannelAccess(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()
	uid := utility.GenerateUUID()

	authCtrl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger,
		ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	channelCtrl := channel.Controller{Db: db, Validator: validatorRef, Logger: logger}
	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}

	r := gin.Default()
	ch := r.Group("/api/v1/channels", middleware.Authorize(db.Postgresql))
	ch.POST("/:channelId/join", channelCtrl.JoinChannels)
	ch.POST("/:channelId/integration-channels", channelCtrl.AddIntegrationChannel)
	ch.DELETE("/:channelId/integration-channels", channelCtrl.DeleteChannelIntegration)
	ch.GET("/:channelId/integration-channels/:IntModId", channelCtrl.GetIntegrationChannels)

	// Owner A with an org and two channels.
	ownerID, ownerToken := signupAndLogin(t, r, authCtrl, db, "intowner")
	var owner models.User
	db.Postgresql.Where("id = ?", ownerID).First(&owner)
	orgID, _, _ := tst.CreateOrganisation(t, r, db, orgCtrl, models.CreateOrgRequestModel{
		Name: fmt.Sprintf("IntAccessOrg%s", uid), Description: "d", Email: owner.Email,
		Type: "test", Location: "test", Country: "test",
	}, ownerToken)
	chanX, _ := tst.CreateChannels(t, gin.Default(), channelCtrl, db, models.CreateChannelsRequest{
		Name: fmt.Sprintf("intx_%s", uid), Username: fmt.Sprintf("intx_%s", uid), OrganisationID: orgID, Description: "x",
	}, ownerToken)
	if chanX == "" {
		t.Fatalf("failed to create channel X")
	}
	chanY, _ := tst.CreateChannels(t, gin.Default(), channelCtrl, db, models.CreateChannelsRequest{
		Name: fmt.Sprintf("inty_%s", uid), Username: fmt.Sprintf("inty_%s", uid), OrganisationID: orgID, Description: "y",
	}, ownerToken)
	if chanY == "" {
		t.Fatalf("failed to create channel Y")
	}

	// Outsider B: not in the org or the channel.
	_, outsiderToken := signupAndLogin(t, r, authCtrl, db, "intoutsider")

	// Member C: in the org with the default User role and in channel X.
	memberID, memberToken := signupAndLogin(t, r, authCtrl, db, "intmember")
	var userRole models.OrgRole
	if err := db.Postgresql.Where("name = ?", "User").First(&userRole).Error; err != nil {
		t.Fatalf("failed to load User role: %v", err)
	}
	db.Postgresql.Create(&models.OrgUserManagement{UserID: memberID, OrganisationID: orgID, Status: "active", RoleID: userRole.ID})
	performPostRequest(t, r, fmt.Sprintf("/api/v1/channels/%s/join", chanX), memberToken, map[string]any{}, http.StatusOK)

	// A modifier and an output integration for the request body.
	modifier := models.Integrations{ID: utility.GenerateUUID(), Name: "intmod_" + uid, IntegrationType: "m", OwnerID: ownerID, PreSharedKey: utility.GenerateUUID()[:32]}
	output := models.Integrations{ID: utility.GenerateUUID(), Name: "intout_" + uid, IntegrationType: "o", OwnerID: ownerID, PreSharedKey: utility.GenerateUUID()[:32]}
	if err := db.Postgresql.Create(&modifier).Error; err != nil {
		t.Fatalf("failed to create modifier: %v", err)
	}
	if err := db.Postgresql.Create(&output).Error; err != nil {
		t.Fatalf("failed to create output: %v", err)
	}
	t.Cleanup(func() {
		db.Postgresql.Where("integration_modifier_id = ?", modifier.ID).Delete(&models.IntegrationOutput{})
		db.Postgresql.Where("integration_id = ?", modifier.ID).Delete(&models.IntegrationChannel{})
		db.Postgresql.Where("id IN ?", []string{modifier.ID, output.ID}).Delete(&models.Integrations{})
	})

	intChannelID := "ext-" + uid[:8]
	addBody := func(channelID string) map[string]string {
		return map[string]string{
			"int_modifier_id": modifier.ID, "int_output_id": output.ID,
			"int_channel_id": intChannelID, "int_channel_name": "ext", "channel_id": channelID,
		}
	}
	delBody := map[string]string{"int_modifier_id": modifier.ID, "int_output_id": output.ID, "int_channel_id": intChannelID}
	pathX := fmt.Sprintf("/api/v1/channels/%s/integration-channels", chanX)
	getX := fmt.Sprintf("%s/%s", pathX, modifier.ID)

	countRows := func(channelID string) int64 {
		var n int64
		db.Postgresql.Model(&models.IntegrationChannel{}).
			Where("integration_id = ? AND channel_id = ?", modifier.ID, channelID).Count(&n)
		return n
	}

	t.Run("outsider cannot add", func(t *testing.T) {
		performPostRequest(t, r, pathX, outsiderToken, addBody(""), http.StatusNotFound)
		if countRows(chanX) != 0 {
			t.Fatalf("outsider created a row")
		}
	})

	t.Run("member without manage permission cannot add", func(t *testing.T) {
		performPostRequest(t, r, pathX, memberToken, addBody(""), http.StatusForbidden)
		if countRows(chanX) != 0 {
			t.Fatalf("member created a row")
		}
	})

	t.Run("owner adds; body channel_id is ignored", func(t *testing.T) {
		performPostRequest(t, r, pathX, ownerToken, addBody(chanY), http.StatusCreated)
		if countRows(chanX) != 1 || countRows(chanY) != 0 {
			t.Fatalf("expected the row on channel X only, got X=%d Y=%d", countRows(chanX), countRows(chanY))
		}
	})

	t.Run("outsider cannot read", func(t *testing.T) {
		performGetRequest(t, r, getX, outsiderToken, http.StatusNotFound)
	})

	t.Run("member can read", func(t *testing.T) {
		performGetRequest(t, r, getX, memberToken, http.StatusOK)
	})

	t.Run("outsider cannot delete", func(t *testing.T) {
		performDeleteRequest(t, r, pathX, outsiderToken, delBody, http.StatusNotFound)
		if countRows(chanX) != 1 {
			t.Fatalf("outsider deleted the row")
		}
	})

	t.Run("member without manage permission cannot delete", func(t *testing.T) {
		performDeleteRequest(t, r, pathX, memberToken, delBody, http.StatusForbidden)
		if countRows(chanX) != 1 {
			t.Fatalf("member deleted the row")
		}
	})

	t.Run("owner deletes", func(t *testing.T) {
		performDeleteRequest(t, r, pathX, ownerToken, delBody, http.StatusOK)
		if countRows(chanX) != 0 {
			t.Fatalf("owner delete left the row")
		}
	})
}
