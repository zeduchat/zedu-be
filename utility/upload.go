package utility

import (
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

// allowedPictureTypes maps the image types accepted for uploaded pictures,
// as sniffed from the file content, to the extension they are stored with.
var allowedPictureTypes = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpeg",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// ValidatePicture decodes a base64 data URL and checks that its content is a
// png, jpeg, gif or webp image of at most 5MB. The type is sniffed from the
// decoded bytes, never taken from the data URL's declared MIME type, so a
// renamed or relabelled non-image is rejected. Empty values, remote URLs and
// other non-data-URL strings return no file and no error.
func ValidatePicture(base64Image string) ([]byte, string, error) {
	const maxImageSize = 5 * 1024 * 1024 // 5MB

	if base64Image == "" {
		return nil, "", nil
	}

	if unsupportedURLPrefix(base64Image) {
		return nil, "", nil
	}

	if !strings.HasPrefix(base64Image, "data:") {
		return nil, "", nil
	}

	parts := strings.SplitN(base64Image, ",", 2)
	if len(parts) < 2 || !strings.HasSuffix(parts[0], ";base64") {
		return nil, "", fmt.Errorf("invalid data URL")
	}

	if len(parts[1]) > base64.StdEncoding.EncodedLen(maxImageSize) {
		return nil, "", fmt.Errorf("image size exceeds 5MB limit")
	}

	imageData, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode base64 string: %w", err)
	}

	if len(imageData) > maxImageSize {
		return nil, "", fmt.Errorf("image size exceeds 5MB limit")
	}

	mimeType := http.DetectContentType(imageData)
	ext, ok := allowedPictureTypes[mimeType]
	if !ok {
		return nil, "", fmt.Errorf("invalid image type %s: allowed types are png, jpeg, gif and webp", mimeType)
	}

	return imageData, ext, nil
}

func unsupportedURLPrefix(url string) bool {
	supportedPrefixes := []string{"http://", "https://", "blob:", "ipfs://", "ftp:"}
	for _, prefix := range supportedPrefixes {
		if strings.HasPrefix(url, prefix) {
			return true
		}
	}
	return false
}

func ConvertToBase64(header *multipart.FileHeader) (string, error) {
	openedFile, err := header.Open()
	if err != nil {
		return "", fmt.Errorf("Could not open uploaded image: %v", err)
	}
	defer openedFile.Close()

	imageBytes, err := io.ReadAll(openedFile)
	if err != nil {
		return "", fmt.Errorf("could not read image data: %v", err)
	}

	base64Str := base64.StdEncoding.EncodeToString(imageBytes)
	mimeType := header.Header.Get("Content-Type")

	base64Image := "data:" + mimeType + ";base64," + base64Str
	return base64Image, nil
}
