package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/controller/user"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func User(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	extReq := request.ExternalRequest{Logger: logger, Test: false}
	user := user.Controller{Db: db, Validator: validator, Logger: logger, ExtReq: extReq}

	userUrl := r.Group(fmt.Sprintf("%v", ApiVersion), middleware.Authorize(db.Postgresql))
	adminUrl := r.Group(fmt.Sprintf("%v", ApiVersion), middleware.Authorize(db.Postgresql))
	{
		userUrl.GET("/users/:user_id", user.GetAUser)
		userUrl.DELETE("/users/:user_id", user.DeleteAUser)

		userUrl.PUT("/users/:user_id", middleware.CheckIsDeactivated(db.Postgresql), user.UpdateAUser)
		userUrl.GET("users/organisations", user.GetAUserOrganisation)

		userUrl.GET("/users/:user_id/login-audit", middleware.CheckIsDeactivated(db.Postgresql), user.GetUserLoginAudit)
		userUrl.PUT("/users/revoke-session", middleware.CheckIsDeactivated(db.Postgresql), user.RevokeUserAccessToken)
		userUrl.DELETE("/users/deactivate/:user_id", middleware.CheckIsDeactivated(db.Postgresql), user.DeactiveUser)
		userUrl.GET("/users/:user_id/sessions", middleware.CheckIsDeactivated(db.Postgresql), user.GetUserSessions)
		userUrl.PUT("/users/switch-org", middleware.CheckIsDeactivated(db.Postgresql), user.SwitchUserOrg)
		userUrl.PUT("/users/switch-roles", middleware.CheckIsDeactivated(db.Postgresql), user.AssignRoleToUser)
		userUrl.PUT("/users/reactivate/:user_id", user.ActivateUser)

		userUrl.GET("/users/notification-preferences", user.GetUserNotificationSettings)
		userUrl.PUT("/users/notification-preferences", user.UpdateUserNotificationSettings)
	}
	adminUrl.GET("/users", user.GetAllUsers)

	return r
}


