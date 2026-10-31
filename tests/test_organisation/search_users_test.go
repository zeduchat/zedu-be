package test_organisation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestSearchUsersInOrganisationReturnsDeterministicUsername(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()
	userController := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{
		Logger: logger,
		Test:   true,
	}}
	orgController := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}
	r := gin.Default()

	orgID, token := initialise(currUUID, t, r, db, userController, orgController, true)
	require.NotEmpty(t, orgID)

	var user models.User
	require.NoError(t, db.Postgresql.Where("email = ?", fmt.Sprintf("testuser%v@qa.team", currUUID)).First(&user).Error)
	require.NoError(t, db.Postgresql.Model(&models.User{}).Where("id = ?", user.ID).Update("name", "divine john").Error)

	require.NoError(t, db.Postgresql.Unscoped().Where("userid = ?", user.ID).Delete(&models.Profile{}).Error)

	blank := models.Profile{
		ID:        utility.GenerateUUID(),
		Userid:    user.ID,
		FirstName: "divine",
		LastName:  "john",
	}
	require.NoError(t, db.Postgresql.Create(&blank).Error)

	handle := "dicint"
	populated := models.Profile{
		ID:             utility.GenerateUUID(),
		Userid:         user.ID,
		OrganisationID: &orgID,
		FirstName:      "divine",
		LastName:       "john",
		UserName:       handle,
	}
	require.NoError(t, db.Postgresql.Create(&populated).Error)

	r.Group("/api/v1", middleware.Authorize(db.Postgresql)).
		GET("/organisations/:org_id/users/search", orgController.SearchUsersInOrganisation)

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organisations/%s/users/search?query=divine", orgID), nil)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	var resp struct {
		Data []models.UserInOrgResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Len(t, resp.Data, 1)
	assert.Equal(t, handle, resp.Data[0].UserName)
}
