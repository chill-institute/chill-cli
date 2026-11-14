package cli

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
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

func TestCommandsReserveTimeForFailover(t *testing.T) {
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
			start := time.Now()
			err := command.Execute()
			if firstCalls.Load() != 1 {
				t.Fatalf("first host calls = %d", firstCalls.Load())
			}
			if !mutation && time.Since(start) >= app.rpcTimeout {
				t.Fatal("command exhausted overall budget")
			}
			if mutation {
				if err == nil || secondCalls.Load() != 0 {
					t.Fatalf("mutation error = %v, second calls = %d", err, secondCalls.Load())
				}
			} else if err != nil || secondCalls.Load() != 1 || !strings.Contains(app.stdout.(*bytes.Buffer).String(), "fallback-result") {
				t.Fatalf("search error = %v, second calls = %d, output = %s", err, secondCalls.Load(), app.stdout)
			}
		})
	}
}

func TestRememberedHostDoesNotRewriteCredentialsAfterConcurrentLogout(t *testing.T) {
	entered := make(chan struct{})
	resume := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-resume
		_, _ = w.Write([]byte(`{"results":[]}`))
	}))
	defer server.Close()
	app := failoverTestApp(t, "http://127.0.0.1:1,"+server.URL)
	app.rpcTimeout = 5 * time.Second
	done := make(chan error, 1)
	go func() {
		command := newSearchCommand(app)
		command.SetArgs([]string{"--query", "example"})
		done <- command.Execute()
	}()
	<-entered
	logout := failoverTestApp(t, server.URL)
	logout.opts.configPath = app.opts.configPath
	command := newAuthLogoutCommand(logout)
	command.SetArgs(nil)
	err := command.Execute()
	if err != nil {
		close(resume)
		t.Fatal(errors.Join(err, <-done))
	}
	before, err := os.ReadFile(app.opts.configPath)
	if err != nil {
		close(resume)
		t.Fatal(errors.Join(err, <-done))
	}
	close(resume)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(app.opts.configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("remembering host rewrote credential file")
	}
	cfg, err := app.loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AuthToken != "" || cfg.APIBaseURLs()[0] != server.URL {
		t.Fatalf("token cleared = %t, preferred host = %s", cfg.AuthToken == "", cfg.APIBaseURLs()[0])
	}
}
