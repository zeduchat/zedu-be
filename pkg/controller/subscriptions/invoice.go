package subscriptions

import (
	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/services/subscription"
)

func (base *Controller) DownloadInvoice(ctx *gin.Context) {

	var (
		sessionID = ctx.Param("session_id")
		userID    = ctx.Param("user_id")
	)

	subscription.DownloadInvoice(sessionID, ctx, base.Db.Postgresql, userID)

}
