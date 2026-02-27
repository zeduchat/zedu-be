package agora

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hngprojects/telex_be/internal/config"
)

const (
	agoraRecordingBaseURL    = "https://api.agora.io/v1/apps"
	recordingModeComposite   = "mix"
	recordingStorageVendorS3 = 11
	recordingRegionUS        = 0
)

type recordingClient struct {
	appID          string
	customerID     string
	customerSecret string
}

type acquireRequest struct {
	Cname         string            `json:"cname"`
	Uid           string            `json:"uid"`
	ClientRequest map[string]string `json:"clientRequest"`
}

type acquireResponse struct {
	ResourceId string `json:"resourceId"`
}

type ExtensionParams struct {
	EndPoint string `json:"endpoint"`
}

type storageConfig struct {
	Vendor          int             `json:"vendor"`
	Region          int             `json:"region"`
	Bucket          string          `json:"bucket"`
	AccessKey       string          `json:"accessKey"`
	SecretKey       string          `json:"secretKey"`
	FileNamePrefix  []string        `json:"fileNamePrefix"`
	ExtensionParams ExtensionParams `json:"extensionParams"`
}

type recordingConfig struct {
	MaxIdleTime        int      `json:"maxIdleTime"`
	StreamTypes        int      `json:"streamTypes"`
	ChannelType        int      `json:"channelType"`
	VideoStreamType    int      `json:"videoStreamType"`
	SubscribeAudioUIDs []string `json:"subscribeAudioUids"`
	SubscribeVideoUIDs []string `json:"subscribeVideoUids"`
}

type startRecordingRequest struct {
	Cname         string             `json:"cname"`
	Uid           string             `json:"uid"`
	ClientRequest startClientRequest `json:"clientRequest"`
}

type startClientRequest struct {
	Token           string          `json:"token,omitempty"`
	RecordingConfig recordingConfig `json:"recordingConfig"`
	StorageConfig   storageConfig   `json:"storageConfig"`
}

type startResponse struct {
	Sid        string `json:"sid"`
	ResourceId string `json:"resourceId"`
}

type queryResponse struct {
	ResourceId     string         `json:"resourceId"`
	Sid            string         `json:"sid"`
	ServerResponse serverResponse `json:"serverResponse"`
}

type serverResponse struct {
	Status   int        `json:"status"`
	FileList []fileInfo `json:"fileList"`
}

type fileInfo struct {
	Filename string `json:"filename"`
}

func newRecordingClient() (*recordingClient, error) {
	cfg := config.GetConfig()
	if cfg.Agora.AppId == "" {
		return nil, errors.New("agora app id not configured")
	}
	return &recordingClient{
		appID:          cfg.Agora.AppId,
		customerID:     cfg.Agora.CustomerID,
		customerSecret: cfg.Agora.CustomerSecret,
	}, nil
}

func (rc *recordingClient) doRequest(method, url string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewBuffer(data)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(rc.customerID, rc.customerSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("agora API error (status %d): %s", resp.StatusCode, string(respData))
	}

	return respData, nil
}

func AcquireRecording(buzzID, uid string) (string, error) {
	rc, err := newRecordingClient()
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/%s/cloud_recording/acquire", agoraRecordingBaseURL, rc.appID)
	reqBody := acquireRequest{
		Cname:         buzzID,
		Uid:           uid,
		ClientRequest: map[string]string{},
	}

	respData, err := rc.doRequest(http.MethodPost, url, reqBody)
	if err != nil {
		return "", fmt.Errorf("acquire recording failed: %w", err)
	}

	var resp acquireResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		return "", fmt.Errorf("failed to parse acquire response: %w", err)
	}

	return resp.ResourceId, nil
}

func StartRecording(resourceID, buzzID, uid string, maxIdleSecs int) (string, error) {
	rc, err := newRecordingClient()
	if err != nil {
		return "", err
	}

	cfg := config.GetConfig()
	url := fmt.Sprintf("%s/%s/cloud_recording/resourceid/%s/mode/%s/start",
		agoraRecordingBaseURL, rc.appID, resourceID, recordingModeComposite)

	minioEndpoint := strings.TrimPrefix(cfg.Minio.MinioEndpoint, "https://")
	minioEndpoint = strings.TrimPrefix(minioEndpoint, "http://")

	reqBody := startRecordingRequest{
		Cname: buzzID,
		Uid:   uid,
		ClientRequest: startClientRequest{
			RecordingConfig: recordingConfig{
				MaxIdleTime:        maxIdleSecs,
				StreamTypes:        2,
				ChannelType:        0,
				VideoStreamType:    1,
				SubscribeAudioUIDs: []string{"#allstream#"},
				SubscribeVideoUIDs: []string{"#allstream#"},
			},
			StorageConfig: storageConfig{
				Vendor:         recordingStorageVendorS3,
				Region:         recordingRegionUS,
				Bucket:         cfg.Minio.BucketName,
				AccessKey:      cfg.Minio.AccessKey,
				SecretKey:      cfg.Minio.Secret,
				FileNamePrefix: []string{"buzz-recordings", buzzID},
				ExtensionParams: ExtensionParams{
					EndPoint: minioEndpoint,
				},
			},
		},
	}

	respData, err := rc.doRequest(http.MethodPost, url, reqBody)
	if err != nil {
		return "", fmt.Errorf("start recording failed: %w", err)
	}

	var resp startResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		return "", fmt.Errorf("failed to parse start response: %w", err)
	}

	return resp.Sid, nil
}

func StopRecording(resourceID, sid, buzzID, uid string) error {
	rc, err := newRecordingClient()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s/cloud_recording/resourceid/%s/sid/%s/mode/%s/stop",
		agoraRecordingBaseURL, rc.appID, resourceID, sid, recordingModeComposite)

	reqBody := map[string]interface{}{
		"cname":         buzzID,
		"uid":           uid,
		"clientRequest": map[string]string{},
	}

	_, err = rc.doRequest(http.MethodPost, url, reqBody)
	return err
}

func QueryRecordingStatus(resourceID, sid, buzzID string) (string, []string, error) {
	rc, err := newRecordingClient()
	if err != nil {
		return "", nil, err
	}

	url := fmt.Sprintf("%s/%s/cloud_recording/resourceid/%s/sid/%s/mode/%s/query",
		agoraRecordingBaseURL, rc.appID, resourceID, sid, recordingModeComposite)

	respData, err := rc.doRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", nil, fmt.Errorf("query recording failed: %w", err)
	}

	var resp queryResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		return "", nil, fmt.Errorf("failed to parse query response: %w", err)
	}

	files := make([]string, 0, len(resp.ServerResponse.FileList))
	for _, f := range resp.ServerResponse.FileList {
		files = append(files, f.Filename)
	}

	statusStr := agoraStatusToString(resp.ServerResponse.Status)
	return statusStr, files, nil
}

func agoraStatusToString(status int) string {
	switch status {
	case 0:
		return "idle"
	case 1:
		return "starting"
	case 2:
		return "recording"
	case 3:
		return "stopping"
	case 4:
		return "stopped"
	default:
		return "unknown"
	}
}
