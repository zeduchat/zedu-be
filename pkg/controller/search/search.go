package search

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"

	"github.com/hngprojects/telex_be/external/request"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/services/search"
	"github.com/hngprojects/telex_be/utility"
)

type Controller struct {
	Db        *storage.Database
	Validator *validator.Validate
	Logger    *utility.Logger
	ExtReq    request.ExternalRequest
}

func (base *Controller) Search(c *gin.Context) {
	claims, exists := c.Get("userClaims")
	if !exists {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "unable to get user claims", "could not perform search", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}
	userClaims := claims.(jwt.MapClaims)
	userId := userClaims["user_id"].(string)

	orgId := c.Param("orgId")
	if !utility.IsValidUUID(orgId) {
		rd := utility.BuildErrorResponse(http.StatusBadRequest, "error", "invalid organisation id", "organisation could not be found", nil)
		c.JSON(http.StatusBadRequest, rd)
		return
	}

	query := c.Query("query")
	sortby := c.Query("sortby")

	req := search.SearchRequest{
		DB:     base.Db,
		Ctx:    c,
		Logger: base.Logger,
		UserID: userId,
		OrgID:  orgId,
		Query:  query,
		SortBy: sortby,
	}

	searchResult, pagination, code, err := search.Search(req)

	if err != nil && code == http.StatusNotFound {
		resp := utility.BuildErrorResponse(code, http.StatusText(code), err.Error(), err, nil)
		c.JSON(code, resp)
		return
	} else if err != nil {
		resp := utility.BuildErrorResponse(code, http.StatusText(code), err.Error(), err, nil)
		c.JSON(code, resp)
		return
	}
	resp := utility.BuildSuccessResponse(http.StatusOK, "success", searchResult, pagination)
	c.JSON(code, resp)
}
