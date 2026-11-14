package cli

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chill-institute/chill-cli/v2/pkg/config"
)

func failoverTestApp(t *testing.T, hosts string) *appContext {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	store, err := config.NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(config.Config{APIBaseURL: hosts, AuthToken: "test-token"}); err != nil {
		t.Fatal(err)
	}
	return &appContext{
		opts:  &appOptions{configPath: path, output: outputJSON},
		stdin: strings.NewReader(""), stdout: &bytes.Buffer{}, stderr: io.Discard,
		rpcTimeout: 400 * time.Millisecond,
	}
}

func TestCommandsDoNotFailOverOnTimeout(t *testing.T) {
	for _, mutation := range []bool{false, true} {
		name := "search"
		if mutation {
			name = "add-transfer"
		}
		t.Run(name, func(t *testing.T) {
			var firstCalls, secondCalls atomic.Int32
			release := make(chan struct{})
			first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				firstCalls.Add(1)
				<-release
			}))
			defer first.Close()
			defer close(release)
			second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				secondCalls.Add(1)
				_, _ = w.Write([]byte(`{"results":[{"title":"fallback-result"}]}`))
			}))
			defer second.Close()
			app := failoverTestApp(t, first.URL+","+second.URL)
			command := newSearchCommand(app)
			command.SetArgs([]string{"--query", "example"})
			if mutation {
				command = newAddTransferCommand(app)
				command.SetArgs([]string{"--url", "https://example.test/file.torrent"})
			}
			err := command.Execute()
			if firstCalls.Load() != 1 {
				t.Fatalf("first host calls = %d", firstCalls.Load())
			}
			if err == nil || secondCalls.Load() != 0 {
				t.Fatalf("error = %v, second calls = %d", err, secondCalls.Load())
			}
		})
	}
}

func TestSearchFailsOverOn503InConfiguredOrder(t *testing.T) {
	for _, override := range []bool{false, true} {
		name := "settings"
		if override {
			name = "api-url"
		}
		t.Run(name, func(t *testing.T) {
			var firstCalls, secondCalls atomic.Int32
			first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				firstCalls.Add(1)
				w.WriteHeader(http.StatusServiceUnavailable)
			}))
			defer first.Close()
			second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				secondCalls.Add(1)
				_, _ = w.Write([]byte(`{"results":[{"title":"fallback-result"}]}`))
			}))
			defer second.Close()
			hosts := first.URL + "," + second.URL
			app := failoverTestApp(t, hosts)
			if override {
				app = failoverTestApp(t, second.URL)
				app.opts.apiURL = hosts
			}
			for range 2 {
				command := newSearchCommand(app)
				command.SetArgs([]string{"--query", "example"})
				if err := command.Execute(); err != nil {
					t.Fatal(err)
				}
			}
			if firstCalls.Load() != 2 || secondCalls.Load() != 2 || !strings.Contains(app.stdout.(*bytes.Buffer).String(), "fallback-result") {
				t.Fatalf("first=%d second=%d output=%s", firstCalls.Load(), secondCalls.Load(), app.stdout)
			}
		})
	}
}
