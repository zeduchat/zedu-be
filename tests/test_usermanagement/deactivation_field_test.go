package test_channel

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
)

func TestOrgUserSoftDeactivationAndIsDeactivatedField(t *testing.T) {
	orgId, adminToken, memberUserID, _ := setupTwoUsersInOrg(t)

	db := storage.Connection()
	logger := tst.Setup()

	orgCtrl := organisation.Controller{
		Db:        db,
		Validator: validator.New(),
		Logger:    logger,
	}

	// 1. Initially verify that member is not deactivated (is_deactivated = false)
	t.Run("Verify is_deactivated initially false", func(t *testing.T) {
		r := gin.Default()
		orgUrl := r.Group("/api/v1/organisations", middleware.Authorize(db.Postgresql))
		orgUrl.GET("/:org_id/users", orgCtrl.GetUsersBotsInOrganisation)

		reqURI := url.URL{Path: fmt.Sprintf("/api/v1/organisations/%s/users", orgId)}
		req, _ := http.NewRequest(http.MethodGet, reqURI.String(), nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)
		data := tst.ParseResponse(rr)
		usersList, ok := data["data"].([]interface{})
		if !ok || len(usersList) == 0 {
			t.Fatalf("expected users list in response, got %v", data["data"])
		}

		foundMember := false
		for _, uItem := range usersList {
			uMap, _ := uItem.(map[string]interface{})
			if uMap["id"] == memberUserID {
				foundMember = true
				if isDeac, ok := uMap["is_deactivated"].(bool); !ok || isDeac {
					t.Errorf("expected is_deactivated=false initially for member, got %v", uMap["is_deactivated"])
				}
			}
		}
		if !foundMember {
			t.Errorf("member %s not found in org users list", memberUserID)
		}
	})

	// 2. Remove member from org (soft deactivation)
	t.Run("Soft deactivate member via RemoveMemberFromOrganisation", func(t *testing.T) {
		r := gin.Default()
		orgUrl := r.Group("/api/v1/organisations", middleware.Authorize(db.Postgresql))
		orgUrl.DELETE("/:org_id/users/:user_id", orgCtrl.RemoveMemberFromOrganisation)

		reqURI := url.URL{Path: fmt.Sprintf("/api/v1/organisations/%s/users/%s", orgId, memberUserID)}
		req, _ := http.NewRequest(http.MethodDelete, reqURI.String(), nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)

		// Verify database state: row still exists, is_deactivated = true, status = inactive
		isDeactivated, status := getOrgMemberState(t, db, memberUserID, orgId)
		if !isDeactivated {
			t.Errorf("expected is_deactivated=true after RemoveMemberFromOrganisation, got %v", isDeactivated)
		}
		if status != "inactive" {
			t.Errorf("expected status='inactive', got %q", status)
		}
	})

	// 3. Verify get org users returns is_deactivated = true for the member
	t.Run("Verify is_deactivated is true in GetUsersBotsInOrganisation", func(t *testing.T) {
		r := gin.Default()
		orgUrl := r.Group("/api/v1/organisations", middleware.Authorize(db.Postgresql))
		orgUrl.GET("/:org_id/users", orgCtrl.GetUsersBotsInOrganisation)

		reqURI := url.URL{Path: fmt.Sprintf("/api/v1/organisations/%s/users", orgId)}
		req, _ := http.NewRequest(http.MethodGet, reqURI.String(), nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		tst.AssertStatusCode(t, rr.Code, http.StatusOK)
		data := tst.ParseResponse(rr)
		usersList, ok := data["data"].([]interface{})
		if !ok || len(usersList) == 0 {
			t.Fatalf("expected users list in response, got %v", data["data"])
		}

		foundMember := false
		for _, uItem := range usersList {
			uMap, _ := uItem.(map[string]interface{})
			if uMap["id"] == memberUserID {
				foundMember = true
				if isDeac, ok := uMap["is_deactivated"].(bool); !ok || !isDeac {
					t.Errorf("expected is_deactivated=true after deactivation, got %v", uMap["is_deactivated"])
				}
			}
		}
		if !foundMember {
			t.Errorf("member %s should still exist in org profile with is_deactivated=true", memberUserID)
		}
	})

	// 4. Verify MessageDocument hydration includes IsDeactivated
	t.Run("Verify MessageDocument profile hydration includes IsDeactivated", func(t *testing.T) {
		msgs := []models.MessageDocument{
			{
				ID:             "msg-1",
				UserID:         memberUserID,
				OrganisationID: orgId,
			},
		}

		hydrated := models.HydrateMessageProfiles(db.Postgresql, msgs)
		if len(hydrated) == 0 {
			t.Fatalf("expected 1 hydrated message, got 0")
		}

		if !hydrated[0].IsDeactivated {
			t.Errorf("expected hydrated message IsDeactivated=true for deactivated user, got false")
		}
	})

	t.Run("Verify GetUserOrganisations excludes deactivated organisation", func(t *testing.T) {
		var orgModel models.Organisation
		orgs, err := orgModel.GetUserOrganisations(db.Postgresql, memberUserID)
		if err != nil {
			t.Fatalf("unexpected error getting user orgs: %v", err)
		}
		for _, o := range orgs {
			if o.ID == orgId {
				t.Errorf("deactivated organisation %s should not appear in user org listings", orgId)
			}
		}
	})
}
