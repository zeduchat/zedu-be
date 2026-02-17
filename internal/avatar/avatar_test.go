package avatar

import (
	"strings"
	"testing"
)

func TestGenerateDefaultAvatarURL(t *testing.T) {
	tests := []struct {
		name   string
		userID string
		want   string // partial match since config might be empty or present
	}{
		{
			name:   "User ID 1",
			userID: "user-123",
		},
		{
			name:   "User ID 2",
			userID: "user-456",
		},
		{
			name:   "Empty User ID",
			userID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateDefaultAvatarURL(tt.userID)
			if !strings.Contains(got, "default_avatars/default_avatar_") {
				t.Errorf("GenerateDefaultAvatarURL() = %v, does not contain expected path", got)
			}
			if !strings.HasSuffix(got, ".png") {
				t.Errorf("GenerateDefaultAvatarURL() = %v, does not end with .png", got)
			}

			// Verify deterministic behavior
			got2 := GenerateDefaultAvatarURL(tt.userID)
			if got != got2 {
				t.Errorf("GenerateDefaultAvatarURL() is not deterministic; got %v then %v", got, got2)
			}
		})
	}
}
