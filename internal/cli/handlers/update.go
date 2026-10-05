package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
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
	return handleUpdate(ctx, out, errOut, checkOnly, GetVersion(), updateDeps{
		fetch: fetchLatestVersion, lookPath: exec.LookPath,
		install: func(ctx context.Context, path string) error {
			cmd := exec.CommandContext(ctx, path, "install", goInstallTarget)
			cmd.Stdout, cmd.Stderr = out, errOut
			return cmd.Run()
		},
	})
}

type updateDeps struct {
	fetch    func(context.Context) (string, error)
	lookPath func(string) (string, error)
	install  func(context.Context, string) error
}

func handleUpdate(ctx context.Context, out, errOut io.Writer, checkOnly bool, current string, deps updateDeps) error {
	if err := writeUpdateStatus(out, "Current version: %s\n", current); err != nil {
		return err
	}

	lookupCtx, cancel := context.WithTimeout(ctx, updateTimeout)
	latest, err := deps.fetch(lookupCtx)
	cancel()
	if err != nil {
		return fmt.Errorf("check for updates: %w", err)
	}
	if err := writeUpdateStatus(out, "Latest version:  %s\n", latest); err != nil {
		return err
	}

	comparison, determinate := compareStableVersions(current, latest)
	if !determinate {
		return writeUpdateStatus(out, "Version comparison indeterminate; installation skipped\n")
	}
	if comparison >= 0 {
		return writeUpdateStatus(out, "Already up to date\n")
	}
	if checkOnly {
		return writeUpdateStatus(out, "Update available — run 'ccl update' to install\n")
	}

	goPath, err := deps.lookPath("go")
	if err != nil {
		return errors.New("go command not found — install Go from https://golang.org/dl/ to self-update")
	}
	if err := writeUpdateStatus(out, "Updating...\n"); err != nil {
		return err
	}
	if err := deps.install(ctx, goPath); err != nil {
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

// compareStableVersions accepts numeric major.minor.patch versions only.
func compareStableVersions(current, latest string) (int, bool) {
	parse := func(v string) ([3]uint64, bool) {
		var result [3]uint64
		parts := strings.Split(strings.TrimPrefix(v, "v"), ".")
		if len(parts) != 3 {
			return result, false
		}
		for i, part := range parts {
			if part == "" || (len(part) > 1 && part[0] == '0') {
				return result, false
			}
			for _, c := range part {
				if c < '0' || c > '9' {
					return result, false
				}
			}
			n, err := strconv.ParseUint(part, 10, 64)
			if err != nil {
				return result, false
			}
			result[i] = n
		}
		return result, true
	}
	a, ok := parse(current)
	if !ok {
		return 0, false
	}
	b, ok := parse(latest)
	if !ok {
		return 0, false
	}
	for i := range a {
		if a[i] < b[i] {
			return -1, true
		}
		if a[i] > b[i] {
			return 1, true
		}
	}
	return 0, true
}
