package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
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
	dataChan := make(chan external_models.StreamChunk, 10) // buffered channel

	var (
		data   = r.Data
		logger = r.Logger
		name   = r.Name
		err    error
	)

	buf := new(bytes.Buffer)
	if data != nil {
		err = json.NewEncoder(buf).Encode(data)
		if err != nil {
			logger.Error("encoding error", name, err.Error())
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
		switch r.Headers["Content-Type"] {
		case "application/x-www-form-urlencoded", "application/json":
			req, err = http.NewRequestWithContext(r.Context, r.Method, r.Path, data.(io.Reader))
		default:
			req, err = http.NewRequestWithContext(r.Context, r.Method, r.Path, buf)
		}
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

		buffer := make([]byte, 4096)
		for {
			select {
			case <-r.Context.Done():
				logger.Info("streaming request cancelled", name)
				dataChan <- external_models.StreamChunk{Error: r.Context.Err()}
				return
			default:
				n, err := res.Body.Read(buffer)
				if n > 0 {
					chunk := make([]byte, n)
					copy(chunk, buffer[:n])
					dataChan <- external_models.StreamChunk{Data: chunk}
				}
				if err == io.EOF {
					logger.Info("streaming completed", name)
					dataChan <- external_models.StreamChunk{Done: true}
					return
				}
				if err != nil {
					logger.Error("streaming read error", name, err.Error())
					dataChan <- external_models.StreamChunk{Error: err}
					return
				}
			}
		}
	}()

	return dataChan, nil
}
