package utility

func LogAndPrint(logger *Logger, data any, args ...any) {
	if len(args) < 1 {
		logger.Info(data)
		return
	}
	logger.Info(data, args)
}
