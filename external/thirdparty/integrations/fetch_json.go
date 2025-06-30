package integrations

func (r *RequestObj) RetriveJsonData() (map[string]any, error) {
	var (
		outBoundResponse map[string]any
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

func (r *RequestObj) SendAgentApiKey() (map[string]any, error) {
	var (
		logger           = r.Logger
		outBoundResponse map[string]any
	)

	headers := map[string]string{
		"Content-Type": "application/json",
	}

	err := r.getNewSendRequestObject(r.RequestData, headers, "/auth_callback").SendRequest(&outBoundResponse)
	if err != nil {
		logger.Error("failed to send api key to agent", outBoundResponse, err.Error())
		return outBoundResponse, err
	}

	return outBoundResponse, nil
}
