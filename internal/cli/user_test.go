package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chill-institute/chill-cli/internal/config"
)

func TestUserTransferAddUsesStoredToken(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v4/chill.v4.UserService/AddTransfer" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer saved-token" {
			t.Fatalf("Authorization = %q", got)
		}
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload["url"] != "magnet:?xt=urn:btih:123" {
			t.Fatalf("url = %v", payload["url"])
		}
		catalogOrigin, ok := payload["catalogOrigin"].(map[string]any)
		if !ok || catalogOrigin["tvShowsSource"] != "TV_SHOWS_SOURCE_NETFLIX" {
			t.Fatalf("catalogOrigin = %#v", payload["catalogOrigin"])
		}
		_, _ = writer.Write([]byte(`{"status":"queued"}`))
	}))
	defer server.Close()

	configPath := filepath.Join(t.TempDir(), "config.json")
	store, err := config.NewStore(configPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Save(config.Config{APIBaseURL: server.URL, AuthToken: "saved-token"}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	stdout := &bytes.Buffer{}
	command := newUserCommand(&appContext{
		opts:   &appOptions{configPath: configPath, output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"transfer", "add", "--url", "magnet:?xt=urn:btih:123", "--tv-source", "netflix"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(stdout.String(), `"status":"queued"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestUserTransferGetUsesStoredToken(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v4/chill.v4.UserService/GetTransfer" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer saved-token" {
			t.Fatalf("Authorization = %q", got)
		}
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload["id"] != "42" && payload["id"] != float64(42) {
			t.Fatalf("id = %#v", payload["id"])
		}
		_, _ = writer.Write([]byte(`{"transfer":{"id":"42","name":"The Secret Agent","status":"COMPLETED","percentDone":100,"isFinished":true,"fileUrl":"https://put.io/files/42"}}`))
	}))
	defer server.Close()

	configPath := filepath.Join(t.TempDir(), "config.json")
	store, err := config.NewStore(configPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Save(config.Config{APIBaseURL: server.URL, AuthToken: "saved-token"}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	stdout := &bytes.Buffer{}
	command := newUserCommand(&appContext{
		opts:   &appOptions{configPath: configPath, output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"transfer", "get", "42"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(stdout.String(), `"id":"42"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}
