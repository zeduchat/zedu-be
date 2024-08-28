package test_blog

import (
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

func TestDeleteBlogs(t *testing.T) {
	_, blogController := SetupBlogTestRouter()
	db := blogController.Db.Postgresql
	currUUID := utility.GenerateUUID()
	password, _ := utility.HashPassword("password")

	regularUser := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "Regular User1",
		Email:    fmt.Sprintf("user%v@qa.team", currUUID),
		Password: password,
	}

	blogCategory := models.BlogCategory{
		ID:   utility.GenerateUUID(),
		Name: "testCategory" + utility.GenerateUUID(),
	}

	blog := models.Blog{
		ID:         utility.GenerateUUID(),
		Title:      "sometitle",
		Content:    "something soemthing",
		CategoryID: blogCategory.ID,
		AuthorID:   regularUser.ID,
		AuthorName: regularUser.Name,
	}

	db.Create(&regularUser)
	db.Create(&blogCategory)
	db.Create(&blog)

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

	token1 := tst.GetLoginToken(t, router, *authController, loginData)

	tests := []struct {
		Name         string
		BlogID       string
		ExpectedCode int
		Message      string
		Headers      map[string]string
	}{
		{
			Name:         "Successful Deletion of Blog",
			BlogID:       blog.ID,
			ExpectedCode: http.StatusNoContent,
			Message:      "blog successfully deleted",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token1,
			},
		},
		{
			Name:         "Invalid Blog ID Format",
			BlogID:       "invalid-id-erttt",
			ExpectedCode: http.StatusBadRequest,
			Message:      "invalid blog id format",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token1,
			},
		},
		{
			Name:         "Blog Not Found",
			BlogID:       utility.GenerateUUID(),
			ExpectedCode: http.StatusNotFound,
			Message:      "blog not found",
			Headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer " + token1,
			},
		},
		{
			Name:         "User Not Authorized to Delete blog",
			BlogID:       blog.ID,
			ExpectedCode: http.StatusUnauthorized,
			Message:      "Token could not be found!",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/blogs/%s", test.BlogID), nil)

			for i, v := range test.Headers {
				req.Header.Set(i, v)
			}

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code == http.StatusNoContent {
				return
			}

			tst.AssertStatusCode(t, resp.Code, test.ExpectedCode)
			response := tst.ParseResponse(resp)
			tst.AssertResponseMessage(t, response["message"].(string), test.Message)

		})
	}
}
