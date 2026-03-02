package agora

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	gomp4 "github.com/yapingcat/gomedia/go-mp4"
	gompeg2 "github.com/yapingcat/gomedia/go-mpeg2"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/hngprojects/telex_be/internal/config"
	"github.com/hngprojects/telex_be/utility"
)

func MergeAndUploadRecording(ctx context.Context, logger *utility.Logger, m3u8ObjectKey string) (string, int64, error) {
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
		return "", 0, fmt.Errorf("parse m3u8: %w", err)
	}
	if len(segmentURLs) == 0 {
		logger.Error("[Agora-Record] No segments found in playlist: %s", m3u8URL)
		return "", 0, fmt.Errorf("no segments found in playlist: %s", m3u8URL)
	}
	logger.Info("[Agora-Record] Found %d segments to merge", len(segmentURLs))

	logger.Info("[Agora-Record] Merging segments ...")
	tsData, err := mergeSegments(ctx, logger, segmentURLs)
	if err != nil {
		logger.Error("[Agora-Record] Failed to merge segments: %v", err)
		return "", 0, fmt.Errorf("merge segments: %w", err)
	}
	logger.Info("[Agora-Record] Merge complete — raw TS size: %d bytes", len(tsData))

	logger.Info("[Agora-Record] Remuxing TS to MP4 ...")
	mp4Data, err := remuxTStoMP4(tsData)
	if err != nil {
		logger.Error("[Agora-Record] Failed to remux TS to MP4: %v", err)
		return "", 0, fmt.Errorf("remux ts to mp4: %w", err)
	}
	logger.Info("[Agora-Record] Remux complete — MP4 size: %d bytes", len(mp4Data))

	mp4Key := strings.TrimSuffix(m3u8ObjectKey, ".m3u8") + ".mp4"

	logger.Info("[Agora-Record] Uploading mp4 to MinIO: %s", mp4Key)
	uploadedSize, err := uploadToMinIO(ctx, logger, endpoint, useSSL, cfg.Minio.BucketName, cfg.Minio.AccessKey, cfg.Minio.Secret, mp4Key, mp4Data)
	if err != nil {
		logger.Error("[Agora-Record] Upload failed: %v", err)
		return "", 0, fmt.Errorf("upload to minio: %w", err)
	}
	logger.Info("[Agora-Record] Upload complete: %s (%d bytes)", mp4Key, uploadedSize)

	return mp4Key, uploadedSize, nil
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

func remuxTStoMP4(tsData []byte) ([]byte, error) {
	tmp, err := os.CreateTemp("", "agora-remux-*.mp4")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	muxer, err := gomp4.CreateMp4Muxer(tmp)
	if err != nil {
		return nil, fmt.Errorf("create mp4 muxer: %w", err)
	}

	videoTrack := uint32(0)
	audioTrack := uint32(0)

	demuxer := gompeg2.NewTSDemuxer()

	demuxer.OnFrame = func(cid gompeg2.TS_STREAM_TYPE, frame []byte, pts, dts uint64) {
		switch cid {
		case gompeg2.TS_STREAM_H264:
			if videoTrack == 0 {
				videoTrack = muxer.AddVideoTrack(gomp4.MP4_CODEC_H264)
			}
			if err := muxer.Write(videoTrack, frame, pts, dts); err != nil {
				fmt.Printf("warn: write video frame: %v\n", err)
			}

		case gompeg2.TS_STREAM_H265:
			if videoTrack == 0 {
				videoTrack = muxer.AddVideoTrack(gomp4.MP4_CODEC_H265)
			}
			if err := muxer.Write(videoTrack, frame, pts, dts); err != nil {
				fmt.Printf("warn: write video frame: %v\n", err)
			}

		case gompeg2.TS_STREAM_AAC:
			if audioTrack == 0 {
				audioTrack = muxer.AddAudioTrack(gomp4.MP4_CODEC_AAC)
			}
			if err := muxer.Write(audioTrack, frame, pts, dts); err != nil {
				fmt.Printf("warn: write audio frame: %v\n", err)
			}

		case gompeg2.TS_STREAM_AUDIO_MPEG1, gompeg2.TS_STREAM_AUDIO_MPEG2:
			if audioTrack == 0 {
				audioTrack = muxer.AddAudioTrack(gomp4.MP4_CODEC_MP3)
			}
			if err := muxer.Write(audioTrack, frame, pts, dts); err != nil {
				fmt.Printf("warn: write audio frame: %v\n", err)
			}
		}
	}

	if err := demuxer.Input(bytes.NewReader(tsData)); err != nil && err != io.EOF {
		return nil, fmt.Errorf("ts demux: %w", err)
	}

	if err := muxer.WriteTrailer(); err != nil {
		return nil, fmt.Errorf("write mp4 trailer: %w", err)
	}

	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek temp file: %w", err)
	}

	mp4Data, err := io.ReadAll(tmp)
	if err != nil {
		return nil, fmt.Errorf("read temp file: %w", err)
	}

	return mp4Data, nil
}

func uploadToMinIO(ctx context.Context, logger *utility.Logger, endpoint string, useSSL bool, bucket, accessKey, secretKey, objectKey string, data []byte) (int64, error) {
	mc, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		logger.Error("[Agora-Record] Failed to create MinIO client: %v", err)
		return 0, fmt.Errorf("minio client: %w", err)
	}

	logger.Info("[Agora-Record] Putting object %s into bucket %s", objectKey, bucket)
	info, err := mc.PutObject(ctx, bucket, objectKey,
		bytes.NewReader(data),
		int64(len(data)),
		minio.PutObjectOptions{ContentType: "video/mp4"},
	)
	if err != nil {
		logger.Error("[Agora-Record] PutObject failed for %s: %v", objectKey, err)
		return 0, err
	}
	return info.Size, nil
}
