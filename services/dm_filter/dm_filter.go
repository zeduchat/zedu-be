package dmfilter

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
	"github.com/hngprojects/telex_be/pkg/repository/storage/elastic"
)

func FilterData(db *storage.Database, userId, orgId string, c *gin.Context) ([]elastic.DmFilter, *elastic.PaginationResponse, int, error) {

	dms, pg, err := models.FilterDms(db, userId, orgId, c)

	if err != nil {
		if err.Error() == "Organisation does not exist" {
			return nil, nil, http.StatusNotFound, err
		} else if err.Error() == "User does not exist in the organisation" {
			return nil, nil, http.StatusBadRequest, err
		}

		return nil, nil, http.StatusInternalServerError, err
	}
	return dms, pg, http.StatusOK, nil
}
