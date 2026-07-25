package utility

import (
	"fmt"
	"log"
)

func LogInfo(l *Logger, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if l != nil {
		l.Info(msg)
	} else {
		log.Println(msg)
	}
}

func LogError(l *Logger, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if l != nil {
		l.Error(msg)
	} else {
		log.Println(msg)
	}
}

func LogAndPrint(logger *Logger, data any, args ...any) {
	if logger == nil {
		return
	}
	if len(args) < 1 {
		logger.Info(data)
		return
	}
	logger.Info(data, args)
}
