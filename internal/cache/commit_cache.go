// internal/cache/commit_cache.go
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CommitCache provides caching for LLM-generated commit messages
type CommitCache struct {
	cacheDir string
	maxAge   time.Duration
	enabled  bool
}

// CacheEntry represents a cached commit message
type CacheEntry struct {
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
	Provider  string    `json:"provider"`
	Stats     struct {
		ChangedFiles int `json:"changed_files"`
		Additions    int `json:"additions"`
		Deletions    int `json:"deletions"`
	} `json:"stats"`
}

// NewCommitCache creates a new commit message cache
func NewCommitCache(configDir string) (*CommitCache, error) {
	cacheDir := filepath.Join(configDir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &CommitCache{
		cacheDir: cacheDir,
		maxAge:   24 * time.Hour, // Cache entries expire after 24 hours
		enabled:  true,
	}, nil
}

// Get retrieves a cached commit message if available
func (c *CommitCache) Get(changes string) (*CacheEntry, error) {
	if !c.enabled {
		return nil, nil
	}

	key := c.generateKey(changes)
	cachePath := filepath.Join(c.cacheDir, key+".json")

	// Check if cache file exists and is not expired
	info, err := os.Stat(cachePath)
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to check cache file: %w", err)
	}

	// Check if cache is expired
	if time.Since(info.ModTime()) > c.maxAge {
		// Clean up expired cache entry
		os.Remove(cachePath)
		return nil, nil
	}

	// Read cache file
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("failed to parse cache entry: %w", err)
	}

	return &entry, nil
}

// Set stores a commit message in cache
func (c *CommitCache) Set(changes string, message string, provider string, stats struct {
	ChangedFiles int
	Additions    int
	Deletions    int
}) error {
	if !c.enabled {
		return nil
	}

	key := c.generateKey(changes)
	cachePath := filepath.Join(c.cacheDir, key+".json")

	entry := CacheEntry{
		Message:   message,
		CreatedAt: time.Now(),
		Provider:  provider,
		Stats: struct {
			ChangedFiles int `json:"changed_files"`
			Additions    int `json:"additions"`
			Deletions    int `json:"deletions"`
		}{
			ChangedFiles: stats.ChangedFiles,
			Additions:    stats.Additions,
			Deletions:    stats.Deletions,
		},
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// Cleanup removes expired cache entries
func (c *CommitCache) Cleanup() error {
	entries, err := os.ReadDir(c.cacheDir)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	now := time.Now()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if now.Sub(info.ModTime()) > c.maxAge {
			cachePath := filepath.Join(c.cacheDir, entry.Name())
			os.Remove(cachePath)
		}
	}

	return nil
}

// generateKey creates a cache key from changes
func (c *CommitCache) generateKey(changes string) string {
	hash := sha256.New()
	hash.Write([]byte(changes))
	return hex.EncodeToString(hash.Sum(nil))
}
