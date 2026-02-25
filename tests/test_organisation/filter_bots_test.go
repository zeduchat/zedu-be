package test_organisation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
	"github.com/stretchr/testify/assert"
)

func TestGetUsersBotsInOrganisation_FilterBots(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()
	user := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{
		Logger: logger,
		Test:   true,
	}}
	org := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}
	r := gin.Default()

	// Initialize creates a user and an organisation
	// This automatically adds default bots to the organisation
	orgID, token := initialise(currUUID, t, r, db, user, org, true)

	orgUrl := r.Group("/api/v1", middleware.Authorize(db.Postgresql))
	{
		orgUrl.GET("/organisations/:org_id/users", org.GetUsersBotsInOrganisation)
	}

	tests := []struct {
		Name          string
		Query         string
		ExpectedCount int  // Minimum expected count (user + potentially 0 bots)
		CheckBots     bool // If true, check if bots are present/absent based on expectation
		ExpectBots    bool
	}{
		{
			Name:       "Default: Should NOT include bots",
			Query:      "",
			CheckBots:  true,
			ExpectBots: false,
		},
		{
			Name:       "Explicit: Should include bots",
			Query:      "?include_bots=true",
			CheckBots:  true,
			ExpectBots: true,
		},
		{
			Name:       "Explicit: Should NOT include bots",
			Query:      "?include_bots=false",
			CheckBots:  true,
			ExpectBots: false,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/users%s", orgID, test.Query), nil)
			if err != nil {
				t.Fatal(err)
			}

			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusOK, rr.Code)

			var response map[string]interface{}
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			assert.NoError(t, err)

			data, ok := response["data"].([]interface{})
			if !ok {
				t.Fatalf("Response data is not a list")
			}

			if test.CheckBots {
				hasBot := false
				hasUser := false
				for _, item := range data {
					entity, ok := item.(map[string]interface{})
					if !ok {
						continue
					}
					if entityType, ok := entity["entity_type"].(string); ok {
						if entityType == "bot" {
							hasBot = true
						}
						if entityType == "user" {
							hasUser = true
						}
					}
				}

				// Always expect at least one user (the creator)
				assert.True(t, hasUser, "Response should contain at least one user")

				if test.ExpectBots {
					assert.True(t, hasBot, "Response should contain bots when requested")
				} else {
					assert.False(t, hasBot, "Response should NOT contain bots by default")
				}
			}
		})
	}
}
