package cli

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chill-institute/chill-cli/internal/config"
)

func TestBuildAddTransferRequest(t *testing.T) {
	t.Parallel()

	app := &appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: &strings.Builder{},
		stderr: &strings.Builder{},
	}

	request, err := buildAddTransferRequest(app, " magnet:?xt=urn:btih:test ", "", "", "")
	if err != nil {
		t.Fatalf("buildAddTransferRequest(flags) error = %v", err)
	}
	if request["url"] != "magnet:?xt=urn:btih:test" {
		t.Fatalf("request = %#v", request)
	}

	request, err = buildAddTransferRequest(app, "", `{"url":" magnet:?xt=urn:btih:test ","catalogOrigin":{"moviesSource":"MOVIES_SOURCE_TRAKT"}}`, "", "")
	if err != nil {
		t.Fatalf("buildAddTransferRequest(json) error = %v", err)
	}
	if request["url"] != "magnet:?xt=urn:btih:test" {
		t.Fatalf("request = %#v", request)
	}
	catalogOrigin, ok := request["catalogOrigin"].(map[string]any)
	if !ok || catalogOrigin["moviesSource"] != "MOVIES_SOURCE_TRAKT" {
		t.Fatalf("catalogOrigin = %#v", request["catalogOrigin"])
	}

	request, err = buildAddTransferRequest(app, "magnet:?xt=urn:btih:test", "", "trakt", "")
	if err != nil {
		t.Fatalf("buildAddTransferRequest(movie source) error = %v", err)
	}
	catalogOrigin, ok = request["catalogOrigin"].(map[string]any)
	if !ok || catalogOrigin["moviesSource"] != "MOVIES_SOURCE_TRAKT" {
		t.Fatalf("catalogOrigin = %#v", request["catalogOrigin"])
	}

	request, err = buildAddTransferRequest(app, "magnet:?xt=urn:btih:test", "", "", "Netflix")
	if err != nil {
		t.Fatalf("buildAddTransferRequest(tv source) error = %v", err)
	}
	catalogOrigin, ok = request["catalogOrigin"].(map[string]any)
	if !ok || catalogOrigin["tvShowsSource"] != "TV_SHOWS_SOURCE_NETFLIX" {
		t.Fatalf("catalogOrigin = %#v", request["catalogOrigin"])
	}

	for _, tc := range []struct {
		name        string
		transferURL string
		rawRequest  string
		movieSource string
		tvSource    string
	}{
		{name: "ambiguous", transferURL: "magnet:?xt=urn:btih:test", rawRequest: `{"url":"magnet:?xt=urn:btih:test"}`},
		{name: "json with movie source", rawRequest: `{"url":"magnet:?xt=urn:btih:test"}`, movieSource: "trakt"},
		{name: "both sources", transferURL: "magnet:?xt=urn:btih:test", movieSource: "trakt", tvSource: "netflix"},
		{name: "invalid movie source", transferURL: "magnet:?xt=urn:btih:test", movieSource: "not-real"},
		{name: "invalid tv source", transferURL: "magnet:?xt=urn:btih:test", tvSource: "not-real"},
		{name: "json missing url", rawRequest: `{"id":"nope"}`},
		{name: "json invalid url type", rawRequest: `{"url":true}`},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := buildAddTransferRequest(app, tc.transferURL, tc.rawRequest, tc.movieSource, tc.tvSource); err == nil {
				t.Fatalf("buildAddTransferRequest(%s) error = nil, want error", tc.name)
			}
		})
	}
}

func TestLoadCurrentUserSettings(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			_, _ = writer.Write([]byte(`{"search":{"filterNastyResults":true}}`))
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

		settings, err := loadCurrentUserSettings(&appContext{opts: &appOptions{configPath: configPath, output: outputJSON}})
		if err != nil {
			t.Fatalf("loadCurrentUserSettings() error = %v", err)
		}
		search, ok := settings["search"].(map[string]any)
		if !ok || search["filterNastyResults"] != true {
			t.Fatalf("settings = %#v", settings)
		}
	})

	t.Run("null body becomes empty object", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			_, _ = writer.Write([]byte(`null`))
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

		settings, err := loadCurrentUserSettings(&appContext{opts: &appOptions{configPath: configPath, output: outputJSON}})
		if err != nil {
			t.Fatalf("loadCurrentUserSettings() error = %v", err)
		}
		if len(settings) != 0 {
			t.Fatalf("settings = %#v, want empty object", settings)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			_, _ = writer.Write([]byte(`{`))
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

		if _, err := loadCurrentUserSettings(&appContext{opts: &appOptions{configPath: configPath, output: outputJSON}}); err == nil {
			t.Fatal("loadCurrentUserSettings() error = nil, want decode error")
		}
	})
}
