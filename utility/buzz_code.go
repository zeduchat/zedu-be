package utility

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

func ExtractBuzzCode(buzzID string) string {
	parts := strings.Split(buzzID, "-")
	if len(parts) == 0 {
		return ""
	}
	return strings.ToUpper(parts[len(parts)-1])
}

func ResolveBuzzCode(db *gorm.DB, buzzCodeOrID string) (string, error) {
	if buzzCodeOrID == "" {
		return "", errors.New("buzz code cannot be empty")
	}

	if IsValidUUID(buzzCodeOrID) {
		return buzzCodeOrID, nil
	}

	buzzCode := strings.ToUpper(buzzCodeOrID)

	var buzzID string
	err := db.Table("buzzs").
		Select("id").
		Where("UPPER(SPLIT_PART(id::text, '-', 5)) = ?", buzzCode).
		Scan(&buzzID).Error

	if err != nil {
		return "", err
	}

	if buzzID == "" {
		return "", errors.New("buzz not found")
	}

	return buzzID, nil
}

func IsValidBuzzCode(buzzCode string) bool {
	if len(buzzCode) != 12 {
		return false
	}

	for _, char := range buzzCode {
		if !((char >= '0' && char <= '9') || (char >= 'A' && char <= 'F') || (char >= 'a' && char <= 'f')) {
			return false
		}
	}

	return true
}

func IsValidBuzzCodeOrUUID(input string) bool {
	return IsValidUUID(input) || IsValidBuzzCode(input)
}
