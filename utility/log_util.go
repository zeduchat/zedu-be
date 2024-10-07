package utility

func LogAndPrint(logger *Logger, data interface{}, args ...interface{}) {
	if len(args) < 1 {
		logger.Info(data)
		return
	}
	logger.Info(data, args)
}

