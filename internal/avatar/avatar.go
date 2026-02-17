package avatar

import (
	"fmt"
	"hash/crc32"

	"github.com/hngprojects/telex_be/internal/config"
)

// GenerateDefaultAvatarURL returns a deterministic default avatar URL for a userID.
// This uses an external deterministic avatar provider (DiceBear) so all services
// can independently compute the same URL without a DB lookup.
func GenerateDefaultAvatarURL(userID string) string {
	// Calculate CRC32 checksum of the userID
	checksum := crc32.ChecksumIEEE([]byte(userID))
	// Map to 1-7 range
	n := (checksum % 7) + 1
	conf := config.GetConfig()
	endpoint := ""
	bucketName := "telexstagingbucket"

	if conf != nil {
		endpoint = conf.Minio.MinioEndpoint
		if conf.App.Mode == "prod" {
			bucketName = "telexprodbucket"
		}
	}
	return fmt.Sprintf("https://%s/%s/public/default_avatars/default_avatar_%d.png", endpoint, bucketName, n)
}
