package config

import "strings"

// docsEnvironments are the APP_NAME values that serve the API docs without
// SWAGGER_ENABLED being set.
var docsEnvironments = map[string]bool{
	"local":       true,
	"dev":         true,
	"development": true,
	"test":        true,
}

// DocsEnabled reports whether the Swagger UI and the OpenAPI spec may be
// served. They are on when SWAGGER_ENABLED is true or APP_NAME names a local
// or development environment, and off everywhere else, including when
// APP_NAME is unset.
func (a *App) DocsEnabled() bool {
	if a == nil {
		return false
	}
	if a.SwaggerEnabled {
		return true
	}
	return docsEnvironments[strings.ToLower(strings.TrimSpace(a.Name))]
}
