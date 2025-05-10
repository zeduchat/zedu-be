package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	fcmtokens "github.com/hngprojects/telex_be/pkg/controller/fcmTokens"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func FcmToken(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	fcm := fcmtokens.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	fcmtokenUrl := r.Group(fmt.Sprintf("%v/fcmtoken", ApiVersion), middleware.Authorize(db.Postgresql))

	{
		fcmtokenUrl.POST("", fcm.CreateFcmToken)
	}

	return r
}


// channel uri: https://learn.microsoft.com/en-us/previous-versions/windows/apps/hh465412(v=win.10)
// access token and sending push notfication: https://learn.microsoft.com/en-us/previous-versions/windows/apps/hh868252(v=win.10)
// https://github.com/oniestel/go-wns/blob/master/client.go