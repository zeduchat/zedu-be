package test_threads

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/thread"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/tests"
)

func SetupThreadsTestRouter() (*gin.Engine, *thread.Controller) {
	gin.SetMode(gin.TestMode)

	logger := tests.Setup()
	db := storage.Connection()
	validator := validator.New()

	threadController := &thread.Controller{
		Db:        db,
		Validator: validator,
		Logger:    logger,
		ExtReq: request.ExternalRequest{
			Logger: logger,
			Test:   true,
		},
	}

	r := gin.Default()
	SetupThreadsRoutes(r, threadController)
	return r, threadController
}

func SetupThreadsRoutes(r *gin.Engine, threadController *thread.Controller) {

	r.PUT("/api/v1/threads/:thread_id/channels/:channel_id", middleware.Authorize(threadController.Db.Postgresql),
		threadController.UpdateAThread)
	r.GET("/api/v1/threads/:thread_id/channels/:channel_id", middleware.Authorize(threadController.Db.Postgresql),
		threadController.GetUserSingleThreads)
	r.GET("/api/v1/threads/channels/:channel_id", middleware.Authorize(threadController.Db.Postgresql),
		threadController.GetAllChannelThreads)
	r.GET("/api/v1/threads/organisations/:org_id", middleware.Authorize(threadController.Db.Postgresql),
		threadController.GetAllUserOrgThreads)

}
