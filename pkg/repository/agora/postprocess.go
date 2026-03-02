package agora

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/utility"
)

func MergeAndUploadRecording(ctx context.Context, logger *utility.Logger, m3u8ObjectKey string) (string, error) {
	cfg := config.GetConfig()

	endpoint := strings.TrimPrefix(cfg.Minio.MinioEndpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")
	useSSL := cfg.Minio.UseSSL

	scheme := "http"
	if useSSL {
		scheme = "https"
	}
	m3u8URL := fmt.Sprintf("%s://%s/%s/%s", scheme, endpoint, cfg.Minio.BucketName, m3u8ObjectKey)

	logger.Info("[Agora-Record] Starting post-processing for: %s", m3u8ObjectKey)

	logger.Info("[Agora-Record] Fetching playlist from: %s", m3u8URL)
	segmentURLs, err := parseM3U8(ctx, m3u8URL)
	if err != nil {
		logger.Error("[Agora-Record] Failed to parse playlist: %v", err)
		return "", fmt.Errorf("parse m3u8: %w", err)
	}
	if len(segmentURLs) == 0 {
		logger.Error("[Agora-Record] No segments found in playlist: %s", m3u8URL)
		return "", fmt.Errorf("no segments found in playlist: %s", m3u8URL)
	}
	logger.Info("[Agora-Record] Found %d segments to merge", len(segmentURLs))

	logger.Info("[Agora-Record] Merging segments ...")
	merged, err := mergeSegments(ctx, logger, segmentURLs)
	if err != nil {
		logger.Error("[Agora-Record] Failed to merge segments: %v", err)
		return "", fmt.Errorf("merge segments: %w", err)
	}
	logger.Info("[Agora-Record] Merge complete — total size: %d bytes", len(merged))

	mp4Key := strings.TrimSuffix(m3u8ObjectKey, ".m3u8") + ".mp4"

	logger.Info("[Agora-Record] Uploading mp4 to MinIO: %s", mp4Key)
	if err := uploadToMinIO(ctx, logger, endpoint, useSSL, cfg.Minio.BucketName, cfg.Minio.AccessKey, cfg.Minio.Secret, mp4Key, merged); err != nil {
		logger.Error("[Agora-Record] Upload failed: %v", err)
		return "", fmt.Errorf("upload to minio: %w", err)
	}
	logger.Info("[Agora-Record] Upload complete: %s", mp4Key)

	return mp4Key, nil
}

func parseM3U8(ctx context.Context, rawURL string) ([]string, error) {
	base, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d fetching playlist", resp.StatusCode)
	}

	var segments []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		segURL, err := base.Parse(line)
		if err != nil {
			return nil, fmt.Errorf("bad segment URL %q: %w", line, err)
		}
		segments = append(segments, segURL.String())
	}
	return segments, scanner.Err()
}

func mergeSegments(ctx context.Context, logger *utility.Logger, segmentURLs []string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	var buf bytes.Buffer
	total := len(segmentURLs)

	for i, segURL := range segmentURLs {
		logger.Info("[Agora-Record] Downloading segment %d/%d", i+1, total)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, segURL, nil)
		if err != nil {
			return nil, fmt.Errorf("segment %d request: %w", i+1, err)
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("segment %d download: %w", i+1, err)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("segment %d: HTTP %d", i+1, resp.StatusCode)
		}

		if _, err := io.Copy(&buf, resp.Body); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("segment %d read: %w", i+1, err)
		}
		resp.Body.Close()
	}

	return buf.Bytes(), nil
}

func uploadToMinIO(ctx context.Context, logger *utility.Logger, endpoint string, useSSL bool, bucket, accessKey, secretKey, objectKey string, data []byte) error {
	mc, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		logger.Error("[Agora-Record] Failed to create MinIO client: %v", err)
		return fmt.Errorf("minio client: %w", err)
	}

	logger.Info("[Agora-Record] Putting object %s into bucket %s", objectKey, bucket)
	_, err = mc.PutObject(ctx, bucket, objectKey,
		bytes.NewReader(data),
		int64(len(data)),
		minio.PutObjectOptions{ContentType: "video/mp4"},
	)
	if err != nil {
		logger.Error("[Agora-Record] PutObject failed for %s: %v", objectKey, err)
	}
	return err
}
