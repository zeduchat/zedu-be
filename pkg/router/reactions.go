package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/reactions"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Reactions(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	reactions := reactions.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}
	reactionsUrl := r.Group(fmt.Sprintf("%v/reactions", ApiVersion), middleware.Authorize(db.Postgresql))

	{
		reactionsUrl.POST("/:channel_id", reactions.CreateReaction)
		reactionsUrl.GET("/:reaction_id/thread/:thread_id", reactions.GetThreadReactions)
		reactionsUrl.GET("/:reaction_id/reply/:message_id", reactions.GetReplyReactions)
		reactionsUrl.DELETE("/:reaction_id/thread/:thread_id", reactions.DeleteThreadReactions)
		reactionsUrl.DELETE("/:reaction_id/reply/:message_id", reactions.DeleteReplyReactions)
	}

	return r
}
