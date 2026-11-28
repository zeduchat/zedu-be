package test_profile

import (
	"bytes"
	"encoding/base64"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hngprojects/telex_be/internal/models"
	"github.com/hngprojects/telex_be/pkg/controller/auth"
	tst "github.com/hngprojects/telex_be/tests"
	"github.com/hngprojects/telex_be/utility"
)

var (
	samplePNG, _ = base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==")
	sampleJPEG   = append([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00}, make([]byte, 32)...)
	sampleGIF    = append([]byte("GIF89a"), make([]byte, 32)...)
	sampleWEBP   = append([]byte("RIFF\x24\x00\x00\x00WEBPVP8 "), make([]byte, 32)...)
	samplePE     = append([]byte("MZ\x90\x00\x03\x00\x00\x00\x04\x00\x00\x00\xff\xff\x00\x00This program cannot be run in DOS mode.\r\n"), make([]byte, 200)...)
)

func dataURL(declared string, data []byte) string {
	return "data:" + declared + ";base64," + base64.StdEncoding.EncodeToString(data)
}

// TestValidatePictureSniffsContent checks that uploaded pictures are typed by
// their bytes, not by the MIME type the client declared (#75).
func TestValidatePictureSniffsContent(t *testing.T) {
	accepted := []struct {
		name string
		in   string
		ext  string
	}{
		{"png", dataURL("image/png", samplePNG), ".png"},
		{"jpeg", dataURL("image/jpeg", sampleJPEG), ".jpeg"},
		{"jpg label", dataURL("image/jpg", sampleJPEG), ".jpeg"},
		{"gif", dataURL("image/gif", sampleGIF), ".gif"},
		{"webp", dataURL("image/webp", sampleWEBP), ".webp"},
		{"real png declared as octet-stream", dataURL("application/octet-stream", samplePNG), ".png"},
	}
	for _, tc := range accepted {
		t.Run("accepts "+tc.name, func(t *testing.T) {
			data, ext, err := utility.ValidatePicture(tc.in)
			require.NoError(t, err)
			assert.Equal(t, tc.ext, ext)
			assert.NotEmpty(t, data)
		})
	}

	rejected := []struct {
		name string
		in   string
	}{
		{"exe declared as png", dataURL("image/png", samplePE)},
		{"exe declared as jpeg", dataURL("image/jpeg", samplePE)},
		{"exe with its own type", dataURL("application/x-msdownload", samplePE)},
		{"html declared as png", dataURL("image/png", []byte("<html><script>alert(1)</script></html>"))},
		{"oversized png", dataURL("image/png", append(append([]byte{}, samplePNG...), make([]byte, 5*1024*1024)...))},
		{"bad base64", "data:image/png;base64,***"},
		{"not base64 encoded", "data:image/png,abc"},
	}
	for _, tc := range rejected {
		t.Run("rejects "+tc.name, func(t *testing.T) {
			data, ext, err := utility.ValidatePicture(tc.in)
			assert.Error(t, err)
			assert.Empty(t, ext)
			assert.Nil(t, data)
		})
	}

	for _, in := range []string{"", "https://cdn.example.com/a.png", "not a data url"} {
		data, ext, err := utility.ValidatePicture(in)
		assert.NoError(t, err)
		assert.Empty(t, ext)
		assert.Nil(t, data)
	}
}

// TestUpdateProfileRejectsNonImageAvatar checks that PATCH /profile rejects a
// non-image avatar file named and labelled as a png before it is stored (#75).
func TestUpdateProfileRejectsNonImageAvatar(t *testing.T) {
	router, profileController := SetupProfileTestRouter()
	db := profileController.Db.Postgresql

	password, _ := utility.HashPassword("password")
	user := models.User{
		ID:       utility.GenerateUUID(),
		Name:     "Avatar Type User",
		Email:    "avatartype" + utility.GenerateUUID() + "@qa.team",
		Password: password,
	}
	require.NoError(t, db.Create(&user).Error)
	prof := models.Profile{ID: utility.GenerateUUID(), Userid: user.ID}
	require.NoError(t, db.Create(&prof).Error)
	t.Cleanup(func() {
		db.Unscoped().Where("id = ?", prof.ID).Delete(&models.Profile{})
		db.Unscoped().Where("id = ?", user.ID).Delete(&models.User{})
	})

	authController := auth.Controller{Db: profileController.Db, Validator: profileController.Validator, Logger: profileController.Logger, ExtReq: profileController.ExtReq}
	token := tst.GetLoginToken(t, router, authController, models.LoginRequestModel{Email: user.Email, Password: "password"})

	// Read the stored column after login, which assigns a default avatar.
	storedAvatar := func() []string {
		var urls []string
		require.NoError(t, db.Model(&models.Profile{}).Where("id = ?", prof.ID).Pluck("avatar_url", &urls).Error)
		return urls
	}
	before := storedAvatar()

	send := func(write func(w *multipart.Writer)) *httptest.ResponseRecorder {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		write(w)
		require.NoError(t, w.Close())
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/profile", &b)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		return rr
	}

	t.Run("exe renamed to png as avatar_file", func(t *testing.T) {
		rr := send(func(w *multipart.Writer) {
			h := make(textproto.MIMEHeader)
			h.Set("Content-Disposition", `form-data; name="avatar_file"; filename="malware.png"`)
			h.Set("Content-Type", "image/png")
			part, err := w.CreatePart(h)
			require.NoError(t, err)
			_, _ = part.Write(samplePE)
		})
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.True(t, strings.Contains(rr.Body.String(), "allowed types are png, jpeg, gif and webp"), rr.Body.String())
	})

	t.Run("exe as png data URL in avatar_url", func(t *testing.T) {
		rr := send(func(w *multipart.Writer) {
			_ = w.WriteField("avatar_url", dataURL("image/png", samplePE))
		})
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	assert.Equal(t, before, storedAvatar(), "the rejected file must not replace the avatar")
}
