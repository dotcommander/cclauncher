package handlers

import (
	"bytes"
	"context"
	"errors"
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
		install: func(install context.Context, path string) error {
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
				fetch: func(context.Context) (string, error) { return "1.0.1", nil }, lookPath: func(string) (string, error) { t.Fatal("compiler lookup"); return "", nil }, install: func(context.Context, string) error { t.Fatal("install"); return nil },
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
