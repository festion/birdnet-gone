// vicohome.go: HTTP handlers for VicoHome bird feeder camera integration.
//
// The vicohome.Poller runs in the realtime analysis goroutine and accumulates
// the most recent camera image URL per detected species in memory. This file
// exposes that map to the kiosk frontend, which reads it via
// `GET /api/v2/vicohome/images`.
package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// VicoHomeImageProvider exposes the most-recent camera image URL per species.
//
// Implemented by the running *vicohome.Poller. The realtime startup sequence
// constructs the poller after the API server is up, then injects it into the
// Controller via SetVicoHomeImageProvider.
type VicoHomeImageProvider interface {
	// GetCameraImages returns a thread-safe snapshot of the current
	// lowercase species name → image URL map. Implementations must return a
	// copy so the caller can read concurrently with later Poller updates.
	GetCameraImages() map[string]string
}

// SetVicoHomeImageProvider wires (or replaces) the running poller into the
// API controller. Safe to call from any goroutine; the next request will see
// the new provider. Passing nil unwires the provider, after which the
// endpoint will respond with an empty object.
func (c *Controller) SetVicoHomeImageProvider(p VicoHomeImageProvider) {
	c.vicohomeProviderMu.Lock()
	defer c.vicohomeProviderMu.Unlock()
	c.vicohomeProvider = p
}

// initVicoHomeRoutes registers VicoHome-related API endpoints under /api/v2.
func (c *Controller) initVicoHomeRoutes() {
	c.Group.GET("/vicohome/images", c.GetVicoHomeImages)
}

// GetVicoHomeImages handles GET /api/v2/vicohome/images.
//
// Returns the current map of lowercase species name → image URL. When no
// provider has been wired yet (e.g., VicoHome disabled in config, or the
// poller has not yet finished its first cycle), responds 200 with "{}" so
// the kiosk view degrades gracefully — the frontend uses the absence of a
// key as the signal to fall back to its built-in thumbnail.
func (c *Controller) GetVicoHomeImages(ctx echo.Context) error {
	c.vicohomeProviderMu.RLock()
	p := c.vicohomeProvider
	c.vicohomeProviderMu.RUnlock()

	if p == nil {
		return ctx.JSON(http.StatusOK, map[string]string{})
	}
	images := p.GetCameraImages()
	if images == nil {
		images = map[string]string{}
	}
	return ctx.JSON(http.StatusOK, images)
}
