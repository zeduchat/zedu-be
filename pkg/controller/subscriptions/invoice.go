package subscriptions

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/services/subscription"
	"github.com/hngprojects/telex_be/utility"
)

func (base *Controller) DownloadInvoice(ctx *gin.Context) {

	var (
		sessionID = ctx.Param("session_id")
		userID    = ctx.Param("user_id")
	)

	err := subscription.DownloadInvoice(sessionID, ctx, base.Db.Postgresql, userID)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusInternalServerError, "error", err.Error(), err, nil)
		base.Logger.Error(err)
		ctx.JSON(http.StatusInternalServerError, rd)
		return
	}

}
