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
		userUrl.PUT("/users/:user_id", user.UpdateAUser)
		userUrl.GET("users/organisations", user.GetAUserOrganisation)

		userUrl.PUT("/users/:user_id/roles/:role_id", user.AssignRoleToUser)
		userUrl.PUT("/users/:user_id/identity/:role_id", user.UpdateUserIdentity)
		userUrl.GET("/users/:user_id/login-audit", user.GetUserLoginAudit)
		userUrl.PUT("/users/revoke-session", user.RevokeUserAccessToken)
		userUrl.DELETE("/users/deactivate/:user_id", user.DeactiveUser)
		userUrl.GET("/users/:user_id/sessions", user.GetUserSessions)
	}
	adminUrl.GET("/users", user.GetAllUsers)

	return r
}
