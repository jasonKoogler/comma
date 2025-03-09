// cmd/update.go
package cmd

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/jasonKoogler/comma/internal/update"
	"github.com/spf13/cobra"
)

var (
	forceUpdate bool
	checkOnly   bool
	setRepoURL  string
	showRepo    bool
	updateCmd   = &cobra.Command{
		Use:   "update",
		Short: "Check for and install updates",
		Long: `Check for updates to Comma and optionally install them.
By default, this command will check for updates and install them if available.
Use --check-only to only check for updates without installing.`,
		RunE: runUpdate,
	}
)

func init() {
	updateCmd.Flags().BoolVarP(&forceUpdate, "force", "f", false, "Force update even if already on latest version")
	updateCmd.Flags().BoolVarP(&checkOnly, "check-only", "c", false, "Only check for updates without installing")
	updateCmd.Flags().StringVar(&setRepoURL, "set-repo", "", "Set a custom repository URL for updates")
	updateCmd.Flags().BoolVar(&showRepo, "show-repo", false, "Show the current repository URL for updates")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	if appContext == nil || appContext.ConfigManager == nil {
		return fmt.Errorf("application context not initialized")
	}

	configDir := appContext.ConfigDir

	// Handle showing the current repository URL
	if showRepo {
		repoConfigPath := filepath.Join(configDir, "update_repo.txt")
		data, err := os.ReadFile(repoConfigPath)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("Using default repository URL: https://api.github.com/repos/jasonKoogler/comma/releases/latest")
			} else {
				fmt.Printf("Error reading repository URL: %v\n", err)
			}
		} else {
			customRepo := strings.TrimSpace(string(data))
			if customRepo == "" {
				fmt.Println("Using default repository URL: https://api.github.com/repos/jasonKoogler/comma/releases/latest")
			} else {
				fmt.Printf("Current repository URL: %s\n", customRepo)
			}
		}
		return nil
	}

	// Handle setting a custom repository URL
	if setRepoURL != "" {
		repoConfigPath := filepath.Join(configDir, "update_repo.txt")
		if err := os.MkdirAll(filepath.Dir(repoConfigPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		if err := os.WriteFile(repoConfigPath, []byte(setRepoURL), 0644); err != nil {
			return fmt.Errorf("failed to save repository URL: %w", err)
		}

		fmt.Printf("✓ Update repository URL set to: %s\n", setRepoURL)
		return nil
	}

	checker := update.NewVersionChecker(version, configDir)

	fmt.Println("Checking for updates...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := checker.CheckForUpdates(ctx)
	if err != nil {
		// Check if it's a 404 error, which likely means the repository doesn't exist
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "unexpected status code") {
			fmt.Println("⚠️  Update check failed: Repository or releases not found")
			fmt.Println("This could be because:")
			fmt.Println("  1. The repository doesn't exist or is private")
			fmt.Println("  2. There are no releases published yet")
			fmt.Println("  3. The update URL is incorrect")
			fmt.Println("\nTo configure a custom repository for updates, use:")
			fmt.Println("  comma update --set-repo https://api.github.com/repos/username/repo/releases/latest")
			fmt.Println("\nTo see the current repository URL, use:")
			fmt.Println("  comma update --show-repo")
			return nil
		}
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if info == nil && !forceUpdate {
		fmt.Println("✓ You're already using the latest version!")
		return nil
	}

	if info != nil {
		fmt.Print(checker.GetUpdateMessage(info))
	}

	if checkOnly {
		return nil
	}

	// If no update available but force flag is set
	if info == nil && forceUpdate {
		fmt.Println("Forcing update even though you're on the latest version...")
	}

	// Confirm update
	if !cmd.Flags().Changed("force") {
		fmt.Print("Do you want to update now? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Update cancelled.")
			return nil
		}
	}

	return performUpdate(info)
}

func performUpdate(info *update.UpdateInfo) error {
	fmt.Println("Starting update process...")

	// Determine update method based on how Comma was installed
	// This is a simplified example - actual implementation would need to detect installation method

	// Check if installed via Go
	if _, err := exec.LookPath("go"); err == nil {
		fmt.Println("Updating using Go...")
		cmd := exec.Command("go", "install", "github.com/jasonKoogler/comma@latest")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	// Check if we're running from a binary that can self-update
	execPath, err := os.Executable()
	if err == nil {
		// Get the download URL for the current platform
		downloadURL := getDownloadURL(info)
		if downloadURL != "" {
			return selfUpdate(execPath, downloadURL)
		}
	}

	// Fallback: Direct user to manual update
	fmt.Println("Automatic update not available for your installation method.")
	fmt.Println("Please update manually by downloading from:")
	if info != nil {
		fmt.Println(info.DownloadURL)
	} else {
		fmt.Println("https://github.com/jasonKoogler/comma/releases/latest")
	}

	// Platform-specific instructions
	switch runtime.GOOS {
	case "darwin":
		fmt.Println("\nOn macOS, you can also use Homebrew:")
		fmt.Println("  brew upgrade comma")
	case "linux":
		fmt.Println("\nOn Linux, you can also use the installation script:")
		fmt.Println("  curl -sSL https://raw.githubusercontent.com/jasonKoogler/comma/main/install.sh | bash")
	}

	return nil
}

// getDownloadURL returns the appropriate download URL for the current platform
func getDownloadURL(info *update.UpdateInfo) string {
	if info == nil {
		return ""
	}

	// Base URL for releases
	baseURL := "https://github.com/jasonKoogler/comma/releases/download"
	version := info.LatestVersion
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	// Determine platform-specific binary name
	var binaryName string
	switch runtime.GOOS {
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			binaryName = "comma_linux_amd64.tar.gz"
		case "arm64":
			binaryName = "comma_linux_arm64.tar.gz"
		default:
			return "" // Unsupported architecture
		}
	case "darwin":
		switch runtime.GOARCH {
		case "amd64":
			binaryName = "comma_darwin_amd64.tar.gz"
		case "arm64":
			binaryName = "comma_darwin_arm64.tar.gz"
		default:
			return "" // Unsupported architecture
		}
	case "windows":
		switch runtime.GOARCH {
		case "amd64":
			binaryName = "comma_windows_amd64.zip"
		default:
			return "" // Unsupported architecture
		}
	default:
		return "" // Unsupported OS
	}

	return fmt.Sprintf("%s/%s/%s", baseURL, version, binaryName)
}

// selfUpdate downloads and replaces the current executable with a new version
func selfUpdate(execPath, downloadURL string) error {
	fmt.Printf("Downloading update from %s...\n", downloadURL)

	// Create a temporary directory for the download
	tempDir, err := os.MkdirTemp("", "comma-update")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Download the archive
	archivePath := filepath.Join(tempDir, "update.tar.gz")
	if strings.HasSuffix(downloadURL, ".zip") {
		archivePath = filepath.Join(tempDir, "update.zip")
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	// Create request
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "comma-updater")

	// Download the file
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// Create the file
	out, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %w", err)
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save download: %w", err)
	}
	out.Close()

	// Extract the archive
	extractDir := filepath.Join(tempDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return fmt.Errorf("failed to create extraction directory: %w", err)
	}

	fmt.Println("Extracting update...")
	if strings.HasSuffix(archivePath, ".zip") {
		if err := extractZip(archivePath, extractDir); err != nil {
			return fmt.Errorf("failed to extract zip: %w", err)
		}
	} else {
		if err := extractTarGz(archivePath, extractDir); err != nil {
			return fmt.Errorf("failed to extract tar.gz: %w", err)
		}
	}

	// Find the binary in the extracted files
	binaryName := "comma"
	if runtime.GOOS == "windows" {
		binaryName = "comma.exe"
	}

	// Look for the binary in the extracted directory
	var newBinaryPath string
	err = filepath.Walk(extractDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Base(path) == binaryName {
			newBinaryPath = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to find binary in extracted files: %w", err)
	}

	if newBinaryPath == "" {
		return fmt.Errorf("could not find %s binary in the downloaded package", binaryName)
	}

	// Make the new binary executable
	if runtime.GOOS != "windows" {
		if err := os.Chmod(newBinaryPath, 0755); err != nil {
			return fmt.Errorf("failed to make binary executable: %w", err)
		}
	}

	fmt.Println("Installing update...")

	// On Windows, we can't replace a running executable directly
	// So we need to use a different approach
	if runtime.GOOS == "windows" {
		// Create a batch file that will replace the executable after we exit
		batchFile := filepath.Join(tempDir, "update.bat")
		batchContent := fmt.Sprintf(`@echo off
ping -n 3 127.0.0.1 > nul
copy /Y "%s" "%s"
del "%s"
`, newBinaryPath, execPath, batchFile)

		if err := os.WriteFile(batchFile, []byte(batchContent), 0755); err != nil {
			return fmt.Errorf("failed to create update script: %w", err)
		}

		// Execute the batch file and exit
		cmd := exec.Command("cmd", "/C", "start", "/b", batchFile)
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to start update script: %w", err)
		}

		fmt.Println("Update will be applied when the application exits.")
		return nil
	}

	// On Unix systems, we can replace the binary directly
	// First, rename the current binary as a backup
	backupPath := execPath + ".bak"
	if err := os.Rename(execPath, backupPath); err != nil {
		return fmt.Errorf("failed to create backup of current binary: %w", err)
	}

	// Copy the new binary to the original location
	if err := copyFile(newBinaryPath, execPath); err != nil {
		// Try to restore the backup if the copy fails
		os.Rename(backupPath, execPath)
		return fmt.Errorf("failed to install new binary: %w", err)
	}

	// Remove the backup
	os.Remove(backupPath)

	fmt.Println("✓ Update successfully installed!")
	fmt.Println("Please restart Comma to use the new version.")

	return nil
}

// extractTarGz extracts a tar.gz file to a destination directory
func extractTarGz(archivePath, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			dir := filepath.Dir(target)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}

			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}

	return nil
}

// extractZip extracts a zip file to a destination directory
func extractZip(archivePath, destDir string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		target := filepath.Join(destDir, file.Name)

		if file.FileInfo().IsDir() {
			os.MkdirAll(target, file.Mode())
			continue
		}

		dir := filepath.Dir(target)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}

		fileReader, err := file.Open()
		if err != nil {
			return err
		}

		targetFile, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			fileReader.Close()
			return err
		}

		if _, err := io.Copy(targetFile, fileReader); err != nil {
			fileReader.Close()
			targetFile.Close()
			return err
		}

		fileReader.Close()
		targetFile.Close()
	}

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}
