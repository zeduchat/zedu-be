package profile

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/middleware"
	"gorm.io/gorm"
)


func GetUserProfile( db *gorm.DB, c *gin.Context) (*models.Profile, int, error) {
    var user models.Profile

    userId, err := middleware.GetUserClaims(c, db, "user_id")
	if err != nil {
		return nil, http.StatusNotFound, err
	}

    fmt.Printf("UserID: %s\n", userId)

	userID, ok := userId.(string)
	if !ok {
		return nil, http.StatusBadRequest, errors.New("user_id is not of type string")
	}

	userProfile, err := user.GetUserProfile(db, userID)
	
	if err != nil {
         return nil, http.StatusNotFound, err
	}
    
	return &userProfile, http.StatusOK, nil
}