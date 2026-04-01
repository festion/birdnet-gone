// api_test.go: Package api provides tests for API v2 endpoints.

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
	"github.com/tphakala/birdnet-go/internal/datastore"
)

// assertSystemMetrics validates system metrics in health check response.
func assertSystemMetrics(t *testing.T, metricsMap map[string]any) {
	t.Helper()
	if cpu, cpuExists := metricsMap["cpu_usage"]; cpuExists {
		cpuValue, ok := cpu.(float64)
		assert.True(t, ok, "CPU usage should be a number")
		assert.GreaterOrEqual(t, cpuValue, float64(0), "CPU usage should be non-negative")
		assert.LessOrEqual(t, cpuValue, float64(100), "CPU usage should be <= 100%")
	}
	if memory, memExists := metricsMap["memory_usage"]; memExists {
		memValue, ok := memory.(float64)
		assert.True(t, ok, "Memory usage should be a number")
		assert.GreaterOrEqual(t, memValue, float64(0), "Memory usage should be non-negative")
	}
}

// assertDiskSpace validates disk space metrics in health check response.
func assertDiskSpace(t *testing.T, diskSpace any) {
	t.Helper()
	diskMap, ok := diskSpace.(map[string]any)
	assert.True(t, ok, "Disk space should be an object")
	if total, totalExists := diskMap["total"]; totalExists {
		assert.GreaterOrEqual(t, total.(float64), float64(0), "Total disk space should be non-negative")
	}
	if free, freeExists := diskMap["free"]; freeExists {
		assert.GreaterOrEqual(t, free.(float64), float64(0), "Free disk space should be non-negative")
	}
}

// assertUptime validates uptime field in health check response.
func assertUptime(t *testing.T, uptime any) {
	t.Helper()
	switch v := uptime.(type) {
	case float64:
		assert.GreaterOrEqual(t, v, float64(0), "Uptime should be non-negative")
	case string:
		assert.NotEmpty(t, v, "Uptime string should not be empty")
	default:
		assert.Fail(t, "Uptime should be a number or string")
	}
}

// TestHealthCheck tests the health check endpoint
func TestHealthCheck(t *testing.T) {
	// Setup
	e, mockDS, controller := setupTestEnvironment(t)

	// Setup mock expectations for database check
	mockDS.On("GetLastDetections", 1).Return([]datastore.Note{}, nil)

	// Setup mock expectations for detection count (today's date, full day)
	today := time.Now().Format("2006-01-02")
	mockDS.On("CountHourlyDetections", today, "0", 24).Return(int64(0), nil)

	// Add system metrics to the controller settings
	controller.Settings.Version = "1.2.3"
	controller.Settings.BuildDate = "2023-05-15"

	// Create a request to the health check endpoint
	req := httptest.NewRequest(http.MethodGet, "/api/v2/health", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/v2/health")

	// Record start time to measure response time
	startTime := time.Now()

	// Test
	require.NoError(t, controller.HealthCheck(c))
	{
		// Calculate response time
		responseTime := time.Since(startTime)

		// Check response time is reasonable (under 100ms for a simple health check)
		assert.Less(t, responseTime.Milliseconds(), int64(100), "Health check should respond quickly")

		// Check response code
		assert.Equal(t, http.StatusOK, rec.Code)

		// Check content type header
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

		// Parse response body
		var response map[string]any
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		// Check required fields
		assert.Contains(t, response, "status", "Response must contain status field")
		assert.Equal(t, "1.2.3", response["version"], "Version should match controller settings")
		assert.Equal(t, "2023-05-15", response["build_date"], "Build date should match controller settings")

		// Check audio health fields are present
		assert.Contains(t, response, "audio_active", "Response must contain audio_active field")
		assert.Contains(t, response, "last_audio_level", "Response must contain last_audio_level field")
		assert.Contains(t, response, "last_audio_level_age_seconds", "Response must contain last_audio_level_age_seconds field")
		assert.Contains(t, response, "detections_today", "Response must contain detections_today field")

		// Check database status if present
		if dbStatus, exists := response["database_status"]; exists {
			assert.Equal(t, "connected", dbStatus, "Database status should be 'connected'")
		}

		// Check system metrics if present
		if metrics, exists := response["system"]; exists {
			metricsMap, ok := metrics.(map[string]any)
			assert.True(t, ok, "System metrics should be an object")
			assertSystemMetrics(t, metricsMap)
			if diskSpace, diskExists := metricsMap["disk_space"]; diskExists {
				assertDiskSpace(t, diskSpace)
			}
		}

		// Check uptime if present
		if uptime, exists := response["uptime"]; exists {
			assertUptime(t, uptime)
		}

		// Check for additional fields that might be useful
		if env, exists := response["environment"]; exists {
			assert.IsType(t, "", env, "Environment should be a string")
		}

		if timestamp, exists := response["timestamp"]; exists {
			_, err := time.Parse(time.RFC3339, timestamp.(string))
			require.NoError(t, err, "Timestamp should be in RFC3339 format")
		}
	}

	// Verify mock expectations
	mockDS.AssertExpectations(t)
}

// TestHealthCheckEnhanced tests the enhanced health endpoint with audio state reporting
func TestHealthCheckEnhanced(t *testing.T) {
	t.Run("healthy_with_active_audio", func(t *testing.T) {
		e, mockDS, controller := setupTestEnvironment(t)

		// Setup mock expectations
		mockDS.On("GetLastDetections", 1).Return([]datastore.Note{}, nil)
		today := time.Now().Format("2006-01-02")
		mockDS.On("CountHourlyDetections", today, "0", 24).Return(int64(42), nil)

		// Simulate active audio with a non-zero level
		controller.UpdateAudioHealth(75)

		req := httptest.NewRequest(http.MethodGet, "/api/v2/health", http.NoBody)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		require.NoError(t, controller.HealthCheck(c))
		assert.Equal(t, http.StatusOK, rec.Code)

		var response map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

		// Audio health fields must be present with correct values
		assert.Equal(t, "healthy", response["status"])
		assert.Equal(t, true, response["audio_active"])
		assert.InDelta(t, 75, response["last_audio_level"], 0)
		assert.InDelta(t, 42, response["detections_today"], 0)

		// Age should be very small (just set)
		ageSeconds, ok := response["last_audio_level_age_seconds"].(float64)
		require.True(t, ok, "last_audio_level_age_seconds should be a number")
		assert.LessOrEqual(t, ageSeconds, float64(2), "Audio level age should be recent")

		mockDS.AssertExpectations(t)
	})

	t.Run("degraded_with_silence", func(t *testing.T) {
		e, mockDS, controller := setupTestEnvironment(t)

		mockDS.On("GetLastDetections", 1).Return([]datastore.Note{}, nil)
		today := time.Now().Format("2006-01-02")
		mockDS.On("CountHourlyDetections", today, "0", 24).Return(int64(0), nil)

		// Simulate silence (level 0, which means possible BOYA desync)
		controller.UpdateAudioHealth(0)

		req := httptest.NewRequest(http.MethodGet, "/api/v2/health", http.NoBody)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		require.NoError(t, controller.HealthCheck(c))

		var response map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

		assert.Equal(t, "degraded", response["status"], "Silence should report degraded status")
		assert.Equal(t, true, response["audio_active"])
		assert.InDelta(t, 0, response["last_audio_level"], 0)

		mockDS.AssertExpectations(t)
	})

	t.Run("unhealthy_no_audio", func(t *testing.T) {
		e, mockDS, controller := setupTestEnvironment(t)

		mockDS.On("GetLastDetections", 1).Return([]datastore.Note{}, nil)
		today := time.Now().Format("2006-01-02")
		mockDS.On("CountHourlyDetections", today, "0", 24).Return(int64(0), nil)

		// No audio health updates — audioActive remains false (default)

		req := httptest.NewRequest(http.MethodGet, "/api/v2/health", http.NoBody)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		require.NoError(t, controller.HealthCheck(c))

		var response map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

		assert.Equal(t, "unhealthy", response["status"], "No audio data should report unhealthy")
		assert.Equal(t, false, response["audio_active"])

		mockDS.AssertExpectations(t)
	})

	t.Run("unhealthy_stale_audio", func(t *testing.T) {
		e, mockDS, controller := setupTestEnvironment(t)

		mockDS.On("GetLastDetections", 1).Return([]datastore.Note{}, nil)
		today := time.Now().Format("2006-01-02")
		mockDS.On("CountHourlyDetections", today, "0", 24).Return(int64(5), nil)

		// Simulate stale audio data by setting the time far in the past
		controller.audioHealthMu.Lock()
		controller.lastAudioLevel = 50
		controller.lastAudioLevelTime = time.Now().Add(-10 * time.Minute)
		controller.audioActive = true
		controller.audioHealthMu.Unlock()

		req := httptest.NewRequest(http.MethodGet, "/api/v2/health", http.NoBody)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		require.NoError(t, controller.HealthCheck(c))

		var response map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))

		assert.Equal(t, "unhealthy", response["status"], "Stale audio data should report unhealthy")
		assert.Equal(t, true, response["audio_active"])

		// Age should reflect the staleness
		ageSeconds, ok := response["last_audio_level_age_seconds"].(float64)
		require.True(t, ok, "last_audio_level_age_seconds should be a number")
		assert.GreaterOrEqual(t, ageSeconds, float64(300), "Audio level age should be >= 5 minutes")

		mockDS.AssertExpectations(t)
	})
}

// TestHandleError tests error handling functionality
func TestHandleError(t *testing.T) {
	// Setup
	e, _, controller := setupTestEnvironment(t)

	// Create a request context
	req := httptest.NewRequest(http.MethodGet, "/api/v2/health", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Test error handling
	err := controller.HandleError(c, echo.NewHTTPError(http.StatusBadRequest, "Test error"),
		"Error message", http.StatusBadRequest)

	// Assertions
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	// Parse response body
	var response ErrorResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Check response content
	assert.Equal(t, "code=400, message=Test error", response.Error)
	assert.Equal(t, "Error message", response.Message)
	assert.Equal(t, http.StatusBadRequest, response.Code)
}
