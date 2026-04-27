package analysis

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tphakala/birdnet-go/internal/conf"
	"github.com/tphakala/birdnet-go/internal/mqtt"
)

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

	startVicoHomePolling(&wg, settings, mqttClient, quitChan)

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

			startVicoHomePolling(&wg, settings, mqttClient, quitChan)

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

	startVicoHomePolling(&wg, settings, nil, quitChan)

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

	startVicoHomePolling(&wg, settings, mqttClient, quitChan)

	assertNoGoroutineStarted(t, &wg)
	assert.False(t, mqttClient.IsConnected(), "fake remained disconnected")
}
