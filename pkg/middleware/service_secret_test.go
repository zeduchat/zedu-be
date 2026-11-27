package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequireServiceSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name, configured, sent string
		want                   int
	}{
		{"correct secret allowed", "s3cret-value", "s3cret-value", http.StatusOK},
		{"missing header denied", "s3cret-value", "", http.StatusUnauthorized},
		{"wrong secret denied", "s3cret-value", "s3cret-valuX", http.StatusUnauthorized},
		{"different length denied", "s3cret-value", "s3", http.StatusUnauthorized},
		{"unconfigured secret denies everything", "", "", http.StatusUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/cb", RequireServiceSecret(ServiceSecretHeader, tc.configured), func(c *gin.Context) { c.Status(http.StatusOK) })
			req := httptest.NewRequest(http.MethodPost, "/cb", nil)
			if tc.sent != "" {
				req.Header.Set(ServiceSecretHeader, tc.sent)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("got %d, want %d", w.Code, tc.want)
			}
		})
	}
}
