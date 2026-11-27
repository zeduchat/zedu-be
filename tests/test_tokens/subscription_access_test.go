package test_tokens

import (
	"bytes"
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
	"github.com/hngprojects/telex_be/pkg/controller/channel"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	tkn "github.com/hngprojects/telex_be/pkg/controller/token"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func subSignup(t *testing.T, r *gin.Engine, authCtrl auth.Controller, prefix string) (models.CreateUserRequestModel, string) {
	id := utility.GenerateUUID()
	signup := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("%s_%v@qa.team", prefix, id),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   prefix,
		LastName:    "user",
		Password:    "password",
		UserName:    fmt.Sprintf("%s_%v", prefix, id),
	}
	tst.SignupUser(t, r, authCtrl, signup, false)
	token := tst.GetLoginToken(t, r, authCtrl, models.LoginRequestModel{Email: signup.Email, Password: signup.Password})
	return signup, token
}

func subOrg(t *testing.T, r *gin.Engine, db *storage.Database, orgCtrl organisation.Controller, email, token string) string {
	orgID, _, _ := tst.CreateOrganisation(t, r, db, orgCtrl, models.CreateOrgRequestModel{
		Name:     fmt.Sprintf("SubOrg_%s", utility.GenerateUUID()),
		Email:    email,
		Type:     "test",
		Location: "lagos",
		Country:  "ng",
	}, token)
	require.NotEmpty(t, orgID)
	return orgID
}

// TestSubscriptionTokenAccess checks that a Centrifugo subscription token is
// signed only for channels the caller may read (#2492).
func TestSubscriptionTokenAccess(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)

	validatorRef := validator.New()
	db := storage.Connection()
	authCtrl := auth.Controller{Db: db, Validator: validatorRef, Logger: logger, ExtReq: request.ExternalRequest{Logger: logger, Test: true}}
	orgCtrl := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}
	chanCtrl := channel.Controller{Db: db, Validator: validatorRef, Logger: logger}
	tokenCtrl := tkn.Controller{Db: db, Validator: validatorRef, Logger: logger}

	rA := gin.Default()
	signupA, tokenA := subSignup(t, rA, authCtrl, "suba")
	orgA := subOrg(t, rA, db, orgCtrl, signupA.Email, tokenA)
	chanA, _ := tst.CreateChannels(t, rA, chanCtrl, db, models.CreateChannelsRequest{
		Name:           fmt.Sprintf("subchan-%s", utility.GenerateUUID()[:8]),
		Username:       signupA.UserName,
		OrganisationID: orgA,
		IsPrivate:      true,
	}, tokenA)
	require.NotEmpty(t, chanA)
	var userA models.User
	require.NoError(t, db.Postgresql.Where("email = ?", signupA.Email).First(&userA).Error)

	rB := gin.Default()
	signupB, tokenB := subSignup(t, rB, authCtrl, "subb")
	subOrg(t, rB, db, orgCtrl, signupB.Email, tokenB)

	r := gin.Default()
	r.POST("/api/v1/token/subscription", middleware.Authorize(db.Postgresql), tokenCtrl.GetSubToken)
	r.POST("/api/v1/centrifugo/subscription", middleware.UnauthenticatedUserTracker(logger), tokenCtrl.GetSubTokenUnAuth)

	post := func(path, channelID string, headers map[string]string) *httptest.ResponseRecorder {
		var b bytes.Buffer
		require.NoError(t, json.NewEncoder(&b).Encode(models.ChannelSubTokenReq{Channel: channelID}))
		req, err := http.NewRequest(http.MethodPost, path, &b)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		return rr
	}

	cases := []struct {
		name     string
		channel  string
		token    string
		wantCode int
	}{
		{"channel member subscribes to own channel", chanA, tokenA, http.StatusOK},
		{"other org user subscribes to private channel", chanA, tokenB, http.StatusForbidden},
		{"org member subscribes to org feed", orgA, tokenA, http.StatusOK},
		{"other org user subscribes to org feed", orgA, tokenB, http.StatusForbidden},
		{"other user subscribes to personal channel", orgA + "/" + userA.ID, tokenB, http.StatusBadRequest},
		{"unknown uuid", utility.GenerateUUID(), tokenA, http.StatusForbidden},
		{"unknown buzz code", "ABCDEF012345", tokenA, http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := post("/api/v1/token/subscription", tc.channel, map[string]string{"Authorization": "Bearer " + tc.token})
			require.Equal(t, tc.wantCode, rr.Code, rr.Body.String())
			if tc.wantCode != http.StatusOK {
				assert.NotContains(t, rr.Body.String(), `"token"`)
			}
		})
	}

	// Widget visitor: may subscribe only to the channel on its own widget row.
	visitorIP := fmt.Sprintf("10.%d.%d.%d", utility.GetRandomNumbersInRange(1, 250), utility.GetRandomNumbersInRange(1, 250), utility.GetRandomNumbersInRange(1, 250))
	widget := models.UnauthenticatedUser{
		UserIdentifier:  visitorIP,
		ChannelID:       utility.GenerateUUID(),
		Subtoken:        utility.GenerateUUID(),
		LastAgentChatID: utility.GenerateUUID(),
	}
	require.NoError(t, db.Postgresql.Create(&widget).Error)
	defer db.Postgresql.Where("user_identifier = ?", visitorIP).Delete(&models.UnauthenticatedUser{})
	widgetHeaders := map[string]string{"X-Forwarded-For": visitorIP, "X-Notification-Token": widget.Subtoken}

	t.Run("widget visitor subscribes to its own channel", func(t *testing.T) {
		rr := post("/api/v1/centrifugo/subscription", widget.ChannelID, widgetHeaders)
		require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	})
	t.Run("widget visitor subscribes to a private channel", func(t *testing.T) {
		rr := post("/api/v1/centrifugo/subscription", chanA, widgetHeaders)
		require.Equal(t, http.StatusForbidden, rr.Code, rr.Body.String())
		assert.NotContains(t, rr.Body.String(), `"token"`)
	})
}
