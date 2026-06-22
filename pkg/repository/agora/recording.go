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
	"github.com/hngprojects/telex_be/utility"
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
	Cname         string               `json:"cname"`
	Uid           string               `json:"uid"`
	ClientRequest acquireClientRequest `json:"clientRequest"`
}

type acquireClientRequest struct {
	Scene               int `json:"scene"`
	ResourceExpiredHour int `json:"resourceExpiredHour"`
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

type extensionServiceParam struct {
	URL              string `json:"url"`
	AudioProfile     int    `json:"audioProfile"`
	VideoWidth       int    `json:"videoWidth"`
	VideoHeight      int    `json:"videoHeight"`
	MaxRecordingHour int    `json:"maxRecordingHour"`
}

type extensionService struct {
	ServiceName       string                `json:"serviceName"`
	ErrorHandlePolicy string                `json:"errorHandlePolicy"`
	ServiceParam      extensionServiceParam `json:"serviceParam"`
}

type extensionServiceConfig struct {
	ErrorHandlePolicy string             `json:"errorHandlePolicy"`
	ExtensionServices []extensionService `json:"extensionServices"`
}

type transcodingConfig struct {
	Width            int `json:"width"`
	Height           int `json:"height"`
	Fps              int `json:"fps"`
	Bitrate          int `json:"bitrate"`
	MixedVideoLayout int `json:"mixedVideoLayout"`
}

type recordingConfig struct {
	MaxIdleTime       int               `json:"maxIdleTime"`
	StreamTypes       int               `json:"streamTypes"`
	ChannelType       int               `json:"channelType"`
	VideoStreamType   int               `json:"videoStreamType"`
	SubscribeUidGroup int               `json:"subscribeUidGroup"`
	TranscodingConfig transcodingConfig `json:"transcodingConfig"`
}

type recordingFileConfig struct {
	AvFileType []string `json:"avFileType"`
}

type startRecordingRequest struct {
	Cname         string             `json:"cname"`
	Uid           string             `json:"uid"`
	ClientRequest startClientRequest `json:"clientRequest"`
}

type startClientRequest struct {
	RecordingConfig        *recordingConfig        `json:"recordingConfig,omitempty"`
	ExtensionServiceConfig extensionServiceConfig `json:"extensionServiceConfig"`
	RecordingFileConfig    recordingFileConfig    `json:"recordingFileConfig"`
	StorageConfig          storageConfig          `json:"storageConfig"`
}

type startResponse struct {
	Sid        string `json:"sid"`
	ResourceId string `json:"resourceId"`
}

type stopServerResponse struct {
	FileList        json.RawMessage `json:"fileList"`
	FileListMode    string          `json:"fileListMode"`
	UploadingStatus string          `json:"uploadingStatus"`
}

type stopResponse struct {
	Cname          string             `json:"cname"`
	ResourceId     string             `json:"resourceId"`
	Sid            string             `json:"sid"`
	Uid            string             `json:"uid"`
	ServerResponse stopServerResponse `json:"serverResponse"`
}

type queryServerResponse struct {
	Status          int             `json:"status"`
	FileList        json.RawMessage `json:"fileList"`
	FileListMode    string          `json:"fileListMode"`
	UploadingStatus string          `json:"uploadingStatus"`
}

type queryResponse struct {
	ResourceId     string              `json:"resourceId"`
	Sid            string              `json:"sid"`
	ServerResponse queryServerResponse `json:"serverResponse"`
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

func AcquireRecording(logger *utility.Logger, buzzID string, uid string) (string, error) {
	rc, err := newRecordingClient()
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/%s/cloud_recording/acquire", agoraRecordingBaseURL, rc.appID)
	reqBody := acquireRequest{
		Cname: buzzID,
		Uid:   uid,
		ClientRequest: acquireClientRequest{
			Scene:               1,
			ResourceExpiredHour: 3,
		},
	}

	respData, err := rc.doRequest(http.MethodPost, url, reqBody)
	if err != nil {
		logger.Error("[Agora] Failed to acquire recording for buzz %s: %v", buzzID, err)
		return "", fmt.Errorf("acquire recording failed: %w", err)
	}

	logger.Info("[Agora] Acquired recording for buzz %s", buzzID)

	var resp acquireResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		logger.Error("[Agora] Failed to parse acquire response for buzz %s: %v", buzzID, err)
		return "", fmt.Errorf("failed to parse acquire response: %w", err)
	}

	return resp.ResourceId, nil
}

func StartRecording(logger *utility.Logger, resourceID, buzzID, webpageURL, uid string, maxIdleSecs int) (string, error) {
	rc, err := newRecordingClient()
	if err != nil {
		logger.Error("[Agora] Failed to create recording client: %v", err)
		return "", err
	}

	cfg := config.GetConfig()
	url := fmt.Sprintf("%s/%s/cloud_recording/resourceid/%s/mode/web/start",
		agoraRecordingBaseURL, rc.appID, resourceID)

	minioEndpoint := strings.TrimPrefix(cfg.Minio.MinioEndpoint, "https://")
	minioEndpoint = strings.TrimPrefix(minioEndpoint, "http://")

	reqBody := startRecordingRequest{
		Cname: buzzID,
		Uid:   uid,
		ClientRequest: startClientRequest{
			RecordingConfig: &recordingConfig{
				MaxIdleTime:       maxIdleSecs,
				StreamTypes:       2,
				ChannelType:       0,
				VideoStreamType:   0,
				SubscribeUidGroup: 0,
				TranscodingConfig: transcodingConfig{
					Width:            1280,
					Height:           720,
					Fps:              30,
					Bitrate:          2000,
					MixedVideoLayout: 1,
				},
			},
			ExtensionServiceConfig: extensionServiceConfig{
				ErrorHandlePolicy: "error_abort",
				ExtensionServices: []extensionService{
					{
						ServiceName:       "web_recorder_service",
						ErrorHandlePolicy: "error_abort",
						ServiceParam: extensionServiceParam{
							// URL:              webpageURL,
							URL: "https://www.youtube.com/watch?v=KJkcH0J_TO4",
							AudioProfile:     0,
							VideoWidth:       1280,
							VideoHeight:      720,
							MaxRecordingHour: 3,
						},
					},
				},
			},
			RecordingFileConfig: recordingFileConfig{
				AvFileType: []string{"hls", "mp4"},
			},
			StorageConfig: storageConfig{
				Vendor:         recordingStorageVendorS3,
				Region:         recordingRegionUS,
				Bucket:         cfg.Minio.BucketName,
				AccessKey:      cfg.Minio.AccessKey,
				SecretKey:      cfg.Minio.Secret,
				FileNamePrefix: []string{"call-recordings", buzzID},
				ExtensionParams: ExtensionParams{
					EndPoint: minioEndpoint,
				},
			},
		},
	}

	respData, err := rc.doRequest(http.MethodPost, url, reqBody)
	if err != nil {
		logger.Error("[Agora] Failed to start recording for buzz %s: %v", buzzID, err)
		return "", fmt.Errorf("start recording failed: %w", err)
	}

	logger.Info("[Agora] Started recording for buzz %s", buzzID)

	var resp startResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		logger.Error("[Agora] Failed to parse start response for buzz %s: %v", buzzID, err)
		return "", fmt.Errorf("failed to parse start response: %w", err)
	}

	return resp.Sid, nil
}

func StopRecording(resourceID, sid, buzzID, uid string) ([]string, error) {
	rc, err := newRecordingClient()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/%s/cloud_recording/resourceid/%s/sid/%s/mode/web/stop",
		agoraRecordingBaseURL, rc.appID, resourceID, sid)

	reqBody := map[string]interface{}{
		"cname": buzzID,
		"uid":   uid,
		"clientRequest": map[string]interface{}{},
	}

	respData, err := rc.doRequest(http.MethodPost, url, reqBody)
	if err != nil {
		return nil, err
	}

	var resp stopResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse stop response: %w", err)
	}

	return parseFileList(resp.ServerResponse.FileList), nil
}

func QueryRecordingStatus(logger *utility.Logger, resourceID, sid, buzzID string) (string, []string, error) {
	rc, err := newRecordingClient()
	if err != nil {
		return "", nil, err
	}

	url := fmt.Sprintf("%s/%s/cloud_recording/resourceid/%s/sid/%s/mode/web/query",
		agoraRecordingBaseURL, rc.appID, resourceID, sid)

	respData, err := rc.doRequest(http.MethodGet, url, nil)
	if err != nil {
		logger.Error("[Agora] Failed to query recording for buzz %s: %v", buzzID, err)
		return "", nil, fmt.Errorf("query recording failed: %w", err)
	}

	var resp queryResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		return "", nil, fmt.Errorf("failed to parse query response: %w", err)
	}

	files := parseFileList(resp.ServerResponse.FileList)
	statusStr := agoraStatusToString(resp.ServerResponse.Status)
	return statusStr, files, nil
}

func agoraStatusToString(status int) string {
	switch status {
	case 0:
		return "idle"
	case 1:
		return "failed"
	case 2:
		return "exited"
	case 3:
		return "joining"
	case 4:
		return "recording"
	case 5:
		return "files_generated"
	case 6:
		return "uploaded"
	case 7:
		return "partly_uploaded"
	default:
		return "unknown"
	}
}

type fileDetail struct {
	FileName  string `json:"fileName"`
	TrackType string `json:"trackType"`
}

func parseFileList(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}

	// Try parsing as array of objects
	var details []fileDetail
	if err := json.Unmarshal(raw, &details); err == nil {
		var filenames []string
		for _, d := range details {
			if d.FileName != "" {
				filenames = append(filenames, d.FileName)
			}
		}
		if len(filenames) > 0 {
			return filenames
		}
	}

	// Try parsing as single string
	var singleStr string
	if err := json.Unmarshal(raw, &singleStr); err == nil {
		if singleStr != "" {
			return []string{singleStr}
		}
	}

	// Try parsing as array of strings
	var strArr []string
	if err := json.Unmarshal(raw, &strArr); err == nil {
		return strArr
	}

	return nil
}


