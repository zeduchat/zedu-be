package utility

import (
	crand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/url"
	"regexp"
	"time"

	"github.com/gofrs/uuid"
)

var table = [...]byte{'1', '2', '3', '4', '5', '6', '7', '8', '9', '0'}

func GetRandomNumbersInRange(min, max int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max-min) + min
}

func RandomString(length int) string {
	u, _ := uuid.NewV4()
	uuidStr := u.String()
	// Regular expression pattern to match all non-alphanumeric characters
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		return ""
	}
	// Replacing all non-alphanumeric characters with an empty string
	processedString := reg.ReplaceAllString(uuidStr+uuidStr[:length%36], "")
	if len(processedString) >= length {
		return processedString[:length]
	}
	// Padding the processed string with random alphanumeric characters to make it the desired length
	alphanumeric := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	padding := make([]byte, length-len(processedString))
	rand.Read(padding)
	for i, b := range padding {
		padding[i] = alphanumeric[b%byte(len(alphanumeric))]
	}
	return processedString + string(padding)
}

func GenerateOTP(length int) (string, error) {
	b := make([]byte, length)
	if _, err := io.ReadFull(crand.Reader, b); err != nil {
		return "", err
	}
	for i := 0; i < len(b); i++ {
		b[i] = table[int(b[i])%len(table)]
	}
	return string(b), nil
}

func GenerateInvitationToken() (string, error) {
	bytes := make([]byte, 6)
	_, err := crand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func GenerateToken(seed string) (string, error) {
	// Generate a UUID based on the seed
	namespace := uuid.NamespaceURL
	id := uuid.NewV5(namespace, seed)
	return id.String(), nil
}

func GenerateInvitationLink(baseurl, orgID, token string) string {
	return baseurl + fmt.Sprintf("accept_org_invitation?org_id=%s&invitation_token=%s", orgID, token)
}

func GenerateGeneralInvitationLink(baseurl, orgID, token string) string {
	return baseurl + fmt.Sprintf("accept_general_invitation?org_id=%s&invitation_token=%s", orgID, token)
}

func GenerateUUIDFromString(strSeed string) (string, error) {
	u, err := url.Parse(strSeed)
	if err != nil {
		return "", err
	}

	url_path := u.Hostname() + u.Path

	namespace := uuid.NamespaceURL
	id := uuid.NewV5(namespace, url_path)

	return id.String(), nil
}

func GenerateUUIDFromSeed(strSeed string) string {

	namespace := uuid.NamespaceURL
	id := uuid.NewV5(namespace, strSeed)

	return id.String()
}


func GenerateUserColor(userID, username string) string {
	// Use userID if available, otherwise use username
	input := userID
	if input == "" {
		input = username
	}
	if input == "" {
		// Fallback to a default color if both are empty
		return "#8E8E93"
	}

	// Create SHA256 hash of the input
	hash := sha256.Sum256([]byte(input))

	// Convert first 4 bytes of hash to a uint32 for hue calculation
	hueValue := uint32(hash[0])<<24 | uint32(hash[1])<<16 | uint32(hash[2])<<8 | uint32(hash[3])

	// Map to hue range 0-360
	hue := float64(hueValue % 360)

	// Fixed saturation and lightness for consistent visual quality
	saturation := 70.0  // 70%
	lightness := 55.0   // 55%

	// Convert HSL to hex
	return hslToHex(hue, saturation, lightness)
}

// hslToHex converts HSL color values to hex format
// hue: 0-360, saturation: 0-100, lightness: 0-100
// returns: hex color string in format "#RRGGBB"
func hslToHex(hue, saturation, lightness float64) string {
	// Normalize values
	h := hue / 60.0
	s := saturation / 100.0
	l := lightness / 100.0

	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h, 2)-1))
	m := l - c/2

	var r, g, b float64

	switch {
	case h < 1:
		r, g, b = c, x, 0
	case h < 2:
		r, g, b = x, c, 0
	case h < 3:
		r, g, b = 0, c, x
	case h < 4:
		r, g, b = 0, x, c
	case h < 5:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}

	// Convert to RGB (0-255)
	red := uint8((r + m) * 255)
	green := uint8((g + m) * 255)
	blue := uint8((b + m) * 255)

	return fmt.Sprintf("#%02X%02X%02X", red, green, blue)
}
