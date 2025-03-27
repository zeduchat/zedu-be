package dmfilter

import (
	"net/http"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/repository/storage"
)

func FilterData(db *storage.Database, userId, orgId string) (*models.RawValues, int, error) {

	dms, err := models.FilterDms(db, userId, orgId)

	if err != nil {
		if err.Error() == "Organisation does not exist" {
			return nil, http.StatusNotFound, err
		} else if err.Error() == "User does not exist in the organisation" {
			return nil, http.StatusBadRequest, err
		}

		return nil, http.StatusInternalServerError, err
	}
	return dms, http.StatusOK, nil
}
