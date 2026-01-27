package test_admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hngprojects/telex_be/internal/models"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestAdminRoleChangeFlow(t *testing.T) {
	r, _, _, db := SetupAdminTestRouter()

	// Create a Super Admin and get token
	superAdminID, superAdminToken := CreateSuperAdminAndGetTokenWithID(t, r, db)

	// Create a Target Admin (to be upgraded)
	targetAdminID := utility.GenerateUUID()
	targetAdmin := models.Admin{
		ID:       targetAdminID,
		Name:     "Target Admin",
		Email:    "target@qa.team",
		Role:     models.RoleAdmin,
		IsActive: true,
	}
	db.Postgresql.Create(&targetAdmin)

	t.Cleanup(func() {
		CleanupSpecificTestData(db.Postgresql, superAdminID, []string{})
		db.Postgresql.Exec("DELETE FROM admins WHERE id = ?", targetAdminID)
		db.Postgresql.Exec("DELETE FROM role_change_confirmations WHERE target_admin_id = ?", targetAdminID)
		db.Postgresql.Exec("DELETE FROM audit_logs WHERE resource_id = ?", targetAdminID)
	})

	t.Run("Initiate Role Change - Missing Confirmation Flag", func(t *testing.T) {
		payload := map[string]any{
			"new_role": models.RoleSuperAdmin,
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/backoffice/admins/%s/role/initiate", targetAdminID), bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Initiate Role Change - Confirmation False", func(t *testing.T) {
		isConfirmed := false
		payload := models.ChangeAdminRoleRequest{
			NewRole:     models.RoleSuperAdmin,
			IsConfirmed: &isConfirmed,
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/backoffice/admins/%s/role/initiate", targetAdminID), bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusPreconditionRequired, rr.Code)
	})

	var confirmationToken string

	t.Run("Initiate Role Change - Success (Acceptance)", func(t *testing.T) {
		isConfirmed := true
		payload := models.ChangeAdminRoleRequest{
			NewRole:     models.RoleSuperAdmin,
			IsConfirmed: &isConfirmed,
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/backoffice/admins/%s/role/initiate", targetAdminID), bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusAccepted, rr.Code)

		response := tst.ParseResponse(rr)
		data := response["data"].(map[string]any)
		confirmationToken = data["token"].(string)
		assert.NotEmpty(t, confirmationToken)
	})

	t.Run("Confirm Role Change - Success", func(t *testing.T) {
		payload := models.ConfirmRoleChangeRequest{
			ConfirmationToken: confirmationToken,
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/backoffice/admins/role/confirm", bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+superAdminToken)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		// Verify Database Change
		var updatedAdmin models.Admin
		db.Postgresql.First(&updatedAdmin, "id = ?", targetAdminID)
		assert.Equal(t, models.RoleSuperAdmin, updatedAdmin.Role)
	})

	t.Run("Verify Audit Logs", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/backoffice/admins/audit-logs?resource_id=%s", targetAdminID), nil)
		req.Header.Set("Authorization", "Bearer "+superAdminToken)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		response := tst.ParseResponse(rr)
		data := response["data"].(map[string]any)
		logs := data["logs"].([]any)

		assert.NotEmpty(t, logs)
		firstLog := logs[0].(map[string]any)
		assert.Equal(t, string(models.ActionAdminUpdate), firstLog["action"])
	})
}

func TestRoleChangeSecurity_Unauthorized(t *testing.T) {
	r, _, _, db := SetupAdminTestRouter()

	// Create a regular admin (NOT a superadmin)
	regularAdminToken := CreateAdminAndGetToken(t, r, db, models.RoleAdmin)
	targetAdminID := utility.GenerateUUID()

	t.Run("Regular Admin Blocked From Initiating", func(t *testing.T) {
		isConfirmed := true
		payload := models.ChangeAdminRoleRequest{
			NewRole:     models.RoleSuperAdmin,
			IsConfirmed: &isConfirmed,
		}

		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/backoffice/admins/%s/role/initiate", targetAdminID), bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+regularAdminToken)

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusForbidden, rr.Code)
	})
}
