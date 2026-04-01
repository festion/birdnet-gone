package vicohome

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tphakala/birdnet-go/internal/logger"
)

// nopLogger is a no-op logger for tests that don't need to verify log output.
type nopLogger struct{}

func (nopLogger) Module(_ string) logger.Logger                      { return nopLogger{} }
func (nopLogger) Trace(_ string, _ ...logger.Field)                  {}
func (nopLogger) Debug(_ string, _ ...logger.Field)                  {}
func (nopLogger) Info(_ string, _ ...logger.Field)                   {}
func (nopLogger) Warn(_ string, _ ...logger.Field)                   {}
func (nopLogger) Error(_ string, _ ...logger.Field)                  {}
func (nopLogger) With(_ ...logger.Field) logger.Logger               { return nopLogger{} }
func (nopLogger) WithContext(_ context.Context) logger.Logger        { return nopLogger{} }
func (nopLogger) Log(_ logger.LogLevel, _ string, _ ...logger.Field) {}
func (nopLogger) Flush() error                                       { return nil }

// publishedMessage records a single MQTT publish call.
type publishedMessage struct {
	topic   string
	payload string
	retain  bool
}

// mockMQTTPublisher is a test double for MQTTPublisher that records all published messages.
type mockMQTTPublisher struct {
	mu       sync.Mutex
	messages []publishedMessage
	err      error // if set, all publish calls return this error
}

func newMockMQTTPublisher() *mockMQTTPublisher {
	return &mockMQTTPublisher{
		messages: make([]publishedMessage, 0),
	}
}

// Publish implements MQTTPublisher. Uses retain=false by default.
func (m *mockMQTTPublisher) Publish(_ context.Context, topic, payload string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.messages = append(m.messages, publishedMessage{topic: topic, payload: payload, retain: false})
	return nil
}

// PublishWithRetain implements MQTTPublisher.
func (m *mockMQTTPublisher) PublishWithRetain(_ context.Context, topic, payload string, retain bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.messages = append(m.messages, publishedMessage{topic: topic, payload: payload, retain: retain})
	return nil
}

// getMessages returns a copy of all published messages.
func (m *mockMQTTPublisher) getMessages() []publishedMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]publishedMessage, len(m.messages))
	copy(result, m.messages)
	return result
}

// findByTopic returns the first message matching the given topic, or nil.
func (m *mockMQTTPublisher) findByTopic(topic string) *publishedMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.messages {
		if m.messages[i].topic == topic {
			return &m.messages[i]
		}
	}
	return nil
}

func TestPublishDiscovery(t *testing.T) {
	mock := newMockMQTTPublisher()
	pub := NewPublisher(mock, nopLogger{})

	err := pub.PublishDiscovery(context.Background())
	require.NoError(t, err)

	messages := mock.getMessages()

	// Should publish exactly 5 discovery configs
	require.Len(t, messages, 5)

	// All discovery messages should be retained
	for _, msg := range messages {
		assert.True(t, msg.retain, "discovery message on %s should be retained", msg.topic)
	}

	// Verify binary_sensor (status)
	statusMsg := mock.findByTopic(fmt.Sprintf("%s/binary_sensor/%s/status/config", discoveryPrefix, nodeID))
	require.NotNil(t, statusMsg, "status discovery message should be published")
	var statusPayload haDiscoveryBinarySensor
	err = json.Unmarshal([]byte(statusMsg.payload), &statusPayload)
	require.NoError(t, err)
	assert.Equal(t, "VicoHome Bridge", statusPayload.Name)
	assert.Equal(t, nodeID+"_status", statusPayload.UniqueID)
	assert.Equal(t, TopicStatus, statusPayload.StateTopic)
	assert.Equal(t, statusOnline, statusPayload.PayloadOn)
	assert.Equal(t, statusOffline, statusPayload.PayloadOff)
	assert.Equal(t, "connectivity", statusPayload.DeviceClass)
	assert.Equal(t, deviceIdentifier, statusPayload.Device.Identifiers[0])

	// Verify sensor (latest_species)
	speciesMsg := mock.findByTopic(fmt.Sprintf("%s/sensor/%s/latest_species/config", discoveryPrefix, nodeID))
	require.NotNil(t, speciesMsg, "species discovery message should be published")
	var speciesPayload haDiscoverySensor
	err = json.Unmarshal([]byte(speciesMsg.payload), &speciesPayload)
	require.NoError(t, err)
	assert.Equal(t, "Latest Species", speciesPayload.Name)
	assert.Equal(t, "{{ value_json.bird_name }}", speciesPayload.ValueTemplate)
	assert.Equal(t, TopicLatest, speciesPayload.JSONAttributesTopic)
	assert.Equal(t, "mdi:bird", speciesPayload.Icon)
	assert.Equal(t, TopicStatus, speciesPayload.AvailabilityTopic)

	// Verify sensor (confidence)
	confMsg := mock.findByTopic(fmt.Sprintf("%s/sensor/%s/confidence/config", discoveryPrefix, nodeID))
	require.NotNil(t, confMsg, "confidence discovery message should be published")
	var confPayloadData haDiscoverySensor
	err = json.Unmarshal([]byte(confMsg.payload), &confPayloadData)
	require.NoError(t, err)
	assert.Equal(t, "Detection Confidence", confPayloadData.Name)
	assert.Equal(t, "%", confPayloadData.UnitOfMeasurement)
	assert.Equal(t, "mdi:percent", confPayloadData.Icon)

	// Verify sensor (detections_today)
	countMsg := mock.findByTopic(fmt.Sprintf("%s/sensor/%s/detections_today/config", discoveryPrefix, nodeID))
	require.NotNil(t, countMsg, "detections_today discovery message should be published")
	var countPayloadData haDiscoverySensor
	err = json.Unmarshal([]byte(countMsg.payload), &countPayloadData)
	require.NoError(t, err)
	assert.Equal(t, "Detections Today", countPayloadData.Name)
	assert.Equal(t, "{{ value_json.detections_today }}", countPayloadData.ValueTemplate)
	assert.Equal(t, "mdi:counter", countPayloadData.Icon)

	// Verify camera (snapshot)
	camMsg := mock.findByTopic(fmt.Sprintf("%s/camera/%s/snapshot/config", discoveryPrefix, nodeID))
	require.NotNil(t, camMsg, "snapshot discovery message should be published")
	var camPayload haDiscoveryCamera
	err = json.Unmarshal([]byte(camMsg.payload), &camPayload)
	require.NoError(t, err)
	assert.Equal(t, "Bird Snapshot", camPayload.Name)
	assert.Equal(t, TopicSnapshot, camPayload.Topic)
	assert.Equal(t, "b64", camPayload.ImageEncoding)
}

func TestPublishDetection(t *testing.T) {
	mock := newMockMQTTPublisher()
	pub := NewPublisher(mock, nopLogger{})

	event := &Event{
		TraceID:        "trace-001",
		Timestamp:      1700000000,
		BirdName:       "Northern Cardinal",
		BirdLatin:      "Cardinalis cardinalis", //nolint:misspell // scientific name
		BirdConfidence: 0.95,
		KeyShotURL:     "https://img.example.com/keyshot.jpg",
		ImageURL:       "https://img.example.com/full.jpg",
		VideoURL:       "https://vid.example.com/clip.mp4",
		DeviceName:     "Bird Feeder Cam",
	}

	err := pub.PublishDetection(context.Background(), event)
	require.NoError(t, err)

	messages := mock.getMessages()
	require.Len(t, messages, 2)

	// First message: retained "latest"
	assert.Equal(t, TopicLatest, messages[0].topic)
	assert.True(t, messages[0].retain, "latest topic should be retained")

	// Second message: non-retained "detection"
	assert.Equal(t, TopicDetection, messages[1].topic)
	assert.False(t, messages[1].retain, "detection topic should not be retained")

	// Both should have identical payloads
	assert.Equal(t, messages[0].payload, messages[1].payload)

	// Verify payload content
	var payload detectionPayload
	err = json.Unmarshal([]byte(messages[0].payload), &payload)
	require.NoError(t, err)
	assert.Equal(t, "Northern Cardinal", payload.BirdName)
	assert.Equal(t, "Cardinalis cardinalis", payload.ScientificName) //nolint:misspell // scientific name
	assert.InDelta(t, 0.95, payload.Confidence, 0.001)
	assert.Equal(t, "https://img.example.com/keyshot.jpg", payload.ImageURL) // uses keyshot
	assert.Equal(t, "https://vid.example.com/clip.mp4", payload.VideoURL)
	assert.Equal(t, int64(1700000000), payload.Timestamp)
	assert.Equal(t, "Bird Feeder Cam", payload.DeviceName)
	assert.Equal(t, "trace-001", payload.TraceID)
	assert.Equal(t, detectionSource, payload.Source)
}

func TestPublishDetection_FallbackToImageURL(t *testing.T) {
	mock := newMockMQTTPublisher()
	pub := NewPublisher(mock, nopLogger{})

	event := &Event{
		TraceID:    "trace-002",
		BirdName:   "Blue Jay",
		KeyShotURL: "", // empty keyshot
		ImageURL:   "https://img.example.com/fallback.jpg",
	}

	err := pub.PublishDetection(context.Background(), event)
	require.NoError(t, err)

	var payload detectionPayload
	err = json.Unmarshal([]byte(mock.getMessages()[0].payload), &payload)
	require.NoError(t, err)
	assert.Equal(t, "https://img.example.com/fallback.jpg", payload.ImageURL)
}

func TestPublishStats(t *testing.T) {
	mock := newMockMQTTPublisher()
	pub := NewPublisher(mock, nopLogger{})

	err := pub.PublishStats(context.Background(), 42)
	require.NoError(t, err)

	messages := mock.getMessages()
	require.Len(t, messages, 1)

	assert.Equal(t, TopicStats, messages[0].topic)
	assert.True(t, messages[0].retain, "stats topic should be retained")

	var payload statsPayload
	err = json.Unmarshal([]byte(messages[0].payload), &payload)
	require.NoError(t, err)
	assert.Equal(t, 42, payload.DetectionsToday)
}

func TestPublishOnline(t *testing.T) {
	mock := newMockMQTTPublisher()
	pub := NewPublisher(mock, nopLogger{})

	err := pub.PublishOnline(context.Background())
	require.NoError(t, err)

	messages := mock.getMessages()
	require.Len(t, messages, 1)
	assert.Equal(t, TopicStatus, messages[0].topic)
	assert.Equal(t, statusOnline, messages[0].payload)
	assert.True(t, messages[0].retain)
}

func TestPublishOffline(t *testing.T) {
	mock := newMockMQTTPublisher()
	pub := NewPublisher(mock, nopLogger{})

	err := pub.PublishOffline(context.Background())
	require.NoError(t, err)

	messages := mock.getMessages()
	require.Len(t, messages, 1)
	assert.Equal(t, TopicStatus, messages[0].topic)
	assert.Equal(t, statusOffline, messages[0].payload)
	assert.True(t, messages[0].retain)
}

func TestPublishDiscovery_ErrorPropagation(t *testing.T) {
	mock := newMockMQTTPublisher()
	mock.err = fmt.Errorf("connection lost")
	pub := NewPublisher(mock, nopLogger{})

	err := pub.PublishDiscovery(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection lost")
}

func TestPublishDetection_ErrorPropagation(t *testing.T) {
	mock := newMockMQTTPublisher()
	mock.err = fmt.Errorf("broker unreachable")
	pub := NewPublisher(mock, nopLogger{})

	err := pub.PublishDetection(context.Background(), &Event{BirdName: "test"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "broker unreachable")
}

func TestPublishSnapshot(t *testing.T) {
	// Set up a test HTTP server that returns a known image payload
	imageData := []byte("fake-png-image-data")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(imageData)
	}))
	defer server.Close()

	mqttMock := newMockMQTTPublisher()
	pub := NewPublisher(mqttMock, nopLogger{})

	pub.PublishSnapshot(context.Background(), server.URL+"/snapshot.png")

	// Verify the snapshot was published to the correct topic with retain
	snapshotMsg := mqttMock.findByTopic(TopicSnapshot)
	require.NotNil(t, snapshotMsg, "snapshot should be published to %s", TopicSnapshot)
	assert.True(t, snapshotMsg.retain, "snapshot should be retained")

	// Verify the payload is the base64-encoded image data
	expectedPayload := base64.StdEncoding.EncodeToString(imageData)
	assert.Equal(t, expectedPayload, snapshotMsg.payload)
}

func TestPublishSnapshot_EmptyURL(t *testing.T) {
	mqttMock := newMockMQTTPublisher()
	pub := NewPublisher(mqttMock, nopLogger{})

	pub.PublishSnapshot(context.Background(), "")

	// No messages should be published for an empty URL
	messages := mqttMock.getMessages()
	assert.Empty(t, messages, "empty URL should not produce any MQTT messages")
}

func TestPublishSnapshot_DownloadError(t *testing.T) {
	// Use a URL that will fail to connect
	mqttMock := newMockMQTTPublisher()
	pub := NewPublisher(mqttMock, nopLogger{})

	pub.PublishSnapshot(context.Background(), "http://127.0.0.1:1/nonexistent")

	// No messages should be published when download fails
	messages := mqttMock.getMessages()
	assert.Empty(t, messages, "failed download should not produce any MQTT messages")
}

func TestPublishSnapshot_Non200Status(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	mqttMock := newMockMQTTPublisher()
	pub := NewPublisher(mqttMock, nopLogger{})

	pub.PublishSnapshot(context.Background(), server.URL+"/missing.png")

	// No messages should be published for non-200 responses
	messages := mqttMock.getMessages()
	assert.Empty(t, messages, "non-200 response should not produce any MQTT messages")
}
