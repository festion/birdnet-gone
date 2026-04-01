package vicohome

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test constants to avoid magic strings
const (
	testEmail    = "test@example.com"
	testPassword = "testpass123" //nolint:gosec // test credential, not real
	testToken    = "fake-token-abc123"
)

// newTestServer creates an httptest.Server with the provided handler and returns both the server and a client configured to use it.
func newTestServer(t *testing.T, handler http.Handler) (*httptest.Server, *Client) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := NewClientWithBaseURL(testEmail, testPassword, srv.URL)
	return srv, client
}

// writeJSON is a test helper that writes JSON to the response writer.
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// writeLoginSuccess writes a successful login response with the test token.
func writeLoginSuccess(w http.ResponseWriter) {
	resp := loginResponse{Result: resultSuccess}
	resp.Data.Token.Token = testToken
	writeJSON(w, resp)
}

// writeEmptyEvents writes a successful empty events response.
func writeEmptyEvents(w http.ResponseWriter) {
	resp := eventsResponse{}
	resp.Data.List = []rawEvent{}
	writeJSON(w, resp)
}

func TestAuthenticate_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc(loginPath, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)

		var req loginRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.NoError(t, err)
		assert.Equal(t, testEmail, req.Email)
		assert.Equal(t, testPassword, req.Password)
		assert.Equal(t, loginTypeDefault, req.LoginType)

		writeLoginSuccess(w)
	})

	_, client := newTestServer(t, mux)
	err := client.Authenticate(context.Background())
	require.NoError(t, err)

	assert.True(t, client.hasValidToken())
	assert.Equal(t, testToken, client.getToken())
}

func TestAuthenticate_FailedLogin(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc(loginPath, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{
			"result": -1,
			"msg":    "invalid credentials",
		})
	})

	_, client := newTestServer(t, mux)
	err := client.Authenticate(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "VicoHome login failed")
	assert.False(t, client.hasValidToken())
}

func TestAuthenticate_EmptyToken(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc(loginPath, func(w http.ResponseWriter, _ *http.Request) {
		resp := loginResponse{Result: resultSuccess}
		resp.Data.Token.Token = "" // Empty token
		writeJSON(w, resp)
	})

	_, client := newTestServer(t, mux)
	err := client.Authenticate(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty token")
}

func TestAuthenticate_NetworkError(t *testing.T) {
	// Create a client pointing to a non-existent server
	client := NewClientWithBaseURL(testEmail, testPassword, "http://127.0.0.1:1")
	err := client.Authenticate(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "login request failed")
}

func TestTokenCaching(t *testing.T) {
	var authCount atomic.Int32

	mux := http.NewServeMux()
	mux.HandleFunc(loginPath, func(w http.ResponseWriter, _ *http.Request) {
		authCount.Add(1)
		writeLoginSuccess(w)
	})
	mux.HandleFunc(eventsPath, func(w http.ResponseWriter, _ *http.Request) {
		writeEmptyEvents(w)
	})

	_, client := newTestServer(t, mux)

	// First call should authenticate
	_, err := client.GetEvents(context.Background(), time.Now().Add(-time.Hour))
	require.NoError(t, err)
	assert.Equal(t, int32(1), authCount.Load())

	// Second call should reuse cached token
	_, err = client.GetEvents(context.Background(), time.Now().Add(-time.Hour))
	require.NoError(t, err)
	assert.Equal(t, int32(1), authCount.Load(), "second call should not re-authenticate")
}

func TestAutoRetryOnAuthError(t *testing.T) {
	var callCount atomic.Int32

	mux := http.NewServeMux()
	mux.HandleFunc(loginPath, func(w http.ResponseWriter, _ *http.Request) {
		writeLoginSuccess(w)
	})
	mux.HandleFunc(eventsPath, func(w http.ResponseWriter, _ *http.Request) {
		count := callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")

		if count == 1 {
			// First call returns auth error
			writeJSON(w, map[string]any{
				"code": -1024,
				"data": map[string]any{"list": []any{}},
			})
			return
		}

		// Subsequent calls succeed
		writeEmptyEvents(w)
	})

	_, client := newTestServer(t, mux)

	events, err := client.GetEvents(context.Background(), time.Now().Add(-time.Hour))
	require.NoError(t, err)
	assert.Empty(t, events)
	assert.Equal(t, int32(2), callCount.Load(), "should have retried after auth error")
}

func TestGetEvents_WithBirdDetections(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc(loginPath, func(w http.ResponseWriter, _ *http.Request) {
		writeLoginSuccess(w)
	})
	mux.HandleFunc(eventsPath, func(w http.ResponseWriter, r *http.Request) {
		// Verify authorization header
		assert.Equal(t, testToken, r.Header.Get("Authorization"))

		resp := eventsResponse{}
		resp.Data.List = []rawEvent{
			{
				TraceID:      "trace-001",
				Timestamp:    1700000000,
				ImageURL:     "https://img.example.com/full.jpg",
				VideoURL:     "https://vid.example.com/clip.mp4",
				DeviceName:   "Bird Feeder Cam",
				SerialNumber: "SN123456",
				Keyshots: []rawKeyshot{
					{ImageURL: "https://img.example.com/keyshot.jpg"},
				},
				SubcategoryInfoList: []rawSubcategory{
					{
						ObjectType:  "bird",
						ObjectName:  "Northern Cardinal",
						BirdStdName: "Cardinalis cardinalis", //nolint:misspell // scientific name
						Confidence:  0.95,
					},
				},
			},
			{
				TraceID:      "trace-002",
				Timestamp:    1700000060,
				ImageURL:     "https://img.example.com/motion.jpg",
				DeviceName:   "Bird Feeder Cam",
				SerialNumber: "SN123456",
				SubcategoryInfoList: []rawSubcategory{
					{
						ObjectType: "animal",
						ObjectName: "cat",
						Confidence: 0.8,
					},
				},
			},
		}
		writeJSON(w, resp)
	})

	_, client := newTestServer(t, mux)

	events, err := client.GetEvents(context.Background(), time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.Len(t, events, 2)

	// First event: bird detection
	assert.Equal(t, "trace-001", events[0].TraceID)
	assert.Equal(t, int64(1700000000), events[0].Timestamp)
	assert.Equal(t, "Northern Cardinal", events[0].BirdName)
	assert.Equal(t, "Cardinalis cardinalis", events[0].BirdLatin) //nolint:misspell // scientific name
	assert.InDelta(t, 0.95, events[0].BirdConfidence, 0.001)
	assert.Equal(t, "https://img.example.com/keyshot.jpg", events[0].KeyShotURL)
	assert.Equal(t, "https://img.example.com/full.jpg", events[0].ImageURL)
	assert.Equal(t, "https://vid.example.com/clip.mp4", events[0].VideoURL)
	assert.Equal(t, "Bird Feeder Cam", events[0].DeviceName)
	assert.Equal(t, "SN123456", events[0].SerialNumber)

	// Second event: non-bird (no bird fields populated)
	assert.Equal(t, "trace-002", events[1].TraceID)
	assert.Empty(t, events[1].BirdName)
	assert.Empty(t, events[1].BirdLatin)
	assert.InDelta(t, 0.0, events[1].BirdConfidence, 0.001)
	assert.Empty(t, events[1].KeyShotURL)
}

func TestGetEvents_DefaultSince(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc(loginPath, func(w http.ResponseWriter, _ *http.Request) {
		writeLoginSuccess(w)
	})
	mux.HandleFunc(eventsPath, func(w http.ResponseWriter, r *http.Request) {
		var req eventsRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.NoError(t, err)

		// When since is zero, should look back 24 hours
		assert.NotEmpty(t, req.StartTimestamp)
		assert.NotEmpty(t, req.EndTimestamp)
		assert.Equal(t, eventsLanguage, req.Language)
		assert.Equal(t, eventsCountry, req.CountryNo)

		writeEmptyEvents(w)
	})

	_, client := newTestServer(t, mux)

	events, err := client.GetEvents(context.Background(), time.Time{}) // zero time
	require.NoError(t, err)
	assert.Empty(t, events)
}

func TestGetEvents_InvalidJSON(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc(loginPath, func(w http.ResponseWriter, _ *http.Request) {
		writeLoginSuccess(w)
	})
	mux.HandleFunc(eventsPath, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{invalid json`))
	})

	_, client := newTestServer(t, mux)

	_, err := client.GetEvents(context.Background(), time.Now().Add(-time.Hour))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse events response")
}

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected bool
	}{
		{
			name:     "auth error code -1024",
			body:     `{"code": -1024}`,
			expected: true,
		},
		{
			name:     "auth error result -1025",
			body:     `{"result": -1025}`,
			expected: true,
		},
		{
			name:     "auth error code -1026",
			body:     `{"code": -1026}`,
			expected: true,
		},
		{
			name:     "auth error code -1027",
			body:     `{"code": -1027}`,
			expected: true,
		},
		{
			name:     "success code 0",
			body:     `{"code": 0}`,
			expected: false,
		},
		{
			name:     "other error code",
			body:     `{"code": -500}`,
			expected: false,
		},
		{
			name:     "invalid JSON",
			body:     `{invalid}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAuthError([]byte(tt.body))
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseEvent(t *testing.T) {
	raw := &rawEvent{
		TraceID:      "t-123",
		Timestamp:    1700000000,
		ImageURL:     "https://img.example.com/full.jpg",
		VideoURL:     "https://vid.example.com/clip.mp4",
		DeviceName:   "Feeder",
		SerialNumber: "SN-001",
		Keyshots: []rawKeyshot{
			{ImageURL: "https://img.example.com/key.jpg"},
			{ImageURL: "https://img.example.com/key2.jpg"}, // should only use first
		},
		SubcategoryInfoList: []rawSubcategory{
			{ObjectType: "person", ObjectName: "human", Confidence: 0.9},
			{ObjectType: objectTypeBird, ObjectName: "Blue Jay", BirdStdName: "Cyanocitta cristata", Confidence: 0.88},
		},
	}

	ev := parseEvent(raw)

	assert.Equal(t, "t-123", ev.TraceID)
	assert.Equal(t, int64(1700000000), ev.Timestamp)
	assert.Equal(t, "Blue Jay", ev.BirdName)
	assert.Equal(t, "Cyanocitta cristata", ev.BirdLatin)
	assert.InDelta(t, 0.88, ev.BirdConfidence, 0.001)
	assert.Equal(t, "https://img.example.com/key.jpg", ev.KeyShotURL)
	assert.Equal(t, "https://img.example.com/full.jpg", ev.ImageURL)
	assert.Equal(t, "https://vid.example.com/clip.mp4", ev.VideoURL)
}

func TestParseEvent_NoBirdSubcategory(t *testing.T) {
	raw := &rawEvent{
		TraceID:   "t-456",
		Timestamp: 1700000100,
		SubcategoryInfoList: []rawSubcategory{
			{ObjectType: "animal", ObjectName: "squirrel", Confidence: 0.75},
		},
	}

	ev := parseEvent(raw)

	assert.Equal(t, "t-456", ev.TraceID)
	assert.Empty(t, ev.BirdName)
	assert.Empty(t, ev.BirdLatin)
	assert.InDelta(t, 0.0, ev.BirdConfidence, 0.001)
}

func TestParseEvent_NoKeyshots(t *testing.T) {
	raw := &rawEvent{
		TraceID:  "t-789",
		Keyshots: nil,
	}

	ev := parseEvent(raw)
	assert.Empty(t, ev.KeyShotURL)
}

func TestContextCancellation(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc(loginPath, func(w http.ResponseWriter, _ *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		writeLoginSuccess(w)
	})

	_, client := newTestServer(t, mux)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := client.Authenticate(ctx)
	require.Error(t, err)
}
