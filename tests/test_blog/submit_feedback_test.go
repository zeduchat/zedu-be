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
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

func TestSubmitFeedback(t *testing.T) {
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

	db.Create(&regularUser)
	db.Create(&blogCategory)
	db.Create(&blog)

	setup := func() *gin.Engine {
		router, _ := SetupBlogTestRouter()
		return router
	}

	router := setup()

	tests := []struct {
		Name         string
		RequestBody  models.BlogFeedbackReq
		ExpectedCode int
		Message      string
		Headers      map[string]string
	}{
		{
			Name: "Successful feedback submission",
			RequestBody: models.BlogFeedbackReq{
				BlogID:   blog.ID,
				Feedback: true,
			},
			ExpectedCode: http.StatusOK,
			Message:      "blog feedback submitted successfully",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			Name: "Invalid blog id format",
			RequestBody: models.BlogFeedbackReq{
				BlogID:   "someting-soething",
				Feedback: true,
			},
			ExpectedCode: http.StatusBadRequest,
			Message:      "invalid blog id format",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var b bytes.Buffer
			json.NewEncoder(&b).Encode(test.RequestBody)

			req, _ := http.NewRequest(http.MethodPost, "/api/v1/blogs/feedback", &b)

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
