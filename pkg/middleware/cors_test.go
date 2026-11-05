package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newCORSTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORS())
	router.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return router
}

func corsRequest(router http.Handler, method, origin string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/health", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func TestCORSAllowsExactAndSubdomainOrigins(t *testing.T) {
	router := newCORSTestRouter()

	allowed := []string{
		"https://zedu.chat",
		"https://app.zedu.chat",
		"https://x.preview.groups.zedu.chat",
		"https://ziki.im",
		"https://staging.telex.im",
		"https://telex-auth.vercel.app",
		"https://zeduchat.vercel.app",
		"https://pr-123.zeduchat.vercel.app",
		"http://localhost:3000",
		"http://localhost:5173",
		"http://127.0.0.1:8080",
	}

	for _, origin := range allowed {
		t.Run(origin, func(t *testing.T) {
			response := corsRequest(router, http.MethodGet, origin)

			if got := response.Header().Get("Access-Control-Allow-Origin"); got != origin {
				t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, origin)
			}
			if got := response.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
				t.Fatalf("Access-Control-Allow-Credentials = %q, want true", got)
			}
		})
	}
}

func TestCORSRefusesLookAlikeAndPlainHTTPOrigins(t *testing.T) {
	router := newCORSTestRouter()

	refused := []string{
		"https://evilzedu.chat",
		"https://zedu.chat.evil.com",
		"https://notzedu.chat",
		"http://x.zedu.chat",
		"http://app.zedu.chat",
		"",
	}

	for _, origin := range refused {
		t.Run(origin, func(t *testing.T) {
			response := corsRequest(router, http.MethodGet, origin)

			if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
				t.Fatalf("Access-Control-Allow-Origin = %q, want empty", got)
			}
			if got := response.Header().Get("Access-Control-Allow-Credentials"); got != "" {
				t.Fatalf("Access-Control-Allow-Credentials = %q, want empty", got)
			}
		})
	}
}

func TestCORSSetsVaryForEveryRequest(t *testing.T) {
	router := newCORSTestRouter()

	for _, origin := range []string{"https://app.zedu.chat", "https://evilzedu.chat", ""} {
		response := corsRequest(router, http.MethodGet, origin)

		if got := response.Header().Values("Vary"); len(got) != 1 || got[0] != "Origin" {
			t.Fatalf("Vary = %v, want [Origin] for origin %q", got, origin)
		}
	}
}

func TestCORSDoesNotLeakOriginBetweenRequests(t *testing.T) {
	router := newCORSTestRouter()

	allowed := corsRequest(router, http.MethodGet, "https://app.zedu.chat")
	if got := allowed.Header().Get("Access-Control-Allow-Origin"); got != "https://app.zedu.chat" {
		t.Fatalf("first request origin = %q", got)
	}

	refused := corsRequest(router, http.MethodGet, "https://evilzedu.chat")
	if got := refused.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("second request leaked origin %q", got)
	}

	withoutOrigin := corsRequest(router, http.MethodGet, "")
	if got := withoutOrigin.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("request without Origin leaked origin %q", got)
	}
}

func TestCORSPreflightReturnsNoContent(t *testing.T) {
	router := newCORSTestRouter()

	response := corsRequest(router, http.MethodOptions, "https://app.zedu.chat")

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if got := response.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Fatal("Access-Control-Allow-Methods missing on preflight")
	}
}
