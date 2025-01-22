package integrations

func (r *RequestObj) RetriveJsonData() (map[string]interface{}, error) {
	var (
		outBoundResponse map[string]interface{}
		logger           = r.Logger
	)

	logger.Info("Retrieving JSON data")

	err := r.getNewSendRequestObject(nil, map[string]string{}, "").SendRequest(&outBoundResponse)
	if err != nil {
		logger.Error("failed to fetch json content", outBoundResponse, err.Error())
		return outBoundResponse, err
	}

	return outBoundResponse, nil
}
