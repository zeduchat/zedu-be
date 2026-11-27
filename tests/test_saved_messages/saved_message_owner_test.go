package test_saved_messages

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/router"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

// TestSavedMessageOwnerScope checks that GET and DELETE
// /organisations/:org_id/saved/message/:smId only reach the caller's own saved
// message in that organisation (#1554).
func TestSavedMessageOwnerScope(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()
	v := validator.New()
	authController := auth.Controller{Db: db, Validator: v, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	orgController := organisation.Controller{Db: db, Validator: v, Logger: logger}

	r := gin.Default()
	router.SavedMessages(r, "/api/v1", v, db, logger)

	newOwner := func() (orgID, userID, token string) {
		// Each owner gets its own setup engine: the test helpers register
		// their routes on the engine they are given.
		setup := gin.Default()
		id := utility.GenerateUUID()
		signup := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("savedmsg_%v@qa.team", id),
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
			FirstName:   "saved",
			LastName:    "owner",
			Password:    "password",
			UserName:    fmt.Sprintf("savedmsg_%v", id),
		}
		tst.SignupUser(t, setup, authController, signup, false)
		token = tst.GetLoginToken(t, setup, authController, models.LoginRequestModel{Email: signup.Email, Password: signup.Password})
		orgID, _, userID = tst.CreateOrganisation(t, setup, db, orgController, models.CreateOrgRequestModel{
			Name:        fmt.Sprintf("org-%v", id),
			Email:       fmt.Sprintf("org_%v@qa.team", id),
			Description: "saved message test org",
			Type:        "Tech",
			Location:    "Lagos",
			Country:     "Nigeria",
		}, token)
		require.NotEmpty(t, orgID)
		return orgID, userID, token
	}

	orgA, userA, tokenA := newOwner()
	orgB, _, tokenB := newOwner()

	newSaved := func() string {
		m := models.SavedMessage{
			ID:         utility.GenerateUUID(),
			ChannelsID: utility.GenerateUUID(),
			OrgId:      orgA,
			UserID:     userA,
			ThreadID:   utility.GenerateUUID(),
			Type:       "message",
		}
		require.NoError(t, db.Postgresql.Create(&m).Error)
		t.Cleanup(func() { db.Postgresql.Unscoped().Where("id = ?", m.ID).Delete(&models.SavedMessage{}) })
		return m.ID
	}
	exists := func(id string) bool {
		var n int64
		db.Postgresql.Unscoped().Model(&models.SavedMessage{}).Where("id = ?", id).Count(&n)
		return n > 0
	}
	do := func(method, orgID, smID, token string) *httptest.ResponseRecorder {
		req, err := http.NewRequest(method, fmt.Sprintf("/api/v1/organisations/%s/saved/message/%s", orgID, smID), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr
	}

	sm := newSaved()

	t.Run("owner GET allowed", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, do(http.MethodGet, orgA, sm, tokenA).Code)
	})
	t.Run("other user GET with own org denied", func(t *testing.T) {
		assert.Equal(t, http.StatusNotFound, do(http.MethodGet, orgB, sm, tokenB).Code)
	})
	t.Run("other user DELETE with own org denied", func(t *testing.T) {
		rr := do(http.MethodDelete, orgB, sm, tokenB)
		assert.Equal(t, http.StatusNotFound, rr.Code, rr.Body.String())
		assert.True(t, exists(sm), "another user's saved message must not be deleted")
	})
	t.Run("owner with wrong org in path denied", func(t *testing.T) {
		// Make A a member of B so the membership check passes and only the
		// org scope of the saved-message lookup can refuse it.
		require.NoError(t, db.Postgresql.Exec("INSERT INTO user_organisations (user_id, organisation_id) VALUES (?, ?)", userA, orgB).Error)
		t.Cleanup(func() {
			db.Postgresql.Exec("DELETE FROM user_organisations WHERE user_id = ? AND organisation_id = ?", userA, orgB)
		})
		rr := do(http.MethodDelete, orgB, sm, tokenA)
		assert.Equal(t, http.StatusNotFound, rr.Code, rr.Body.String())
		assert.True(t, exists(sm))
	})
	t.Run("owner DELETE allowed", func(t *testing.T) {
		do(http.MethodDelete, orgA, sm, tokenA)
		assert.False(t, exists(sm), "owner's saved message should be deleted")
	})
}
