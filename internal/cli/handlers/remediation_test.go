package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/charmbracelet/colorprofile"
	"github.com/dotcommander/cclauncher/internal/config"
)

func TestProviderDelimiterOwnership(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		args, want []string
		provider   string
	}{
		{[]string{"--", "--provider", "x", "--", "-p"}, []string{"--provider", "x", "--", "-p"}, ""},
		{[]string{"-p", "x", "--help"}, []string{"--help"}, "x"},
		{[]string{"--provider=x", "--", "--provider=y"}, []string{"--provider=y"}, "x"},
	} {
		t.Run(tc.provider+tc.args[0], func(t *testing.T) {
			t.Parallel()
			p, args, err := extractProviderFromArgs(tc.args)
			if err != nil || p != tc.provider || !reflect.DeepEqual(args, tc.want) {
				t.Fatalf("got %s %v %v", p, args, err)
			}
		})
	}
}

func TestNullIsNotTerminal(t *testing.T) {
	t.Parallel()
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	if isInteractive(f) || isTerminalOutput(f) || isTerminalOutput(&bytes.Buffer{}) {
		t.Fatal("nonterminal detected as terminal")
	}
}

func TestDoctorEmptyJSONArray(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	if err := HandleDoctor(context.Background(), &out, &config.Config{Providers: map[string]config.Provider{}}, DoctorOptions{JSON: true}); err != nil {
		t.Fatal(err)
	}
	var decoded []any
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil || decoded == nil || len(decoded) != 0 {
		t.Fatalf("output=%s err=%v", out.String(), err)
	}
}

type doctorRoundTrip func(*http.Request) (*http.Response, error)

func (f doctorRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestDoctorProbeConcurrencyAndOrdering(t *testing.T) {
	t.Parallel()
	for _, limit := range []int{1, 2} {
		t.Run(string(rune('0'+limit)), func(t *testing.T) {
			t.Parallel()
			names := []string{"a", "b", "c"}
			providers := map[string]config.Provider{}
			for _, name := range names {
				providers[name] = config.Provider{BaseURL: "https://" + name}
			}
			entered := make(chan struct{}, len(names))
			release := make(chan struct{})
			var active, maxActive atomic.Int32
			client := &http.Client{Transport: doctorRoundTrip(func(r *http.Request) (*http.Response, error) {
				n := active.Add(1)
				for old := maxActive.Load(); n > old && !maxActive.CompareAndSwap(old, n); old = maxActive.Load() {
				}
				entered <- struct{}{}
				select {
				case <-release:
				case <-r.Context().Done():
				}
				active.Add(-1)
				return &http.Response{StatusCode: 200, Body: http.NoBody, Header: make(http.Header)}, nil
			})}
			done := make(chan []string, 1)
			go func() {
				results, err := probeProviders(context.Background(), client, names, providers, limit)
				if err != nil {
					done <- nil
					return
				}
				got := []string{}
				for _, r := range results {
					got = append(got, r.Provider)
				}
				done <- got
			}()
			for range limit {
				<-entered
			}
			close(release)
			got := <-done
			if !reflect.DeepEqual(got, names) || maxActive.Load() > int32(limit) {
				t.Fatalf("order=%v concurrency=%d", got, maxActive.Load())
			}
		})
	}
}

func TestDoctorProbeCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := probeProviders(ctx, &http.Client{}, []string{"a"}, map[string]config.Provider{}, 1)
	if err != context.Canceled {
		t.Fatal(err)
	}
}

func TestColorProfileUsesSuppliedWriter(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	want := colorprofile.Detect(&out, os.Environ())
	if got := profileWriter(&out).Profile; got != want {
		t.Fatalf("profile=%v want=%v", got, want)
	}
}

func TestDoctorCancellationStopsActiveProbesAndQueuedRequests(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	started := make(chan struct{}, 3)
	stopped := make(chan struct{}, 3)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		started <- struct{}{}
		// The blocked probe exits only when its client request is canceled.
		<-r.Context().Done()
		stopped <- struct{}{}
	}))
	t.Cleanup(server.Close)
	providers := map[string]config.Provider{}
	names := []string{"a", "b", "queued"}
	for _, name := range names {
		providers[name] = config.Provider{BaseURL: server.URL}
	}
	done := make(chan error, 1)
	go func() {
		_, err := probeProviders(ctx, server.Client(), names, providers, 2)
		done <- err
	}()
	// Both workers are occupied before canceling, leaving the third job queued.
	<-started
	<-started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
	<-stopped
	<-stopped
	if got := requests.Load(); got != 2 {
		t.Fatalf("network requests = %d; queued request ran after cancellation", got)
	}
}
