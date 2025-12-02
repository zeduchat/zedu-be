package test_buzz

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/buzz"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	tst "github.com/hngprojects/telex_be/tests"
)

func SetupBuzzTestRouter() (*gin.Engine, *buzz.Controller) {
	gin.SetMode(gin.TestMode)

	logger := tst.Setup()
	db := storage.Connection()
	validator := validator.New()

	buzzController := &buzz.Controller{
		Db:        db,
		Validator: validator,
		Logger:    logger,
	}

	r := gin.Default()
	buzzGroup := r.Group("/api/v1/buzz", middleware.Authorize(buzzController.Db.Postgresql))
	{
		buzzGroup.PUT("/:id/camera", buzzController.UpdateCamera)
		buzzGroup.POST("/:id/notes", buzzController.CreateNote)
		buzzGroup.GET("/:id/notes", buzzController.GetNotes)
		buzzGroup.PATCH("/:id/notes/:note_id", buzzController.UpdateNote)
		buzzGroup.POST("/create", buzzController.Create)
		buzzGroup.POST("/:id/leave", buzzController.LeaveBuzz)
		buzzGroup.POST("/:id/end", buzzController.EndBuzz)
		buzzGroup.POST("/search-members", buzzController.SearchChannelMembers)
		buzzGroup.POST("/invite", buzzController.InviteUsersToBuzz)
		buzzGroup.POST("/invitation/respond", buzzController.RespondToInvitation)
		buzzGroup.GET("/invitations/pending", buzzController.GetPendingInvitations)
	}

	return r, buzzController
}

// SetupBuzzInvitationTestRouter sets up router with invitation endpoints
func SetupBuzzInvitationTestRouter() (*gin.Engine, *buzz.Controller) {
	return SetupBuzzTestRouter()
}

func SetupBuzzEndTestRouter() (*gin.Engine, *buzz.Controller) {
	return SetupBuzzTestRouter()
}

// getTestUserID retrieves the user ID from the database by email
func getTestUserID(db *storage.Database, email string) string {
	var user models.User
	if err := db.Postgresql.Where("email = ?", email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ""
		}
		return ""
	}
	return user.ID
}
