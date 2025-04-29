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

func TestFeedbackCount(t *testing.T) {
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

	blog := models.Blog{
		ID:         utility.GenerateUUID(),
		Title:      "sometitle",
		Content:    "something soemthing",
		CategoryID: blogCategory.ID,
		AuthorID:   regularUser.ID,
	}

	feedback1 := models.BlogFeedback{
		ID:       utility.GenerateUUID(),
		BlogID:   blog.ID,
		Feedback: true,
	}

	feedback2 := models.BlogFeedback{
		ID:       utility.GenerateUUID(),
		BlogID:   blog.ID,
		Feedback: false,
	}

	db.Create(&regularUser)
	db.Create(&blogCategory)
	db.Create(&blog)
	db.Create(&feedback1)
	db.Create(&feedback2)

	setup := func() *gin.Engine {
		router, _ := SetupBlogTestRouter()
		return router
	}

	router := setup()

	tests := []struct {
		Name         string
		BlogId       string
		ExpectedCode int
		Message      string
		Headers      map[string]string
	}{
		{
			Name:         "Successful Retrieval of Feedback Count",
			BlogId:       blog.ID,
			ExpectedCode: http.StatusOK,
			Message:      "blog feedback counts retrieved successfully",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			Name:         "Invalid blog id format",
			BlogId:       "invalid-blog-id",
			ExpectedCode: http.StatusBadRequest,
			Message:      "invalid blog id format",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {

			req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/blogs/%s/feedback/count", test.BlogId), nil)

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
