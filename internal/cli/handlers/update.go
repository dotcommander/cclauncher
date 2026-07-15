package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"time"
)

const (
	githubReleaseURL = "https://api.github.com/repos/dotcommander/cclauncher/releases/latest"
	goInstallTarget  = "github.com/dotcommander/cclauncher/cmd/ccl@latest"
	updateTimeout    = 10 * time.Second
)

// HandleUpdate checks GitHub for a newer release and, unless --check was set,
// runs `go install` to update the binary.
func HandleUpdate(ctx context.Context, out, errOut io.Writer, checkOnly bool) error {
	if _, err := exec.LookPath("go"); err != nil {
		return errors.New("go command not found — install Go from https://golang.org/dl/ to self-update")
	}

	current := GetVersion()
	if err := writeUpdateStatus(out, "Current version: %s\n", current); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	latest, err := fetchLatestVersion(ctx)
	if err != nil {
		return fmt.Errorf("check for updates: %w", err)
	}
	if err := writeUpdateStatus(out, "Latest version:  %s\n", latest); err != nil {
		return err
	}

	if current == latest {
		return writeUpdateStatus(out, "Already up to date\n")
	}
	if checkOnly {
		return writeUpdateStatus(out, "Update available — run 'ccl update' to install\n")
	}

	if err := writeUpdateStatus(out, "Updating...\n"); err != nil {
		return err
	}
	install := exec.CommandContext(ctx, "go", "install", goInstallTarget)
	install.Stdout, install.Stderr = out, errOut
	if err := install.Run(); err != nil {
		return fmt.Errorf("go install: %w", err)
	}

	if err := writeUpdateStatus(out, "Updated to %s\n", latest); err != nil {
		return err
	}
	return writeUpdateStatus(out, "Restart your terminal or run 'hash -r' to pick up the new binary\n")
}

func writeUpdateStatus(out io.Writer, format string, args ...any) error {
	if _, err := fmt.Fprintf(out, format, args...); err != nil {
		return fmt.Errorf("write update status: %w", err)
	}
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
	defer func() { _ = resp.Body.Close() }()

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
		return "", errors.New("release has empty tag_name")
	}
	return release.TagName, nil
}
