package audit_utility

import (
	"time"
)

func GetCurrentTime() time.Time {
	return time.Now()
}

func GetStringDateTime() string {
	currentTime := GetCurrentTime()
	layout := "2006-01-02 15:04:05"
	formattedTime := currentTime.Format(layout)
	return formattedTime
}
