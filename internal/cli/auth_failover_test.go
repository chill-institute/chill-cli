package cli

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/chill-institute/chill-cli/v2/pkg/config"
)

func TestAuthLoginStartsBrowserFlowOnFirstReachableHost(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	unreachable := "http://" + listener.Addr().String()
	_ = listener.Close()

	reachable := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/healthz":
			writer.WriteHeader(http.StatusOK)
		case "/v4/chill.v4.UserService/GetUserProfile":
			_, _ = writer.Write([]byte(`{"user_id":"user-123"}`))
		default:
			t.Errorf("unexpected path %q", request.URL.Path)
		}
	}))
	defer reachable.Close()

	configPath := filepath.Join(t.TempDir(), "config.json")
	var openedURL string
	app := &appContext{
		opts:            &appOptions{configPath: configPath, apiURL: unreachable + "," + reachable.URL, output: outputJSON},
		stdin:           strings.NewReader(""),
		stdout:          &bytes.Buffer{},
		stderr:          &bytes.Buffer{},
		authFlowTimeout: 2 * time.Second,
		openURL: func(rawURL string) error {
			openedURL = rawURL
			return completeLoopbackLogin(t, rawURL, "failover-token")
		},
	}

	command := newAuthLoginCommand(app)
	command.SetArgs([]string{"--local-browser"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.HasPrefix(openedURL, reachable.URL+"/auth/putio/start?") {
		t.Fatalf("login URL = %q, want it on %s", openedURL, reachable.URL)
	}
	store, err := config.NewStore(configPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.AuthToken != "failover-token" || cfg.LastWorkingAPIBaseURL != reachable.URL {
		t.Fatalf("saved config = %+v", cfg)
	}
}

func completeLoopbackLogin(t *testing.T, loginURL string, token string) error {
	t.Helper()
	parsed, err := url.Parse(loginURL)
	if err != nil {
		return err
	}
	callback, err := http.Get(parsed.Query().Get("success_url"))
	if err != nil {
		return err
	}
	page, err := io.ReadAll(callback.Body)
	_ = callback.Body.Close()
	if err != nil {
		return err
	}
	match := regexp.MustCompile(`data-token-endpoint="([^"]+)"`).FindStringSubmatch(string(page))
	if len(match) != 2 {
		t.Fatalf("token endpoint not found in callback page")
	}
	response, err := http.Post(match[1], "application/json", strings.NewReader(`{"auth_token":"`+token+`"}`))
	if err != nil {
		return err
	}
	return response.Body.Close()
}
