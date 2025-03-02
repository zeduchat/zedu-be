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
		orgID     = ctx.Param("org_id")
	)

	err := subscription.DownloadInvoice(sessionID, ctx, base.Db.Postgresql, base.Db.Redis, orgID)
	if err != nil {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", err.Error(), err, nil)
		base.Logger.Error("An error occured while fetching invoice: %v", err)
		ctx.JSON(http.StatusBadRequest, rd)
		return
	}

}
