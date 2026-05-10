package analysis

import (
	"context"
	"sync"

	apiv2 "github.com/tphakala/birdnet-go/internal/api/v2"
	"github.com/tphakala/birdnet-go/internal/conf"
	"github.com/tphakala/birdnet-go/internal/logger"
	"github.com/tphakala/birdnet-go/internal/mqtt"
	"github.com/tphakala/birdnet-go/internal/vicohome"
)

// startVicoHomePolling launches the VicoHome poller in a goroutine tracked by wg.
//
// It is a no-op when the integration is disabled, when credentials are missing,
// or when the supplied MQTT client is unavailable. The poller exits cleanly when
// quitChan is closed.
//
// When registerImageProvider is non-nil, the freshly-constructed poller is
// registered with the API controller before the goroutine starts. This is what
// connects the in-memory image cache to GET /api/v2/vicohome/images, which the
// kiosk view consumes. Pass nil to skip the wiring (tests, or a deployment
// without the HTTP server).
func startVicoHomePolling(
	wg *sync.WaitGroup,
	settings *conf.Settings,
	mqttClient mqtt.Client,
	registerImageProvider func(apiv2.VicoHomeImageProvider),
	quitChan chan struct{},
) {
	if !settings.VicoHome.Enabled {
		return
	}
	if settings.VicoHome.Email == "" || settings.VicoHome.Password == "" {
		GetLogger().Warn("VicoHome enabled but credentials missing, skipping",
			logger.String("operation", "vicohome_skip_no_credentials"))
		return
	}
	if mqttClient == nil || !mqttClient.IsConnected() {
		GetLogger().Warn("VicoHome enabled but MQTT client unavailable, skipping",
			logger.String("operation", "vicohome_skip_no_mqtt"))
		return
	}

	apiClient := vicohome.NewClient(settings.VicoHome.Email, settings.VicoHome.Password)
	pollerLog := logger.Global().Module("vicohome")
	poller := vicohome.NewPoller(settings.VicoHome, apiClient, mqttClient, pollerLog)

	// Register the poller with the API controller so the kiosk view can
	// retrieve image URLs via GET /api/v2/vicohome/images. This must happen
	// before Start() so the very first poll cycle's images are immediately
	// reachable through the HTTP endpoint.
	if registerImageProvider != nil {
		registerImageProvider(poller)
	}

	wg.Go(func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Bridge quitChan → context cancellation. The bridge goroutine exits
		// when quitChan closes OR when the poller returns (signalled by done).
		done := make(chan struct{})
		defer close(done)
		go func() {
			select {
			case <-quitChan:
				cancel()
			case <-done:
			}
		}()

		poller.Start(ctx)
	})
}
