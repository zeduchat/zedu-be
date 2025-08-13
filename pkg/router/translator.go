package router

import (
	"fmt"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/assets"
	"github.com/hngprojects/telex_be/pkg/controller/translator"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Translator(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	transCtrl := translator.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	SetupStaticRoutes(r, ApiVersion)

	translatorURL := r.Group(fmt.Sprintf("%v/translator", ApiVersion))
	{
		translatorURL.POST("/", transCtrl.GenerateTranslation)
		translatorURL.GET("/tester", transCtrl.TranslationTester)
	}

	promptURL := r.Group(fmt.Sprintf("%v/prompts", ApiVersion))
	{
		promptURL.POST("/", transCtrl.CreatePrompt)
		promptURL.GET("/", transCtrl.GetPrompts)
		promptURL.GET("/:prompt_id", transCtrl.GetPrompt)
		promptURL.GET("/steps", transCtrl.FetchUniqueSteps)
	}

	return r
}

func SetupStaticRoutes(r *gin.Engine, apiVersion string) {

	staticFS, err := fs.Sub(assets.StaticFiles, "static")
	if err != nil {
		panic("Failed to create static file system: " + err.Error())
	}

	// Serve static files
	r.StaticFS("/static", http.FS(staticFS))
}
