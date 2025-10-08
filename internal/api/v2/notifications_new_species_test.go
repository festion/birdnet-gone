package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tphakala/birdnet-go/internal/conf"
	"github.com/tphakala/birdnet-go/internal/notification"
)

// parseJSONResponse unmarshals JSON response body into target struct
func parseJSONResponse(body []byte, target interface{}) error {
	return json.Unmarshal(body, target)
}

func TestCreateTestNewSpeciesNotification_ServiceNotInitialized(t *testing.T) {
	// Skip this test if notification service is already initialized
	// This test specifically validates the uninitialized service error path
	if notification.IsInitialized() {
		t.Skip("notification service already initialized; skipping service-not-initialized path")
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v2/notifications/test/new-species", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	controller := &Controller{}

	err := controller.CreateTestNewSpeciesNotification(c)
	require.NoError(t, err)

	// Verify we get the expected service unavailable error
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Contains(t, rec.Body.String(), "Notification service not available")
}

func TestCreateTestNewSpeciesNotification_Success(t *testing.T) {
	// Initialize notification service for testing using correct API
	config := &notification.ServiceConfig{
		Debug:              true,
		MaxNotifications:   100,
		CleanupInterval:    30 * time.Minute,
		RateLimitWindow:    1 * time.Minute,
		RateLimitMaxEvents: 10,
	}

	// Try to set up isolated service for testing
	service := notification.NewService(config)
	err := notification.SetServiceForTesting(service)
	if err != nil {
		// Service already exists, use it
		service = notification.GetService()
		require.NotNil(t, service, "Expected notification service to be available")
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v2/notifications/test/new-species", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	controller := &Controller{}
	controller.Settings = &conf.Settings{}
	controller.Settings.Security.Host = "localhost"
	controller.Settings.WebServer.Port = "8080"
	controller.Settings.Main.TimeAs24h = true
	// Set default templates from config.yaml
	controller.Settings.Notification.Templates.NewSpecies.Title = "New Species: {{.CommonName}}"
	controller.Settings.Notification.Templates.NewSpecies.Message = "First detection of {{.CommonName}} ({{.ScientificName}}) with {{.ConfidencePercent}}% confidence at {{.DetectionTime}}. View: {{.DetectionURL}}"

	err = controller.CreateTestNewSpeciesNotification(c)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Parse response to verify notification structure
	var response notification.Notification
	err = parseJSONResponse(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify notification fields match detection_consumer.go patterns
	assert.Equal(t, notification.TypeDetection, response.Type)
	assert.Equal(t, notification.PriorityHigh, response.Priority)
	assert.Equal(t, "detection", response.Component)
	assert.Equal(t, notification.StatusUnread, response.Status)
	// Verify default title format matches config.yaml template
	assert.Equal(t, "New Species: Test Bird Species", response.Title)
	// Verify message matches default template: includes confidence and time, not location
	assert.Contains(t, response.Message, "First detection of Test Bird Species")
	assert.Contains(t, response.Message, "Testus birdicus")
	assert.Contains(t, response.Message, "99% confidence")
	assert.NotEmpty(t, response.ID)
	assert.False(t, response.Timestamp.IsZero())

	// Verify metadata fields match detection_consumer.go
	require.NotNil(t, response.Metadata)
	assert.Equal(t, "Test Bird Species", response.Metadata["species"])
	assert.Equal(t, "Testus birdicus", response.Metadata["scientific_name"])
	assert.InDelta(t, 0.99, response.Metadata["confidence"], 0.001)
	assert.Equal(t, "Test Location (Sample Data)", response.Metadata["location"])
	assert.Equal(t, true, response.Metadata["is_new_species"])
	assert.InDelta(t, 0, response.Metadata["days_since_first_seen"], 0.001)

	// Verify 24-hour expiry
	require.NotNil(t, response.ExpiresAt)
	expectedExpiry := response.Timestamp.Add(24 * time.Hour)
	assert.WithinDuration(t, expectedExpiry, *response.ExpiresAt, time.Second)
}
