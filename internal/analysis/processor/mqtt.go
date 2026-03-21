// mqtt.go: MQTT-related functionality for the processor
package processor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tphakala/birdnet-go/internal/conf"
	"github.com/tphakala/birdnet-go/internal/datastore"
	"github.com/tphakala/birdnet-go/internal/logger"
	"github.com/tphakala/birdnet-go/internal/mqtt"
	"github.com/tphakala/birdnet-go/internal/myaudio"
)

const (
	// mqttConnectionTimeout is the timeout for MQTT connection attempts
	mqttConnectionTimeout = 30 * time.Second
	// discoveryPublishTimeout is the timeout for publishing discovery messages
	discoveryPublishTimeout = 30 * time.Second
)

// lastPublishedSources tracks source IDs from the last discovery publish.
// Used to detect and clean up sources that no longer exist on reconnect.
var (
	lastPublishedSourcesMu sync.Mutex
	lastPublishedSources   []datastore.AudioSource
)

// GetMQTTClient safely returns the current MQTT client
func (p *Processor) GetMQTTClient() mqtt.Client {
	p.mqttMutex.RLock()
	defer p.mqttMutex.RUnlock()
	return p.MqttClient
}

// SetMQTTClient safely sets a new MQTT client
func (p *Processor) SetMQTTClient(client mqtt.Client) {
	p.mqttMutex.Lock()
	defer p.mqttMutex.Unlock()
	p.MqttClient = client
}

// DisconnectMQTTClient safely disconnects and removes the MQTT client
func (p *Processor) DisconnectMQTTClient() {
	p.mqttMutex.Lock()
	client := p.MqttClient
	p.MqttClient = nil
	p.mqttMutex.Unlock()

	if client != nil {
		client.Disconnect()
	}
}

// PublishMQTT safely publishes a message using the MQTT client if available
func (p *Processor) PublishMQTT(ctx context.Context, topic, payload string) error {
	p.mqttMutex.RLock()
	client := p.MqttClient
	p.mqttMutex.RUnlock()

	if client != nil && client.IsConnected() {
		return client.Publish(ctx, topic, payload)
	}
	return fmt.Errorf("MQTT client not available or not connected")
}

// initializeMQTT initializes the MQTT client if enabled in settings
func (p *Processor) initializeMQTT(settings *conf.Settings) {
	if settings == nil {
		return
	}
	if !settings.Realtime.MQTT.Enabled {
		return
	}

	log := GetLogger()
	// Create a new MQTT client using the settings and metrics
	mqttClient, err := mqtt.NewClient(settings, p.Metrics)
	if err != nil {
		log.Error("failed to create MQTT client", logger.Error(err))
		return
	}

	// Register Home Assistant discovery handler if enabled
	if settings.Realtime.MQTT.HomeAssistant.Enabled {
		p.registerHomeAssistantDiscovery(mqttClient, settings)
	}

	// Create a context with a timeout for the connection attempt
	ctx, cancel := context.WithTimeout(context.Background(), mqttConnectionTimeout)
	defer cancel() // Ensure the cancel function is called to release resources

	// Attempt to connect to the MQTT broker
	if err := mqttClient.Connect(ctx); err != nil {
		log.Error("failed to connect to MQTT broker", logger.Error(err))
		return
	}

	// Set the client only if connection was successful
	p.SetMQTTClient(mqttClient)
}

// RegisterHomeAssistantDiscovery registers the OnConnect handler for Home Assistant discovery.
// This is called during MQTT initialization and after MQTT reconfiguration.
func (p *Processor) RegisterHomeAssistantDiscovery(client mqtt.Client, settings *conf.Settings) {
	if client == nil || settings == nil {
		return
	}
	if !settings.Realtime.MQTT.HomeAssistant.Enabled {
		return
	}
	p.registerHomeAssistantDiscovery(client, settings)
}

// registerHomeAssistantDiscovery registers the OnConnect handler for Home Assistant discovery.
func (p *Processor) registerHomeAssistantDiscovery(client mqtt.Client, settings *conf.Settings) {
	log := GetLogger()
	haSettings := settings.Realtime.MQTT.HomeAssistant

	// Register the OnConnect handler
	client.RegisterOnConnectHandler(func() {
		log.Info("MQTT connected, publishing Home Assistant discovery messages")

		// Create a context for publishing
		ctx, cancel := context.WithTimeout(context.Background(), discoveryPublishTimeout)
		defer cancel()

		// Publish discovery messages using the helper
		if err := p.publishHomeAssistantDiscovery(ctx, client, settings); err != nil {
			log.Error("Failed to publish Home Assistant discovery",
				logger.Error(err))
		}
	})

	log.Info("Home Assistant discovery handler registered",
		logger.String("discovery_prefix", haSettings.DiscoveryPrefix),
		logger.String("device_name", haSettings.DeviceName))
}

// publishHomeAssistantDiscovery publishes Home Assistant discovery messages.
// This is the shared implementation used by both the OnConnect handler and manual trigger.
func (p *Processor) publishHomeAssistantDiscovery(ctx context.Context, client mqtt.Client, settings *conf.Settings) error {
	log := GetLogger()
	haSettings := settings.Realtime.MQTT.HomeAssistant
	discoveryConfig := mqtt.DiscoveryConfig{
		DiscoveryPrefix: haSettings.DiscoveryPrefix,
		BaseTopic:       settings.Realtime.MQTT.Topic,
		DeviceName:      haSettings.DeviceName,
		NodeID:          settings.Main.Name,
		Version:         settings.Version,
	}

	sources := p.getAudioSourcesForDiscovery()
	if len(sources) == 0 {
		log.Info("No audio sources registered yet, skipping Home Assistant discovery")
		return nil
	}

	publisher := mqtt.NewDiscoveryPublisher(client, &discoveryConfig)

	// Clean up stale sources from previous publish
	staleSources := findStaleSources(sources)
	if len(staleSources) > 0 {
		log.Info("Cleaning up stale discovery sources",
			logger.Int("stale_count", len(staleSources)))
		if err := publisher.RemoveSourceDiscovery(ctx, staleSources); err != nil {
			log.Warn("Failed to remove some stale discovery sources", logger.Error(err))
		}
	}

	// Publish discovery for current sources
	if err := publisher.PublishDiscovery(ctx, sources, settings); err != nil {
		return err
	}

	// Track what we published for next comparison
	lastPublishedSourcesMu.Lock()
	lastPublishedSources = sources
	lastPublishedSourcesMu.Unlock()

	return nil
}

// findStaleSources returns sources from the last publish that are not in the current list.
func findStaleSources(currentSources []datastore.AudioSource) []datastore.AudioSource {
	lastPublishedSourcesMu.Lock()
	previous := lastPublishedSources
	lastPublishedSourcesMu.Unlock()

	if len(previous) == 0 {
		return nil
	}

	// Build set of current source IDs
	currentIDs := make(map[string]bool, len(currentSources))
	for _, src := range currentSources {
		currentIDs[src.ID] = true
	}

	// Find sources in previous that aren't in current
	stale := make([]datastore.AudioSource, 0)
	for _, src := range previous {
		if !currentIDs[src.ID] {
			stale = append(stale, src)
		}
	}

	return stale
}

// TriggerHomeAssistantDiscovery manually triggers Home Assistant discovery messages.
// This can be called from the API to force republishing of discovery messages.
func (p *Processor) TriggerHomeAssistantDiscovery(ctx context.Context) error {
	log := GetLogger()

	// Guard against nil settings during startup/teardown
	if p.Settings == nil {
		return fmt.Errorf("settings not initialized")
	}

	// Check if MQTT is enabled and Home Assistant discovery is enabled
	if !p.Settings.Realtime.MQTT.Enabled {
		return fmt.Errorf("MQTT is not enabled")
	}
	if !p.Settings.Realtime.MQTT.HomeAssistant.Enabled {
		return fmt.Errorf("home assistant discovery is not enabled")
	}

	// Get the MQTT client
	client := p.GetMQTTClient()
	if client == nil || !client.IsConnected() {
		return fmt.Errorf("MQTT client not connected")
	}

	log.Info("Manually triggering Home Assistant discovery")

	// Publish discovery messages with timeout
	publishCtx, cancel := context.WithTimeout(ctx, discoveryPublishTimeout)
	defer cancel()

	if err := p.publishHomeAssistantDiscovery(publishCtx, client, p.Settings); err != nil {
		log.Error("Failed to publish Home Assistant discovery", logger.Error(err))
		return fmt.Errorf("failed to publish discovery: %w", err)
	}

	return nil
}

// getAudioSourcesForDiscovery retrieves audio sources from the registry for HA discovery.
func (p *Processor) getAudioSourcesForDiscovery() []datastore.AudioSource {
	registry := myaudio.GetRegistry()
	registrySources := registry.ListSources()

	// Convert myaudio.AudioSource to datastore.AudioSource
	sources := make([]datastore.AudioSource, 0, len(registrySources))
	for _, src := range registrySources {
		sources = append(sources, datastore.AudioSource{
			ID:          src.ID,
			SafeString:  src.SafeString,
			DisplayName: src.DisplayName,
		})
	}

	return sources
}
