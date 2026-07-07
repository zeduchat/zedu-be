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
	"github.com/hngprojects/telex_be/internal/models"
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
	FileNamePrefix  []string        `json:"fileNamePrefix,omitempty"`
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
	Width            int                       `json:"width"`
	Height           int                       `json:"height"`
	Fps              int                       `json:"fps"`
	Bitrate          int                       `json:"bitrate"`
	MixedVideoLayout int                       `json:"mixedVideoLayout"`
	LayoutConfig     []models.LayoutConfigItem `json:"layoutConfig,omitempty"`
}

type recordingConfig struct {
	ChannelType        int               `json:"channelType"`
	StreamTypes        int               `json:"streamTypes"`
	AudioProfile       int               `json:"audioProfile"`
	VideoStreamType    int               `json:"videoStreamType"`
	MaxIdleTime        int               `json:"maxIdleTime"`
	SubscribeAudioUids []string          `json:"subscribeAudioUids"`
	SubscribeVideoUids []string          `json:"subscribeVideoUids"`
	TranscodingConfig  transcodingConfig `json:"transcodingConfig"`
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
	Token                  string                  `json:"token,omitempty"`
	RecordingConfig        *recordingConfig        `json:"recordingConfig,omitempty"`
	ExtensionServiceConfig *extensionServiceConfig `json:"extensionServiceConfig,omitempty"`
	RecordingFileConfig    recordingFileConfig    `json:"recordingFileConfig"`
	StorageConfig          storageConfig          `json:"storageConfig"`
}

type startResponse struct {
	Sid        string `json:"sid"`
	ResourceId string `json:"resourceId"`
}

type extensionServicePayloadFile struct {
	Filename       string `json:"filename"`
	SliceStartTime int64  `json:"sliceStartTime"`
}

type extensionServicePayload struct {
	FileList        []extensionServicePayloadFile `json:"fileList"`
	UploadingStatus string                        `json:"uploadingStatus"`
	State           string                        `json:"state"`
}

type extensionServiceState struct {
	ServiceName string                  `json:"serviceName"`
	Payload     extensionServicePayload `json:"payload"`
}

type stopServerResponse struct {
	FileList              json.RawMessage         `json:"fileList"`
	FileListMode          string                  `json:"fileListMode"`
	UploadingStatus       string                  `json:"uploadingStatus"`
	ExtensionServiceState []extensionServiceState `json:"extensionServiceState"`
}

type stopResponse struct {
	Cname          string             `json:"cname"`
	ResourceId     string             `json:"resourceId"`
	Sid            string             `json:"sid"`
	Uid            string             `json:"uid"`
	ServerResponse stopServerResponse `json:"serverResponse"`
}

type queryServerResponse struct {
	Status                int                     `json:"status"`
	FileList              json.RawMessage         `json:"fileList"`
	FileListMode          string                  `json:"fileListMode"`
	UploadingStatus       string                  `json:"uploadingStatus"`
	ExtensionServiceState []extensionServiceState `json:"extensionServiceState"`
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
			Scene:               0,
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

func StartRecording(logger *utility.Logger, resourceID, buzzID, rtcToken, uid string, maxIdleSecs int) (string, error) {
	rc, err := newRecordingClient()
	if err != nil {
		logger.Error("[Agora] Failed to create recording client: %v", err)
		return "", err
	}

	cfg := config.GetConfig()
	url := fmt.Sprintf("%s/%s/cloud_recording/resourceid/%s/mode/mix/start",
		agoraRecordingBaseURL, rc.appID, resourceID)

	minioEndpoint := strings.TrimPrefix(cfg.Minio.MinioEndpoint, "https://")
	minioEndpoint = strings.TrimPrefix(minioEndpoint, "http://")

	reqBody := startRecordingRequest{
		Cname: buzzID,
		Uid:   uid,
		ClientRequest: startClientRequest{
			Token: rtcToken,
			RecordingConfig: &recordingConfig{
				ChannelType:        0,
				StreamTypes:        2,
				AudioProfile:       1,
				VideoStreamType:    0,
				MaxIdleTime:        maxIdleSecs,
				SubscribeAudioUids: []string{"#allstream#"},
				SubscribeVideoUids: []string{"#allstream#"},
				TranscodingConfig: transcodingConfig{
					Width:            1920,
					Height:           1080,
					Fps:              30,
					Bitrate:          4000,
					MixedVideoLayout: 1,
				},
			},
			ExtensionServiceConfig: nil,
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

func StopRecording(logger *utility.Logger, resourceID, sid, buzzID, uid string) ([]string, error) {
	rc, err := newRecordingClient()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/%s/cloud_recording/resourceid/%s/sid/%s/mode/mix/stop",
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

	logger.Info("[Agora-Stop-Response] raw JSON: %s", string(respData))

	var resp stopResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse stop response: %w", err)
	}

	files := parseFileList(resp.ServerResponse.FileList)
	if len(files) == 0 {
		files = parseFileListFromStates(resp.ServerResponse.ExtensionServiceState)
	}

	logger.Info("[Agora-Stop-Response] parsed files: %v", files)

	return files, nil
}

func QueryRecordingStatus(logger *utility.Logger, resourceID, sid, buzzID string) (string, []string, error) {
	rc, err := newRecordingClient()
	if err != nil {
		return "", nil, err
	}

	url := fmt.Sprintf("%s/%s/cloud_recording/resourceid/%s/sid/%s/mode/mix/query",
		agoraRecordingBaseURL, rc.appID, resourceID, sid)

	respData, err := rc.doRequest(http.MethodGet, url, nil)
	if err != nil {
		logger.Error("[Agora] Failed to query recording for buzz %s: %v", buzzID, err)
		return "", nil, fmt.Errorf("query recording failed: %w", err)
	}

	logger.Info("[Agora-Query-Response] raw JSON: %s", string(respData))

	var resp queryResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		return "", nil, fmt.Errorf("failed to parse query response: %w", err)
	}

	files := parseFileList(resp.ServerResponse.FileList)
	if len(files) == 0 {
		files = parseFileListFromStates(resp.ServerResponse.ExtensionServiceState)
	}
	statusStr := agoraStatusToString(resp.ServerResponse.Status)
	logger.Info("[Agora-Query-Response] status: %s, parsed files: %v", statusStr, files)
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

func parseFileListFromStates(states []extensionServiceState) []string {
	var filenames []string
	for _, state := range states {
		if state.ServiceName == "web_recorder_service" {
			for _, file := range state.Payload.FileList {
				if file.Filename != "" {
					filenames = append(filenames, file.Filename)
				}
			}
		}
	}
	return filenames
}

type updateLayoutClientRequest struct {
	MixedVideoLayout int                       `json:"mixedVideoLayout"`
	LayoutConfig     []models.LayoutConfigItem `json:"layoutConfig,omitempty"`
}

type updateLayoutRequest struct {
	Cname         string                    `json:"cname"`
	Uid           string                    `json:"uid"`
	ClientRequest updateLayoutClientRequest `json:"clientRequest"`
}

func UpdateLayout(logger *utility.Logger, resourceID, sid, buzzID, uid string, layoutConfig []models.LayoutConfigItem) error {
	rc, err := newRecordingClient()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s/cloud_recording/resourceid/%s/sid/%s/mode/mix/updateLayout",
		agoraRecordingBaseURL, rc.appID, resourceID, sid)

	mixedVideoLayout := 1
	if len(layoutConfig) == 0 {
		mixedVideoLayout = 1
	}

	reqBody := updateLayoutRequest{
		Cname: buzzID,
		Uid:   uid,
		ClientRequest: updateLayoutClientRequest{
			MixedVideoLayout: mixedVideoLayout,
			LayoutConfig:     layoutConfig,
		},
	}

	respData, err := rc.doRequest(http.MethodPost, url, reqBody)
	if err != nil {
		logger.Error("[Agora] Failed to update layout for buzz %s: %v", buzzID, err)
		return fmt.Errorf("update layout failed: %w", err)
	}

	logger.Info("[Agora] Layout updated for buzz %s. Response: %s", buzzID, string(respData))
	return nil
}


