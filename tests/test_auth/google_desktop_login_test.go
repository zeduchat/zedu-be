package test_auth

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/hngprojects/telex_be/internal/models"
)

func TestGoogleDesktopValidation(t *testing.T) {
	v := validator.New()

	tests := []struct {
		name          string
		request       models.GoogleRequestModel
		expectedError bool
	}{
		{
			name: "Valid request with empty application_type",
			request: models.GoogleRequestModel{
				Token:           "some-valid-token",
				ApplicationType: "",
			},
			expectedError: false,
		},
		{
			name: "Valid request with desktop application_type",
			request: models.GoogleRequestModel{
				Token:           "some-valid-token",
				ApplicationType: "desktop",
			},
			expectedError: false,
		},
		{
			name: "Invalid request with invalid application_type",
			request: models.GoogleRequestModel{
				Token:           "some-valid-token",
				ApplicationType: "mobile",
			},
			expectedError: true,
		},
		{
			name: "Invalid request with missing token",
			request: models.GoogleRequestModel{
				Token:           "",
				ApplicationType: "desktop",
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Struct(&tt.request)
			if (err != nil) != tt.expectedError {
				t.Errorf("expected error: %v, got: %v", tt.expectedError, err)
			}
		})
	}
}
