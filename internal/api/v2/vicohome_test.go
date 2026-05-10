// vicohome_test.go: tests for the /api/v2/vicohome/* endpoints.
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubVicoHomeProvider is a test double for VicoHomeImageProvider.
type stubVicoHomeProvider struct {
	images map[string]string
}

func (s *stubVicoHomeProvider) GetCameraImages() map[string]string {
	return s.images
}

func setupVicoHomeTestEnv(t *testing.T) (*echo.Echo, *Controller) {
	t.Helper()
	e := echo.New()
	return e, &Controller{Group: e.Group("/api/v2")}
}

// When no provider is wired, the endpoint must still respond 200 with an
// empty JSON object so the kiosk view degrades gracefully (e.g., before the
// poller has finished its first cycle, or when VicoHome is disabled).
func TestGetVicoHomeImages_NoProvider(t *testing.T) {
	t.Attr("component", "vicohome")
	t.Attr("type", "integration")

	e, controller := setupVicoHomeTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/vicohome/images", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	require.NoError(t, controller.GetVicoHomeImages(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

	var got map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Empty(t, got)
}

// When the wired provider returns image URLs, the endpoint serves them as a
// JSON object keyed by lowercase species name (the contract Kiosk.svelte
// expects: `cameraImages[detection.commonName.toLowerCase()]`).
func TestGetVicoHomeImages_WithImages(t *testing.T) {
	t.Attr("component", "vicohome")
	t.Attr("type", "integration")

	e, controller := setupVicoHomeTestEnv(t)
	controller.SetVicoHomeImageProvider(&stubVicoHomeProvider{
		images: map[string]string{
			"northern cardinal": "https://cdn.vicohome.io/snap1.jpg",
			"blue jay":          "https://cdn.vicohome.io/snap2.jpg",
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/vicohome/images", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	require.NoError(t, controller.GetVicoHomeImages(c))
	assert.Equal(t, http.StatusOK, rec.Code)

	var got map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "https://cdn.vicohome.io/snap1.jpg", got["northern cardinal"])
	assert.Equal(t, "https://cdn.vicohome.io/snap2.jpg", got["blue jay"])
}

// A provider that returns an empty map must serialize as "{}", not "null".
// The kiosk frontend uses `cameraImages[name] ?? ''` — null would be tolerated
// but `{}` is the contract and avoids surprises in other consumers.
func TestGetVicoHomeImages_EmptyMap(t *testing.T) {
	t.Attr("component", "vicohome")
	t.Attr("type", "integration")

	e, controller := setupVicoHomeTestEnv(t)
	controller.SetVicoHomeImageProvider(&stubVicoHomeProvider{
		images: map[string]string{},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/vicohome/images", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	require.NoError(t, controller.GetVicoHomeImages(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, "{}", rec.Body.String())
}

// SetVicoHomeImageProvider replaces a previously-wired provider so that
// subsequent reads observe the new instance. This matters for hot-reload
// flows that may rebuild the poller.
func TestSetVicoHomeImageProvider_Replaces(t *testing.T) {
	t.Attr("component", "vicohome")
	t.Attr("type", "unit")

	e, controller := setupVicoHomeTestEnv(t)

	controller.SetVicoHomeImageProvider(&stubVicoHomeProvider{
		images: map[string]string{"a": "url-a"},
	})
	controller.SetVicoHomeImageProvider(&stubVicoHomeProvider{
		images: map[string]string{"b": "url-b"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/vicohome/images", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	require.NoError(t, controller.GetVicoHomeImages(c))
	var got map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, map[string]string{"b": "url-b"}, got)
}
