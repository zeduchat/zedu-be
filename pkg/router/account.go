package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/pkg/controller/account"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/utility"
)

func Account(r *gin.Engine, ApiVersion string, validator *validator.Validate, db *storage.Database, logger *utility.Logger) *gin.Engine {
	accountController := account.Controller{Db: db, Validator: validator, Logger: logger}

	accountUrl := r.Group(fmt.Sprintf("%v/account", ApiVersion), middleware.Authorize(db.Postgresql))
	{
		accountUrl.POST("/delete-account", accountController.CreateAccountDeletionRequest)
	}

	return r
}
