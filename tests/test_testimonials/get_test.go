package test_testimonial

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

func TestGetTestimonials(t *testing.T) {

	_, testimonialController := SetupTestimonialTestRouter()
	db := testimonialController.Db.Postgresql
	currUUID := utility.GenerateUUID()
	password, _ := utility.HashPassword("password")

	user := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "Regular User",
		Email:    fmt.Sprintf("user%v@qa.team", currUUID),
		Password: password,
	}

	testimonial := models.Testimonial{
		ID:          utility.GenerateUUID(),
		UserID:      user.ID,
		CompanyName: "somecompany",
		Name:        user.Name,
		Content:     "i love whats going on here",
		ImageURL:    "imageehere",
	}

	db.Create(&user)
	db.Create(&testimonial)

	setup := func() *gin.Engine {
		router, _ := SetupTestimonialTestRouter()
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
			Name:         "Successful retrieval of Testimonials",
			ExpectedCode: http.StatusOK,
			Message:      "testimonials retrieved successfully",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/testimonials", nil)

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
