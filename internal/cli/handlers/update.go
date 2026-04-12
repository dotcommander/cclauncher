package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/dotcommander/cclauncher/internal/config"
	"github.com/spf13/cobra"
)

const (
	githubReleaseURL = "https://api.github.com/repos/dotcommander/cclauncher/releases/latest"
	goInstallTarget  = "github.com/dotcommander/cclauncher/cmd/ccl@latest"
	updateTimeout    = 10 * time.Second
)

// HandleUpdate checks GitHub for a newer release and, unless --check was set,
// runs `go install` to update the binary.
func HandleUpdate(cmd *cobra.Command, _ []string) error {
	checkOnly, _ := cmd.Flags().GetBool("check")

	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("go command not found — install Go from https://golang.org/dl/ to self-update")
	}

	current := config.Version
	fmt.Printf("Current version: %s\n", current)

	ctx, cancel := context.WithTimeout(cmd.Context(), updateTimeout)
	defer cancel()

	latest, err := fetchLatestVersion(ctx)
	if err != nil {
		return fmt.Errorf("check for updates: %w", err)
	}
	fmt.Printf("Latest version:  %s\n", latest)

	if current == latest {
		fmt.Println("Already up to date")
		return nil
	}
	if checkOnly {
		fmt.Println("Update available — run 'ccl update' to install")
		return nil
	}

	fmt.Println("Updating...")
	install := exec.CommandContext(ctx, "go", "install", goInstallTarget)
	install.Stdout, install.Stderr = os.Stdout, os.Stderr
	if err := install.Run(); err != nil {
		return fmt.Errorf("go install: %w", err)
	}

	fmt.Printf("Updated to %s\n", latest)
	fmt.Println("Restart your terminal or run 'hash -r' to pick up the new binary")
	return nil
}

// fetchLatestVersion returns the tag name of the latest GitHub release.
func fetchLatestVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubReleaseURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "ccl-updater")
	req.Header.Set("Accept", "application/vnd.github+json")

	// Belt-and-suspenders: local client Timeout backs up the context deadline,
	// per project convention (http.DefaultClient has no timeout).
	client := &http.Client{Timeout: updateTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github returned %s", resp.Status)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("decode release: %w", err)
	}
	if release.TagName == "" {
		return "", fmt.Errorf("release has empty tag_name")
	}
	return release.TagName, nil
}
