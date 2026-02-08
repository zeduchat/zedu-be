package test_credits

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/credits"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

type UserOrganisation struct {
	UserID         string `gorm:"type:uuid;not null" json:"user_id"`
	OrganisationID string `gorm:"type:uuid;not null" json:"organisation_id"`
}

func SetupCreditTestRouter() (*gin.Engine, *auth.Controller, *credits.Controller, *utility.Logger, *storage.Database) {
	gin.SetMode(gin.TestMode)
	logger := utility.NewLogger()
	db := storage.Connection()
	validator := validator.New()

	authController := &auth.Controller{
		Db:        db,
		Validator: validator,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	creditController := &credits.Controller{
		Db:        db,
		Validator: validator,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	r := gin.Default()
	SetupAuthRoutes(r, authController)
	SetupCreditRoutes(r, creditController)
	return r, authController, creditController, logger, db
}

func SetupAuthRoutes(r *gin.Engine, authController *auth.Controller) {
	apiVersion := "/api/v1"
	authUrl := r.Group(fmt.Sprintf("%s/auth", apiVersion))
	{
		authUrl.POST("/register", authController.RegisterUser)
		authUrl.POST("/login", authController.LoginUser)
	}
}

func SetupCreditRoutes(r *gin.Engine, creditController *credits.Controller) {
	apiVersion := "/api/v1"

	creditUrl := r.Group(fmt.Sprintf("%s/credits", apiVersion),
		middleware.Authorize(creditController.Db.Postgresql))
	{
		creditUrl.POST("/purchase", creditController.PurchaseCredits)
		creditUrl.GET("/packages", creditController.GetCreditPackages)
		creditUrl.GET("/usage", creditController.GetOrgCreditUsage)
		creditUrl.GET("/usage-report/:org_id", creditController.GetOrgCreditReport)
		creditUrl.GET("/transactions/:org_id", creditController.GetOrgCreditTransactions)
	}
}

func CreateUserWithOrganization(t *testing.T, r *gin.Engine, authCtl *auth.Controller, db *storage.Database) (string, string, string, string) {
	currUUID := utility.GenerateUUID()
	user := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("credituser%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "Credit",
		LastName:    "TestUser",
		Password:    "password",
		UserName:    fmt.Sprintf("credituser%v", currUUID),
	}

	var b bytes.Buffer
	json.NewEncoder(&b).Encode(user)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", &b)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated && rr.Code != http.StatusOK {
		t.Fatalf("Registration failed with status %d. Response: %s", rr.Code, rr.Body.String())
	}

	loginData := models.LoginRequestModel{
		Email:    user.Email,
		Password: user.Password,
	}

	b.Reset()
	json.NewEncoder(&b).Encode(loginData)
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/auth/login", &b)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Login failed with status %d. Response: %s", rr.Code, rr.Body.String())
	}

	response := tst.ParseResponse(rr)
	dataM := response["data"].(map[string]any)
	token := dataM["access_token"].(string)
	userID := tst.GetUserIDFromToken(t, token, db)

	orgID := utility.GenerateUUID()
	org := models.Organisation{
		ID:            orgID,
		Name:          fmt.Sprintf("Test Credit Org %s", utility.RandomString(5)),
		Email:         fmt.Sprintf("creditorg%s@qa.team", utility.RandomString(5)),
		CreditBalance: 100.00,
		OwnerID:       userID,
		Type:          "test",
		Country:       "NG",
	}

	err := db.Postgresql.Create(&org).Error
	if err != nil {
		t.Fatalf("Failed to create organization: %v", err)
	}

	userOrg := UserOrganisation{
		UserID:         userID,
		OrganisationID: orgID,
	}
	err = db.Postgresql.Create(&userOrg).Error
	if err != nil {
		t.Fatalf("Failed to create user organization: %v", err)
	}

	orgUUID, err := uuid.FromString(orgID)
	if err != nil {
		t.Fatalf("Failed to parse org ID: %v", err)
	}
	err = db.Postgresql.Model(&models.User{}).Where("id = ?", userID).Update("current_org", orgUUID).Error
	if err != nil {
		t.Fatalf("Failed to update user current org: %v", err)
	}

	b.Reset()
	json.NewEncoder(&b).Encode(loginData)
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/auth/login", &b)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Second login failed with status %d. Response: %s", rr.Code, rr.Body.String())
	}

	response = tst.ParseResponse(rr)
	dataM = response["data"].(map[string]any)
	token = dataM["access_token"].(string)

	return userID, orgID, token, user.Email
}

func GetSeededCreditPackage(t *testing.T, db *gorm.DB, packageName string) *models.CreditPackage {
	var pkg models.CreditPackage
	err := db.Where("name = ?", packageName).First(&pkg).Error
	if err != nil {
		t.Fatalf("Failed to get seeded credit package '%s': %v", packageName, err)
	}
	return &pkg
}

func CleanupCreditTestData(db *gorm.DB, userID string, orgID string) {
	if orgID != "" {
		db.Exec("DELETE FROM user_organisations WHERE organisation_id = ?", orgID)
		db.Exec("DELETE FROM credit_transactions WHERE organisation_id = ?", orgID)
		db.Exec("DELETE FROM credit_usages WHERE organisation_id = ?", orgID)
		db.Exec("DELETE FROM organisations WHERE id = ?", orgID)
	}

	if userID != "" {
		db.Exec("DELETE FROM user_channels WHERE user_id = ?", userID)
		db.Exec("DELETE FROM user_organisations WHERE user_id = ?", userID)
		db.Exec("DELETE FROM access_tokens WHERE owner_id = ?", userID)
		db.Exec("DELETE FROM profiles WHERE userid = ?", userID)
		db.Exec("DELETE FROM users WHERE id = ?", userID)
	}
}
