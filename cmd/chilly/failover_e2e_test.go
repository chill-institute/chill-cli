package main

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"sync/atomic"
	"testing"
)

type fakeAPIHost struct {
	server *httptest.Server
	hits   atomic.Int32
}

func newFakeAPIHost(t *testing.T, status int) *fakeAPIHost {
	t.Helper()
	host := &fakeAPIHost{}
	host.server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		host.hits.Add(1)
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(status)
		if status == http.StatusOK {
			_, _ = writer.Write([]byte(`{"user_id":"user-e2e","transfer":{"id":"1"}}`))
			return
		}
		_, _ = writer.Write([]byte(`{"code":"unavailable","message":"down"}`))
	}))
	t.Cleanup(host.server.Close)
	return host
}

func unreachableAPIURL(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	address := listener.Addr().String()
	_ = listener.Close()
	return "http://" + address
}

func configureHosts(t *testing.T, baseURLs string) string {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "config.json")
	if stdout, stderr, exitCode := runCLI("--config", configPath, "--output", "json", "settings", "set", "api-base-url", baseURLs); exitCode != 0 {
		t.Fatalf("settings set exit = %d; stdout = %s; stderr = %s", exitCode, stdout, stderr)
	}
	if stdout, stderr, exitCode := runCLI("--config", configPath, "--output", "json", "auth", "login", "--token", "test-token", "--skip-verify"); exitCode != 0 {
		t.Fatalf("auth login exit = %d; stdout = %s; stderr = %s", exitCode, stdout, stderr)
	}
	return configPath
}

func TestCLIFailsOverInConfiguredOrderOnEveryCall(t *testing.T) {
	unavailable := newFakeAPIHost(t, http.StatusServiceUnavailable)
	healthy := newFakeAPIHost(t, http.StatusOK)
	configPath := configureHosts(t, unavailable.server.URL+","+healthy.server.URL)

	for run := 1; run <= 2; run++ {
		if _, stderr, exitCode := runCLI("--config", configPath, "--output", "json", "whoami"); exitCode != 0 {
			t.Fatalf("run %d: whoami exit = %d; stderr = %s", run, exitCode, stderr)
		}
	}
	if got := unavailable.hits.Load(); got != 2 {
		t.Fatalf("unavailable host hits = %d, want 2 (configured host should be tried first)", got)
	}
	if got := healthy.hits.Load(); got != 2 {
		t.Fatalf("healthy host hits = %d, want 2", got)
	}

	stdout, stderr, exitCode := runCLI("--config", configPath, "--output", "json", "settings", "show")
	if exitCode != 0 {
		t.Fatalf("settings show exit = %d; stderr = %s", exitCode, stderr)
	}
	var shown struct {
		APIBaseURLs []string `json:"api_base_urls"`
	}
	if err := json.Unmarshal([]byte(stdout), &shown); err != nil {
		t.Fatalf("decode settings show: %v", err)
	}
	if want := []string{unavailable.server.URL, healthy.server.URL}; !slices.Equal(shown.APIBaseURLs, want) {
		t.Fatalf("api_base_urls = %v, want %v", shown.APIBaseURLs, want)
	}
}

func TestCLIFailsOverOnConnectFailureInListedOrder(t *testing.T) {
	second := newFakeAPIHost(t, http.StatusOK)
	third := newFakeAPIHost(t, http.StatusOK)
	configPath := configureHosts(t, unreachableAPIURL(t)+","+second.server.URL+","+third.server.URL)

	if _, stderr, exitCode := runCLI("--config", configPath, "--output", "json", "add-transfer", "--url", "magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567"); exitCode != 0 {
		t.Fatalf("add-transfer exit = %d; stderr = %s", exitCode, stderr)
	}
	if second.hits.Load() != 1 || third.hits.Load() != 0 {
		t.Fatalf("hits second = %d, third = %d; want 1, 0", second.hits.Load(), third.hits.Load())
	}
}

func TestCLIDoesNotReplayTransferOnAnotherHost(t *testing.T) {
	unavailable := newFakeAPIHost(t, http.StatusServiceUnavailable)
	fallback := newFakeAPIHost(t, http.StatusOK)
	configPath := configureHosts(t, unavailable.server.URL+","+fallback.server.URL)

	_, stderr, exitCode := runCLI("--config", configPath, "--output", "json", "add-transfer", "--url", "magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567")
	if exitCode == 0 {
		t.Fatal("add-transfer exit = 0, want failure from unavailable host")
	}
	if envelope := decodeCLIError(t, stderr); envelope.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("error envelope = %#v", envelope)
	}
	if unavailable.hits.Load() != 1 || fallback.hits.Load() != 0 {
		t.Fatalf("hits unavailable = %d, fallback = %d; want 1, 0", unavailable.hits.Load(), fallback.hits.Load())
	}
}

func TestCLIReturnsApplicationErrorsWithoutFailover(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusInternalServerError} {
		failing := newFakeAPIHost(t, status)
		fallback := newFakeAPIHost(t, http.StatusOK)
		configPath := configureHosts(t, failing.server.URL+","+fallback.server.URL)

		_, stderr, exitCode := runCLI("--config", configPath, "--output", "json", "whoami")
		if exitCode == 0 {
			t.Fatalf("status %d: whoami exit = 0", status)
		}
		if envelope := decodeCLIError(t, stderr); envelope.StatusCode != status {
			t.Fatalf("status %d: error envelope = %#v", status, envelope)
		}
		if fallback.hits.Load() != 0 {
			t.Fatalf("status %d: fallback hits = %d, want 0", status, fallback.hits.Load())
		}
	}
}

func TestCLICustomSingleHostDoesNotFailOver(t *testing.T) {
	unavailable := newFakeAPIHost(t, http.StatusServiceUnavailable)
	configPath := configureHosts(t, unavailable.server.URL)

	_, stderr, exitCode := runCLI("--config", configPath, "--output", "json", "whoami")
	if exitCode == 0 {
		t.Fatal("whoami exit = 0, want failure")
	}
	if envelope := decodeCLIError(t, stderr); envelope.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("error envelope = %#v", envelope)
	}

	stdout, _, _ := runCLI("--config", configPath, "--output", "json", "doctor", "--offline")
	var report struct {
		API struct {
			BaseURLs []string `json:"base_urls"`
		} `json:"api"`
	}
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("decode doctor: %v; stdout = %s", err, stdout)
	}
	if !slices.Equal(report.API.BaseURLs, []string{unavailable.server.URL}) {
		t.Fatalf("doctor api.base_urls = %v", report.API.BaseURLs)
	}
}
