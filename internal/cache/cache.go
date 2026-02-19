// Package cache provides a file-based response cache with TTL and auto-purge.
package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Cache is a file-based cache that stores entries as JSON files in a directory.
type Cache struct {
	dir string
	ttl time.Duration
}

type cacheEntry struct {
	Data      []byte    `json:"data"`
	CreatedAt time.Time `json:"created_at"`
}

// New creates a new Cache with the given directory and TTL.
// The directory is created if it does not exist.
func New(dir string, ttl time.Duration) *Cache {
	os.MkdirAll(dir, 0755)
	return &Cache{dir: dir, ttl: ttl}
}

// Key produces a deterministic SHA-256 hex key from the given parts
// (e.g. query, profile, mode, provider).
func Key(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Get retrieves a cached entry by key. Returns the data and true on a cache
// hit, or nil and false on a miss (including expired or corrupted entries).
func (c *Cache) Get(key string) ([]byte, bool) {
	path := filepath.Join(c.dir, key+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		os.Remove(path)
		return nil, false
	}

	if time.Since(entry.CreatedAt) > c.ttl {
		os.Remove(path)
		return nil, false
	}

	return entry.Data, true
}

// Set stores data in the cache under the given key.
func (c *Cache) Set(key string, data []byte) error {
	entry := cacheEntry{
		Data:      data,
		CreatedAt: time.Now(),
	}
	encoded, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	path := filepath.Join(c.dir, key+".json")
	return os.WriteFile(path, encoded, 0644)
}

// Purge removes expired and corrupted entries from the cache directory.
// It returns the number of entries removed.
func (c *Cache) Purge() int {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return 0
	}
	removed := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(c.dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var entry cacheEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			os.Remove(path)
			removed++
			continue
		}
		if time.Since(entry.CreatedAt) > c.ttl {
			os.Remove(path)
			removed++
		}
	}
	return removed
}

// DefaultDir returns the default cache directory, respecting XDG_CACHE_HOME.
func DefaultDir() string {
	if dir := os.Getenv("XDG_CACHE_HOME"); dir != "" {
		return filepath.Join(dir, "websearch")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "websearch")
}
