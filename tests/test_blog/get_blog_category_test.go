package test_blog

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestGetBlogCategories(t *testing.T) {
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

	setup := func() *gin.Engine {
		router, _ := SetupBlogTestRouter()
		return router
	}

	router := setup()

	tests := []struct {
		Name         string
		ExpectedCode int
		Message      string
		Headers      map[string]string
	}{
		{
			Name:         "Successful Retrieval of Blog Categories",
			ExpectedCode: http.StatusOK,
			Message:      "blog categories retrieved successfully",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/blog_categories", nil)

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

func TestGetBlogCategoryById(t *testing.T) {
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

	setup := func() *gin.Engine {
		router, _ := SetupBlogTestRouter()
		return router
	}

	router := setup()

	tests := []struct {
		Name         string
		ID           string
		ExpectedCode int
		Message      string
		Headers      map[string]string
	}{
		{
			Name:         "Successful retrieval of blog category",
			ID:           blogCategory.ID,
			ExpectedCode: http.StatusOK,
			Message:      "blog category retrieved successfully",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			Name:         "Invalid Blog Category Id Format",
			ID:           "invalid-category-id",
			ExpectedCode: http.StatusBadRequest,
			Message:      "invalid category id format",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			Name:         "Non Existent Blog Category",
			ID:           utility.GenerateUUID(),
			ExpectedCode: http.StatusNotFound,
			Message:      "blog category not found",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/blog_categories/%s", test.ID), nil)

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
