package test_credits

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/hngprojects/telex_be/internal/models"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestGetOrgCreditUsage_Success(t *testing.T) {
	_ = tst.Setup()
	gin.SetMode(gin.TestMode)

	r, authCtl, _, _, db := SetupCreditTestRouter()

	userID, orgID, token, _ := CreateUserWithOrganization(t, r, authCtl, db)

	t.Cleanup(func() {
		CleanupCreditTestData(db.Postgresql, userID, orgID)
	})

	agentID := utility.GenerateUUID()
	creditUsage := models.CreditUsage{
		ID:             utility.GenerateUUID(),
		OrganisationID: orgID,
		Amount:         10.50,
		AgentID:        agentID,
		UserID:         &userID,
	}
	err := db.Postgresql.Create(&creditUsage).Error
	assert.NoError(t, err, "Failed to create test credit usage")

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/credits/usage", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Organisation-Id", orgID)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Logf("Response body: %s", rr.Body.String())
	}

	assert.Equal(t, http.StatusOK, rr.Code)

	response := tst.ParseResponse(rr)
	assert.Equal(t, "success", response["status"])
	assert.NotNil(t, response["data"])

	data, ok := response["data"].([]interface{})
	assert.True(t, ok, "data should be an array")
	assert.GreaterOrEqual(t, len(data), 1, "should have at least one credit usage record")
}

func TestGetOrgCreditUsage_Unauthorized(t *testing.T) {
	_ = tst.Setup()
	gin.SetMode(gin.TestMode)

	r, _, _, _, _ := SetupCreditTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/credits/usage", nil)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)

	response := tst.ParseResponse(rr)
	assert.NotEqual(t, "success", response["status"])
}

func TestGetOrgCreditUsage_WithPagination(t *testing.T) {
	_ = tst.Setup()
	gin.SetMode(gin.TestMode)

	r, authCtl, _, _, db := SetupCreditTestRouter()

	userID, orgID, token, _ := CreateUserWithOrganization(t, r, authCtl, db)

	t.Cleanup(func() {
		CleanupCreditTestData(db.Postgresql, userID, orgID)
	})

	agentID := utility.GenerateUUID()
	for i := 0; i < 5; i++ {
		creditUsage := models.CreditUsage{
			ID:             utility.GenerateUUID(),
			OrganisationID: orgID,
			Amount:         float64(i) + 1.0,
			AgentID:        agentID,
			UserID:         &userID,
		}
		err := db.Postgresql.Create(&creditUsage).Error
		assert.NoError(t, err, "Failed to create test credit usage")
	}

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/credits/usage?page=1&limit=3", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Organisation-Id", orgID)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	response := tst.ParseResponse(rr)
	assert.Equal(t, "success", response["status"])

	data, ok := response["data"].([]interface{})
	assert.True(t, ok)
	assert.LessOrEqual(t, len(data), 3, "should respect pagination limit")

	pagination, ok := response["pagination"].(map[string]interface{})
	assert.True(t, ok, "should have pagination metadata")
	assert.NotNil(t, pagination["current_page"])
	assert.NotNil(t, pagination["total_pages"])
	assert.NotNil(t, pagination["page_size"])
	assert.NotNil(t, pagination["total_items"])
}

func TestGetOrgCreditUsage_EmptyResults(t *testing.T) {
	_ = tst.Setup()
	gin.SetMode(gin.TestMode)

	r, authCtl, _, _, db := SetupCreditTestRouter()

	userID, orgID, token, _ := CreateUserWithOrganization(t, r, authCtl, db)

	t.Cleanup(func() {
		CleanupCreditTestData(db.Postgresql, userID, orgID)
	})

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/credits/usage", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Organisation-Id", orgID)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	response := tst.ParseResponse(rr)
	assert.Equal(t, "success", response["status"])

	data, ok := response["data"].([]interface{})
	assert.True(t, ok)
	assert.Equal(t, 0, len(data), "should return empty array for org with no credit usage")
}

func TestGetOrgCreditUsage_MultipleAgents(t *testing.T) {
	_ = tst.Setup()
	gin.SetMode(gin.TestMode)

	r, authCtl, _, _, db := SetupCreditTestRouter()

	userID, orgID, token, _ := CreateUserWithOrganization(t, r, authCtl, db)

	t.Cleanup(func() {
		CleanupCreditTestData(db.Postgresql, userID, orgID)
	})

	agent1ID := utility.GenerateUUID()
	agent2ID := utility.GenerateUUID()

	creditUsage1 := models.CreditUsage{
		ID:             utility.GenerateUUID(),
		OrganisationID: orgID,
		Amount:         15.00,
		AgentID:        agent1ID,
		UserID:         &userID,
	}
	err := db.Postgresql.Create(&creditUsage1).Error
	assert.NoError(t, err)

	creditUsage2 := models.CreditUsage{
		ID:             utility.GenerateUUID(),
		OrganisationID: orgID,
		Amount:         25.00,
		AgentID:        agent2ID,
		UserID:         &userID,
	}
	err = db.Postgresql.Create(&creditUsage2).Error
	assert.NoError(t, err)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/credits/usage", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Organisation-Id", orgID)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	response := tst.ParseResponse(rr)
	assert.Equal(t, "success", response["status"])

	data, ok := response["data"].([]interface{})
	assert.True(t, ok)
	assert.GreaterOrEqual(t, len(data), 2, "should return credit usage for multiple agents")
}
