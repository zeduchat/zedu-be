package blog

import (
	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/utility"
	"gorm.io/gorm"
)

func SubmitFeedback(req models.BlogFeedbackReq, db *gorm.DB) error {
	blogFeedback := models.BlogFeedback{
		ID:       utility.GenerateUUID(),
		BlogID:   req.BlogID,
		Feedback: req.Feedback,
	}

	err := blogFeedback.Create(db)

	if err != nil {
		return err
	}

	return nil
}
