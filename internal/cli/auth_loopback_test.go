package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/chill-institute/chill-cli/internal/config"
)

func TestAuthLoginWebTokenFlowPromptsForTokenAndVerifies(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v4/chill.v4.UserService/GetUserProfile" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer web-token" {
			t.Fatalf("Authorization = %q", got)
		}
		_, _ = writer.Write([]byte(`{"user_id":"user-123"}`))
	}))
	defer server.Close()

	configPath := filepath.Join(t.TempDir(), "config.json")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	app := &appContext{
		opts:            &appOptions{configPath: configPath, apiURL: server.URL, output: outputJSON},
		stdin:           strings.NewReader(""),
		stdout:          stdout,
		stderr:          stderr,
		authFlowTimeout: 2 * time.Second,
		isInputTerminal: func(io.Reader) bool { return true },
		readSecret: func(io.Reader) (string, error) {
			return "web-token", nil
		},
		openURL: func(string) error {
			t.Fatal("openURL should not be called for the default web token flow")
			return nil
		},
	}

	command := newAuthLoginCommand(app)
	command.SetArgs(nil)
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	store, err := config.NewStore(configPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.AuthToken != "web-token" {
		t.Fatalf("AuthToken = %q, want %q", cfg.AuthToken, "web-token")
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("output json decode error: %v", err)
	}
	if output["status"] != "ok" {
		t.Fatalf("status = %v", output["status"])
	}
	if output["saved"] != true {
		t.Fatalf("saved = %v", output["saved"])
	}
	if !strings.Contains(stderr.String(), "1. Open this URL in a signed-in browser:") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "2. Copy the setup token.") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "3. Paste it below.") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Paste setup token: ") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "/auth/cli-token") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestAuthLoginLocalBrowserFlowCapturesLoopbackToken(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v4/chill.v4.UserService/GetUserProfile" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer loopback-token" {
			t.Fatalf("Authorization = %q", got)
		}
		_, _ = writer.Write([]byte(`{"user_id":"user-123"}`))
	}))
	defer server.Close()

	configPath := filepath.Join(t.TempDir(), "config.json")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	app := &appContext{
		opts:            &appOptions{configPath: configPath, apiURL: server.URL, output: outputNDJSON},
		stdin:           strings.NewReader(""),
		stdout:          stdout,
		stderr:          stderr,
		authFlowTimeout: 2 * time.Second,
		openURL: func(rawURL string) error {
			loginURL, err := url.Parse(rawURL)
			if err != nil {
				t.Fatalf("url.Parse(loginURL) error = %v", err)
			}
			if loginURL.Path != "/auth/putio/start" {
				t.Fatalf("login path = %q", loginURL.Path)
			}

			successURL := strings.TrimSpace(loginURL.Query().Get("success_url"))
			if successURL == "" {
				t.Fatal("missing success_url")
			}

			callbackResponse, err := http.Get(successURL)
			if err != nil {
				t.Fatalf("http.Get(successURL) error = %v", err)
			}
			defer func() {
				_ = callbackResponse.Body.Close()
			}()

			if callbackResponse.StatusCode != http.StatusOK {
				t.Fatalf("callback status = %d, want %d", callbackResponse.StatusCode, http.StatusOK)
			}

			body, err := io.ReadAll(callbackResponse.Body)
			if err != nil {
				t.Fatalf("io.ReadAll(callbackResponse.Body) error = %v", err)
			}

			match := regexp.MustCompile(`data-token-endpoint="([^"]+)"`).FindStringSubmatch(string(body))
			if len(match) != 2 {
				t.Fatalf("token endpoint not found in callback page: %s", string(body))
			}

			postBody := strings.NewReader(`{"auth_token":"loopback-token"}`)
			postResponse, err := http.Post(match[1], "application/json", postBody)
			if err != nil {
				t.Fatalf("http.Post(tokenEndpoint) error = %v", err)
			}
			defer func() {
				_ = postResponse.Body.Close()
			}()

			if postResponse.StatusCode != http.StatusOK {
				t.Fatalf("token post status = %d, want %d", postResponse.StatusCode, http.StatusOK)
			}

			return nil
		},
	}

	command := newAuthLoginCommand(app)
	command.SetArgs([]string{"--local-browser"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	store, err := config.NewStore(configPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.AuthToken != "loopback-token" {
		t.Fatalf("AuthToken = %q, want %q", cfg.AuthToken, "loopback-token")
	}
	if strings.Contains(stdout.String(), "Open this URL") {
		t.Fatalf("stdout = %q, want machine-readable output only", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Open this URL to authenticate:") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestAuthLoginWebTokenFlowRejectsNonInteractiveInput(t *testing.T) {
	t.Parallel()

	app := &appContext{
		opts:            &appOptions{output: outputJSON},
		stdin:           strings.NewReader(""),
		stdout:          &bytes.Buffer{},
		stderr:          &bytes.Buffer{},
		isInputTerminal: func(io.Reader) bool { return false },
	}

	_, err := app.loginWithWebToken(config.Config{APIBaseURL: "https://api.chill.institute"}, true)
	if err == nil || !strings.Contains(err.Error(), "requires a terminal for token entry") {
		t.Fatalf("loginWithWebToken() error = %v", err)
	}
}

func TestAuthLoginWebTokenFlowSupportsPromptOnly(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	app := &appContext{
		opts:            &appOptions{output: outputNDJSON},
		stdin:           strings.NewReader(""),
		stdout:          stdout,
		stderr:          stderr,
		isInputTerminal: func(io.Reader) bool { return true },
		readSecret: func(io.Reader) (string, error) {
			return "manual-token", nil
		},
		openURL: func(string) error {
			t.Fatal("openURL should not be called for the default web token flow")
			return nil
		},
	}

	token, err := app.loginWithWebToken(config.Config{APIBaseURL: "https://api.chill.institute"}, false)
	if err != nil {
		t.Fatalf("loginWithWebToken() error = %v", err)
	}
	if token != "manual-token" {
		t.Fatalf("token = %q, want %q", token, "manual-token")
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "/auth/cli-token") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestLoopbackAuthWaitForTokenReturnsServeError(t *testing.T) {
	t.Parallel()

	flow := &loopbackAuthFlow{tokenCh: make(chan string, 1)}
	errCh := make(chan error, 1)
	errCh <- errors.New("boom")

	_, err := flow.waitForToken(context.Background(), errCh)
	if err == nil || !strings.Contains(err.Error(), "serve oauth callback") {
		t.Fatalf("waitForToken() error = %v", err)
	}
}

func TestLoopbackAuthWaitForTokenTimesOut(t *testing.T) {
	t.Parallel()

	flow := &loopbackAuthFlow{tokenCh: make(chan string, 1)}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	_, err := flow.waitForToken(ctx, make(chan error))
	if err == nil || !strings.Contains(err.Error(), "timed out waiting for browser authentication") {
		t.Fatalf("waitForToken() error = %v", err)
	}
}

func TestLoopbackHandleCallbackRejectsWrongMethod(t *testing.T) {
	t.Parallel()

	flow := &loopbackAuthFlow{
		baseURL:   "http://127.0.0.1:9999",
		tokenPath: "/auth/token/test",
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/auth/callback/test", nil)
	flow.handleCallback(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
	if allow := recorder.Header().Get("Allow"); allow != http.MethodGet {
		t.Fatalf("Allow = %q, want %q", allow, http.MethodGet)
	}
}

func TestLoopbackHandleTokenValidatesRequests(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		method     string
		body       string
		statusCode int
	}{
		{name: "wrong method", method: http.MethodGet, body: "", statusCode: http.StatusMethodNotAllowed},
		{name: "invalid json", method: http.MethodPost, body: "{", statusCode: http.StatusBadRequest},
		{name: "missing token", method: http.MethodPost, body: `{}`, statusCode: http.StatusBadRequest},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			flow := &loopbackAuthFlow{tokenCh: make(chan string, 1)}
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(tc.method, "/auth/token/test", strings.NewReader(tc.body))

			flow.handleToken(recorder, request)

			if recorder.Code != tc.statusCode {
				t.Fatalf("status = %d, want %d", recorder.Code, tc.statusCode)
			}
		})
	}
}

func TestLoopbackHandleTokenRejectsSecondToken(t *testing.T) {
	t.Parallel()

	flow := &loopbackAuthFlow{tokenCh: make(chan string, 1)}

	firstRecorder := httptest.NewRecorder()
	firstRequest := httptest.NewRequest(http.MethodPost, "/auth/token/test", strings.NewReader(`{"auth_token":"token-1"}`))
	flow.handleToken(firstRecorder, firstRequest)
	if firstRecorder.Code != http.StatusOK {
		t.Fatalf("first status = %d, want %d", firstRecorder.Code, http.StatusOK)
	}

	secondRecorder := httptest.NewRecorder()
	secondRequest := httptest.NewRequest(http.MethodPost, "/auth/token/test", strings.NewReader(`{"auth_token":"token-2"}`))
	flow.handleToken(secondRecorder, secondRequest)
	if secondRecorder.Code != http.StatusConflict {
		t.Fatalf("second status = %d, want %d", secondRecorder.Code, http.StatusConflict)
	}
}
