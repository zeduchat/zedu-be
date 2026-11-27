package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hngprojects/telex_be/utility"
)

// ServiceSecretHeader carries the shared secret on service-to-service callbacks.
const ServiceSecretHeader = "X-Service-Secret"

// RequireServiceSecret aborts with 401 unless the request carries `header`
// equal to `secret`. Both values are hashed before a constant-time compare,
// so neither the content nor the length leaks through timing. It fails
// closed: an empty configured secret rejects every request.
func RequireServiceSecret(header string, secret string) gin.HandlerFunc {
	expected := sha256.Sum256([]byte(secret))
	configured := secret != ""

	return func(c *gin.Context) {
		got := sha256.Sum256([]byte(c.GetHeader(header)))
		if !configured || subtle.ConstantTimeCompare(got[:], expected[:]) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized,
				utility.BuildErrorResponse(http.StatusUnauthorized, "error", "invalid service credentials", "Unauthorized", nil))
			return
		}
		c.Next()
	}
}
