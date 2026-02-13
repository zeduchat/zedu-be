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
)

func TestGetOrganisationUserRole(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()
	user := auth.Controller{Db: db, Validator: validatorRef,
		Logger: logger, ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		}}
	org := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}
	r := gin.Default()

	orgID, token := initialise(currUUID, t, r, db, user, org, false)

	tests := []struct {
		Name         string
		OrgID        string
		ExpectedCode int
		Message      string
		Headers      map[string]string
	}{
		{
			Name:         "Successful retrieval with user role info",
			OrgID:        orgID,
			ExpectedCode: http.StatusOK,
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
		},
	}

	orgUrl := r.Group("/api/v1", middleware.Authorize(db.Postgresql))
	{
		orgUrl.GET("/organisations/:org_id", org.GetOrganisation)
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s", test.OrgID), nil)
			if err != nil {
				t.Fatal(err)
			}

			for i, v := range test.Headers {
				req.Header.Set(i, v)
			}

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			if rr.Code != test.ExpectedCode {
				tst.AssertStatusCode(t, rr.Code, test.ExpectedCode)
			}

			var data map[string]any
			if err := json.NewDecoder(rr.Body).Decode(&data); err != nil {
				t.Fatalf("Failed to decode response body: %v", err)
			}

			code, ok := data["status_code"].(float64)
			if !ok {
				t.Fatalf("Expected status_code to be float64, got %T", data["status_code"])
			}
			if int(code) != test.ExpectedCode {
				tst.AssertStatusCode(t, int(code), test.ExpectedCode)
			}

			if test.Name == "Successful retrieval with user role info" && test.ExpectedCode == http.StatusOK {
				responseData, ok := data["data"].(map[string]any)
				if !ok {
					t.Fatalf("Expected data to be a map, got %T", data["data"])
				}

				currentUserRoleInfo, ok := responseData["user_role"].(map[string]any)
				if !ok {
					t.Fatalf("Expected user_role to be a map, got %T", responseData["user_role"])
				}

				roleID, hasRoleID := currentUserRoleInfo["role_id"]
				if !hasRoleID {
					t.Error("Expected user_role to contain 'role_id' field")
				}
				if roleID == nil || roleID == "" {
					t.Error("Expected role_id to have a value")
				}

				roleName, hasRoleName := currentUserRoleInfo["role_name"]
				if !hasRoleName {
					t.Error("Expected user_role to contain 'role_name' field")
				}
				if roleName == nil || roleName == "" {
					t.Error("Expected role_name to have a value")
				}

				permissions, hasPermissions := currentUserRoleInfo["permissions"]
				if !hasPermissions {
					t.Error("Expected user_role to contain 'permissions' field")
				}
				if permissions == nil {
					t.Error("Expected permissions to not be nil")
				}

				t.Logf("User role info validated successfully - RoleID: %v, RoleName: %v, Permissions: %v", roleID, roleName, permissions)
			}
		})
	}
}
