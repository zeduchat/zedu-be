package avatar

import (
	"strings"
	"testing"

	"github.com/hngprojects/telex_be/internal/config"
)

func TestGenerateDefaultAvatarURL(t *testing.T) {
	// Save original config
	originalConfig := config.Config
	defer func() { config.Config = originalConfig }()

	tests := []struct {
		name       string
		userID     string
		appMode    string
		wantBucket string
	}{
		{
			name:       "Prod Mode",
			userID:     "user-123",
			appMode:    "prod",
			wantBucket: "telexprodbucket",
		},
		{
			name:       "Staging Mode",
			userID:     "user-456",
			appMode:    "staging",
			wantBucket: "telexstagingbucket",
		},
		{
			name:       "Empty Mode (Default to Staging)",
			userID:     "user-789",
			appMode:    "",
			wantBucket: "telexstagingbucket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock config
			config.Config = &config.Configuration{
				App: config.App{
					Mode: tt.appMode,
				},
				Minio: config.Minio{
					MinioEndpoint: "media.zedu.chat",
				},
			}

			got := GenerateDefaultAvatarURL(tt.userID)

			if !strings.Contains(got, tt.wantBucket) {
				t.Errorf("GenerateDefaultAvatarURL() = %v, expected to contain bucket %v", got, tt.wantBucket)
			}

			if !strings.Contains(got, "public/default_avatars/default_avatar_") {
				t.Errorf("GenerateDefaultAvatarURL() = %v, does not contain expected path structure", got)
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
