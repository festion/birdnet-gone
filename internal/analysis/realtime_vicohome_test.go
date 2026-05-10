package analysis

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	apiv2 "github.com/tphakala/birdnet-go/internal/api/v2"
	"github.com/tphakala/birdnet-go/internal/conf"
	"github.com/tphakala/birdnet-go/internal/mqtt"
)

// ignoreProvider is the no-op registerImageProvider used by tests that don't
// care about the wiring (early-exit branches).
func ignoreProvider(_ apiv2.VicoHomeImageProvider) {}

// Compile-time check that fakeMQTTClient satisfies mqtt.Client.
var _ mqtt.Client = (*fakeMQTTClient)(nil)

// fakeMQTTClient is a minimal mqtt.Client used to drive the connected/disconnected
// branches of startVicoHomePolling without standing up a real broker.
type fakeMQTTClient struct {
	connected bool
}

func (f *fakeMQTTClient) Connect(_ context.Context) error              { return nil }
func (f *fakeMQTTClient) Publish(_ context.Context, _, _ string) error { return nil }
func (f *fakeMQTTClient) PublishWithRetain(_ context.Context, _, _ string, _ bool) error {
	return nil
}
func (f *fakeMQTTClient) IsConnected() bool                                          { return f.connected }
func (f *fakeMQTTClient) Disconnect()                                                {}
func (f *fakeMQTTClient) SetControlChannel(_ chan string)                            {}
func (f *fakeMQTTClient) TestConnection(_ context.Context, _ chan<- mqtt.TestResult) {}
func (f *fakeMQTTClient) RegisterOnConnectHandler(_ mqtt.OnConnectHandler)           {}

// assertNoGoroutineStarted asserts that startVicoHomePolling did not enqueue
// any work onto the WaitGroup (i.e., wg.Wait() returns immediately).
func assertNoGoroutineStarted(t *testing.T, wg *sync.WaitGroup) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		// expected: no goroutine was started, wg counter is 0
	case <-time.After(200 * time.Millisecond):
		t.Fatal("startVicoHomePolling started a goroutine when it should have been a no-op")
	}
}

func TestStartVicoHomePolling_DisabledIsNoOp(t *testing.T) {
	t.Parallel()
	settings := &conf.Settings{
		VicoHome: conf.VicoHomeSettings{Enabled: false},
	}
	mqttClient := &fakeMQTTClient{connected: true}
	var wg sync.WaitGroup
	quitChan := make(chan struct{})

	startVicoHomePolling(&wg, settings, mqttClient, ignoreProvider, quitChan)

	assertNoGoroutineStarted(t, &wg)
}

func TestStartVicoHomePolling_MissingCredentialsIsNoOp(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		email    string
		password string
	}{
		{"empty email", "", "secret"},
		{"empty password", "user@example.com", ""},
		{"both empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			settings := &conf.Settings{
				VicoHome: conf.VicoHomeSettings{
					Enabled:  true,
					Email:    tt.email,
					Password: tt.password,
				},
			}
			mqttClient := &fakeMQTTClient{connected: true}
			var wg sync.WaitGroup
			quitChan := make(chan struct{})

			startVicoHomePolling(&wg, settings, mqttClient, ignoreProvider, quitChan)

			assertNoGoroutineStarted(t, &wg)
		})
	}
}

func TestStartVicoHomePolling_NilMQTTClientIsNoOp(t *testing.T) {
	t.Parallel()
	settings := &conf.Settings{
		VicoHome: conf.VicoHomeSettings{
			Enabled:  true,
			Email:    "user@example.com",
			Password: "secret",
		},
	}
	var wg sync.WaitGroup
	quitChan := make(chan struct{})

	startVicoHomePolling(&wg, settings, nil, ignoreProvider, quitChan)

	assertNoGoroutineStarted(t, &wg)
}

func TestStartVicoHomePolling_DisconnectedMQTTClientIsNoOp(t *testing.T) {
	t.Parallel()
	settings := &conf.Settings{
		VicoHome: conf.VicoHomeSettings{
			Enabled:  true,
			Email:    "user@example.com",
			Password: "secret",
		},
	}
	mqttClient := &fakeMQTTClient{connected: false}
	var wg sync.WaitGroup
	quitChan := make(chan struct{})

	startVicoHomePolling(&wg, settings, mqttClient, ignoreProvider, quitChan)

	assertNoGoroutineStarted(t, &wg)
	assert.False(t, mqttClient.IsConnected(), "fake remained disconnected")
}

// When all preconditions are met, startVicoHomePolling must register the
// poller with the API controller via registerImageProvider. The kiosk
// frontend's GET /api/v2/vicohome/images endpoint reads from this provider,
// so a missing call here is a silent UI failure (events are processed but
// camera images never reach the display).
func TestStartVicoHomePolling_RegistersImageProvider(t *testing.T) {
	t.Parallel()
	settings := &conf.Settings{
		VicoHome: conf.VicoHomeSettings{
			Enabled:  true,
			Email:    "user@example.com",
			Password: "secret",
		},
	}
	mqttClient := &fakeMQTTClient{connected: true}
	var wg sync.WaitGroup
	quitChan := make(chan struct{})
	defer close(quitChan)

	var registered apiv2.VicoHomeImageProvider
	register := func(p apiv2.VicoHomeImageProvider) { registered = p }

	startVicoHomePolling(&wg, settings, mqttClient, register, quitChan)

	assert.NotNil(t, registered, "expected the poller to be registered with the controller")
	// The registered provider must satisfy the contract — calling
	// GetCameraImages on a fresh poller should return a non-nil map.
	images := registered.GetCameraImages()
	assert.NotNil(t, images, "GetCameraImages must return a non-nil map (empty is fine)")
	assert.Empty(t, images, "fresh poller has no images yet")
}

// A nil registerImageProvider must be tolerated — callers that don't need
// the API wiring (or are running without an HTTP server) should not crash.
func TestStartVicoHomePolling_NilRegisterIsTolerated(t *testing.T) {
	t.Parallel()
	settings := &conf.Settings{
		VicoHome: conf.VicoHomeSettings{
			Enabled:  true,
			Email:    "user@example.com",
			Password: "secret",
		},
	}
	mqttClient := &fakeMQTTClient{connected: true}
	var wg sync.WaitGroup
	quitChan := make(chan struct{})
	defer close(quitChan)

	assert.NotPanics(t, func() {
		startVicoHomePolling(&wg, settings, mqttClient, nil, quitChan)
	})
}
