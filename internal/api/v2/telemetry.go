// telemetry.go - Handles telemetry and logging endpoints
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
)

// TelemetryLogRequest represents a client-side log/error report
type TelemetryLogRequest struct {
	Level    string                 `json:"level"`    // Log level (debug, info, warn, error)
	Category string                 `json:"category"` // Logger category
	Message  string                 `json:"message"`  // Error/log message
	Stack    string                 `json:"stack"`    // Stack trace (for errors)
	Context  map[string]interface{} `json:"context"`  // Additional context
}

// initTelemetryRoutes registers telemetry-related routes
func (c *Controller) initTelemetryRoutes() {
	telemetryGroup := c.Group.Group("/telemetry")

	// POST /api/v2/telemetry/log - Log client-side errors/events
	telemetryGroup.POST("/log", c.LogTelemetry)
}

// LogTelemetry handles client-side telemetry logging
//
// POST /api/v2/telemetry/log
//
// Accepts structured log events from the frontend for server-side logging
// and optional persistence. This enables centralized error tracking without
// external services like Sentry.
func (c *Controller) LogTelemetry(ctx echo.Context) error {
	var req TelemetryLogRequest

	// Parse request body
	if err := ctx.Bind(&req); err != nil {
		return c.HandleError(ctx, err, "Invalid telemetry request", http.StatusBadRequest)
	}

	// Validate required fields
	if req.Level == "" || req.Category == "" || req.Message == "" {
		return c.HandleError(ctx, nil, "Missing required fields: level, category, message", http.StatusBadRequest)
	}

	// Convert context to JSON for structured logging
	contextJSON, _ := json.Marshal(req.Context)

	// Map log level to slog level
	var level slog.Level
	switch req.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Log the telemetry event
	c.logAPIRequest(
		ctx,
		level,
		"Client telemetry",
		"category", req.Category,
		"message", req.Message,
		"context", string(contextJSON),
	)

	// For errors, include stack trace if available
	if req.Level == "error" && req.Stack != "" {
		c.logAPIRequest(
			ctx,
			slog.LevelError,
			"Client error stack trace",
			"category", req.Category,
			"stack", req.Stack,
		)
	}

	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Telemetry logged successfully",
	})
}
