package test_subscription

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/internal/models/seed"
	"github.com/hngprojects/telex_be/pkg/middleware"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestDeleteSubscription_OnlyOwnerOrBillingManager(t *testing.T) {
	logger := tst.Setup()
	r, authController, subController, _, db := SetupSubscriptionTestRouter()
	r.Group("/api/v1/subscriptions", middleware.Authorize(db.Postgresql)).
		DELETE("/:org_id", subController.DeleteSubscription)

	seed.SeedPlans(logger, db.Postgresql)
	seed.SeedRolesAndPermissions(logger, db.Postgresql)
	plan := GetSeededPlan(t, db.Postgresql, "Free")

	ownerID, orgID, ownerToken, _ := CreateUserWithOrganization(t, r, authController, db)
	outsiderID, outsiderOrgID, outsiderToken, _ := CreateUserWithOrganization(t, r, authController, db)
	memberID, memberOrgID, memberToken, _ := CreateUserWithOrganization(t, r, authController, db)

	orgPlanID := utility.GenerateUUID()
	defer func() {
		db.Postgresql.Where("user_id = ? AND organisation_id = ?", memberID, orgID).Delete(&models.OrgUserManagement{})
		CleanupSubscriptionTestData(db.Postgresql, ownerID, orgID)
		CleanupSubscriptionTestData(db.Postgresql, outsiderID, outsiderOrgID)
		CleanupSubscriptionTestData(db.Postgresql, memberID, memberOrgID)
	}()

	if err := db.Postgresql.Create(&models.OrganisationPlan{
		ID:             orgPlanID,
		OrganisationID: orgID,
		PlanID:         plan.ID,
		StartedAt:      time.Now(),
		Status:         "Active",
	}).Error; err != nil {
		t.Fatalf("create org plan: %v", err)
	}
	if err := db.Postgresql.Model(&models.Organisation{}).Where("id = ?", orgID).Update("org_plan_id", orgPlanID).Error; err != nil {
		t.Fatalf("set org plan: %v", err)
	}

	var userRole models.OrgRole
	userRole, err := userRole.GetAOrgRoleByName(db.Postgresql, models.OrgRoleNameUser)
	if err != nil {
		t.Fatalf("load User role: %v", err)
	}
	if err := db.Postgresql.Create(&models.OrgUserManagement{
		UserID:         memberID,
		OrganisationID: orgID,
		RoleID:         userRole.ID,
		Status:         "active",
	}).Error; err != nil {
		t.Fatalf("add member: %v", err)
	}

	del := func(token string) *httptest.ResponseRecorder {
		req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/subscriptions/%s", orgID), nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr
	}
	planStillSet := func() bool {
		var org models.Organisation
		db.Postgresql.Select("org_plan_id").Where("id = ?", orgID).First(&org)
		return org.OrgPlanID == orgPlanID
	}

	tst.AssertStatusCode(t, del(outsiderToken).Code, http.StatusForbidden)
	if !planStillSet() {
		t.Fatal("outsider cancelled the plan")
	}

	tst.AssertStatusCode(t, del(memberToken).Code, http.StatusForbidden)
	if !planStillSet() {
		t.Fatal("plain member cancelled the plan")
	}

	tst.AssertStatusCode(t, del(ownerToken).Code, http.StatusOK)
	if planStillSet() {
		t.Fatal("owner could not cancel the plan")
	}
}
