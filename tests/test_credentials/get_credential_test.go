package test_credentials

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/credentials"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

// signupWithOrg registers a user on a fresh engine (the tst helpers register
// routes, so each user needs its own), creates an org and returns a token
// whose current org is that org, the user id and the org id.
func signupWithOrg(t *testing.T, db *storage.Database, logger *utility.Logger) (string, string, string) {
	currUUID := utility.GenerateUUID()
	v := validator.New()
	authCtrl := auth.Controller{Db: db, Validator: v, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	orgCtrl := organisation.Controller{Db: db, Validator: v, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	signup := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testuser%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "test",
		LastName:    "user",
		Password:    "password",
		UserName:    fmt.Sprintf("test_username%v", currUUID),
	}
	r := gin.Default()
	tst.SignupUser(t, r, authCtrl, signup, false)
	token := tst.GetLoginToken(t, r, authCtrl, models.LoginRequestModel{Email: signup.Email, Password: signup.Password})
	orgID, _, _ := tst.CreateOrganisation(t, r, db, orgCtrl, models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("Org %v", currUUID),
		Email:       signup.Email,
		Description: "credential access test",
		Type:        "type1",
		Location:    "wakanda",
		Country:     "wakanda",
	}, token)
	// Creating an org makes it the user's current org; log in again so the
	// token's org_id claim is that org.
	token = tst.GetLoginToken(t, r, authCtrl, models.LoginRequestModel{Email: signup.Email, Password: signup.Password})
	return token, tst.GetUserIDFromToken(t, token, db), orgID
}

func insertCredential(t *testing.T, db *storage.Database, userID, orgID string) string {
	key := []byte(config.GetConfig().Server.EncryptionKey)
	enc, err := models.EncryptJSON(models.JSONBMap{"api_key": "secret-value"}, key)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	cred := models.Credential{
		ID:          utility.GenerateUUID(),
		Name:        "test-cred",
		OrgId:       orgID,
		AgentId:     utility.GenerateUUID(),
		UserId:      userID,
		SkillId:     utility.GenerateUUID(),
		Credentials: enc,
	}
	if err := db.Postgresql.Create(&cred).Error; err != nil {
		t.Fatalf("insert credential: %v", err)
	}
	return cred.ID
}

func TestGetCredentialByIDAccess(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	db := storage.Connection()

	// The test env may not set ENCRYPTION_KEY. Use a 32-byte test key for the
	// duration of the test so the owner's credential can be decrypted.
	cfg := config.GetConfig()
	if len(cfg.Server.EncryptionKey) != 32 {
		prev := cfg.Server.EncryptionKey
		cfg.Server.EncryptionKey = "0123456789abcdef0123456789abcdef"
		t.Cleanup(func() { cfg.Server.EncryptionKey = prev })
	}

	tokenA, userA, orgA := signupWithOrg(t, db, logger)
	tokenB, _, orgB := signupWithOrg(t, db, logger)

	ownCred := insertCredential(t, db, userA, orgA)
	// Same owner, but saved under an org that isn't the token's current org.
	otherOrgCred := insertCredential(t, db, userA, orgB)

	ctrl := credentials.Controller{Db: db, Validator: validator.New(), Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	r := gin.Default()
	r.GET("/api/v1/credentials/:credential_id", middleware.Authorize(db.Postgresql), ctrl.GetCredentialByID)

	cases := []struct {
		name    string
		token   string
		id      string
		want    int
		secrets bool
	}{
		{"owner reads own credential", tokenA, ownCred, http.StatusOK, true},
		{"other user gets 404", tokenB, ownCred, http.StatusNotFound, false},
		{"owner outside the credential's org gets 404", tokenA, otherOrgCred, http.StatusNotFound, false},
		{"unknown id gets 404", tokenA, utility.GenerateUUID(), http.StatusNotFound, false},
		{"malformed id gets 400", tokenA, "not-a-uuid", http.StatusBadRequest, false},
		{"no token gets 401", "", ownCred, http.StatusUnauthorized, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/credentials/"+tc.id, nil)
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			tst.AssertStatusCode(t, rr.Code, tc.want)

			body := tst.ParseResponse(rr)
			data, _ := body["data"].(map[string]any)
			creds, _ := data["credentials"].(map[string]any)
			if tc.secrets {
				if creds["api_key"] != "secret-value" {
					t.Errorf("expected decrypted credentials for the owner, got %v", data)
				}
			} else if creds["api_key"] != nil {
				t.Errorf("secrets leaked in a %d response: %v", rr.Code, data)
			}
		})
	}
}
