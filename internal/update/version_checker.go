// internal/update/version_checker.go
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
)

// UpdateInfo holds information about the latest version
type UpdateInfo struct {
	LatestVersion string    `json:"latest_version"`
	ReleaseDate   time.Time `json:"release_date"`
	ReleaseNotes  string    `json:"release_notes"`
	DownloadURL   string    `json:"download_url"`
	CheckedAt     time.Time `json:"checked_at"`
}

// VersionChecker checks for new versions of the application
type VersionChecker struct {
	currentVersion string
	configDir      string
	updateURL      string
	cacheDuration  time.Duration
}

// NewVersionChecker creates a new version checker
func NewVersionChecker(currentVersion, configDir string) *VersionChecker {
	return &VersionChecker{
		currentVersion: strings.TrimPrefix(currentVersion, "v"),
		configDir:      configDir,
		updateURL:      "https://api.github.com/repos/jasonKoogler/comma/releases/latest",
		cacheDuration:  24 * time.Hour, // Check once per day
	}
}

// CheckForUpdates checks if a newer version is available
func (vc *VersionChecker) CheckForUpdates(ctx context.Context) (*UpdateInfo, error) {
	// First check if we have cached update info
	cachedInfo, err := vc.loadCachedInfo()
	if err == nil && time.Since(cachedInfo.CheckedAt) < vc.cacheDuration {
		// Use cached info if it's recent enough
		if vc.isNewerVersion(cachedInfo.LatestVersion) {
			return cachedInfo, nil
		}
		return nil, nil // No update available
	}

	// Need to check for updates
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", vc.updateURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "comma-git-client")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse GitHub API response
	var release struct {
		TagName     string    `json:"tag_name"`
		PublishedAt time.Time `json:"published_at"`
		Body        string    `json:"body"`
		HTMLURL     string    `json:"html_url"`
	}

	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Clean version string (remove 'v' prefix if present)
	latestVersion := strings.TrimPrefix(release.TagName, "v")

	// Create update info
	info := &UpdateInfo{
		LatestVersion: latestVersion,
		ReleaseDate:   release.PublishedAt,
		ReleaseNotes:  release.Body,
		DownloadURL:   release.HTMLURL,
		CheckedAt:     time.Now(),
	}

	// Cache the update info
	vc.cacheUpdateInfo(info)

	// Check if the latest version is newer
	if vc.isNewerVersion(latestVersion) {
		return info, nil
	}

	return nil, nil // No update available
}

// isNewerVersion checks if the latest version is newer than the current version
func (vc *VersionChecker) isNewerVersion(latestVersion string) bool {
	current, err := semver.NewVersion(vc.currentVersion)
	if err != nil {
		// If current version is not valid semver, assume latest is newer
		return true
	}

	latest, err := semver.NewVersion(latestVersion)
	if err != nil {
		// If latest version is not valid semver, assume no update
		return false
	}

	return latest.GreaterThan(current)
}

// cacheUpdateInfo saves update information to cache
func (vc *VersionChecker) cacheUpdateInfo(info *UpdateInfo) error {
	cacheDir := filepath.Join(vc.configDir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	cachePath := filepath.Join(cacheDir, "update_info.json")
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal update info: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// loadCachedInfo loads cached update information
func (vc *VersionChecker) loadCachedInfo() (*UpdateInfo, error) {
	cachePath := filepath.Join(vc.configDir, "cache", "update_info.json")
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var info UpdateInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to parse cached update info: %w", err)
	}

	return &info, nil
}

// GetUpdateMessage returns a formatted message about an available update
func (vc *VersionChecker) GetUpdateMessage(info *UpdateInfo) string {
	return fmt.Sprintf(`
🎉 Update Available: v%s (released %s)
   Current version: v%s
   
   %s
   
   Download: %s
   
   Run 'comma update' to update automatically.
`, info.LatestVersion, info.ReleaseDate.Format("Jan 2, 2006"),
		vc.currentVersion, info.ReleaseNotes, info.DownloadURL)
}
