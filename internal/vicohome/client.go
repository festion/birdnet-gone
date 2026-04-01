// Package vicohome provides a Go client for the VicoHome cloud API,
// MQTT publishing with Home Assistant auto-discovery, and a polling loop
// that replaces the Python vicohome-bridge script.
package vicohome

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tphakala/birdnet-go/internal/errors"
)

// VicoHome API constants
const (
	// DefaultBaseURL is the base URL for the US VicoHome cloud API.
	DefaultBaseURL = "https://api-us.vicohome.io"

	// loginPath is the API path for authentication.
	loginPath = "/account/login"

	// eventsPath is the API path for fetching library events.
	eventsPath = "/library/newselectlibrary"

	// tokenExpiryDuration is how long a cached token is considered valid.
	// VicoHome tokens last 24 hours; we expire at 23 to avoid edge cases.
	tokenExpiryDuration = 23 * time.Hour

	// httpTimeout is the default timeout for HTTP requests to the VicoHome API.
	httpTimeout = 15 * time.Second

	// loginTypeDefault is the default login type for VicoHome authentication.
	loginTypeDefault = 0

	// resultSuccess indicates a successful API response.
	resultSuccess = 0

	// defaultLookbackHours is how far back to fetch events by default.
	defaultLookbackHours = 24

	// eventsLanguage is the language parameter for event queries.
	eventsLanguage = "en"

	// eventsCountry is the country parameter for event queries.
	eventsCountry = "US"

	// objectTypeBird is the subcategory object type for bird detections.
	objectTypeBird = "bird"
)

// Auth error codes returned by VicoHome API when the token is expired or invalid.
var authErrorCodes = map[int]bool{
	-1024: true,
	-1025: true,
	-1026: true,
	-1027: true,
}

// Event represents a single bird detection event from the VicoHome API.
type Event struct {
	TraceID        string  // unique identifier for the event
	Timestamp      int64   // Unix timestamp of the detection
	BirdName       string  // common name of the detected bird
	BirdLatin      string  // scientific/Latin name of the bird
	BirdConfidence float64 // confidence score of the detection (0-1)
	KeyShotURL     string  // URL of the key shot image
	ImageURL       string  // URL of the full image
	VideoURL       string  // URL of the video clip
	DeviceName     string  // name of the camera device
	SerialNumber   string  // serial number of the camera device
}

// loginRequest is the JSON body for the VicoHome login endpoint.
type loginRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	LoginType int    `json:"loginType"`
}

// loginResponse represents the JSON response from the login endpoint.
type loginResponse struct {
	Result int    `json:"result"`
	Msg    string `json:"msg"`
	Data   struct {
		Token struct {
			Token string `json:"token"`
		} `json:"token"`
	} `json:"data"`
}

// eventsRequest is the JSON body for the events endpoint.
type eventsRequest struct {
	StartTimestamp string `json:"startTimestamp"`
	EndTimestamp   string `json:"endTimestamp"`
	Language       string `json:"language"`
	CountryNo      string `json:"countryNo"`
}

// eventsResponse represents the top-level JSON response from the events endpoint.
type eventsResponse struct {
	Code   int `json:"code"`
	Result int `json:"result"`
	Data   struct {
		List []rawEvent `json:"list"`
	} `json:"data"`
}

// rawEvent represents a single event in the API response.
type rawEvent struct {
	TraceID             string           `json:"traceId"`
	Timestamp           int64            `json:"timestamp"`
	ImageURL            string           `json:"imageUrl"`
	VideoURL            string           `json:"videoUrl"`
	DeviceName          string           `json:"deviceName"`
	SerialNumber        string           `json:"serialNumber"`
	Keyshots            []rawKeyshot     `json:"keyshots"`
	SubcategoryInfoList []rawSubcategory `json:"subcategoryInfoList"`
}

// rawKeyshot represents a keyshot entry in the API response.
type rawKeyshot struct {
	ImageURL string `json:"imageUrl"`
}

// rawSubcategory represents a subcategory info entry in the API response.
type rawSubcategory struct {
	ObjectType  string  `json:"objectType"`
	ObjectName  string  `json:"objectName"`
	BirdStdName string  `json:"birdStdName"`
	Confidence  float64 `json:"confidence"`
}

// Client is the VicoHome API client that handles authentication and event fetching.
type Client struct {
	email      string
	password   string
	baseURL    string
	httpClient *http.Client

	mu          sync.RWMutex
	token       string
	tokenExpiry time.Time
}

// NewClient creates a new VicoHome API client with the given credentials.
func NewClient(email, password string) *Client {
	return &Client{
		email:    email,
		password: password,
		baseURL:  DefaultBaseURL,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
	}
}

// NewClientWithBaseURL creates a new VicoHome API client with a custom base URL.
// This is primarily used for testing with httptest.NewServer.
func NewClientWithBaseURL(email, password, baseURL string) *Client {
	c := NewClient(email, password)
	c.baseURL = baseURL
	return c
}

// Authenticate performs login against the VicoHome API and caches the token.
func (c *Client) Authenticate(ctx context.Context) error {
	reqBody := loginRequest{
		Email:     c.email,
		Password:  c.password,
		LoginType: loginTypeDefault,
	}

	bodyBytes, err := json.Marshal(reqBody) //nolint:gosec // G117: password field is required for authentication
	if err != nil {
		return errors.Newf("failed to marshal login request: %v", err).
			Component("vicohome").
			Category(errors.CategoryNetwork).
			Context("operation", "authenticate").
			Build()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+loginPath, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return errors.Newf("failed to create login request: %v", err).
			Component("vicohome").
			Category(errors.CategoryNetwork).
			Context("operation", "authenticate").
			Build()
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Newf("login request failed: %v", err).
			Component("vicohome").
			Category(errors.CategoryNetwork).
			Context("operation", "authenticate").
			Build()
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Newf("failed to read login response: %v", err).
			Component("vicohome").
			Category(errors.CategoryNetwork).
			Context("operation", "authenticate").
			Build()
	}

	var loginResp loginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return errors.Newf("failed to parse login response: %v", err).
			Component("vicohome").
			Category(errors.CategoryNetwork).
			Context("operation", "authenticate").
			Build()
	}

	if loginResp.Result != resultSuccess {
		return errors.Newf("VicoHome login failed: %s (code %d)", loginResp.Msg, loginResp.Result).
			Component("vicohome").
			Category(errors.CategoryNetwork).
			Context("operation", "authenticate").
			Context("result_code", loginResp.Result).
			Build()
	}

	token := loginResp.Data.Token.Token
	if token == "" {
		return errors.Newf("VicoHome login returned empty token").
			Component("vicohome").
			Category(errors.CategoryNetwork).
			Context("operation", "authenticate").
			Build()
	}

	c.mu.Lock()
	c.token = token
	c.tokenExpiry = time.Now().Add(tokenExpiryDuration)
	c.mu.Unlock()

	return nil
}

// hasValidToken reports whether the client has a non-expired cached token.
func (c *Client) hasValidToken() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token != "" && time.Now().Before(c.tokenExpiry)
}

// getToken returns the current cached token.
func (c *Client) getToken() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token
}

// GetEvents fetches bird detection events from the VicoHome API since the given time.
// If since is zero, events from the last 24 hours are fetched.
func (c *Client) GetEvents(ctx context.Context, since time.Time) ([]Event, error) {
	if since.IsZero() {
		since = time.Now().UTC().Add(-time.Duration(defaultLookbackHours) * time.Hour)
	}

	body, err := c.doAuthenticatedRequest(ctx, eventsPath, eventsRequest{
		StartTimestamp: fmt.Sprintf("%d", since.Unix()),
		EndTimestamp:   fmt.Sprintf("%d", time.Now().UTC().Unix()),
		Language:       eventsLanguage,
		CountryNo:      eventsCountry,
	})
	if err != nil {
		return nil, err
	}

	var evResp eventsResponse
	if err := json.Unmarshal(body, &evResp); err != nil {
		return nil, errors.Newf("failed to parse events response: %v", err).
			Component("vicohome").
			Category(errors.CategoryNetwork).
			Context("operation", "get_events").
			Build()
	}

	events := make([]Event, 0, len(evResp.Data.List))
	for i := range evResp.Data.List {
		events = append(events, parseEvent(&evResp.Data.List[i]))
	}

	return events, nil
}

// doAuthenticatedRequest performs an authenticated POST request with auto-retry on auth errors.
func (c *Client) doAuthenticatedRequest(ctx context.Context, path string, reqBody any) ([]byte, error) {
	// Ensure we have a valid token
	if !c.hasValidToken() {
		if err := c.Authenticate(ctx); err != nil {
			return nil, err
		}
	}

	body, err := c.doPostWithToken(ctx, path, reqBody)
	if err != nil {
		return nil, err
	}

	// Check for auth error codes and retry once
	if isAuthError(body) {
		if err := c.Authenticate(ctx); err != nil {
			return nil, err
		}
		return c.doPostWithToken(ctx, path, reqBody)
	}

	return body, nil
}

// doPostWithToken performs a POST request with the current auth token.
func (c *Client) doPostWithToken(ctx context.Context, path string, reqBody any) ([]byte, error) {
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, errors.Newf("failed to marshal request body: %v", err).
			Component("vicohome").
			Category(errors.CategoryNetwork).
			Context("operation", "api_request").
			Context("path", path).
			Build()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, errors.Newf("failed to create request: %v", err).
			Component("vicohome").
			Category(errors.CategoryNetwork).
			Context("operation", "api_request").
			Context("path", path).
			Build()
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", c.getToken())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, errors.Newf("API request failed: %v", err).
			Component("vicohome").
			Category(errors.CategoryNetwork).
			Context("operation", "api_request").
			Context("path", path).
			Build()
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Newf("failed to read response: %v", err).
			Component("vicohome").
			Category(errors.CategoryNetwork).
			Context("operation", "api_request").
			Context("path", path).
			Build()
	}

	return body, nil
}

// isAuthError checks if the response body contains a VicoHome auth error code.
func isAuthError(body []byte) bool {
	var resp struct {
		Code   int `json:"code"`
		Result int `json:"result"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return false
	}
	// Check both code and result fields, as the API uses them inconsistently
	return authErrorCodes[resp.Code] || authErrorCodes[resp.Result]
}

// parseEvent converts a raw API event into a normalized Event struct.
func parseEvent(raw *rawEvent) Event {
	ev := Event{
		TraceID:      raw.TraceID,
		Timestamp:    raw.Timestamp,
		ImageURL:     raw.ImageURL,
		VideoURL:     raw.VideoURL,
		DeviceName:   raw.DeviceName,
		SerialNumber: raw.SerialNumber,
	}

	// Extract bird info from subcategory list
	for i := range raw.SubcategoryInfoList {
		sub := &raw.SubcategoryInfoList[i]
		if sub.ObjectType == objectTypeBird {
			ev.BirdName = sub.ObjectName
			ev.BirdLatin = sub.BirdStdName
			ev.BirdConfidence = sub.Confidence
			break
		}
	}

	// Extract keyshot URL
	if len(raw.Keyshots) > 0 {
		ev.KeyShotURL = raw.Keyshots[0].ImageURL
	}

	return ev
}
