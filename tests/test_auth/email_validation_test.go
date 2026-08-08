package test_auth

import (
	"testing"

	"github.com/hngprojects/telex_be/utility"
)

func TestSignupEmailValidation(t *testing.T) {
	tests := []struct {
		name          string
		email         string
		expectError   bool
		expectedError string
		expectedEmail string
	}{
		{
			name:          "Valid Email",
			email:         "testuser@gmail.com",
			expectError:   false,
			expectedEmail: "testuser@gmail.com",
		},
		{
			name:          "Valid Email With Alias Tag Preserved",
			email:         "testuser+tag123@gmail.com",
			expectError:   false,
			expectedEmail: "testuser+tag123@gmail.com",
		},
		{
			name:          "Disposable Email Mailinator",
			email:         "testuser@mailinator.com",
			expectError:   true,
			expectedError: "disposable email domains are not allowed for registration",
		},
		{
			name:          "Disposable Email TempMail",
			email:         "testuser@tempmail.com",
			expectError:   true,
			expectedError: "disposable email domains are not allowed for registration",
		},
		{
			name:          "Invalid DNS Domain",
			email:         "testuser@nonexistentdomainxyz999.com",
			expectError:   true,
			expectedError: "invalid email address",
		},
		{
			name:          "Invalid DNS Domain ddfs.co",
			email:         "testuser@ddfs.co",
			expectError:   true,
			expectedError: "invalid email address",
		},
		{
			name:          "Invalid Format Missing Domain",
			email:         "testuser@",
			expectError:   true,
			expectedError: "invalid email address",
		},
		{
			name:          "Invalid Format Plain String",
			email:         "notanemail",
			expectError:   true,
			expectedError: "invalid email address",
		},
		{
			name:          "Empty Email",
			email:         "",
			expectError:   true,
			expectedError: "invalid email address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := utility.ValidateSignupEmail(tt.email)
			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error for email '%s', got nil", tt.email)
				}
				if err.Error() != tt.expectedError {
					t.Errorf("Expected error '%s', got '%s'", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error for email '%s', got '%v'", tt.email, err)
				}
				if res != tt.expectedEmail {
					t.Errorf("Expected normalized email '%s', got '%s'", tt.expectedEmail, res)
				}
			}
		})
	}
}
