package utility

import (
	"errors"
	"time"
)

var (
	ErrInvalidFromDate = errors.New("invalid from date format, use YYYY-MM-DD")
	ErrInvalidToDate   = errors.New("invalid to date format, use YYYY-MM-DD")
)

// ParseDateRange parses optional from/to date strings into time pointers
func ParseDateRange(fromStr, toStr string) (*time.Time, *time.Time, error) {
	var from, to *time.Time

	if fromStr != "" {
		parsedFrom, err := time.Parse("2006-01-02", fromStr)
		if err != nil {
			return nil, nil, ErrInvalidFromDate
		}
		from = &parsedFrom
	}

	if toStr != "" {
		parsedTo, err := time.Parse("2006-01-02", toStr)
		if err != nil {
			return nil, nil, ErrInvalidToDate
		}
		to = &parsedTo
	}

	return from, to, nil
}
