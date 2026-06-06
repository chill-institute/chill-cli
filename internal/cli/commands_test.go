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

func TestAuthLoginWithTokenSavesConfig(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v4/chill.v4.UserService/GetUserProfile" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("Authorization = %q", got)
		}
		_, _ = writer.Write([]byte(`{"user_id":"user-123"}`))
	}))
	defer server.Close()

	configPath := filepath.Join(t.TempDir(), "config.json")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	app := &appContext{
		opts:    &appOptions{configPath: configPath, apiURL: server.URL, output: outputJSON},
		stdin:   strings.NewReader(""),
		stdout:  stdout,
		stderr:  stderr,
		openURL: func(string) error { return nil },
	}

	command := newAuthLoginCommand(app)
	command.SetArgs([]string{"--token", "test-token"})
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
	if cfg.AuthToken != "test-token" {
		t.Fatalf("AuthToken = %q, want %q", cfg.AuthToken, "test-token")
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
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestCompletionCommandGeneratesZshScript(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newRootCommand(&appContext{
		opts:   &appOptions{output: outputPretty},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"completion", "zsh"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	rendered := stdout.String()
	if !strings.Contains(rendered, "#compdef chilly") {
		t.Fatalf("expected zsh completion header in %q", rendered)
	}
	if !strings.Contains(rendered, "_chilly") {
		t.Fatalf("expected zsh completion function in %q", rendered)
	}
}

func TestCompletionCommandSupportsFishAndRejectsUnknownShells(t *testing.T) {
	t.Parallel()

	t.Run("fish", func(t *testing.T) {
		t.Parallel()

		stdout := &bytes.Buffer{}
		command := newRootCommand(&appContext{
			opts:   &appOptions{output: outputPretty},
			stdin:  strings.NewReader(""),
			stdout: stdout,
			stderr: &bytes.Buffer{},
		})
		command.SetArgs([]string{"completion", "fish"})
		if err := command.Execute(); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if !strings.Contains(stdout.String(), "complete -c chilly") {
			t.Fatalf("expected fish completion content in %q", stdout.String())
		}
	})

	t.Run("unsupported", func(t *testing.T) {
		t.Parallel()

		command := newRootCommand(&appContext{
			opts:   &appOptions{output: outputPretty},
			stdin:  strings.NewReader(""),
			stdout: &bytes.Buffer{},
			stderr: &bytes.Buffer{},
		})
		command.SetArgs([]string{"completion", "tcsh"})
		if err := command.Execute(); err == nil {
			t.Fatal("Execute() error = nil, want unsupported shell error")
		}
	})
}

func TestGetTransferHelpIncludesFieldsExample(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newRootCommand(&appContext{
		opts:   &appOptions{output: outputPretty},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"get-transfer", "--help"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	rendered := stdout.String()
	if !strings.Contains(rendered, "get-transfer 42 --fields") {
		t.Fatalf("expected get-transfer fields example in %q", rendered)
	}
}

func TestRootHelpIncludesExamples(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newRootCommand(&appContext{
		opts:   &appOptions{output: outputPretty},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"--help"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	rendered := stdout.String()
	if !strings.Contains(rendered, "Examples:") {
		t.Fatalf("expected examples in root help: %q", rendered)
	}
	if !strings.Contains(rendered, `chilly schema command search --output json`) {
		t.Fatalf("expected schema example in root help: %q", rendered)
	}
}

func TestUserHelpUsesProductFacingCopy(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newRootCommand(&appContext{
		opts:   &appOptions{output: outputPretty},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"user", "--help"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	rendered := stdout.String()
	if !strings.Contains(rendered, "Run user account commands through chill.institute") {
		t.Fatalf("expected product-facing user help: %q", rendered)
	}
	if strings.Contains(rendered, "Bearer auth") {
		t.Fatalf("unexpected internal auth phrasing in user help: %q", rendered)
	}
	if !strings.Contains(rendered, "chilly user settings get --output json") {
		t.Fatalf("expected example in user help: %q", rendered)
	}
}

func TestAuthLogoutClearsToken(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.json")
	store, err := config.NewStore(configPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Save(config.Config{APIBaseURL: "http://localhost:8080", AuthToken: "token-1"}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	stdout := &bytes.Buffer{}
	app := &appContext{
		opts:    &appOptions{configPath: configPath, output: outputJSON},
		stdin:   strings.NewReader(""),
		stdout:  stdout,
		stderr:  &bytes.Buffer{},
		openURL: func(string) error { return nil },
	}

	command := newAuthLogoutCommand(app)
	command.SetArgs(nil)
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.AuthToken != "" {
		t.Fatalf("AuthToken = %q, want empty", cfg.AuthToken)
	}
}

func TestAuthLogoutDryRunKeepsToken(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.json")
	store, err := config.NewStore(configPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Save(config.Config{APIBaseURL: "https://api.chill.institute", AuthToken: "token-1"}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	stdout := &bytes.Buffer{}
	app := &appContext{
		opts:   &appOptions{configPath: configPath, output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	}

	command := newAuthLogoutCommand(app)
	command.SetArgs([]string{"--dry-run"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.AuthToken != "token-1" {
		t.Fatalf("AuthToken = %q, want %q", cfg.AuthToken, "token-1")
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("output json decode error: %v", err)
	}
	if output["dry_run"] != true {
		t.Fatalf("dry_run = %v, want true", output["dry_run"])
	}
}

func TestSearchCommandUsesStoredToken(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v4/chill.v4.UserService/Search" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer saved-token" {
			t.Fatalf("Authorization = %q", got)
		}
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload["query"] != "blade runner" {
			t.Fatalf("query = %v", payload["query"])
		}
		_, _ = writer.Write([]byte(`{"query":"blade runner","results":[]}`))
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
	command := newSearchCommand(&appContext{
		opts:    &appOptions{configPath: configPath, output: outputJSON},
		stdin:   strings.NewReader(""),
		stdout:  stdout,
		stderr:  &bytes.Buffer{},
		openURL: func(string) error { return nil },
	})
	command.SetArgs([]string{"--query", "blade runner"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("output json decode error: %v", err)
	}
	if output["query"] != "blade runner" {
		t.Fatalf("output query = %v", output["query"])
	}
}

func TestGetTransferCommandUsesStoredToken(t *testing.T) {
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
		if payload["id"] != float64(42) {
			t.Fatalf("id = %v", payload["id"])
		}
		_, _ = writer.Write([]byte(`{"transfer":{"id":"42","status":"COMPLETED"}}`))
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
	command := newGetTransferCommand(&appContext{
		opts:   &appOptions{configPath: configPath, output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"42"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(stdout.String(), `"status":"COMPLETED"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestSearchCommandRejectsUnsafeIndexerID(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := Run([]string{
		"search",
		"--query", "dune",
		"--indexer-id", "../yts",
		"--output", "json",
	}, strings.NewReader(""), stdout, stderr)
	if exitCode != int(exitCodeUsage) {
		t.Fatalf("exitCode = %d, want %d", exitCode, exitCodeUsage)
	}

	var output map[string]any
	if err := json.Unmarshal(stderr.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal(stderr) error = %v", err)
	}
	if output["code"] != "invalid_indexer_id" {
		t.Fatalf("code = %v, want %q", output["code"], "invalid_indexer_id")
	}
}

func TestSearchCommandFieldsFiltersResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`{"results":[{"title":"Dune","magnetLink":"magnet:?xt=urn:btih:dune","size":"1.4 GB"}],"request_id":"req-123"}`))
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
	command := newSearchCommand(&appContext{
		opts:   &appOptions{configPath: configPath, output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"--query", "dune", "--fields", "results.title,request_id"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("output json decode error: %v", err)
	}
	if _, ok := output["request_id"]; !ok {
		t.Fatal("expected request_id in filtered output")
	}
	results, ok := output["results"].([]any)
	if !ok || len(results) != 1 {
		t.Fatalf("results = %#v", output["results"])
	}
	first, ok := results[0].(map[string]any)
	if !ok {
		t.Fatalf("first result = %#v", results[0])
	}
	if first["title"] != "Dune" {
		t.Fatalf("first.title = %v", first["title"])
	}
	if _, ok := first["magnetLink"]; ok {
		t.Fatalf("unexpected magnetLink in %#v", first)
	}
}

func TestSearchCommandPrettyOutputShowsReadableSummary(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`{"query":"dune","results":[{"title":"Dune","magnetLink":"magnet:?xt=urn:btih:dune","size":"1.4 GB","seeders":120}]}`))
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
	command := newSearchCommand(&appContext{
		opts:   &appOptions{configPath: configPath, output: outputPretty},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"--query", "dune"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	rendered := stripANSI(stdout.String())
	for _, expected := range []string{"Results: 1", "Query: dune", "1. Dune", "Size: 1.4 GB", "Magnet: magnet:?xt=urn:btih:dune"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("pretty output missing %q in %q", expected, rendered)
		}
	}
}

func TestSearchCommandPrettyOutputTruncatesLongLists(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`{"query":"dune","results":[{"title":"One"},{"title":"Two"},{"title":"Three"},{"title":"Four"},{"title":"Five"},{"title":"Six"},{"title":"Seven"},{"title":"Eight"},{"title":"Nine"},{"title":"Ten"},{"title":"Eleven"}]}`))
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
	command := newSearchCommand(&appContext{
		opts:   &appOptions{configPath: configPath, output: outputPretty},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"--query", "dune"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	rendered := stripANSI(stdout.String())
	if strings.Contains(rendered, "11. Eleven") {
		t.Fatalf("pretty output should truncate long lists: %q", rendered)
	}
	if !strings.Contains(rendered, "... 1 more results omitted.") {
		t.Fatalf("pretty output missing truncation notice: %q", rendered)
	}
}

func TestSettingsSetAndGetAPIBaseURL(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.json")
	store, err := config.NewStore(configPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Save(config.Default()); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	setOutput := &bytes.Buffer{}
	app := &appContext{
		opts:    &appOptions{configPath: configPath, output: outputJSON},
		stdin:   strings.NewReader(""),
		stdout:  setOutput,
		stderr:  &bytes.Buffer{},
		openURL: func(string) error { return nil },
	}
	setCommand := newSettingsCommand(app)
	setCommand.SetArgs([]string{"set", "api-base-url", "https://api.chill.test"})
	if err := setCommand.Execute(); err != nil {
		t.Fatalf("set Execute() error = %v", err)
	}

	getOutput := &bytes.Buffer{}
	app.stdout = getOutput
	getCommand := newSettingsCommand(app)
	getCommand.SetArgs([]string{"get", "api-base-url"})
	if err := getCommand.Execute(); err != nil {
		t.Fatalf("get Execute() error = %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(getOutput.Bytes(), &output); err != nil {
		t.Fatalf("output json decode error: %v", err)
	}
	if output["value"] != "https://api.chill.test" {
		t.Fatalf("value = %v, want %q", output["value"], "https://api.chill.test")
	}
}

func TestSettingsSetDryRunDoesNotSaveAPIBaseURL(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.json")
	store, err := config.NewStore(configPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Save(config.Default()); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	stdout := &bytes.Buffer{}
	app := &appContext{
		opts:   &appOptions{configPath: configPath, output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	}

	command := newSettingsCommand(app)
	command.SetArgs([]string{"set", "api-base-url", "https://api.chill.test", "--dry-run"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.APIBaseURL != config.Default().APIBaseURL {
		t.Fatalf("APIBaseURL = %q, want %q", cfg.APIBaseURL, config.Default().APIBaseURL)
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("output json decode error: %v", err)
	}
	if output["command"] != "settings set" {
		t.Fatalf("command = %v, want %q", output["command"], "settings set")
	}
}

func TestSettingsShowRedactsToken(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.json")
	store, err := config.NewStore(configPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Save(config.Config{APIBaseURL: "https://api.chill.test", AuthToken: "secret-token"}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	stdout := &bytes.Buffer{}
	command := newSettingsCommand(&appContext{
		opts:    &appOptions{configPath: configPath, profile: config.DefaultProfile(), output: outputJSON},
		stdin:   strings.NewReader(""),
		stdout:  stdout,
		stderr:  &bytes.Buffer{},
		openURL: func(string) error { return nil },
	})
	command.SetArgs([]string{"show"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("output json decode error: %v", err)
	}
	if output["auth_token"] != redactedToken {
		t.Fatalf("auth_token = %v, want %q", output["auth_token"], redactedToken)
	}
	if output["profile"] != config.DefaultProfile() {
		t.Fatalf("profile = %v, want %q", output["profile"], config.DefaultProfile())
	}
}

func TestSettingsPathIncludesProfile(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.json")
	stdout := &bytes.Buffer{}
	command := newSettingsCommand(&appContext{
		opts:    &appOptions{configPath: configPath, profile: "production", output: outputJSON},
		stdin:   strings.NewReader(""),
		stdout:  stdout,
		stderr:  &bytes.Buffer{},
		openURL: func(string) error { return nil },
	})
	command.SetArgs([]string{"path"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("output json decode error: %v", err)
	}
	if output["path"] != configPath {
		t.Fatalf("path = %v, want %q", output["path"], configPath)
	}
	if output["profile"] != "production" {
		t.Fatalf("profile = %v, want %q", output["profile"], "production")
	}
}

func TestDoctorOfflineReportsLocalState(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.json")
	store, err := config.NewStore(configPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Save(config.Config{APIBaseURL: "https://api.chill.institute", AuthToken: "saved-token"}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	stdout := &bytes.Buffer{}
	command := newDoctorCommand(&appContext{
		opts:    &appOptions{configPath: configPath, profile: "production", output: outputJSON},
		stdin:   strings.NewReader(""),
		stdout:  stdout,
		stderr:  &bytes.Buffer{},
		openURL: func(string) error { return nil },
	})
	command.SetArgs([]string{"--offline"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("output json decode error: %v", err)
	}
	if output["status"] != "warn" {
		t.Fatalf("status = %v, want %q", output["status"], "warn")
	}

	configPayload, ok := output["config"].(map[string]any)
	if !ok {
		t.Fatalf("config = %#v, want object", output["config"])
	}
	if configPayload["profile"] != "production" {
		t.Fatalf("config.profile = %v, want %q", configPayload["profile"], "production")
	}

	authPayload, ok := output["auth"].(map[string]any)
	if !ok {
		t.Fatalf("auth = %#v, want object", output["auth"])
	}
	if authPayload["status"] != "skipped" {
		t.Fatalf("auth.status = %v, want %q", authPayload["status"], "skipped")
	}
}

func TestDoctorChecksStoredAuth(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v4/chill.v4.UserService/GetUserProfile" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer saved-token" {
			t.Fatalf("Authorization = %q", got)
		}
		writer.Header().Set("X-Request-Id", "req-doctor")
		_, _ = writer.Write([]byte(`{"username":"dunefan","email":"dune@example.com","userId":"123"}`))
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
	command := newDoctorCommand(&appContext{
		opts:    &appOptions{configPath: configPath, output: outputJSON},
		stdin:   strings.NewReader(""),
		stdout:  stdout,
		stderr:  &bytes.Buffer{},
		openURL: func(string) error { return nil },
	})
	command.SetArgs(nil)
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("output json decode error: %v", err)
	}
	if output["status"] != "ok" {
		t.Fatalf("status = %v, want %q", output["status"], "ok")
	}

	authPayload, ok := output["auth"].(map[string]any)
	if !ok {
		t.Fatalf("auth = %#v, want object", output["auth"])
	}
	if authPayload["status"] != "ok" {
		t.Fatalf("auth.status = %v, want %q", authPayload["status"], "ok")
	}
	if authPayload["request_id"] != "req-doctor" {
		t.Fatalf("auth.request_id = %v, want %q", authPayload["request_id"], "req-doctor")
	}

	userPayload, ok := authPayload["user"].(map[string]any)
	if !ok {
		t.Fatalf("auth.user = %#v, want object", authPayload["user"])
	}
	if userPayload["username"] != "dunefan" {
		t.Fatalf("auth.user.username = %v, want %q", userPayload["username"], "dunefan")
	}
}

func TestDoctorHandlesInvalidAndErroredAuthChecks(t *testing.T) {
	t.Parallel()

	t.Run("invalid auth", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("X-Request-Id", "req-invalid")
			writer.WriteHeader(http.StatusUnauthorized)
			_, _ = writer.Write([]byte(`{"code":"invalid_auth_token","message":"expired"}`))
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
		command := newDoctorCommand(&appContext{
			opts:   &appOptions{configPath: configPath, output: outputJSON},
			stdin:  strings.NewReader(""),
			stdout: stdout,
			stderr: &bytes.Buffer{},
		})
		command.SetArgs(nil)
		if err := command.Execute(); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		var output map[string]any
		if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
			t.Fatalf("output json decode error: %v", err)
		}
		if output["status"] != "warn" {
			t.Fatalf("status = %v, want warn", output["status"])
		}
		authPayload := output["auth"].(map[string]any)
		if authPayload["status"] != "invalid" || authPayload["request_id"] != "req-invalid" {
			t.Fatalf("auth = %#v", authPayload)
		}
	})

	t.Run("api error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("X-Request-Id", "req-error")
			writer.WriteHeader(http.StatusInternalServerError)
			_, _ = writer.Write([]byte(`{"code":"backend_error","message":"boom"}`))
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
		command := newDoctorCommand(&appContext{
			opts:   &appOptions{configPath: configPath, output: outputJSON},
			stdin:  strings.NewReader(""),
			stdout: stdout,
			stderr: &bytes.Buffer{},
		})
		command.SetArgs(nil)
		if err := command.Execute(); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}

		var output map[string]any
		if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
			t.Fatalf("output json decode error: %v", err)
		}
		if output["status"] != "error" {
			t.Fatalf("status = %v, want error", output["status"])
		}
		authPayload := output["auth"].(map[string]any)
		if authPayload["status"] != "error" || authPayload["request_id"] != "req-error" {
			t.Fatalf("auth = %#v", authPayload)
		}
	})
}

func TestMoviesUsesStoredToken(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v4/chill.v4.UserService/GetMovies" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer saved-token" {
			t.Fatalf("Authorization = %q", got)
		}
		_, _ = writer.Write([]byte(`{"movies":[{"title":"Dune"}]}`))
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
	command := newMoviesCommand(&appContext{
		opts:   &appOptions{configPath: configPath, output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs(nil)
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(stdout.String(), `"title":"Dune"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestMoviesFieldsFiltersResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`{"movies":[{"title":"Dune","year":2021}]}`))
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
	command := newMoviesCommand(&appContext{
		opts:   &appOptions{configPath: configPath, output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"--fields", "movies.title"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if strings.Contains(stdout.String(), `"year"`) {
		t.Fatalf("stdout should not include year: %q", stdout.String())
	}
}

func TestMoviesPrettyOutputShowsReadableSummary(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`{"movies":[{"title":"Dune","year":2021},{"title":"Arrival","year":2016}]}`))
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
	command := newMoviesCommand(&appContext{
		opts:   &appOptions{configPath: configPath, output: outputPretty},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs(nil)
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	rendered := stripANSI(stdout.String())
	for _, expected := range []string{"Movies: 2", "1. Dune (2021)", "2. Arrival (2016)"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("pretty output missing %q in %q", expected, rendered)
		}
	}
}

func TestTVShowsUsesStoredToken(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v4/chill.v4.UserService/GetTVShows" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer saved-token" {
			t.Fatalf("Authorization = %q", got)
		}
		_, _ = writer.Write([]byte(`{"shows":[{"title":"The Pitt","imdbId":"tt31938062"}]}`))
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
	command := newTVShowsCommand(&appContext{
		opts:   &appOptions{configPath: configPath, output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs(nil)
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(stdout.String(), `"title":"The Pitt"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestTVShowsSourceSendsRequestOverride(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v4/chill.v4.UserService/GetTVShows" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload["source"] != "TV_SHOWS_SOURCE_AMC_PLUS" {
			t.Fatalf("payload = %#v, want AMC+ source", payload)
		}
		_, _ = writer.Write([]byte(`{"source":"TV_SHOWS_SOURCE_AMC_PLUS","shows":[{"title":"Dark Winds","imdbId":"tt15017118"}]}`))
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
	command := newTVShowsCommand(&appContext{
		opts:   &appOptions{configPath: configPath, output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"--source", "amc-plus", "--fields", "source,shows.title"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(stdout.String(), `"source":"TV_SHOWS_SOURCE_AMC_PLUS"`) ||
		!strings.Contains(stdout.String(), `"title":"Dark Winds"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestTVShowsSourceRejectsUnsafeValueBeforeNetwork(t *testing.T) {
	t.Parallel()

	called := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
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

	command := newTVShowsCommand(&appContext{
		opts:   &appOptions{configPath: configPath, output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"--source", "hulu?source=netflix"})
	if err := command.Execute(); err == nil {
		t.Fatal("Execute() error = nil, want invalid source")
	}
	if called {
		t.Fatal("server was called for invalid source")
	}
}

func TestTVShowDetailUsesIMDbID(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v4/chill.v4.UserService/GetTVShowDetail" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload["imdbId"] != "tt31938062" {
			t.Fatalf("imdbId = %v", payload["imdbId"])
		}
		_, _ = writer.Write([]byte(`{"show":{"title":"The Pitt","imdbId":"tt31938062"},"seasons":[{"seasonNumber":1,"name":"Season 1"}]}`))
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
	command := newTVShowsCommand(&appContext{
		opts:   &appOptions{configPath: configPath, output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"detail", "tt31938062"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(stdout.String(), `"title":"The Pitt"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestTVShowSeasonUsesIMDbIDAndSeasonNumber(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v4/chill.v4.UserService/GetTVShowSeason" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload["imdbId"] != "tt31938062" || payload["seasonNumber"] != float64(1) {
			t.Fatalf("payload = %#v", payload)
		}
		_, _ = writer.Write([]byte(`{"season":{"name":"Season 1","seasonNumber":1},"episodes":[{"episodeNumber":1,"name":"Pilot"}]}`))
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
	command := newTVShowsCommand(&appContext{
		opts:   &appOptions{configPath: configPath, output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"season", "tt31938062", "1"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(stdout.String(), `"name":"Pilot"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestTVShowEpisodeDownloadUsesIMDbIDSeasonAndEpisode(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v4/chill.v4.UserService/GetTVShowEpisodeDownload" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload["imdbId"] != "tt31938062" || payload["seasonNumber"] != float64(1) || payload["episodeNumber"] != float64(1) {
			t.Fatalf("payload = %#v", payload)
		}
		_, _ = writer.Write([]byte(`{"searchQuery":"The Pitt S01E01","download":{"title":"The.Pitt.S01E01.1080p.WEB-DL"}}`))
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
	command := newTVShowsCommand(&appContext{
		opts:   &appOptions{configPath: configPath, output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"episode-download", "tt31938062", "1", "1"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(stdout.String(), `"searchQuery":"The Pitt S01E01"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestTVShowSeasonDownloadsUsesIMDbIDAndSeasonNumber(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v4/chill.v4.UserService/GetTVShowSeasonDownloads" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload["imdbId"] != "tt31938062" || payload["seasonNumber"] != float64(1) {
			t.Fatalf("payload = %#v", payload)
		}
		_, _ = writer.Write([]byte(`{"seasonSearchQuery":"The Pitt S01","seasonPack":{"title":"The.Pitt.S01.COMPLETE"},"episodes":[{"episodeNumber":1}]}`))
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
	command := newTVShowsCommand(&appContext{
		opts:   &appOptions{configPath: configPath, output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"season-downloads", "tt31938062", "1"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(stdout.String(), `"seasonSearchQuery":"The Pitt S01"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestSettingsPathOutputsResolvedStorePath(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.json")
	stdout := &bytes.Buffer{}
	command := newSettingsCommand(&appContext{
		opts:   &appOptions{configPath: configPath, output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"path"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if output["path"] != configPath {
		t.Fatalf("path = %v, want %q", output["path"], configPath)
	}
}

func TestWhoamiFieldsFiltersResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`{"username":"dunefan","email":"dune@example.com","userId":"123"}`))
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
	command := newWhoamiCommand(&appContext{
		opts:   &appOptions{configPath: configPath, output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"--fields", "username,email"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("output json decode error: %v", err)
	}
	if output["username"] != "dunefan" || output["email"] != "dune@example.com" {
		t.Fatalf("output = %#v", output)
	}
	if _, ok := output["userId"]; ok {
		t.Fatalf("unexpected userId in %#v", output)
	}
}

func TestWhoamiPrettyOutputShowsReadableSummary(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`{"username":"dunefan","email":"dune@example.com","userId":"123"}`))
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
	command := newWhoamiCommand(&appContext{
		opts:   &appOptions{configPath: configPath, output: outputPretty},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs(nil)
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	rendered := stripANSI(stdout.String())
	for _, expected := range []string{"Username: dunefan", "Email: dune@example.com", "User ID: 123"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("pretty output missing %q in %q", expected, rendered)
		}
	}
}

func TestUserSettingsGetFieldsFiltersResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v4/chill.v4.UserService/GetUserSettings" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		_, _ = writer.Write([]byte(`{"search":{"filterNastyResults":true,"sortBy":"seeders","sortDirection":"desc"}}`))
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
	command.SetArgs([]string{"settings", "get", "--fields", "search.filterNastyResults,search.sortBy"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("output json decode error: %v", err)
	}
	search, ok := output["search"].(map[string]any)
	if !ok {
		t.Fatalf("output.search = %#v, want object", output["search"])
	}
	if search["filterNastyResults"] != true || search["sortBy"] != "seeders" {
		t.Fatalf("output = %#v", output)
	}
	if _, ok := search["sortDirection"]; ok {
		t.Fatalf("unexpected sortDirection in %#v", output)
	}
}

func TestUserSettingsGetPrettyOutputShowsReadableSummary(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v4/chill.v4.UserService/GetUserSettings" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		_, _ = writer.Write([]byte(`{"search":{"filterNastyResults":true,"sortBy":"SORT_BY_SEEDERS","sortDirection":"SORT_DIRECTION_DESC","disabledIndexerIds":["animetosho"]}}`))
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
		opts:   &appOptions{configPath: configPath, output: outputPretty},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"settings", "get"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	rendered := stripANSI(stdout.String())
	for _, expected := range []string{"User Settings", "search:", "filterNastyResults: true", "sortBy: SORT_BY_SEEDERS", "disabledIndexerIds: animetosho"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("pretty output missing %q in %q", expected, rendered)
		}
	}
}

func TestUserIndexersPrettyOutputShowsReadableSummary(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v4/chill.v4.UserService/GetIndexers" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		_, _ = writer.Write([]byte(`{"indexers":[{"id":"yts","name":"YTS","enabled":true,"status":"INDEXER_STATUS_HEALTHY"},{"id":"tbp","name":"The Pirate Bay","enabled":true,"status":"INDEXER_STATUS_DEGRADED"},{"id":"rarbg","name":"RARBG","enabled":false,"status":"INDEXER_STATUS_DOWN"}]}`))
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
		opts:   &appOptions{configPath: configPath, output: outputPretty},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"indexers"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	rendered := stripANSI(stdout.String())
	for _, expected := range []string{"Indexers: 3", "1. YTS [yts]", "Enabled: enabled", "Indexer Status: healthy", "2. The Pirate Bay [tbp]", "Indexer Status: degraded", "3. RARBG [rarbg]", "Enabled: disabled", "Indexer Status: down"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("pretty output missing %q in %q", expected, rendered)
		}
	}
}

func TestUserIndexersFieldsFiltersResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`{"indexers":[{"id":"yts","name":"YTS","enabled":true,"status":"INDEXER_STATUS_HEALTHY"}]}`))
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
	command.SetArgs([]string{"indexers", "--fields", "indexers.id"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if strings.Contains(stdout.String(), `"name"`) {
		t.Fatalf("stdout should not include name: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"id":"yts"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestUserDownloadFolderPrettyOutputShowsReadableSummary(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v4/chill.v4.UserService/GetDownloadFolder" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		_, _ = writer.Write([]byte(`{"folder":{"id":"42","name":"chill.institute","fileType":"FOLDER","isShared":false}}`))
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
		opts:   &appOptions{configPath: configPath, output: outputPretty},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"download-folder"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	rendered := stripANSI(stdout.String())
	for _, expected := range []string{"Download Folder", "Name: chill.institute", "ID: 42"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("pretty output missing %q in %q", expected, rendered)
		}
	}
}

func TestUserDownloadFolderFieldsFiltersResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`{"folder":{"id":"42","name":"chill.institute","fileType":"FOLDER"}}`))
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
	command.SetArgs([]string{"download-folder", "--fields", "folder.name"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if strings.Contains(stdout.String(), `"id"`) {
		t.Fatalf("stdout should not include id: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"name":"chill.institute"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestUserDownloadFolderSetDryRunReturnsPatchPreview(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newUserCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"download-folder", "set", "42", "--dry-run"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("output json decode error: %v", err)
	}
	if output["command"] != "user download-folder set" {
		t.Fatalf("command = %v, want %q", output["command"], "user download-folder set")
	}
	request, ok := output["request"].(map[string]any)
	if !ok {
		t.Fatalf("request = %#v", output["request"])
	}
	patch, ok := request["patch"].(map[string]any)
	if !ok {
		t.Fatalf("patch = %#v", request["patch"])
	}
	if patch["field"] != "download.folderId" || patch["value"] != "42" {
		t.Fatalf("patch = %#v", patch)
	}
}

func TestUserFolderGetPrettyOutputShowsReadableSummary(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v4/chill.v4.UserService/GetFolder" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload["id"] != float64(42) {
			t.Fatalf("id = %v, want %d", payload["id"], 42)
		}
		_, _ = writer.Write([]byte(`{"parent":{"id":"42","name":"Movies","fileType":"FOLDER"},"files":[{"id":"1","name":"Dune","fileType":"VIDEO"},{"id":"2","name":"Arrival","fileType":"VIDEO"}]}`))
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
		opts:   &appOptions{configPath: configPath, output: outputPretty},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"folder", "get", "42"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	rendered := stripANSI(stdout.String())
	for _, expected := range []string{"Folder", "Name: Movies", "Children: 2", "1. Dune [VIDEO]", "2. Arrival [VIDEO]"} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("pretty output missing %q in %q", expected, rendered)
		}
	}
}

func TestUserFolderGetFieldsFiltersResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`{"parent":{"id":"42","name":"Movies"},"files":[{"id":"1","name":"Dune","fileType":"VIDEO"}]}`))
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
	command.SetArgs([]string{"folder", "get", "42", "--fields", "parent.name,files.name"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if strings.Contains(stdout.String(), `"id"`) {
		t.Fatalf("stdout should not include id: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"name":"Movies"`) || !strings.Contains(stdout.String(), `"name":"Dune"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestUserDownloadFolderClearDryRunReturnsPatchPreview(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	command := newUserCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"download-folder", "clear", "--dry-run"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("output json decode error: %v", err)
	}
	if output["command"] != "user download-folder clear" {
		t.Fatalf("command = %v, want %q", output["command"], "user download-folder clear")
	}
	request, ok := output["request"].(map[string]any)
	if !ok {
		t.Fatalf("request = %#v", output["request"])
	}
	patch, ok := request["patch"].(map[string]any)
	if !ok {
		t.Fatalf("patch = %#v", request["patch"])
	}
	if patch["field"] != "download.folderId" {
		t.Fatalf("patch = %#v", patch)
	}
	if value, ok := patch["value"]; !ok || value != nil {
		t.Fatalf("patch.value = %#v, want nil", patch["value"])
	}
}

func TestUserSettingsSetPatchMergesWithCurrentSettings(t *testing.T) {
	t.Parallel()

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount++
		switch request.URL.Path {
		case "/v4/chill.v4.UserService/GetUserSettings":
			_, _ = writer.Write([]byte(`{"search":{"filterNastyResults":false,"sortBy":"SORT_BY_SEEDERS","sortDirection":"SORT_DIRECTION_DESC"}}`))
		case "/v4/chill.v4.UserService/SaveUserSettings":
			var payload map[string]any
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatalf("decode payload: %v", err)
			}
			settings, ok := payload["settings"].(map[string]any)
			if !ok {
				t.Fatalf("settings = %#v", payload["settings"])
			}
			search, ok := settings["search"].(map[string]any)
			if !ok {
				t.Fatalf("search = %#v", settings["search"])
			}
			if search["filterNastyResults"] != true {
				t.Fatalf("filterNastyResults = %v, want true", search["filterNastyResults"])
			}
			if search["sortBy"] != "SORT_BY_SEEDERS" {
				t.Fatalf("sortBy = %v, want %q", search["sortBy"], "SORT_BY_SEEDERS")
			}
			if search["sortDirection"] != "SORT_DIRECTION_DESC" {
				t.Fatalf("sortDirection = %v, want %q", search["sortDirection"], "SORT_DIRECTION_DESC")
			}
			_, _ = writer.Write([]byte(`{"search":{"filterNastyResults":true,"sortBy":"SORT_BY_SEEDERS","sortDirection":"SORT_DIRECTION_DESC"}}`))
		default:
			t.Fatalf("path = %q", request.URL.Path)
		}
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
	command.SetArgs([]string{"settings", "set", "filter-nasty-results", "true"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if requestCount != 2 {
		t.Fatalf("requestCount = %d, want 2", requestCount)
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("output json decode error: %v", err)
	}
	search, ok := output["search"].(map[string]any)
	if !ok || search["filterNastyResults"] != true {
		t.Fatalf("output = %#v", output)
	}
}

func TestUserSettingsSetPatchDryRunDoesNotCallAPI(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.json")
	stdout := &bytes.Buffer{}
	command := newUserCommand(&appContext{
		opts:   &appOptions{configPath: configPath, output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"settings", "set", "sort-by", "title", "--dry-run"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("output json decode error: %v", err)
	}
	request, ok := output["request"].(map[string]any)
	if !ok {
		t.Fatalf("request = %#v", output["request"])
	}
	patch, ok := request["patch"].(map[string]any)
	if !ok {
		t.Fatalf("patch = %#v", request["patch"])
	}
	if patch["field"] != "search.sortBy" {
		t.Fatalf("patch.field = %v, want %q", patch["field"], "search.sortBy")
	}
	if patch["value"] != "SORT_BY_TITLE" {
		t.Fatalf("patch.value = %v, want %q", patch["value"], "SORT_BY_TITLE")
	}
}

func TestUserSettingsSetRejectsMixedJSONAndPatchModes(t *testing.T) {
	t.Parallel()

	command := newUserCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"settings", "set", "filter-nasty-results", "true", "--json", `{"search":{"filterNastyResults":true}}`})
	err := command.Execute()
	if err == nil {
		t.Fatal("expected error")
	}

	commandError, ok := err.(*cliError)
	if !ok {
		t.Fatalf("err = %#v, want *cliError", err)
	}
	if commandError.Code != "ambiguous_user_settings_update" {
		t.Fatalf("Code = %q, want %q", commandError.Code, "ambiguous_user_settings_update")
	}
}

func TestUserProfilePrettyOutputShowsReadableSummary(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v4/chill.v4.UserService/GetUserProfile" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		_, _ = writer.Write([]byte(`{"username":"dunefan","email":"dune@example.com","userId":"123"}`))
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
		opts:   &appOptions{configPath: configPath, output: outputPretty},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"profile"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(stripANSI(stdout.String()), "Username: dunefan") {
		t.Fatalf("stdout = %q", stdout.String())
	}
}
