package vicohome

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/tphakala/birdnet-go/internal/logger"
)

// MQTT topic and discovery constants matching the existing Python bridge.
const (
	// baseTopic is the root MQTT topic for all VicoHome messages.
	baseTopic = "vicohome"

	// TopicStatus is the topic for bridge online/offline status.
	TopicStatus = baseTopic + "/status"

	// TopicLatest is the topic for the most recent detection (retained).
	TopicLatest = baseTopic + "/latest"

	// TopicDetection is the topic for each new detection (non-retained).
	TopicDetection = baseTopic + "/detection"

	// TopicStats is the topic for daily detection statistics (retained).
	TopicStats = baseTopic + "/stats"

	// TopicSnapshot is the topic for the latest bird image in base64 (retained).
	TopicSnapshot = baseTopic + "/snapshot"

	// discoveryPrefix is the Home Assistant MQTT discovery topic prefix.
	discoveryPrefix = "homeassistant"

	// nodeID is the HA device node identifier for the VicoHome bridge.
	nodeID = "vicohome_bridge"

	// statusOnline is the payload indicating the bridge is online.
	statusOnline = "online"

	// statusOffline is the payload indicating the bridge is offline.
	statusOffline = "offline"

	// deviceIdentifier is the unique identifier for the HA device.
	deviceIdentifier = "vicohome_g02_bridge"

	// deviceName is the display name in Home Assistant.
	deviceName = "VicoHome Bird Feeder"

	// deviceManufacturer is the manufacturer name for the HA device.
	deviceManufacturer = "HeaPets"

	// deviceModel is the model name for the HA device.
	deviceModel = "G02"

	// deviceSWVersion is the software version for the HA device.
	deviceSWVersion = "1.0.0"

	// detectionSource is the source identifier for detection payloads.
	detectionSource = "vicohome_g02"
)

// MQTTPublisher is the interface needed for MQTT publishing.
// It is intentionally minimal to allow easy mocking in tests.
type MQTTPublisher interface {
	// Publish sends a message to the specified topic.
	Publish(ctx context.Context, topic, payload string) error

	// PublishWithRetain sends a message with explicit retain flag control.
	PublishWithRetain(ctx context.Context, topic, payload string, retain bool) error
}

// snapshotDownloadTimeout is the maximum time allowed for downloading a snapshot image.
const snapshotDownloadTimeout = 10 * time.Second

// Publisher handles publishing VicoHome detection data and HA discovery configs to MQTT.
type Publisher struct {
	client     MQTTPublisher
	log        logger.Logger
	httpClient *http.Client
}

// NewPublisher creates a new Publisher with the given MQTT client and logger.
func NewPublisher(client MQTTPublisher, log logger.Logger) *Publisher {
	return &Publisher{
		client: client,
		log:    log,
		httpClient: &http.Client{
			Timeout: snapshotDownloadTimeout,
		},
	}
}

// haDevice is the HA device info payload included in all discovery messages.
type haDevice struct {
	Identifiers  []string `json:"identifiers"`
	Name         string   `json:"name"`
	Manufacturer string   `json:"manufacturer"`
	Model        string   `json:"model"`
	SWVersion    string   `json:"sw_version"`
}

// haDiscoveryBinarySensor is the discovery payload for a binary sensor.
type haDiscoveryBinarySensor struct {
	Name        string   `json:"name"`
	UniqueID    string   `json:"unique_id"`
	Device      haDevice `json:"device"`
	StateTopic  string   `json:"state_topic"`
	PayloadOn   string   `json:"payload_on"`
	PayloadOff  string   `json:"payload_off"`
	DeviceClass string   `json:"device_class"`
}

// haDiscoverySensor is the discovery payload for a standard sensor.
type haDiscoverySensor struct {
	Name                string   `json:"name"`
	UniqueID            string   `json:"unique_id"`
	Device              haDevice `json:"device"`
	StateTopic          string   `json:"state_topic"`
	ValueTemplate       string   `json:"value_template,omitempty"`
	JSONAttributesTopic string   `json:"json_attributes_topic,omitempty"`
	UnitOfMeasurement   string   `json:"unit_of_measurement,omitempty"`
	Icon                string   `json:"icon,omitempty"`
	AvailabilityTopic   string   `json:"availability_topic,omitempty"`
	PayloadAvailable    string   `json:"payload_available,omitempty"`
	PayloadNotAvailable string   `json:"payload_not_available,omitempty"`
}

// haDiscoveryCamera is the discovery payload for a camera entity.
type haDiscoveryCamera struct {
	Name                string   `json:"name"`
	UniqueID            string   `json:"unique_id"`
	Device              haDevice `json:"device"`
	Topic               string   `json:"topic"`
	ImageEncoding       string   `json:"image_encoding"`
	AvailabilityTopic   string   `json:"availability_topic,omitempty"`
	PayloadAvailable    string   `json:"payload_available,omitempty"`
	PayloadNotAvailable string   `json:"payload_not_available,omitempty"`
}

// detectionPayload is the JSON payload published for each detection event.
type detectionPayload struct {
	BirdName       string  `json:"bird_name"`
	ScientificName string  `json:"scientific_name"`
	Confidence     float64 `json:"confidence"`
	ImageURL       string  `json:"image_url"`
	VideoURL       string  `json:"video_url"`
	Timestamp      int64   `json:"timestamp"`
	DeviceName     string  `json:"device_name"`
	TraceID        string  `json:"trace_id"`
	Source         string  `json:"source"`
}

// statsPayload is the JSON payload for daily detection statistics.
type statsPayload struct {
	DetectionsToday int `json:"detections_today"`
}

// newHADevice returns the standard device info for all discovery messages.
func newHADevice() haDevice {
	return haDevice{
		Identifiers:  []string{deviceIdentifier},
		Name:         deviceName,
		Manufacturer: deviceManufacturer,
		Model:        deviceModel,
		SWVersion:    deviceSWVersion,
	}
}

// PublishDiscovery publishes HA auto-discovery configs for all VicoHome entities.
func (p *Publisher) PublishDiscovery(ctx context.Context) error {
	device := newHADevice()

	// Binary sensor: bridge online status
	statusConfig := haDiscoveryBinarySensor{
		Name:        "VicoHome Bridge",
		UniqueID:    nodeID + "_status",
		Device:      device,
		StateTopic:  TopicStatus,
		PayloadOn:   statusOnline,
		PayloadOff:  statusOffline,
		DeviceClass: "connectivity",
	}
	if err := p.publishDiscoveryConfig(ctx, "binary_sensor", "status", statusConfig); err != nil {
		return fmt.Errorf("failed to publish status discovery: %w", err)
	}

	// Sensor: latest bird species
	speciesConfig := haDiscoverySensor{
		Name:                "Latest Species",
		UniqueID:            nodeID + "_latest_species",
		Device:              device,
		StateTopic:          TopicLatest,
		ValueTemplate:       "{{ value_json.bird_name }}",
		JSONAttributesTopic: TopicLatest,
		Icon:                "mdi:bird",
		AvailabilityTopic:   TopicStatus,
		PayloadAvailable:    statusOnline,
		PayloadNotAvailable: statusOffline,
	}
	if err := p.publishDiscoveryConfig(ctx, "sensor", "latest_species", speciesConfig); err != nil {
		return fmt.Errorf("failed to publish species discovery: %w", err)
	}

	// Sensor: confidence
	confidenceConfig := haDiscoverySensor{
		Name:                "Detection Confidence",
		UniqueID:            nodeID + "_confidence",
		Device:              device,
		StateTopic:          TopicLatest,
		ValueTemplate:       "{{ (value_json.confidence * 100) | round(0) if value_json.confidence <= 1 else value_json.confidence | round(0) }}",
		UnitOfMeasurement:   "%",
		Icon:                "mdi:percent",
		AvailabilityTopic:   TopicStatus,
		PayloadAvailable:    statusOnline,
		PayloadNotAvailable: statusOffline,
	}
	if err := p.publishDiscoveryConfig(ctx, "sensor", "confidence", confidenceConfig); err != nil {
		return fmt.Errorf("failed to publish confidence discovery: %w", err)
	}

	// Sensor: detections today
	countConfig := haDiscoverySensor{
		Name:                "Detections Today",
		UniqueID:            nodeID + "_detections_today",
		Device:              device,
		StateTopic:          TopicStats,
		ValueTemplate:       "{{ value_json.detections_today }}",
		Icon:                "mdi:counter",
		AvailabilityTopic:   TopicStatus,
		PayloadAvailable:    statusOnline,
		PayloadNotAvailable: statusOffline,
	}
	if err := p.publishDiscoveryConfig(ctx, "sensor", "detections_today", countConfig); err != nil {
		return fmt.Errorf("failed to publish detections_today discovery: %w", err)
	}

	// Camera: latest bird snapshot
	cameraConfig := haDiscoveryCamera{
		Name:                "Bird Snapshot",
		UniqueID:            nodeID + "_snapshot",
		Device:              device,
		Topic:               TopicSnapshot,
		ImageEncoding:       "b64",
		AvailabilityTopic:   TopicStatus,
		PayloadAvailable:    statusOnline,
		PayloadNotAvailable: statusOffline,
	}
	if err := p.publishDiscoveryConfig(ctx, "camera", "snapshot", cameraConfig); err != nil {
		return fmt.Errorf("failed to publish snapshot discovery: %w", err)
	}

	return nil
}

// publishDiscoveryConfig marshals and publishes a single discovery config message.
func (p *Publisher) publishDiscoveryConfig(ctx context.Context, component, objectID string, config any) error {
	topic := fmt.Sprintf("%s/%s/%s/%s/config", discoveryPrefix, component, nodeID, objectID)
	payload, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal %s/%s discovery config: %w", component, objectID, err)
	}
	return p.client.PublishWithRetain(ctx, topic, string(payload), true)
}

// PublishDetection publishes a bird detection event to both retained and non-retained topics.
func (p *Publisher) PublishDetection(ctx context.Context, event *Event) error {
	imageURL := event.KeyShotURL
	if imageURL == "" {
		imageURL = event.ImageURL
	}

	payload := detectionPayload{
		BirdName:       event.BirdName,
		ScientificName: event.BirdLatin,
		Confidence:     event.BirdConfidence,
		ImageURL:       imageURL,
		VideoURL:       event.VideoURL,
		Timestamp:      event.Timestamp,
		DeviceName:     event.DeviceName,
		TraceID:        event.TraceID,
		Source:         detectionSource,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal detection payload: %w", err)
	}

	payloadStr := string(data)

	// Publish to retained "latest" topic
	if err := p.client.PublishWithRetain(ctx, TopicLatest, payloadStr, true); err != nil {
		return fmt.Errorf("failed to publish to %s: %w", TopicLatest, err)
	}

	// Publish to non-retained "detection" topic
	if err := p.client.PublishWithRetain(ctx, TopicDetection, payloadStr, false); err != nil {
		return fmt.Errorf("failed to publish to %s: %w", TopicDetection, err)
	}

	// Publish snapshot image (best-effort, errors are logged but don't fail the detection)
	p.PublishSnapshot(ctx, imageURL)

	return nil
}

// PublishSnapshot downloads an image from the given URL and publishes it as base64
// to the snapshot MQTT topic with retain=true. If imageURL is empty, this is a no-op.
// Download errors are logged as warnings but do not cause the method to return an error,
// ensuring that snapshot failures never block detection publishing.
func (p *Publisher) PublishSnapshot(ctx context.Context, imageURL string) {
	if imageURL == "" {
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, http.NoBody)
	if err != nil {
		p.log.Warn("Failed to create snapshot request",
			logger.String("url", imageURL),
			logger.String("error_detail", err.Error()))
		return
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		p.log.Warn("Failed to download snapshot",
			logger.String("url", imageURL),
			logger.String("error_detail", err.Error()))
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		p.log.Warn("Snapshot download returned non-200 status",
			logger.String("url", imageURL),
			logger.Int("status_code", resp.StatusCode))
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		p.log.Warn("Failed to read snapshot response body",
			logger.String("url", imageURL),
			logger.String("error_detail", err.Error()))
		return
	}

	encoded := base64.StdEncoding.EncodeToString(body)
	if err := p.client.PublishWithRetain(ctx, TopicSnapshot, encoded, true); err != nil {
		p.log.Warn("Failed to publish snapshot to MQTT",
			logger.String("error_detail", err.Error()))
	}
}

// PublishStats publishes the daily detection count.
func (p *Publisher) PublishStats(ctx context.Context, count int) error {
	payload := statsPayload{DetectionsToday: count}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal stats payload: %w", err)
	}
	return p.client.PublishWithRetain(ctx, TopicStats, string(data), true)
}

// PublishOnline publishes the bridge online status.
func (p *Publisher) PublishOnline(ctx context.Context) error {
	return p.client.PublishWithRetain(ctx, TopicStatus, statusOnline, true)
}

// PublishOffline publishes the bridge offline status.
func (p *Publisher) PublishOffline(ctx context.Context) error {
	return p.client.PublishWithRetain(ctx, TopicStatus, statusOffline, true)
}
