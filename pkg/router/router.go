package router

import (
	"context"
	"net/http"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"riverqueue.com/riverui"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Setup(logger *utility.Logger, validator *validator.Validate, db *storage.Database, appConfiguration *config.App) *gin.Engine {
	if appConfiguration.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()

	// Middlewares
	// r.Use(gin.Logger())
	r.ForwardedByClientIP = true
	r.SetTrustedProxies(config.GetConfig().Server.TrustedProxies)
	r.Use(middleware.Security())
	r.Use(middleware.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())
	r.Use(gzip.Gzip(gzip.DefaultCompression))
	r.MaxMultipartMemory = 1 << 20 // 1MB

	// routers
	ApiVersion := "api/v1"
	Testimonial(r, ApiVersion, validator, db, logger)
	ApiStatus(r, ApiVersion, validator, db, logger)
	Newsletter(r, ApiVersion, validator, db, logger)
	Slack(r, ApiVersion, validator, db, logger)
	Contact(r, ApiVersion, validator, db, logger)
	Blog(r, ApiVersion, validator, db, logger)
	FileManagement(r, ApiVersion, validator, db, logger)
	Avatar(r, ApiVersion, validator, db, logger)
	Health(r, ApiVersion, validator, db, logger)
	Auth(r, ApiVersion, validator, db, logger)
	Channels(r, ApiVersion, validator, db, logger)
	TokenGen(r, ApiVersion, validator, db, logger)
	Organisation(r, ApiVersion, validator, db, logger)
	User(r, ApiVersion, validator, db, logger)
	Credits(r, ApiVersion, validator, db, logger)
	Invite(r, ApiVersion, validator, db, logger)
	Webhook(r, ApiVersion, validator, db, logger)
	Profile(r, ApiVersion, validator, db, logger)
	Threads(r, ApiVersion, validator, db, logger)
	Subscriptions(r, ApiVersion, validator, db, logger)
	WebhookHandler(r, ApiVersion, validator, db, logger)
	HelpCenter(r, ApiVersion, validator, db, logger)
	OptIn(r, ApiVersion, validator, db, logger)
	Dms(r, ApiVersion, validator, db, logger)
	GroupDMs(r, ApiVersion, validator, db, logger)
	Group(r, ApiVersion, validator, db, logger)
	Admin(r, ApiVersion, validator, db, logger)
	Buzz(r, ApiVersion, validator, db, logger)

	Waitlist(r, ApiVersion, validator, db, logger)
	Search(r, ApiVersion, validator, db, logger)
	Agents(r, ApiVersion, validator, db, logger)
	Mongogrations(r, ApiVersion, validator, db, logger)
	FcmToken(r, ApiVersion, validator, db, logger)
	TelexAI(r, ApiVersion, validator, db, logger)
	GetRecentLogs(r, ApiVersion, validator, db, logger)
	SavedMessages(r, ApiVersion, validator, db, logger)
	PinMessages(r, ApiVersion, validator, db, logger)
	Reactions(r, ApiVersion, validator, db, logger)
	WorkflowRoutes(r, ApiVersion, validator, db, logger)
	CredentialRoutes(r, ApiVersion, validator, db, logger)
	ForwardMessage(r, ApiVersion, validator, db, logger)
	AgentSkillTask(r, ApiVersion, validator, db, logger)
	Translator(r, ApiVersion, validator, db, logger)
	Shares(r, ApiVersion, validator, db, logger)

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "HNGi Golang Telex BE",
			"status":  http.StatusOK,
		})
	})

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"name":    "Not Found",
			"message": "Page not found.",
			"code":    404,
			"status":  http.StatusNotFound,
		})
	})

	r.StaticFile("/swagger.yaml", "static/swagger.yaml")
	url := ginSwagger.URL("/swagger.yaml")
	r.GET("/api/docs/*any", func(c *gin.Context) {
		c.Writer.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'sha256-2TOI2ugkuROHHfKZr6kdGv+XxhrVUI8uHycXqXUIR4g='; img-src 'self' data:;")
		ginSwagger.WrapHandler(swaggerFiles.Handler, url)(c)
	})

	handler, err := SetupRiverUI(db)
	if err != nil {
		panic(err)
	}

	if err := handler.Start(context.Background()); err != nil {
		panic(err)
	}

	if db.River != nil {
		wrappedHandler := gin.WrapH(handler)
		r.GET("/river/*any", func(c *gin.Context) {
			c.Writer.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'sha256-2TOI2ugkuROHHfKZr6kdGv+XxhrVUI8uHycXqXUIR4g='; img-src 'self' data:;")
			wrappedHandler(c)
		})
	}

	return r
}

func SetupRiverUI(db *storage.Database) (*riverui.Handler, error) {

	sLogger := utility.NewRotatingLogger()

	handler, err := riverui.NewHandler(&riverui.HandlerOpts{
		DevMode:   false,
		Logger:    sLogger,
		Endpoints: riverui.NewEndpoints(db.River, nil),
		Prefix:    "",
	})
	if err != nil {
		return nil, err
	}

	return handler, nil
}
