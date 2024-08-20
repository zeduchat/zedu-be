package newsletter

import (
	"strings"

	"github.com/hngprojects/telex_be/internal/models"
	"gorm.io/gorm"
)

func NewsLetterSubscribe(newsletter *models.NewsLetter, db *gorm.DB) error {

	newsletter.Email = strings.ToLower(newsletter.Email)

	if err := newsletter.CreateNewsLetter(db, newsletter.Email); err != nil {
		return err
	}

	return nil
}
