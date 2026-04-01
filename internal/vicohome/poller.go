package vicohome

import (
	"context"
	"maps"
	"strings"
	"sync"
	"time"

	"github.com/tphakala/birdnet-go/internal/conf"
	"github.com/tphakala/birdnet-go/internal/logger"
)

// Poller constants
const (
	// defaultPollInterval is the default polling interval in seconds.
	defaultPollInterval = 60

	// lookbackHours is how far back to look for events on each poll cycle.
	lookbackHours = 2

	// maxCameraImages is the maximum number of recent bird images to retain.
	maxCameraImages = 10
)

// EventFetcher is the interface for fetching events from the VicoHome API.
// This allows testing the Poller with a mock client.
type EventFetcher interface {
	// GetEvents fetches bird detection events since the given time.
	GetEvents(ctx context.Context, since time.Time) ([]Event, error)
}

// Poller ties the VicoHome API client and MQTT publisher together.
// It periodically polls the VicoHome API for new bird detection events,
// deduplicates them, and publishes to MQTT.
type Poller struct {
	config    conf.VicoHomeSettings
	client    EventFetcher
	publisher *Publisher
	log       logger.Logger

	mu              sync.RWMutex
	lastSeenTraceID string
	detectionsToday int
	today           string
	cameraImages    map[string]string // lowercase bird name -> image URL
	imageOrder      []string          // tracks insertion order for eviction

	stopCh chan struct{}
	done   chan struct{}
}

// NewPoller creates a new Poller with the given configuration, MQTT publisher, and logger.
func NewPoller(config conf.VicoHomeSettings, client EventFetcher, mqttPub MQTTPublisher, log logger.Logger) *Poller {
	pollInterval := config.PollInterval
	if pollInterval <= 0 {
		pollInterval = defaultPollInterval
	}
	config.PollInterval = pollInterval

	return &Poller{
		config:       config,
		client:       client,
		publisher:    NewPublisher(mqttPub),
		log:          log,
		today:        time.Now().Format("2006-01-02"),
		cameraImages: make(map[string]string),
		imageOrder:   make([]string, 0, maxCameraImages),
		stopCh:       make(chan struct{}),
		done:         make(chan struct{}),
	}
}

// Start begins the polling loop. It blocks until Stop is called or the context is cancelled.
func (p *Poller) Start(ctx context.Context) {
	defer close(p.done)

	p.log.Info("Starting VicoHome poller",
		logger.Int("poll_interval_seconds", p.config.PollInterval))

	// Publish discovery and online status on startup
	if err := p.publisher.PublishDiscovery(ctx); err != nil {
		p.log.Error("Failed to publish HA discovery", logger.String("error_detail", err.Error()))
	}
	if err := p.publisher.PublishOnline(ctx); err != nil {
		p.log.Error("Failed to publish online status", logger.String("error_detail", err.Error()))
	}

	ticker := time.NewTicker(time.Duration(p.config.PollInterval) * time.Second)
	defer ticker.Stop()

	// Run an initial poll immediately
	p.poll(ctx)

	for {
		select {
		case <-ctx.Done():
			p.shutdown(context.Background())
			return
		case <-p.stopCh:
			p.shutdown(ctx)
			return
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

// Stop signals the poller to stop gracefully.
func (p *Poller) Stop() {
	select {
	case <-p.stopCh:
		// Already stopped
	default:
		close(p.stopCh)
	}
	<-p.done
}

// GetCameraImages returns a thread-safe copy of the species name to image URL map.
// Keys are lowercase bird names.
func (p *Poller) GetCameraImages() map[string]string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make(map[string]string, len(p.cameraImages))
	maps.Copy(result, p.cameraImages)
	return result
}

// poll performs a single polling cycle: fetch events, deduplicate, publish new ones.
func (p *Poller) poll(ctx context.Context) {
	p.resetDailyCounterIfNeeded()

	since := time.Now().UTC().Add(-time.Duration(lookbackHours) * time.Hour)
	events, err := p.client.GetEvents(ctx, since)
	if err != nil {
		p.log.Warn("Failed to fetch VicoHome events", logger.String("error_detail", err.Error()))
		return
	}

	newEvents := p.filterNewEvents(events)
	if len(newEvents) == 0 {
		return
	}

	// Publish in chronological order (oldest first)
	for i := len(newEvents) - 1; i >= 0; i-- {
		event := &newEvents[i]
		if event.BirdName == "" {
			continue // Skip non-bird motion events
		}
		if err := p.publisher.PublishDetection(ctx, event); err != nil {
			p.log.Warn("Failed to publish detection",
				logger.String("trace_id", event.TraceID),
				logger.String("error_detail", err.Error()))
			continue
		}
		p.mu.Lock()
		p.detectionsToday++
		p.mu.Unlock()

		p.addCameraImage(event.BirdName, event.KeyShotURL, event.ImageURL)
	}

	// Update last seen and publish stats
	p.mu.Lock()
	p.lastSeenTraceID = newEvents[0].TraceID
	count := p.detectionsToday
	p.mu.Unlock()

	if err := p.publisher.PublishStats(ctx, count); err != nil {
		p.log.Warn("Failed to publish stats", logger.String("error_detail", err.Error()))
	}

	p.log.Info("Processed VicoHome events",
		logger.Int("new_events", len(newEvents)),
		logger.Int("detections_today", count))
}

// filterNewEvents returns events that are newer than the last seen trace ID.
// The API returns events in reverse chronological order (newest first).
func (p *Poller) filterNewEvents(events []Event) []Event {
	p.mu.RLock()
	lastSeen := p.lastSeenTraceID
	p.mu.RUnlock()

	if lastSeen == "" {
		return events
	}

	newEvents := make([]Event, 0, len(events))
	for i := range events {
		if events[i].TraceID == lastSeen {
			break
		}
		newEvents = append(newEvents, events[i])
	}
	return newEvents
}

// resetDailyCounterIfNeeded resets the daily detection counter at midnight.
func (p *Poller) resetDailyCounterIfNeeded() {
	currentDate := time.Now().Format("2006-01-02")
	p.mu.Lock()
	defer p.mu.Unlock()
	if currentDate != p.today {
		p.detectionsToday = 0
		p.today = currentDate
		p.log.Info("Daily counter reset")
	}
}

// addCameraImage adds a bird image to the cache, evicting the oldest if at capacity.
func (p *Poller) addCameraImage(birdName, keyShotURL, imageURL string) {
	url := keyShotURL
	if url == "" {
		url = imageURL
	}
	if url == "" {
		return
	}

	key := strings.ToLower(birdName)

	p.mu.Lock()
	defer p.mu.Unlock()

	// If the key already exists, just update the URL
	if _, exists := p.cameraImages[key]; exists {
		p.cameraImages[key] = url
		return
	}

	// Evict oldest entry if at capacity
	if len(p.imageOrder) >= maxCameraImages {
		oldestKey := p.imageOrder[0]
		delete(p.cameraImages, oldestKey)
		p.imageOrder = p.imageOrder[1:]
	}

	p.cameraImages[key] = url
	p.imageOrder = append(p.imageOrder, key)
}

// shutdown publishes the offline status before exiting.
func (p *Poller) shutdown(ctx context.Context) {
	p.log.Info("Shutting down VicoHome poller")
	if err := p.publisher.PublishOffline(ctx); err != nil {
		p.log.Warn("Failed to publish offline status", logger.String("error_detail", err.Error()))
	}
}
