package test_config

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hngprojects/telex_be/internal/config"
)

// TestDocsEnabled checks that the Swagger UI and OpenAPI spec are only served
// in local/dev environments or when SWAGGER_ENABLED is set (#1594).
func TestDocsEnabled(t *testing.T) {
	cases := []struct {
		name    string
		app     config.App
		enabled bool
	}{
		{"local", config.App{Name: "local"}, true},
		{"dev", config.App{Name: "dev"}, true},
		{"development mixed case", config.App{Name: " Development "}, true},
		{"test", config.App{Name: "test"}, true},
		{"production", config.App{Name: "production", Mode: "release"}, false},
		{"staging", config.App{Name: "staging"}, false},
		{"unset", config.App{}, false},
		{"flag on in production", config.App{Name: "production", SwaggerEnabled: true}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := tc.app
			assert.Equal(t, tc.enabled, app.DocsEnabled())
		})
	}

	var nilApp *config.App
	assert.False(t, nilApp.DocsEnabled())
}
