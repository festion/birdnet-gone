package imagecache

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testImageData is a minimal valid JPEG header used as synthetic image content.
var testImageData = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F'}

// newTestServer creates an httptest server that serves testImageData on every
// request and returns the server along with the number of requests received.
func newTestServer(t *testing.T) (*httptest.Server, *atomic.Int64) {
	t.Helper()

	var requestCount atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		w.Header().Set("Content-Type", jpegContentType)
		_, _ = w.Write(testImageData)
	}))

	t.Cleanup(func() {
		srv.Close()
	})

	return srv, &requestCount
}

func TestNewCache_CreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "cache")

	cache, err := NewCache(dir)

	require.NoError(t, err, "NewCache should succeed")
	require.NotNil(t, cache, "cache must not be nil")

	info, err := os.Stat(dir)
	require.NoError(t, err, "cache directory should exist")
	assert.True(t, info.IsDir(), "cache path should be a directory")
}

func TestGet_ReturnsFalseForUncachedSpecies(t *testing.T) {
	cache, err := NewCache(t.TempDir())
	require.NoError(t, err)

	path, found := cache.Get("Nonexistent Bird")

	assert.False(t, found, "uncached species should not be found")
	assert.Empty(t, path, "path should be empty for uncached species")
}

func TestPut_And_Get_RoundTrip(t *testing.T) {
	srv, _ := newTestServer(t)
	cache, err := NewCache(t.TempDir())
	require.NoError(t, err)

	species := "Northern Cardinal"
	imageURL := srv.URL + "/cardinal.jpg"

	err = cache.Put(species, imageURL)
	require.NoError(t, err, "Put should succeed")

	path, found := cache.Get(species)
	require.True(t, found, "species should be found after Put")
	assert.NotEmpty(t, path, "returned path should not be empty")

	data, err := os.ReadFile(path) //nolint:gosec // test-only: path from cache.Get
	require.NoError(t, err, "cached file should be readable")
	assert.Equal(t, testImageData, data, "cached data should match test image")
}

func TestPut_EmptyURL_IsNoOp(t *testing.T) {
	cache, err := NewCache(t.TempDir())
	require.NoError(t, err)

	err = cache.Put("Any Bird", "")

	require.NoError(t, err, "Put with empty URL should be a no-op")

	_, found := cache.Get("Any Bird")
	assert.False(t, found, "no image should be cached for empty URL")
}

func TestEnsureCached_Idempotent(t *testing.T) {
	srv, requestCount := newTestServer(t)
	cache, err := NewCache(t.TempDir())
	require.NoError(t, err)

	species := "Blue Jay"
	imageURL := srv.URL + "/bluejay.jpg"

	// First call should download.
	err = cache.EnsureCached(species, imageURL)
	require.NoError(t, err, "first EnsureCached should succeed")
	assert.Equal(t, int64(1), requestCount.Load(), "should have made exactly one request")

	// Second call should be a no-op.
	err = cache.EnsureCached(species, imageURL)
	require.NoError(t, err, "second EnsureCached should succeed")
	assert.Equal(t, int64(1), requestCount.Load(), "should not have made a second request")

	// Verify the image is still there.
	path, found := cache.Get(species)
	require.True(t, found, "image should still be cached")
	assert.NotEmpty(t, path)
}

func TestEnsureCached_EmptyURL_IsNoOp(t *testing.T) {
	cache, err := NewCache(t.TempDir())
	require.NoError(t, err)

	err = cache.EnsureCached("Any Bird", "")

	assert.NoError(t, err, "EnsureCached with empty URL should be a no-op")
}

func TestServeCachedImage_ReturnsBytes(t *testing.T) {
	srv, _ := newTestServer(t)
	cache, err := NewCache(t.TempDir())
	require.NoError(t, err)

	species := "American Robin"
	imageURL := srv.URL + "/robin.jpg"

	err = cache.Put(species, imageURL)
	require.NoError(t, err)

	data, contentType, err := cache.ServeCachedImage(species)

	require.NoError(t, err, "ServeCachedImage should succeed for cached species")
	assert.Equal(t, testImageData, data, "returned bytes should match cached image")
	assert.Equal(t, jpegContentType, contentType, "content type should be image/jpeg")
}

func TestServeCachedImage_ErrorsForUncached(t *testing.T) {
	cache, err := NewCache(t.TempDir())
	require.NoError(t, err)

	data, contentType, err := cache.ServeCachedImage("Unknown Bird")

	require.Error(t, err, "should error for uncached species")
	assert.Nil(t, data, "data should be nil on error")
	assert.Empty(t, contentType, "content type should be empty on error")
}

func TestNormalizeSpecies(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple two-word name",
			input:    "Northern Cardinal",
			expected: "Northern_Cardinal",
		},
		{
			name:     "name with apostrophe",
			input:    "Cooper's Hawk",
			expected: "Coopers_Hawk",
		},
		{
			name:     "name with hyphen",
			input:    "Black-capped Chickadee",
			expected: "Blackcapped_Chickadee",
		},
		{
			name:     "name with parentheses",
			input:    "Dark-eyed Junco (Slate-colored)",
			expected: "Darkeyed_Junco_Slatecolored",
		},
		{
			name:     "single word",
			input:    "Osprey",
			expected: "Osprey",
		},
		{
			name:     "already underscored",
			input:    "Blue_Jay",
			expected: "Blue_Jay",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := normalizeSpecies(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestURLHash(t *testing.T) {
	hash := urlHash("https://example.com/bird.jpg")

	assert.Len(t, hash, 8, "hash should be 8 hex characters")
	assert.Regexp(t, `^[0-9a-f]{8}$`, hash, "hash should be lowercase hex")

	// Same input produces same output.
	hash2 := urlHash("https://example.com/bird.jpg")
	assert.Equal(t, hash, hash2, "hash should be deterministic")

	// Different input produces different output.
	hash3 := urlHash("https://example.com/other.jpg")
	assert.NotEqual(t, hash, hash3, "different URLs should produce different hashes")
}

func TestPut_HTTPError_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(func() {
		srv.Close()
	})

	cache, err := NewCache(t.TempDir())
	require.NoError(t, err)

	err = cache.Put("Test Bird", srv.URL+"/missing.jpg")

	require.Error(t, err, "Put should return error on HTTP 404")
	assert.Contains(t, err.Error(), "status 404")
}

func TestPut_MultipleImages_SameSpecies(t *testing.T) {
	srv, _ := newTestServer(t)
	cache, err := NewCache(t.TempDir())
	require.NoError(t, err)

	species := "House Sparrow"
	url1 := srv.URL + "/sparrow1.jpg"
	url2 := srv.URL + "/sparrow2.jpg"

	err = cache.Put(species, url1)
	require.NoError(t, err)

	err = cache.Put(species, url2)
	require.NoError(t, err)

	// Both files should exist in the species directory.
	speciesDir := filepath.Join(cache.dir, normalizeSpecies(species))
	entries, err := os.ReadDir(speciesDir)
	require.NoError(t, err)
	assert.Len(t, entries, 2, "should have two cached images for the species")
}

func TestDetectContentType_ByExtension(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{name: "jpg", path: "bird.jpg", expected: jpegContentType},
		{name: "jpeg", path: "bird.jpeg", expected: jpegContentType},
		{name: "png", path: "bird.png", expected: "image/png"},
		{name: "gif", path: "bird.gif", expected: "image/gif"},
		{name: "webp", path: "bird.webp", expected: "image/webp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := detectContentType(tt.path, nil)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectContentType_FallbackToSniff(t *testing.T) {
	// Unknown extension should fall back to http.DetectContentType.
	result := detectContentType("bird.unknown", testImageData)
	assert.NotEmpty(t, result, "should detect content type by sniffing bytes")
}

func TestNewCache_InvalidPath(t *testing.T) {
	// Attempt to create cache in a path that cannot be created.
	cache, err := NewCache("/dev/null/impossible/path")

	require.Error(t, err, "should error for invalid path")
	assert.Nil(t, cache, "cache should be nil on error")
}

func TestPut_UnreachableServer(t *testing.T) {
	cache, err := NewCache(t.TempDir())
	require.NoError(t, err)

	err = cache.Put("Test Bird", "http://127.0.0.1:1/unreachable.jpg")

	require.Error(t, err, "should error for unreachable server")
	assert.Contains(t, err.Error(), "failed to download image")
}
