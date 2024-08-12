package test_blog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestCreateBlog(t *testing.T) {
	_, blogController := SetupBlogTestRouter()
	db := blogController.Db.Postgresql
	currUUID := utility.GenerateUUID()
	password, _ := utility.HashPassword("password")

	regularUser := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "Regular User",
		Email:    fmt.Sprintf("user%v@qa.team", currUUID),
		Password: password,
	}

	blogCategory := models.BlogCategory{
		ID:   utility.GenerateUUID(),
		Name: "testCategory" + utility.GenerateUUID(),
	}

	db.Create(&regularUser)
	db.Create(&blogCategory)

	setup := func() (*gin.Engine, *auth.Controller) {
		router, blogController := SetupBlogTestRouter()
		authController := auth.Controller{
			Db:        blogController.Db,
			Validator: blogController.Validator,
			Logger:    blogController.Logger,
			ExtReq:    blogController.ExtReq,
		}

		return router, &authController
	}

	router, authController := setup()

	loginData := models.LoginRequestModel{
		Email:    regularUser.Email,
		Password: "password",
	}

	token := tst.GetLoginToken(t, router, *authController, loginData)

	fmt.Print("I AM THE TOKEN HEREEE", token)

	tests := []struct {
		Name         string
		RequestBody  models.BlogCreateReq
		ExpectedCode int
		Message      string
		Headers      map[string]string
	}{
		{
			Name: "Successful creation of blog",
			RequestBody: models.BlogCreateReq{
				Title:      "sometitle",
				Content:    "testcontent",
				CategoryID: blogCategory.ID,
			},
			ExpectedCode: http.StatusCreated,
			Message:      "blog created successfully",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
		},
		{
			Name: "Unauthorized Access",
			RequestBody: models.BlogCreateReq{
				Title:      "sometitle",
				Content:    "testcontent",
				CategoryID: blogCategory.ID,
			},
			ExpectedCode: http.StatusUnauthorized,
			Message:      "Token could not be found!",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			Name: "Invalid Category Id",
			RequestBody: models.BlogCreateReq{
				Title:      "sometitle",
				Content:    "testcontent",
				CategoryID: "invalid-id-uututu",
			},
			ExpectedCode: http.StatusBadRequest,
			Message:      "invalid blog id format",
			Headers: map[string]string{
				"Content-Type": "application/json",
				"Authorization": "Bearer " + token,
			},
		},
		{
			Name: "Blog category Not Found",
			RequestBody: models.BlogCreateReq{
				Title:      "sometitle",
				Content:    "testcontent",
				CategoryID: utility.GenerateUUID(),
			},
			ExpectedCode: http.StatusNotFound,
			Message:      "blog category not found",
			Headers: map[string]string{
				"Content-Type": "application/json",
				"Authorization": "Bearer " + token,
			},
		},
		{
			Name: "Validation failed",
			RequestBody: models.BlogCreateReq{
				Content: "testcontent",
			},
			ExpectedCode: http.StatusUnprocessableEntity,
			Message:      "Validation failed",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var b bytes.Buffer
			json.NewEncoder(&b).Encode(test.RequestBody)

			req, _ := http.NewRequest(http.MethodPost, "/api/v1/blogs", &b)

			for i, v := range test.Headers {
				req.Header.Set(i, v)
			}

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			tst.AssertStatusCode(t, resp.Code, test.ExpectedCode)
			response := tst.ParseResponse(resp)
			tst.AssertResponseMessage(t, response["message"].(string), test.Message)

		})
	}
}
