package external

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/hngprojects/telex_be/external/external_models"
	"github.com/hngprojects/telex_be/utility"
)

func GetNewStreamingRequestObject(logger *utility.Logger, name, path, method, urlPrefix string, decodeMethod string, headers map[string]string, successCode int, data any, timeout bool, ctx context.Context) *SendRequestObject {
	obj := GetNewSendRequestObject(logger, name, path, method, urlPrefix, decodeMethod, headers, successCode, data, timeout)
	obj.IsStreaming = true
	if ctx != nil {
		obj.Context = ctx
	}
	return obj
}

func (r *SendRequestObject) SendStream() (<-chan external_models.StreamChunk, error) {
	dataChan := make(chan external_models.StreamChunk, 10)

	var (
		logger = r.Logger
		name   = r.Name
		err    error
	)
	buf := new(bytes.Buffer)

	switch data := r.Data.(type) {
	case *bytes.Buffer:
		buf = data
	case nil:
		logger.Info("r.Data is nil")
	default:
		err = json.NewEncoder(buf).Encode(r.Data)
		if err != nil {
			close(dataChan)
			return dataChan, err
		}
	}

	if r.UrlPrefix != "" {
		r.Path += r.UrlPrefix
	}

	var req *http.Request
	client := &http.Client{}

	if r.Timeout {
		client = &http.Client{
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout: 30 * time.Second,
				}).DialContext,
				ResponseHeaderTimeout: 30 * time.Second,
			},
			Timeout: 0,
		}
	}

	if r.Method == http.MethodGet {
		req, err = http.NewRequestWithContext(r.Context, r.Method, r.Path, nil)
	} else {
		req, err = http.NewRequestWithContext(r.Context, r.Method, r.Path, buf)
	}

	if err != nil {
		logger.Error("request creation error", name, err.Error())
		close(dataChan)
		return dataChan, err
	}

	for key, value := range r.Headers {
		req.Header.Add(key, value)
	}

	go func() {
		defer close(dataChan)

		logger.Info("streaming request (channel)", name, r.Path, r.Method, r.Headers)

		bodyBytes, _ := json.MarshalIndent(buf, "", "  ")
		fmt.Println("=== JSON BODY ===")
		fmt.Println(string(bodyBytes))

		res, err := client.Do(req)
		if err != nil {
			logger.Error("client do error", name, err.Error())
			dataChan <- external_models.StreamChunk{Error: err}
			return
		}
		defer res.Body.Close()

		ResponseCode = res.StatusCode

		if res.StatusCode != r.SuccessCode {
			errorBody, _ := io.ReadAll(res.Body)
			logger.Error("streaming request failed", name, "status", res.StatusCode, "body", string(errorBody))
			dataChan <- external_models.StreamChunk{Error: fmt.Errorf("streaming request failed for %v, code %v", name, res.StatusCode)}
			return
		}

		scanner := bufio.NewScanner(res.Body)
		var buffer strings.Builder

		for scanner.Scan() {
			select {
			case <-r.Context.Done():
				logger.Info("streaming request cancelled", name)
				dataChan <- external_models.StreamChunk{Error: r.Context.Err()}
				return
			default:
				line := scanner.Text()

				if line == "" {
					if buffer.Len() > 0 {
						dataChan <- external_models.StreamChunk{Data: []byte(buffer.String() + "\n")}
						buffer.Reset()
					}
				} else {
					buffer.WriteString(line + "\n")
				}
			}
		}

		if err := scanner.Err(); err != nil {
			logger.Error("streaming scan error", name, err.Error())
			dataChan <- external_models.StreamChunk{Error: err}
			return
		}

		logger.Info("streaming completed", name)
		dataChan <- external_models.StreamChunk{Done: true}
	}()

	return dataChan, nil
}
