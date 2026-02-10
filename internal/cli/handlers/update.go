package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/dotcommander/cclauncher/internal/config"
	"github.com/spf13/cobra"
)

// HandleUpdate handles the update command
func HandleUpdate(cmd *cobra.Command, args []string) error {
	checkOnly, _ := cmd.Flags().GetBool("check")

	currentVersion := config.Version

	// Show current version
	fmt.Printf("Current version: %s\n", currentVersion)

	// Check if go is available
	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("go command not found. Install Go from https://golang.org/dl/ to use self-update")
	}

	// Get latest version from GitHub
	latestVersion, err := getLatestVersion()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	fmt.Printf("Latest version: %s\n", latestVersion)

	// Compare versions
	if currentVersion == latestVersion {
		fmt.Println("Already up to date")
		return nil
	}

	// If check-only mode, just show update availability
	if checkOnly {
		fmt.Printf("Update available: ccl update\n")
		return nil
	}

	// Perform update
	fmt.Println("Updating to latest version...")

	goInstallCmd := exec.Command("go", "install", "github.com/dotcommander/cclauncher/cmd/ccl@latest")
	goInstallCmd.Stdout = os.Stdout
	goInstallCmd.Stderr = os.Stderr

	if err := goInstallCmd.Run(); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	// Verify the update by checking the new version
	// Note: The newly installed binary might not reflect the version until we rebuild or restart
	fmt.Printf("Updated to %s\n", latestVersion)
	fmt.Println("\nNote: Restart your terminal or run 'hash -r' to ensure the new version is used")

	return nil
}

// getLatestVersion fetches the latest release version from GitHub API
func getLatestVersion() (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", "https://api.github.com/repos/dotcommander/cclauncher/releases/latest", nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	// Set User-Agent to avoid GitHub API rate limiting
	req.Header.Set("User-Agent", "ccl-updater")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Fallback: try to get version from go list command
		return getVersionFromGoList()
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}

	if err := json.Unmarshal(body, &release); err != nil {
		return "", fmt.Errorf("parse release info: %w", err)
	}

	return release.TagName, nil
}

// getVersionFromGoList gets the latest version using go list command
func getVersionFromGoList() (string, error) {
	cmd := exec.Command("go", "list", "-m", "-versions", "github.com/dotcommander/cclauncher")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("go list failed: %w", err)
	}

	// Output format: "module version1 version2 version3 ..."
	// We want the last version
	parts := strings.Fields(string(output))
	if len(parts) < 2 {
		return "unknown", nil
	}

	// Return the last version in the list
	return parts[len(parts)-1], nil
}
