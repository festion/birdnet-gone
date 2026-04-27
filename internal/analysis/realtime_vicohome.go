package analysis

import (
	"context"
	"sync"

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
func startVicoHomePolling(wg *sync.WaitGroup, settings *conf.Settings, mqttClient mqtt.Client, quitChan chan struct{}) {
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
