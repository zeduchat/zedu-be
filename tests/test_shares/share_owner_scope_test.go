package test_shares

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/shares"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

// TestShareHoldingOwnerScope covers #1425: GET and DELETE /shares/:id act only on the
// caller's own holding; another user's holding behaves like a missing one (404).
func TestShareHoldingOwnerScope(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()

	authController := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	sharesController := shares.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}

	r := gin.Default()
	sharesURL := r.Group("/api/v1/shares", middleware.Authorize(db.Postgresql))
	sharesURL.POST("/", sharesController.Create)
	sharesURL.GET("/:id", sharesController.GetShare)
	sharesURL.DELETE("/:id", sharesController.Delete)

	newUser := func(tag string) string {
		currUUID := utility.GenerateUUID()
		ud := models.CreateUserRequestModel{
			Email:       fmt.Sprintf("holding%s%v@qa.team", tag, currUUID),
			PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
			FirstName:   "holding",
			LastName:    tag,
			Password:    "password123",
			UserName:    fmt.Sprintf("holding%s%v", tag, currUUID),
		}
		tst.SignupUser(t, r, authController, ud, false)
		return tst.GetLoginToken(t, r, authController, models.LoginRequestModel{Email: ud.Email, Password: ud.Password})
	}
	ownerToken, otherToken := newUser("owner"), newUser("other")

	do := func(method, path, token string, body any) *httptest.ResponseRecorder {
		var b bytes.Buffer
		if body != nil {
			json.NewEncoder(&b).Encode(body)
		}
		req, _ := http.NewRequest(method, path, &b)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr
	}

	rr := do(http.MethodPost, "/api/v1/shares/", ownerToken, models.ShareRequest{NumberOfShares: 3, PricePurchased: 2.5})
	tst.AssertStatusCode(t, rr.Code, http.StatusCreated)
	shareID := tst.ParseResponse(rr)["data"].(map[string]any)["id"].(string)
	path := "/api/v1/shares/" + shareID

	tst.AssertStatusCode(t, do(http.MethodGet, path, otherToken, nil).Code, http.StatusNotFound)
	tst.AssertStatusCode(t, do(http.MethodDelete, path, otherToken, nil).Code, http.StatusNotFound)
	tst.AssertStatusCode(t, do(http.MethodGet, path, ownerToken, nil).Code, http.StatusOK)
	tst.AssertStatusCode(t, do(http.MethodGet, "/api/v1/shares/"+utility.GenerateUUID(), ownerToken, nil).Code, http.StatusNotFound)
	tst.AssertStatusCode(t, do(http.MethodGet, "/api/v1/shares/not-a-uuid", ownerToken, nil).Code, http.StatusBadRequest)
	tst.AssertStatusCode(t, do(http.MethodDelete, path, ownerToken, nil).Code, http.StatusOK)
	tst.AssertStatusCode(t, do(http.MethodGet, path, ownerToken, nil).Code, http.StatusNotFound)

	db.Postgresql.Unscoped().Where("id = ?", shareID).Delete(&models.Shares{})
}
