package cli

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chill-institute/chill-cli/v2/pkg/config"
	"github.com/chill-institute/chill-cli/v2/pkg/rpc"
)

func TestLoginWithoutHealthEndpoint(t *testing.T) {
	for _, multiple := range []bool{false, true} {
		t.Run(map[bool]string{false: "single custom host", true: "all probes fail"}[multiple], func(t *testing.T) {
			var probes atomic.Int32
			primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/healthz" {
					probes.Add(1)
					http.NotFound(w, r)
					return
				}
				_, _ = w.Write([]byte(`{"user_id":"test-user"}`))
			}))
			defer primary.Close()
			hosts := primary.URL
			if multiple {
				hosts += ",http://127.0.0.1:1"
			}
			app := failoverTestApp(t, hosts)
			app.stderr = &bytes.Buffer{}
			app.isInputTerminal = func(io.Reader) bool { return true }
			app.readSecret = func(io.Reader) (string, error) { return "new-token", nil }
			cmd := newAuthLoginCommand(app)
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			cfg, err := app.loadConfig()
			if err != nil || cfg.AuthToken != "new-token" {
				t.Fatalf("login config=%+v err=%v", cfg, err)
			}
			if !multiple && probes.Load() != 0 {
				t.Fatal("single host was probed")
			}
			if multiple && !strings.Contains(app.stderr.(*bytes.Buffer).String(), "Warning: API health probes failed") {
				t.Fatalf("missing warning: %s", app.stderr)
			}
		})
	}
}

func TestLoginChoiceReplacesRememberedHost(t *testing.T) {
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{}`)) }))
	defer primary.Close()
	app := failoverTestApp(t, primary.URL+",http://127.0.0.1:1")
	store, _ := app.configStore()
	if err := store.RememberWorkingAPIBaseURL("http://127.0.0.1:1"); err != nil {
		t.Fatal(err)
	}
	app.isInputTerminal = func(io.Reader) bool { return true }
	app.readSecret = func(io.Reader) (string, error) { return "token", nil }
	cmd := newAuthLoginCommand(app)
	cmd.SetArgs([]string{"--skip-verify"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	cfg, err := store.Load()
	if err != nil || cfg.APIBaseURLs()[0] != primary.URL {
		t.Fatalf("config=%+v err=%v", cfg, err)
	}
}

func TestSettingsResetClearsRememberedHost(t *testing.T) {
	app := failoverTestApp(t, "https://api.chill.institute")
	store, _ := app.configStore()
	if err := store.RememberWorkingAPIBaseURL("https://api.binge.institute"); err != nil {
		t.Fatal(err)
	}
	cmd := newSettingsSetCommand(app)
	cmd.SetArgs([]string{"api-base-url", "https://api.chill.institute"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	cfg, err := store.Load()
	if err != nil || cfg.APIBaseURLs()[0] != "https://api.chill.institute" {
		t.Fatalf("config=%+v err=%v", cfg, err)
	}
}

func TestAPIOverrideDoesNotChangeStoredPreference(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"results":[]}`)) }))
	defer server.Close()
	app := failoverTestApp(t, "https://api.chill.institute")
	store, _ := app.configStore()
	if err := store.RememberWorkingAPIBaseURL("https://api.binge.institute"); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(store.Path() + ".host")
	if err != nil {
		t.Fatal(err)
	}
	app.opts.apiURL = server.URL + ",http://127.0.0.1:1"
	cmd := newSearchCommand(app)
	cmd.SetArgs([]string{"--query", "example"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(store.Path() + ".host")
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("stored hint changed: %s %v", after, err)
	}
}

func TestLogoutDoesNotSerializeRememberedHost(t *testing.T) {
	app := failoverTestApp(t, "https://api.chill.institute")
	store, _ := app.configStore()
	if err := store.RememberWorkingAPIBaseURL("https://api.binge.institute"); err != nil {
		t.Fatal(err)
	}
	cmd := newAuthLogoutCommand(app)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(store.Path())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "last_working") || strings.Contains(string(data), "binge") || strings.Contains(string(data), "test-token") {
		t.Fatalf("unexpected persisted config: %s", data)
	}
	cfg, err := store.Load()
	if err != nil || cfg.APIBaseURLs()[0] != "https://api.binge.institute" {
		t.Fatalf("config=%+v err=%v", cfg, err)
	}
}

func TestDoctorReportsUnreadableRememberedHost(t *testing.T) {
	app := failoverTestApp(t, "https://api.chill.institute")
	if err := os.Mkdir(app.opts.configPath+".host", 0700); err != nil {
		t.Fatal(err)
	}
	report, err := app.buildDoctorReport(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	api := report["api"].(map[string]any)
	warning, _ := api["remembered_host_warning"].(string)
	pretty, _, renderErr := renderDoctorPretty(report)
	if renderErr != nil || !strings.Contains(pretty, "read remembered API host") {
		t.Fatalf("pretty doctor=%s err=%v", pretty, renderErr)
	}
	if !strings.Contains(warning, "read remembered API host") {
		t.Fatalf("warning=%q", warning)
	}
	cfg, err := config.NewStore(app.opts.configPath)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := cfg.Load()
	if err != nil || loaded.APIBaseURLs()[0] != "https://api.chill.institute" {
		t.Fatalf("config=%+v err=%v", loaded, err)
	}
}

func TestRPCPreservesCallerDeadline(t *testing.T) {
	app := failoverTestApp(t, "https://api.chill.institute")
	app.rpcTimeout = time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	callCtx, stop := app.rpcCallContext(ctx)
	defer stop()
	select {
	case <-callCtx.Done():
		t.Fatalf("caller budget shortened: %v", callCtx.Err())
	case <-time.After(20 * time.Millisecond):
	}
	cancelled, cancelNow := context.WithCancel(ctx)
	cancelNow()
	callCtx, stop = app.rpcCallContext(cancelled)
	defer stop()
	_, err := rpc.NewClient("https://api.example", nil).Call(callCtx, rpc.CallRequest{Procedure: "test/Read", AuthMode: rpc.AuthNone, ReadOnly: true})
	if err == nil {
		t.Fatal("ignored caller cancellation")
	}
}

func TestSlowMutationUsesFullBudget(t *testing.T) {
	var fallbackCalls atomic.Int32
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		_, _ = w.Write([]byte(`{"id":123}`))
	}))
	defer primary.Close()
	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallbackCalls.Add(1)
	}))
	defer fallback.Close()
	app := failoverTestApp(t, primary.URL+","+fallback.URL)
	command := newAddTransferCommand(app)
	command.SetArgs([]string{"--url", "https://example.test/file.torrent"})
	if err := command.Execute(); err != nil || fallbackCalls.Load() != 0 {
		t.Fatalf("mutation error=%v fallback calls=%d", err, fallbackCalls.Load())
	}
}

func TestHealthProbeIsBoundedAndLeavesRequestBudget(t *testing.T) {
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer primary.Close()
	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer fallback.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client := rpc.NewClient(primary.URL, nil, rpc.WithFailoverBaseURLs(fallback.URL))
	start := time.Now()
	host, err := client.FirstReachableBaseURL(ctx)
	if err != nil || host != fallback.URL || time.Since(start) > 3*time.Second {
		t.Fatalf("probe host=%s err=%v elapsed=%v", host, err, time.Since(start))
	}
	if _, err := rpc.NewClient(host, nil).Call(ctx, rpc.CallRequest{Procedure: "test/Read", AuthMode: rpc.AuthNone, ReadOnly: true}); err != nil {
		t.Fatalf("request after probe: %v", err)
	}
}

func TestExplicitHostOrderOverridesRememberedHost(t *testing.T) {
	app := failoverTestApp(t, "https://primary.example,https://fallback.example")
	store, _ := app.configStore()
	if err := store.RememberWorkingAPIBaseURL("https://fallback.example"); err != nil {
		t.Fatal(err)
	}
	app.opts.apiURL = "https://primary.example,https://fallback.example"
	cfg, err := app.loadConfig()
	if err != nil || cfg.APIBaseURLs()[0] != "https://primary.example" {
		t.Fatalf("explicit host ignored: %+v %v", cfg, err)
	}
}
