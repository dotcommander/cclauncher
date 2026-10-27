package handlers

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestStableVersionDecisions(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		current, latest string
		comparison      int
		ok              bool
	}{
		{"1.2.3", "v1.2.3", 0, true}, {"v2.0.0", "1.99.9", 1, true}, {"1.2.9", "1.2.10", -1, true},
		{"dev", "1.0.0", 0, false}, {"1.0.0-beta", "1.0.0", 0, false}, {"1.0", "1.0.0", 0, false}, {"1.0.0", "bad", 0, false},
	} {
		t.Run(tc.current+tc.latest, func(t *testing.T) {
			t.Parallel()
			got, ok := compareStableVersions(tc.current, tc.latest)
			if got != tc.comparison || ok != tc.ok {
				t.Fatalf("got %d,%v", got, ok)
			}
		})
	}
}

func TestUpdateUsesSeparateLookupDeadline(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	var out bytes.Buffer
	fetched := false
	err := handleUpdate(ctx, &out, &out, false, "1.0.0", updateDeps{
		fetch: func(lookup context.Context) (string, error) {
			deadline, ok := lookup.Deadline()
			if !ok || time.Until(deadline) > updateTimeout {
				t.Fatal("lookup is not bounded")
			}
			fetched = true
			return "v1.0.1", nil
		},
		lookPath: func(string) (string, error) {
			if !fetched {
				t.Fatal("looked up compiler before release")
			}
			return "/fake/go", nil
		},
		install: func(install context.Context, path, target string) error {
			if _, ok := install.Deadline(); ok {
				t.Fatal("installation inherited lookup deadline")
			}
			if path != "/fake/go" {
				t.Fatal(path)
			}
			cancel()
			return install.Err()
		},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
}

func TestUpdateSkipsCompilerAndInstallation(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		current string
		check   bool
	}{{"1.0.0", true}, {"2.0.0", false}, {"dev", false}, {"v1.0.1", false}} {
		t.Run(tc.current, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			err := handleUpdate(context.Background(), &out, &out, tc.check, tc.current, updateDeps{
				fetch: func(context.Context) (string, error) { return "1.0.1", nil }, lookPath: func(string) (string, error) { t.Fatal("compiler lookup"); return "", nil }, install: func(context.Context, string, string) error { t.Fatal("install"); return nil },
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestUpdateLookupCallerCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var out bytes.Buffer
	err := handleUpdate(ctx, &out, &out, true, "1.0.0", updateDeps{fetch: func(ctx context.Context) (string, error) { return "", ctx.Err() }})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
}

func TestInstallableVersion(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		version string
		want    bool
	}{
		{"v1.2.3", true}, {"v0.0.1", true},
		{"1.2.3", false}, {"v1.2", false}, {"v1.2.3-beta", false}, {"latest", false}, {"", false},
	} {
		t.Run(tc.version, func(t *testing.T) {
			t.Parallel()
			if got := installableVersion(tc.version); got != tc.want {
				t.Fatalf("installableVersion(%q) = %v, want %v", tc.version, got, tc.want)
			}
		})
	}
}

func TestModuleVersionFromBuildInfo(t *testing.T) {
	t.Parallel()
	output := "/fake/gopath/bin/ccl: go1.26.0\n" +
		"\tpath\tgithub.com/dotcommander/cclauncher\n" +
		"\tmod\tgithub.com/dotcommander/cclauncher\tv1.2.3\th1:abc=\n" +
		"\tdep\tsome.other/module\tv0.9.0\th1:def=\n"
	version, ok := moduleVersionFromBuildInfo(output)
	if !ok || version != "v1.2.3" {
		t.Fatalf("got %q, %v; want v1.2.3, true", version, ok)
	}
	if _, ok := moduleVersionFromBuildInfo("/fake/ccl: go1.26.0\n\tbuild\t-compiler=gc\n"); ok {
		t.Fatal("parsed a version without a mod line")
	}
}

func TestUpdateInstallsFetchedReleaseAndVerifiesBinary(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	var installedTarget, versionedBinary string
	err := handleUpdate(context.Background(), &out, &out, false, "v1.0.0", updateDeps{
		fetch:    func(context.Context) (string, error) { return "v1.0.2", nil },
		lookPath: func(string) (string, error) { return "/fake/go", nil },
		install:  func(_ context.Context, _, target string) error { installedTarget = target; return nil },
		goEnv: func(_ context.Context, _, name string) (string, error) {
			if name == "GOBIN" {
				return "", nil
			}
			return "/fake/gopath", nil
		},
		goVersionM: func(_ context.Context, _, binary string) (string, error) {
			versionedBinary = binary
			return "\tmod\tgithub.com/dotcommander/cclauncher\tv1.0.2\th1:abc=\n", nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if installedTarget != "github.com/dotcommander/cclauncher/cmd/ccl@v1.0.2" {
		t.Fatalf("install target %q", installedTarget)
	}
	if versionedBinary != "/fake/gopath/bin/ccl" {
		t.Fatalf("verified binary %q", versionedBinary)
	}
	if !strings.Contains(out.String(), "Updated to v1.0.2") {
		t.Fatalf("output %q", out.String())
	}
}

func TestUpdateVerifiesBinaryInGOBIN(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	var versionedBinary string
	err := handleUpdate(context.Background(), &out, &out, false, "v1.0.0", updateDeps{
		fetch:    func(context.Context) (string, error) { return "v1.0.2", nil },
		lookPath: func(string) (string, error) { return "/fake/go", nil },
		install:  func(context.Context, string, string) error { return nil },
		goEnv: func(_ context.Context, _, name string) (string, error) {
			if name != "GOBIN" {
				t.Fatal("GOPATH consulted while GOBIN is set")
			}
			return "/fake/gobin", nil
		},
		goVersionM: func(_ context.Context, _, binary string) (string, error) {
			versionedBinary = binary
			return "\tmod\tgithub.com/dotcommander/cclauncher\tv1.0.2\th1:abc=\n", nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if versionedBinary != "/fake/gobin/ccl" {
		t.Fatalf("verified binary %q", versionedBinary)
	}
}

func TestUpdateSkipsUninstallableRelease(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	err := handleUpdate(context.Background(), &out, &out, false, "1.0.0", updateDeps{
		fetch:    func(context.Context) (string, error) { return "1.0.2", nil },
		lookPath: func(string) (string, error) { t.Fatal("compiler lookup"); return "", nil },
		install:  func(context.Context, string, string) error { t.Fatal("install"); return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "1.0.2 is not an installable module version") {
		t.Fatalf("output %q", out.String())
	}
}

func TestUpdateRejectsUnverifiedInstallation(t *testing.T) {
	t.Parallel()
	fakeEnv := func(_ context.Context, _, name string) (string, error) {
		if name == "GOBIN" {
			return "", nil
		}
		return "/fake/gopath", nil
	}
	for _, tc := range []struct {
		name    string
		goEnv   func(context.Context, string, string) (string, error)
		goVerM  func(context.Context, string, string) (string, error)
		wantErr string
	}{
		{
			name:    "install location unresolvable",
			goEnv:   func(context.Context, string, string) (string, error) { return "", errors.New("go env failed") },
			wantErr: "resolve install location",
		},
		{
			name:    "build info unreadable",
			goEnv:   fakeEnv,
			goVerM:  func(context.Context, string, string) (string, error) { return "", errors.New("go version failed") },
			wantErr: "verify installed ccl",
		},
		{
			name:    "binary lacks module version",
			goEnv:   fakeEnv,
			goVerM:  func(_ context.Context, _, _ string) (string, error) { return "/fake/gopath/bin/ccl: go1.26.0\n", nil },
			wantErr: "does not report a github.com/dotcommander/cclauncher module version",
		},
		{
			name:    "binary reports an old version",
			goEnv:   fakeEnv,
			goVerM:  func(_ context.Context, _, _ string) (string, error) { return "\tmod\tgithub.com/dotcommander/cclauncher\tv1.0.1\th1:abc=\n", nil },
			wantErr: "reports version v1.0.1, expected v1.0.2",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			err := handleUpdate(context.Background(), &out, &out, false, "v1.0.1", updateDeps{
				fetch:      func(context.Context) (string, error) { return "v1.0.2", nil },
				lookPath:   func(string) (string, error) { return "/fake/go", nil },
				install:    func(context.Context, string, string) error { return nil },
				goEnv:      tc.goEnv,
				goVersionM: tc.goVerM,
			})
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error=%v, want substring %q", err, tc.wantErr)
			}
			if strings.Contains(out.String(), "Updated to") {
				t.Fatalf("success printed without verification: %q", out.String())
			}
		})
	}
}
