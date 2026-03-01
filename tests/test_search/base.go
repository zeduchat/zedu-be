package test_search

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/router"
	"github.com/hngprojects/telex_be/utility"
)

func SetupSearchRouter(db *storage.Database, logger *utility.Logger) *gin.Engine {
	r := gin.Default()
	validatorRef := validator.New()
	return router.Search(r, "/api/v1", validatorRef, db, logger)
}
