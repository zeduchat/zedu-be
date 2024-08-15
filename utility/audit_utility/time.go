package audit_utility

import (
	"time"
)

func GetCurrentTime() time.Time {
	t := time.Now()
	formattedTimeStr := t.Format("2006-01-02 15:04:05")
	parsedTime, _ := time.Parse("2006-01-02 15:04:05", formattedTimeStr)
	return parsedTime
}

func GetStringDateTime() string {
	currentTime := GetCurrentTime()
	layout := "2006-01-02 15:04:05"
	formattedTime := currentTime.Format(layout)
	return formattedTime
}
