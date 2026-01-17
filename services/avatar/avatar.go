package avatar

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	minioStorage "github.com/hngprojects/telex_be/pkg/repository/storage/minio"
	"github.com/hngprojects/telex_be/utility"
)

const (
	maxAvatarSize = 5 << 20 // 5MB
)

var allowedImageTypes = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/jpg":  true,
	"image/gif":  true,
	"image/webp": true,
}

func UploadAvatar(logger *utility.Logger, file multipart.File, header *multipart.FileHeader) (string, error) {
	if header.Size > maxAvatarSize {
		return "", fmt.Errorf("file size exceeds maximum allowed size of 5MB")
	}
	buffer := make([]byte, 512)
	_, err := file.Read(buffer)
	if err != nil {
		return "", fmt.Errorf("failed to read file header: %w", err)
	}
	mimeType := detectMimeType(buffer)

	if !allowedImageTypes[mimeType] {
		return "", fmt.Errorf("invalid file type: %s. Allowed types: png, jpg, jpeg, gif, webp", mimeType)
	}
	if seeker, ok := file.(io.Seeker); ok {
		_, err := seeker.Seek(0, io.SeekStart)
		if err != nil {
			return "", fmt.Errorf("failed to reset file pointer: %w", err)
		}
	}

	fileHash, err := hashFile(file)
	if err != nil {
		return "", fmt.Errorf("failed to hash file: %w", err)
	}
	if seeker, ok := file.(io.Seeker); ok {
		_, err := seeker.Seek(0, io.SeekStart)
		if err != nil {
			return "", fmt.Errorf("failed to reset file pointer: %w", err)
		}
	}

	extension := strings.ToLower(filepath.Ext(header.Filename))
	if extension == "" {
		extension = ".png" // default extension
	}
	objectName := fileHash + extension

	return minioStorage.UploadAvatar(logger, objectName, file, header.Size, mimeType)
}

func ListAvatars(logger *utility.Logger) ([]minioStorage.AvatarInfo, error) {
	return minioStorage.ListAvatars(logger)
}

func detectMimeType(buffer []byte) string {
	if len(buffer) >= 8 {
		if buffer[0] == 0x89 && buffer[1] == 0x50 && buffer[2] == 0x4E && buffer[3] == 0x47 {
			return "image/png"
		}
		if buffer[0] == 0xFF && buffer[1] == 0xD8 && buffer[2] == 0xFF {
			return "image/jpeg"
		}
		if buffer[0] == 0x47 && buffer[1] == 0x49 && buffer[2] == 0x46 {
			return "image/gif"
		}
		if len(buffer) >= 12 && buffer[0] == 0x52 && buffer[1] == 0x49 && buffer[2] == 0x46 && buffer[3] == 0x46 &&
			buffer[8] == 0x57 && buffer[9] == 0x45 && buffer[10] == 0x42 && buffer[11] == 0x50 {
			return "image/webp"
		}
	}
	return "application/octet-stream"
}

func hashFile(file multipart.File) (string, error) {
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
