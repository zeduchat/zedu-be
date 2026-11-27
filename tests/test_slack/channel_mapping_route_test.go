package test_slack

import (
	"bytes"
	"encoding/json"
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

// TestSlackChannelMappingRoute checks that the real router registers the
// mapping route with a :org_id parameter (#1501), so a request with an actual
// organisation id reaches the handler, the org id is stored, and only members
// of that organisation can create a mapping.
func TestSlackChannelMappingRoute(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()
	v := validator.New()
	authController := auth.Controller{Db: db, Validator: v, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	orgController := organisation.Controller{Db: db, Validator: v, Logger: logger}

	r := gin.Default()
	router.Slack(r, "/api/v1", v, db, logger)

	newOwner := func() (string, string) {
		// Each owner gets its own setup engine: the test helpers register
		// their routes on the engine they are given.
		setup := gin.Default()
		id := utility.GenerateUUID()
		signup := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("slackmap_%v@qa.team", id),
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
			FirstName:   "slack",
			LastName:    "mapper",
			Password:    "password",
			UserName:    fmt.Sprintf("slackmap_%v", id),
		}
		tst.SignupUser(t, setup, authController, signup, false)
		token := tst.GetLoginToken(t, setup, authController, models.LoginRequestModel{Email: signup.Email, Password: signup.Password})
		orgID, _, _ := tst.CreateOrganisation(t, setup, db, orgController, models.CreateOrgRequestModel{
			Name:        fmt.Sprintf("org-%v", id),
			Email:       fmt.Sprintf("org_%v@qa.team", id),
			Description: "slack mapping test org",
			Type:        "Tech",
			Location:    "Lagos",
			Country:     "Nigeria",
		}, token)
		require.NotEmpty(t, orgID)
		return orgID, token
	}

	orgA, tokenA := newOwner()
	orgB, tokenB := newOwner()
	require.NotEqual(t, orgA, orgB)

	post := func(orgID, token string) *httptest.ResponseRecorder {
		var b bytes.Buffer
		require.NoError(t, json.NewEncoder(&b).Encode(models.TelexSlackChannelMappingReq{TelexChannelName: "general", SlackChannelName: "general"}))
		req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/organisation/%s/channels/mapping", orgID), &b)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr
	}

	t.Run("member creates mapping for own org", func(t *testing.T) {
		rr := post(orgA, tokenA)
		require.Equal(t, http.StatusCreated, rr.Code, rr.Body.String())
		var resp struct {
			Data models.TelexSlackChannelMapping `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
		assert.Equal(t, orgA, resp.Data.OrganisationID)

		var stored models.TelexSlackChannelMapping
		require.NoError(t, db.Postgresql.Where("id = ?", resp.Data.ID).First(&stored).Error)
		assert.Equal(t, orgA, stored.OrganisationID)
		db.Postgresql.Delete(&stored)
	})

	t.Run("unauthenticated request reaches auth, not 404", func(t *testing.T) {
		assert.Equal(t, http.StatusUnauthorized, post(orgA, "").Code)
	})

	t.Run("non-member is refused", func(t *testing.T) {
		rr := post(orgA, tokenB)
		assert.Equal(t, http.StatusForbidden, rr.Code, rr.Body.String())
		var n int64
		db.Postgresql.Model(&models.TelexSlackChannelMapping{}).Where("organisation_id = ?", orgA).Count(&n)
		assert.Zero(t, n)
	})

	t.Run("literal {org_id} path no longer matches", func(t *testing.T) {
		assert.Equal(t, http.StatusBadRequest, post("{org_id}", tokenA).Code)
	})
}
