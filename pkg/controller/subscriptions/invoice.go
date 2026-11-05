package subscriptions

import (
	"errors"
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

	resp, err := subscription.DownloadInvoice(sessionID, ctx, base.Db.Postgresql, base.Db.Redis, orgID)
	if err != nil {
		statusCode := http.StatusBadRequest
		if errors.Is(err, subscription.ErrNotOrgMember) || errors.Is(err, subscription.ErrSessionOrgMismatch) {
			statusCode = http.StatusForbidden
		}
		rd := utility.BuildErrorResponse(statusCode, "error", err.Error(), err, nil)
		base.Logger.Error("An error occured while fetching invoice: %v", err)
		ctx.JSON(statusCode, rd)
		return
	}

	base.Logger.Info("invoiced retrived")
	rd := utility.BuildSuccessResponse(http.StatusOK, "Invoiced retrived successfully", resp)
	ctx.JSON(http.StatusOK, rd)
}
