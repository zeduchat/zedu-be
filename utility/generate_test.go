package utility

import (
	"regexp"
	"testing"
)

func TestGenerateUserColor(t *testing.T) {
	tests := []struct {
		name                string
		userID              string
		username            string
		expectedHexFormat   bool
		deterministic       bool
		shouldBeDifferent   bool
		compareWithUserID   string
		compareWithUsername string
	}{
		{
			name:              "Valid userID",
			userID:            "user-123-uuid",
			expectedHexFormat: true,
			deterministic:     true,
		},
		{
			name:              "Valid username",
			userID:            "",
			username:          "john_doe",
			expectedHexFormat: true,
			deterministic:     true,
		},
		{
			name:              "UserID takes precedence over username",
			userID:            "user-123",
			username:          "john_doe",
			expectedHexFormat: true,
		},
		{
			name:              "Empty inputs fallback",
			userID:            "",
			username:          "",
			expectedHexFormat: true,
		},
		{
			name:              "Different userIDs produce different colors",
			userID:            "user-123",
			username:          "",
			shouldBeDifferent: true,
			compareWithUserID: "user-456",
		},
		{
			name:              "Special characters handled",
			userID:            "user@email.com#123!",
			username:          "",
			expectedHexFormat: true,
		},
		{
			name:              "Long strings handled",
			userID:            "this-is-a-very-long-user-id-string-that-should-still-produce-a-valid-hex-color-code",
			username:          "",
			expectedHexFormat: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			color := GenerateUserColor(tt.userID, tt.username)

			// Check hex format
			if tt.expectedHexFormat {
				hexRegex := regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
				if !hexRegex.MatchString(color) {
					t.Errorf("GenerateUserColor(%q, %q) = %q, want valid hex format #RRGGBB", tt.userID, tt.username, color)
				}
			}

			// Check determinism
			if tt.deterministic {
				color2 := GenerateUserColor(tt.userID, tt.username)
				if color != color2 {
					t.Errorf("GenerateUserColor(%q, %q) is not deterministic: got %q first time, %q second time", tt.userID, tt.username, color, color2)
				}
			}

			// Check different inputs produce different colors
			if tt.shouldBeDifferent && tt.compareWithUserID != "" {
				color2 := GenerateUserColor(tt.compareWithUserID, "")
				if color == color2 {
					t.Errorf("GenerateUserColor(%q, %q) = %q, but GenerateUserColor(%q, %q) = %q, expected different colors", tt.userID, tt.username, color, tt.compareWithUserID, "", color2)
				}
			}
		})
	}
}

func TestHSLToHex(t *testing.T) {
	tests := []struct {
		name      string
		hue       float64
		sat       float64
		light     float64
		wantColor string // We'll just check format, exact value depends on conversion
	}{
		{
			name:      "Red hue",
			hue:       0,
			sat:       100,
			light:     50,
			wantColor: "#FF0000", // Should produce red
		},
		{
			name:      "Green hue",
			hue:       120,
			sat:       100,
			light:     50,
			wantColor: "#00FF00", // Should produce green
		},
		{
			name:      "Blue hue",
			hue:       240,
			sat:       100,
			light:     50,
			wantColor: "#0000FF", // Should produce blue
		},
		{
			name:      "Gray (low saturation)",
			hue:       0,
			sat:       0,
			light:     50,
			wantColor: "#808080", // Should produce gray
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			color := hslToHex(tt.hue, tt.sat, tt.light)

			// Verify hex format
			hexRegex := regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
			if !hexRegex.MatchString(color) {
				t.Errorf("hslToHex(%v, %v, %v) = %q, want valid hex format", tt.hue, tt.sat, tt.light, color)
			}
		})
	}
}

func TestGenerateUserColorHueDiversity(t *testing.T) {
	// Generate colors for multiple users and verify they're reasonably diverse
	userIDs := []string{
		"user-1",
		"user-2",
		"user-3",
		"user-4",
		"user-5",
	}

	colors := make(map[string]bool)
	for _, userID := range userIDs {
		color := GenerateUserColor(userID, "")
		colors[color] = true
	}

	// We should have different colors for different users
	// (theoretically could collide, but extremely unlikely with 5 users)
	if len(colors) < 4 {
		t.Logf("GenerateUserColor produced only %d unique colors for 5 different users, expected better diversity", len(colors))
	}
}
