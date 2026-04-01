package vicohome

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tphakala/birdnet-go/internal/conf"
	"github.com/tphakala/birdnet-go/internal/logger"
)

// mockEventFetcher implements EventFetcher for testing.
type mockEventFetcher struct {
	events []Event
	err    error
}

func (m *mockEventFetcher) GetEvents(_ context.Context, _ time.Time) ([]Event, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.events, nil
}

// newTestLogger creates a silent logger for tests.
func newTestLogger(t *testing.T) logger.Logger {
	t.Helper()
	return logger.NewSlogLogger(io.Discard, logger.LogLevelError, nil)
}

// newTestPoller creates a Poller for testing with sensible defaults.
func newTestPoller(t *testing.T, fetcher EventFetcher, mqttPub MQTTPublisher) *Poller {
	t.Helper()
	config := conf.VicoHomeSettings{
		Enabled:      true,
		PollInterval: 1, // 1 second for fast tests
	}
	return NewPoller(config, fetcher, mqttPub, newTestLogger(t))
}

func TestPoller_NewEventsPublished(t *testing.T) {
	events := []Event{
		{
			TraceID:        "trace-002",
			Timestamp:      1700000060,
			BirdName:       "Blue Jay",
			BirdLatin:      "Cyanocitta cristata",
			BirdConfidence: 0.88,
			KeyShotURL:     "https://img.example.com/bluejay.jpg",
			DeviceName:     "Feeder",
		},
		{
			TraceID:        "trace-001",
			Timestamp:      1700000000,
			BirdName:       "Northern Cardinal",
			BirdLatin:      "Cardinalis cardinalis", //nolint:misspell // scientific name
			BirdConfidence: 0.95,
			KeyShotURL:     "https://img.example.com/cardinal.jpg",
			DeviceName:     "Feeder",
		},
	}

	fetcher := &mockEventFetcher{events: events}
	mqttPub := newMockMQTTPublisher()
	poller := newTestPoller(t, fetcher, mqttPub)

	ctx, cancel := context.WithCancel(context.Background())

	// Start poller in a goroutine
	go poller.Start(ctx)

	// Wait for initial poll to complete
	time.Sleep(200 * time.Millisecond)
	cancel()
	poller.Stop()

	// Should have published: discovery (5) + online (1) + 2 detections (2*2=4) + stats (1) + offline (1) = 12
	messages := mqttPub.getMessages()

	// Check that detections were published
	var detectionCount int
	for _, msg := range messages {
		if msg.topic == TopicDetection {
			detectionCount++
		}
	}
	assert.Equal(t, 2, detectionCount, "should have published 2 detection messages")

	// Check stats were published
	statsMsg := mqttPub.findByTopic(TopicStats)
	require.NotNil(t, statsMsg, "stats should have been published")
	assert.Contains(t, statsMsg.payload, `"detections_today":2`)
}

func TestPoller_DeduplicationByTraceID(t *testing.T) {
	events := []Event{
		{
			TraceID:    "trace-001",
			Timestamp:  1700000000,
			BirdName:   "Northern Cardinal",
			KeyShotURL: "https://img.example.com/cardinal.jpg",
		},
	}

	fetcher := &mockEventFetcher{events: events}
	mqttPub := newMockMQTTPublisher()
	poller := newTestPoller(t, fetcher, mqttPub)

	ctx := context.Background()

	// First poll
	poller.poll(ctx)

	firstCount := len(mqttPub.getMessages())
	assert.Positive(t, firstCount, "first poll should publish messages")

	// Second poll with same events
	poller.poll(ctx)

	secondCount := len(mqttPub.getMessages())
	assert.Equal(t, firstCount, secondCount, "second poll should not publish duplicates")
}

func TestPoller_GetCameraImages(t *testing.T) {
	events := []Event{
		{
			TraceID:    "trace-002",
			Timestamp:  1700000060,
			BirdName:   "Blue Jay",
			KeyShotURL: "https://img.example.com/bluejay.jpg",
		},
		{
			TraceID:    "trace-001",
			Timestamp:  1700000000,
			BirdName:   "Northern Cardinal",
			KeyShotURL: "https://img.example.com/cardinal.jpg",
		},
	}

	fetcher := &mockEventFetcher{events: events}
	mqttPub := newMockMQTTPublisher()
	poller := newTestPoller(t, fetcher, mqttPub)

	poller.poll(context.Background())

	images := poller.GetCameraImages()
	assert.Len(t, images, 2)
	assert.Equal(t, "https://img.example.com/cardinal.jpg", images["northern cardinal"])
	assert.Equal(t, "https://img.example.com/bluejay.jpg", images["blue jay"])
}

func TestPoller_CameraImageEviction(t *testing.T) {
	// Create more events than maxCameraImages
	events := make([]Event, maxCameraImages+3)
	for i := range events {
		events[i] = Event{
			TraceID:    "trace-" + time.Now().Format("150405") + "-" + string(rune('a'+i)),
			Timestamp:  int64(1700000000 + (len(events)-1-i)*60), // newest first
			BirdName:   "Bird " + string(rune('A'+i)),
			KeyShotURL: "https://img.example.com/bird" + string(rune('a'+i)) + ".jpg",
		}
	}

	fetcher := &mockEventFetcher{events: events}
	mqttPub := newMockMQTTPublisher()
	poller := newTestPoller(t, fetcher, mqttPub)

	poller.poll(context.Background())

	images := poller.GetCameraImages()
	assert.Len(t, images, maxCameraImages, "should cap at maxCameraImages")
}

func TestPoller_CameraImageUpdate(t *testing.T) {
	// First poll: Cardinal with URL 1
	events1 := []Event{
		{
			TraceID:    "trace-001",
			BirdName:   "Northern Cardinal",
			KeyShotURL: "https://img.example.com/cardinal-v1.jpg",
		},
	}
	fetcher := &mockEventFetcher{events: events1}
	mqttPub := newMockMQTTPublisher()
	poller := newTestPoller(t, fetcher, mqttPub)

	poller.poll(context.Background())

	images := poller.GetCameraImages()
	assert.Equal(t, "https://img.example.com/cardinal-v1.jpg", images["northern cardinal"])

	// Second poll: Cardinal with URL 2
	events2 := []Event{
		{
			TraceID:    "trace-002",
			BirdName:   "Northern Cardinal",
			KeyShotURL: "https://img.example.com/cardinal-v2.jpg",
		},
	}
	fetcher.events = events2
	poller.poll(context.Background())

	images = poller.GetCameraImages()
	assert.Equal(t, "https://img.example.com/cardinal-v2.jpg", images["northern cardinal"])
	assert.Len(t, images, 1, "should not have duplicate entries")
}

func TestPoller_DailyCounterReset(t *testing.T) {
	events := []Event{
		{
			TraceID:    "trace-001",
			BirdName:   "Northern Cardinal",
			KeyShotURL: "https://img.example.com/cardinal.jpg",
		},
	}

	fetcher := &mockEventFetcher{events: events}
	mqttPub := newMockMQTTPublisher()
	poller := newTestPoller(t, fetcher, mqttPub)

	poller.poll(context.Background())

	poller.mu.RLock()
	assert.Equal(t, 1, poller.detectionsToday)
	poller.mu.RUnlock()

	// Simulate date change
	poller.mu.Lock()
	poller.today = "1999-01-01" // a date that is definitely not today
	poller.mu.Unlock()

	// Reset last seen so next poll processes events
	poller.mu.Lock()
	poller.lastSeenTraceID = ""
	poller.mu.Unlock()

	poller.poll(context.Background())

	poller.mu.RLock()
	// Counter was reset to 0, then incremented by 1 for the new detection
	assert.Equal(t, 1, poller.detectionsToday)
	poller.mu.RUnlock()
}

func TestPoller_SkipsNonBirdEvents(t *testing.T) {
	events := []Event{
		{
			TraceID:   "trace-001",
			Timestamp: 1700000000,
			BirdName:  "", // non-bird motion event
		},
	}

	fetcher := &mockEventFetcher{events: events}
	mqttPub := newMockMQTTPublisher()
	poller := newTestPoller(t, fetcher, mqttPub)

	poller.poll(context.Background())

	// Should not have published any detection messages
	var detectionCount int
	for _, msg := range mqttPub.getMessages() {
		if msg.topic == TopicDetection {
			detectionCount++
		}
	}
	assert.Equal(t, 0, detectionCount, "non-bird events should not be published")
}

func TestPoller_DefaultPollInterval(t *testing.T) {
	config := conf.VicoHomeSettings{
		Enabled:      true,
		PollInterval: 0, // should default
	}
	fetcher := &mockEventFetcher{}
	mqttPub := newMockMQTTPublisher()

	poller := NewPoller(config, fetcher, mqttPub, newTestLogger(t))
	assert.Equal(t, defaultPollInterval, poller.config.PollInterval)
}

func TestPoller_FetchError(t *testing.T) {
	fetcher := &mockEventFetcher{
		err: assert.AnError,
	}
	mqttPub := newMockMQTTPublisher()
	poller := newTestPoller(t, fetcher, mqttPub)

	// Should not panic on fetch error
	poller.poll(context.Background())

	// No detection messages should have been published
	var detectionCount int
	for _, msg := range mqttPub.getMessages() {
		if msg.topic == TopicDetection {
			detectionCount++
		}
	}
	assert.Equal(t, 0, detectionCount)
}

func TestFilterNewEvents_Empty(t *testing.T) {
	fetcher := &mockEventFetcher{}
	mqttPub := newMockMQTTPublisher()
	poller := newTestPoller(t, fetcher, mqttPub)

	result := poller.filterNewEvents([]Event{})
	assert.Empty(t, result)
}

func TestFilterNewEvents_NoLastSeen(t *testing.T) {
	fetcher := &mockEventFetcher{}
	mqttPub := newMockMQTTPublisher()
	poller := newTestPoller(t, fetcher, mqttPub)

	events := []Event{
		{TraceID: "a"},
		{TraceID: "b"},
	}

	result := poller.filterNewEvents(events)
	assert.Len(t, result, 2, "with no last seen, all events should be new")
}

func TestFilterNewEvents_WithLastSeen(t *testing.T) {
	fetcher := &mockEventFetcher{}
	mqttPub := newMockMQTTPublisher()
	poller := newTestPoller(t, fetcher, mqttPub)

	poller.mu.Lock()
	poller.lastSeenTraceID = "b"
	poller.mu.Unlock()

	events := []Event{
		{TraceID: "d"}, // newest
		{TraceID: "c"},
		{TraceID: "b"}, // last seen
		{TraceID: "a"}, // older
	}

	result := poller.filterNewEvents(events)
	require.Len(t, result, 2)
	assert.Equal(t, "d", result[0].TraceID)
	assert.Equal(t, "c", result[1].TraceID)
}

func TestPoller_KeyShotFallback(t *testing.T) {
	events := []Event{
		{
			TraceID:    "trace-001",
			BirdName:   "Sparrow",
			KeyShotURL: "",
			ImageURL:   "https://img.example.com/fallback.jpg",
		},
	}

	fetcher := &mockEventFetcher{events: events}
	mqttPub := newMockMQTTPublisher()
	poller := newTestPoller(t, fetcher, mqttPub)

	poller.poll(context.Background())

	images := poller.GetCameraImages()
	assert.Equal(t, "https://img.example.com/fallback.jpg", images["sparrow"])
}
