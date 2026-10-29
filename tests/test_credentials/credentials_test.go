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

func signUpAndLogin(t *testing.T, r *gin.Engine, authCtrl auth.Controller, prefix string) (string, string) {
	id := utility.GenerateUUID()
	signUp := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("%s%v@qa.team", prefix, id),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   prefix,
		LastName:    "user",
		Password:    "password123",
		UserName:    fmt.Sprintf("%s_%v", prefix, id),
	}
	tst.SignupUser(t, r, authCtrl, signUp, false)
	token := tst.GetLoginToken(t, r, authCtrl, models.LoginRequestModel{Email: signUp.Email, Password: signUp.Password})
	return token, signUp.Email
}

func TestGetCredentialByIDOnlyReturnsTheOwnersCredential(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	authCtrl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}
	credCtrl := credentials.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()

	// Owner: a user with an organisation and one stored credential.
	ownerToken, ownerEmail := signUpAndLogin(t, r, authCtrl, "credowner")
	orgID, _, ownerID := tst.CreateOrganisation(t, r, db, orgCtrl, models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("CredOrg%s", utility.GenerateUUID()),
		Description: "Org for credential ownership tests",
		Email:       ownerEmail,
		Type:        "type1",
		Location:    "wakanda",
		Country:     "wakanda",
	}, ownerToken)
	if orgID == "" || ownerID == "" {
		t.Fatal("organisation setup failed")
	}

	// Reading a credential back needs a 32-byte ENCRYPTION_KEY. Environments
	// without one still run the access checks below, which never decrypt.
	key := []byte(config.GetConfig().Server.EncryptionKey)
	hasKey := len(key) == 32
	encrypted := []byte("not-decryptable")
	if hasKey {
		var err error
		encrypted, err = models.EncryptJSON(models.JSONBMap{"api_key": "owner-secret"}, key)
		if err != nil {
			t.Fatalf("encrypting credential: %v", err)
		}
	}
	credential := models.Credential{
		ID:          utility.GenerateUUID(),
		Name:        "test credential",
		OrgId:       orgID,
		UserId:      ownerID,
		AgentId:     utility.GenerateUUID(),
		SkillId:     utility.GenerateUUID(),
		Credentials: encrypted,
	}
	if err := db.Postgresql.Create(&credential).Error; err != nil {
		t.Fatalf("creating credential: %v", err)
	}
	t.Cleanup(func() { db.Postgresql.Delete(&models.Credential{}, "id = ?", credential.ID) })

	// Another signed-in user with no claim to that credential.
	otherToken, _ := signUpAndLogin(t, r, authCtrl, "credother")

	credGroup := r.Group("/api/v1/credentials", middleware.Authorize(db.Postgresql))
	credGroup.GET("/:credential_id", credCtrl.GetCredentialByID)

	get := func(id, token string) *httptest.ResponseRecorder {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/credentials/"+id, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr
	}

	t.Run("owner can read their credential", func(t *testing.T) {
		if !hasKey {
			t.Skip("ENCRYPTION_KEY is not a 32-byte key here, so the credential can't be decrypted")
		}
		rr := get(credential.ID, ownerToken)
		if rr.Code != http.StatusOK {
			t.Fatalf("got status %d, want %d. Response: %s", rr.Code, http.StatusOK, rr.Body.String())
		}
		data, _ := tst.ParseResponse(rr)["data"].(map[string]any)
		creds, _ := data["credentials"].(map[string]any)
		if creds["api_key"] != "owner-secret" {
			t.Fatalf("got credentials %v, want the owner's secret", data["credentials"])
		}
	})

	t.Run("another user gets 404 and no secrets", func(t *testing.T) {
		rr := get(credential.ID, otherToken)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("got status %d, want %d. Response: %s", rr.Code, http.StatusNotFound, rr.Body.String())
		}
		data, _ := tst.ParseResponse(rr)["data"].(map[string]any)
		if data != nil && data["credentials"] != nil {
			t.Fatalf("response leaked credentials: %s", rr.Body.String())
		}
	})

	t.Run("unknown id gets 404", func(t *testing.T) {
		rr := get(utility.GenerateUUID(), ownerToken)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("got status %d, want %d. Response: %s", rr.Code, http.StatusNotFound, rr.Body.String())
		}
	})
}
