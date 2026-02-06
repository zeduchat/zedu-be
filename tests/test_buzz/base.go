package test_buzz

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/buzz"
	"github.com/hngprojects/telex_be/pkg/controller/thread"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

const ApiVersion = "/api/v1"

func SetupBuzzTestRouter(logger *utility.Logger, validator *validator.Validate) (*gin.Engine, *buzz.Controller) {
	gin.SetMode(gin.TestMode)

	db := storage.Connection()

	ctrl := &buzz.Controller{
		Db:        db,
		Validator: validator,
		Logger:    logger,
	}

	extReq := request.ExternalRequest{Logger: logger, Test: true}
	threadCtrl := thread.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	r := gin.Default()
	r.GET("/api/v1/threads/channels/:channel_id/threads", middleware.Authorize(db.Postgresql), threadCtrl.GetAllChannelThreads)

	buzzGroup := r.Group(fmt.Sprintf("%v/buzz", ApiVersion), middleware.Authorize(db.Postgresql), middleware.CheckIsDeactivated(db.Postgresql))
	{
		buzzGroup.POST("/create", ctrl.Create)
		buzzGroup.POST("/org/create", ctrl.CreateOrgBuzz)
		buzzGroup.GET("/org", ctrl.GetOrgBuzzList)
		buzzGroup.POST("/:id/join", ctrl.Join)
		buzzGroup.POST("/token", ctrl.GetAgoraToken)
		buzzGroup.POST("/:id/notes", ctrl.CreateNote)
		buzzGroup.GET("/:id/notes", ctrl.GetNotes)
		buzzGroup.PATCH("/:id/notes/:note_id", ctrl.UpdateNote)
		buzzGroup.PATCH("/:id/camera", ctrl.UpdateCamera)
		buzzGroup.PATCH("/:id/media-state", ctrl.UpdateMediaState)
		buzzGroup.POST("/:id/leave", ctrl.LeaveBuzz)
		buzzGroup.POST("/:id/end", ctrl.EndBuzz)
		buzzGroup.POST("/channel/:channel_id/end", ctrl.EndBuzzByChannel)
		buzzGroup.GET("/:id/metadata", ctrl.GetMetadata)
		buzzGroup.GET("/channel/:channel_id/active", ctrl.GetChannelActiveBuzz)
		buzzGroup.GET("/dm/:dm_id/active", ctrl.GetDMActiveBuzz)
		buzzGroup.GET("/group-dm/:group_dm_id/active", ctrl.GetGroupDMActiveBuzz)
		buzzGroup.POST("/:id/reaction", ctrl.SendReaction)
		buzzGroup.POST("/:id/sticker", ctrl.UpdateSticker)
		buzzGroup.POST("/search-members", ctrl.SearchChannelMembers)
		buzzGroup.POST("/invite", ctrl.InviteUsersToBuzz)
		buzzGroup.POST("/invitation/respond", ctrl.RespondToInvitation)
		buzzGroup.GET("/invitations/pending", ctrl.GetPendingInvitations)

		buzzGroup.POST("/:id/force-end", ctrl.ForceEndBuzz)
		buzzGroup.POST("/:id/message", ctrl.SendBuzzMessage)
	}

	return r, ctrl // Renamed buzzController to ctrl
}

// SetupBuzzInvitationTestRouter sets up router with invitation endpoints
func SetupBuzzInvitationTestRouter(logger *utility.Logger, validator *validator.Validate) (*gin.Engine, *buzz.Controller) {
	return SetupBuzzTestRouter(logger, validator)
}

func SetupBuzzEndTestRouter(logger *utility.Logger, validator *validator.Validate) (*gin.Engine, *buzz.Controller) {
	return SetupBuzzTestRouter(logger, validator)
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
