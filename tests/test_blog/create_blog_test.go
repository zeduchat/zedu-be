package test_blog

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
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
	content := `---
title: "How Telex is Changing the Game for Real-Time Notifications in Applications"
publishedAt: "2024-08-28"
summary: "Telex is setting a new standard for real-time notifications with its advanced features and innovative approach. Explore how this tool is transforming the way applications handle real-time events."
image: "/images/video/banner.png"
previewImage: "/images/about-us.jpg"
---

[Telex](http://telexapp.com/) is redefining the landscape of real-time notifications with its state-of-the-art technology. As businesses increasingly rely on real-time data to drive their operations, Telex provides a groundbreaking solution that addresses the limitations of traditional notification systems.
`

	tests := []struct {
		Name         string
		CategoryID   string
		FileContent  string
		ExpectedCode int
		Message      string
		Headers      map[string]string
	}{
		{
			Name:         "Successful creation of blog",
			CategoryID:   blogCategory.ID,
			FileContent:  content,
			ExpectedCode: http.StatusCreated,
			Message:      "blog created successfully",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
		},
		{
			Name:         "Unauthorized Access",
			CategoryID:   blogCategory.ID,
			FileContent:  content,
			ExpectedCode: http.StatusUnauthorized,
			Message:      "Token could not be found!",
			Headers: map[string]string{
				"Authorization": "",
			},
		},
		{
			Name:         "Invalid Category Id",
			CategoryID:   "invalid-id-uututu",
			FileContent:  content,
			ExpectedCode: http.StatusBadRequest,
			Message:      "invalid category id format",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
		},
		{
			Name:         "Blog category Not Found",
			CategoryID:   utility.GenerateUUID(),
			FileContent:  content,
			ExpectedCode: http.StatusNotFound,
			Message:      "blog category not found",
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var b bytes.Buffer
			writer := multipart.NewWriter(&b)

			// Add form field for category_id
			writer.WriteField("category_id", test.CategoryID)

			// Add file field for content
			part, err := writer.CreateFormFile("content", "blog.md")
			if err != nil {
				t.Fatalf("failed to create form file: %v", err)
			}
			_, err = io.WriteString(part, test.FileContent)
			if err != nil {
				t.Fatalf("failed to write to form file: %v", err)
			}

			writer.Close()

			req, _ := http.NewRequest(http.MethodPost, "/api/v1/blogs", &b)
			req.Header.Set("Content-Type", writer.FormDataContentType())
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
