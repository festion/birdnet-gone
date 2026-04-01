// Package imagecache provides disk-based image caching for bird species.
// It downloads and stores species images by URL hash for offline display
// in kiosk mode, replacing the Python cache_builder.py functionality.
package imagecache

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/tphakala/birdnet-go/internal/errors"
)

const (
	// httpClientTimeout is the timeout for image download requests.
	httpClientTimeout = 10 * 1000 * 1000 * 1000 // 10 seconds in nanoseconds

	// urlHashLength is the number of bytes from the SHA256 hash used for filenames.
	// Produces 8 hex characters (4 bytes * 2 hex chars per byte).
	urlHashLength = 4

	// imageFileExtension is the default file extension for cached images.
	imageFileExtension = ".jpg"

	// jpegContentType is the MIME type for JPEG images.
	jpegContentType = "image/jpeg"

	// dirPermissions is the file mode for cache directories.
	dirPermissions = 0o755

	// filePermissions is the file mode for cached image files.
	filePermissions = 0o644
)

// nonAlphanumericRegex matches characters that are not alphanumeric, spaces, or underscores.
var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9 _]`)

func init() {
	errors.RegisterComponent("imagecache", "imagecache")
}

// Cache provides disk-based image caching for bird species.
// Images are stored in species-specific subdirectories using URL-derived
// hash filenames to support multiple images per species without collisions.
type Cache struct {
	dir        string
	httpClient *http.Client
	mu         sync.RWMutex
}

// NewCache creates a new image cache rooted at cacheDir, creating the
// directory if it does not exist. The cache uses an HTTP client with a
// 10-second timeout for downloading images.
func NewCache(cacheDir string) (*Cache, error) {
	if err := os.MkdirAll(cacheDir, dirPermissions); err != nil {
		return nil, errors.Newf("failed to create cache directory %s: %v", cacheDir, err).
			Component("imagecache").
			Category(errors.CategoryFileIO).
			Context("operation", "create_cache_dir").
			Context("path", cacheDir).
			Build()
	}

	return &Cache{
		dir: cacheDir,
		httpClient: &http.Client{
			Timeout: httpClientTimeout,
		},
	}, nil
}

// Get returns the path to the first cached image for the given species
// and true if one exists, or an empty string and false otherwise.
// Species names are normalized to a filesystem-safe format.
func (c *Cache) Get(species string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	speciesDir := filepath.Join(c.dir, normalizeSpecies(species))

	entries, err := os.ReadDir(speciesDir)
	if err != nil {
		return "", false
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			return filepath.Join(speciesDir, entry.Name()), true
		}
	}

	return "", false
}

// Put downloads the image at imageURL and stores it in the cache under
// the given species. The file is named using the first 8 hex characters
// of the SHA256 hash of the URL. Returns early without error if imageURL
// is empty. Creates the species subdirectory if needed.
func (c *Cache) Put(species, imageURL string) error {
	if imageURL == "" {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	normalized := normalizeSpecies(species)
	speciesDir := filepath.Join(c.dir, normalized)

	if err := os.MkdirAll(speciesDir, dirPermissions); err != nil {
		return errors.Newf("failed to create species directory: %v", err).
			Component("imagecache").
			Category(errors.CategoryFileIO).
			Context("operation", "create_species_dir").
			Context("species", species).
			Build()
	}

	filename := urlHash(imageURL) + imageFileExtension
	destPath := filepath.Join(speciesDir, filename)

	resp, err := c.httpClient.Get(imageURL) //nolint:gosec // URL comes from trusted image provider APIs
	if err != nil {
		return errors.Newf("failed to download image: %v", err).
			Component("imagecache").
			Category(errors.CategoryImageCache).
			Context("operation", "download_image").
			Context("species", species).
			Context("url", imageURL).
			Build()
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return errors.Newf("image download returned status %d", resp.StatusCode).
			Component("imagecache").
			Category(errors.CategoryImageCache).
			Context("operation", "download_image").
			Context("species", species).
			Context("url", imageURL).
			Context("status_code", resp.StatusCode).
			Build()
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Newf("failed to read image response body: %v", err).
			Component("imagecache").
			Category(errors.CategoryImageCache).
			Context("operation", "read_response").
			Context("species", species).
			Build()
	}

	if err := os.WriteFile(destPath, data, filePermissions); err != nil {
		return errors.Newf("failed to write cached image: %v", err).
			Component("imagecache").
			Category(errors.CategoryFileIO).
			Context("operation", "write_image").
			Context("species", species).
			Context("path", destPath).
			Build()
	}

	return nil
}

// EnsureCached is an idempotent variant of Put — it only downloads the
// image if the specific URL has not already been cached (checked by hash
// filename). This is safe for repeated calls with the same arguments.
func (c *Cache) EnsureCached(species, imageURL string) error {
	if imageURL == "" {
		return nil
	}

	c.mu.RLock()
	normalized := normalizeSpecies(species)
	filename := urlHash(imageURL) + imageFileExtension
	destPath := filepath.Join(c.dir, normalized, filename)
	_, err := os.Stat(destPath)
	c.mu.RUnlock()

	if err == nil {
		// File already exists — nothing to do.
		return nil
	}

	return c.Put(species, imageURL)
}

// ServeCachedImage reads the first cached image file for the species and
// returns the raw bytes along with its MIME content type. This is intended
// for serving cached images over HTTP. Returns an error if no cached image
// exists for the species.
func (c *Cache) ServeCachedImage(species string) (imgData []byte, contentType string, err error) {
	path, ok := c.Get(species)
	if !ok {
		return nil, "", errors.Newf("no cached image for species %q", species).
			Component("imagecache").
			Category(errors.CategoryNotFound).
			Context("operation", "serve_cached_image").
			Context("species", species).
			Build()
	}

	data, err := os.ReadFile(path) //nolint:gosec // path is constructed from cache directory, not user input
	if err != nil {
		return nil, "", errors.Newf("failed to read cached image: %v", err).
			Component("imagecache").
			Category(errors.CategoryFileIO).
			Context("operation", "read_cached_image").
			Context("species", species).
			Context("path", path).
			Build()
	}

	contentType = detectContentType(path, data)
	return data, contentType, nil
}

// normalizeSpecies converts a species name to a filesystem-safe directory
// name. Spaces become underscores, and non-alphanumeric characters (except
// underscores) are removed.
//
// Example: "Northern Cardinal" -> "Northern_Cardinal"
func normalizeSpecies(name string) string {
	// Remove non-alphanumeric characters except spaces and underscores
	cleaned := nonAlphanumericRegex.ReplaceAllString(name, "")
	// Replace spaces with underscores
	return strings.ReplaceAll(cleaned, " ", "_")
}

// urlHash returns the first 8 hex characters of the SHA256 hash of the
// given URL. This provides a compact, deterministic filename component
// that avoids filesystem-unsafe characters from raw URLs.
func urlHash(imageURL string) string {
	h := sha256.Sum256([]byte(imageURL))
	return hex.EncodeToString(h[:urlHashLength])
}

// detectContentType returns the MIME content type for the given image file.
// It uses the file extension as the primary signal, falling back to
// http.DetectContentType for unknown extensions.
func detectContentType(path string, data []byte) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg":
		return jpegContentType
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return http.DetectContentType(data)
	}
}
