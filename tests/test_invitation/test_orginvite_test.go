package test_invitation

// package test_invitation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	"github.com/hngprojects/telex_be/pkg/controller/invitation"
	"github.com/hngprojects/telex_be/pkg/controller/organisation"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestOrganisationInvitation(t *testing.T) {
	logger := tst.Setup()
	gin.SetMode(gin.TestMode)
	validatorRef := validator.New()
	db := storage.Connection()
	currUUID := utility.GenerateUUID()

	userSignUpData := models.CreateUserRequestModel{
		Email:       fmt.Sprintf("testuser%v@qa.team", currUUID),
		PhoneNumber: fmt.Sprintf("+234%v", utility.GetRandomNumbersInRange(7000000000, 9099999999)),
		FirstName:   "test",
		LastName:    "user",
		Password:    "password",
		UserName:    fmt.Sprintf("test_username%v", currUUID),
	}
	loginData := models.LoginRequestModel{
		Email:    userSignUpData.Email,
		Password: userSignUpData.Password,
	}
	auth := auth.Controller{Db: db, Validator: validatorRef,
		Logger: logger, ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		}}
	r := gin.Default()
	tst.SignupUser(t, r, auth, userSignUpData, false)

	org := organisation.Controller{Db: db, Validator: validatorRef, Logger: logger}

	token := tst.GetLoginToken(t, r, auth, loginData)
	createOrgData := models.CreateOrgRequestModel{
		Name:        fmt.Sprintf("testorg%v", currUUID),
		Description: "Some Random description",
		Email:       fmt.Sprintf("testuser%v@qa.team", currUUID),
		Type:        "type1",
		Location:    "wakanda",
		Country:     "wakanda",
	}
	orgId, _, _ := tst.CreateOrganisation(t, r, db, org, createOrgData, token)

	role := models.OrgRole{
		ID:          utility.GenerateUUID(),
		Name:        "Admin",
		Description: "Admin",
		OrganisationID: &orgId,
	}

	db.Postgresql.Create(&role)

	createInviteData := models.InvitationCreateReq{
		Emails:         []string{fmt.Sprintf("test%s@example.com", currUUID)},
		OrganisationID: orgId,
		Role:           role.ID,
	}
	invitation := invitation.Controller{Db: db, Validator: validatorRef, Logger: logger}

	invite_token := tst.CreateInvitation(t, r, db, invitation, createInviteData, token)



	tests := []struct {
		Name         string
		RequestBody  interface{}
		ExpectedCode int
		Message      string
		Method       string
		Headers      map[string]string
		RequestURI   url.URL
	}{
		{
			Name: "Organisation Invite Creation Action",
			RequestBody: models.InvitationCreateReq{
				Emails:         []string{fmt.Sprintf("test%s@example.com", currUUID)},
				OrganisationID: orgId,
				Role:           "01915c5c-6417-7620-a80f-b8dde5509881",
			},
			ExpectedCode: http.StatusCreated,
			Message:      "Invitations created successfully",
			Method:       http.MethodPost,
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
				"Content-Type":  "application/json",
			},
			RequestURI: url.URL{Path: "/api/v1/invite"},
		}, {
			Name: "Organization Accept Invite Action",
			RequestBody: models.VerifyInvitationLinkRequest{
				Token: invite_token,
			},
			RequestURI:   url.URL{Path: "/api/v1/invite/verify"},
			ExpectedCode: http.StatusOK,
			Message:      "User invited successfully",
			Method:       http.MethodPost,
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
				"Content-Type":  "application/json",
			},
		}, {
			Name: "Organization Resend Invite Action",
			RequestBody: models.ResendInvitationRequest{
				Emails:         []string{fmt.Sprintf("test%s@example.com", currUUID)},
				OrganisationID: orgId,
			},
			RequestURI:   url.URL{Path: "/api/v1/invite/resend"},
			ExpectedCode: http.StatusOK,
			Message:      "success",
			Method:       http.MethodPost,
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
				"Content-Type":  "application/json",
			},
		},
	}

	for _, test := range tests {
		r := gin.Default()

		invitationURL := r.Group("/api/v1/invite")
		{
			invitationURL.POST("", middleware.Authorize(db.Postgresql), invitation.OrganisationCreateInvite)
			invitationURL.POST("/verify", invitation.OrganisationVerifyInvite)
			invitationURL.POST("/resend", middleware.Authorize(db.Postgresql), invitation.ResendInvitation)

		}

		t.Run(test.Name, func(t *testing.T) {
			var b bytes.Buffer
			json.NewEncoder(&b).Encode(test.RequestBody)

			req, err := http.NewRequest(test.Method, test.RequestURI.String(), &b)
			if err != nil {
				t.Fatal(err)
			}

			for i, v := range test.Headers {
				req.Header.Set(i, v)
			}

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			tst.AssertStatusCode(t, rr.Code, test.ExpectedCode)

			data := tst.ParseResponse(rr)

			code := int(data["status_code"].(float64))
			tst.AssertStatusCode(t, code, test.ExpectedCode)

			if test.Message != "" {
				message := data["message"]
				if message != nil {
					tst.AssertResponseMessage(t, message.(string), test.Message)
				} else {
					tst.AssertResponseMessage(t, "", test.Message)
				}

			}
		})
	}
}
