package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chill-institute/chill-cli/internal/config"
)

func TestResolveAuthLoginInputFromJSON(t *testing.T) {
	t.Parallel()

	app := &appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: &strings.Builder{},
		stderr: &strings.Builder{},
	}

	token, skipOpen, localBrowser, skipVerify, err := resolveAuthLoginInput(app, "", `{"token":" token-123 ","no_browser":true,"local_browser":true,"skip_verify":true}`, false, false, false)
	if err != nil {
		t.Fatalf("resolveAuthLoginInput() error = %v", err)
	}
	if token != "token-123" {
		t.Fatalf("token = %q, want %q", token, "token-123")
	}
	if !skipOpen {
		t.Fatal("skipOpen = false, want true")
	}
	if !localBrowser {
		t.Fatal("localBrowser = false, want true")
	}
	if !skipVerify {
		t.Fatal("skipVerify = false, want true")
	}
}

func TestResolveAuthLoginInputRejectsAmbiguousMixedInput(t *testing.T) {
	t.Parallel()

	app := &appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: &strings.Builder{},
		stderr: &strings.Builder{},
	}

	if _, _, _, _, err := resolveAuthLoginInput(app, "token-123", `{"token":"other"}`, false, false, false); err == nil {
		t.Fatal("resolveAuthLoginInput() error = nil, want ambiguity error")
	}
}

func TestResolveAuthLoginInputRejectsInvalidJSONTypes(t *testing.T) {
	t.Parallel()

	app := &appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: &strings.Builder{},
		stderr: &strings.Builder{},
	}

	testCases := []string{
		`{"token":true}`,
		`{"no_browser":"yes"}`,
		`{"local_browser":"yes"}`,
		`{"skip_verify":"yes"}`,
	}

	for _, rawRequest := range testCases {
		rawRequest := rawRequest
		t.Run(rawRequest, func(t *testing.T) {
			t.Parallel()

			if _, _, _, _, err := resolveAuthLoginInput(app, "", rawRequest, false, false, false); err == nil {
				t.Fatalf("resolveAuthLoginInput(%s) error = nil, want error", rawRequest)
			}
		})
	}
}

func TestResolveAuthLogoutInputAcceptsExplicitTrue(t *testing.T) {
	t.Parallel()

	app := &appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: &strings.Builder{},
		stderr: &strings.Builder{},
	}

	request, err := resolveAuthLogoutInput(app, `{"clear_auth_token":true}`, true)
	if err != nil {
		t.Fatalf("resolveAuthLogoutInput() error = %v", err)
	}
	if request["clear_auth_token"] != true {
		t.Fatalf("request = %#v", request)
	}
	if request["had_auth_token"] != true {
		t.Fatalf("request = %#v", request)
	}
}

func TestResolveAuthLogoutInputRejectsInvalidJSONValue(t *testing.T) {
	t.Parallel()

	app := &appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: &strings.Builder{},
		stderr: &strings.Builder{},
	}

	for _, rawRequest := range []string{`{"clear_auth_token":false}`, `{"clear_auth_token":"yes"}`} {
		rawRequest := rawRequest
		t.Run(rawRequest, func(t *testing.T) {
			t.Parallel()

			if _, err := resolveAuthLogoutInput(app, rawRequest, false); err == nil {
				t.Fatalf("resolveAuthLogoutInput(%s) error = nil, want error", rawRequest)
			}
		})
	}
}

func TestResolveAuthLoginInputReturnsFlagValuesWithoutJSON(t *testing.T) {
	t.Parallel()

	app := &appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: &strings.Builder{},
		stderr: &strings.Builder{},
	}

	token, skipOpen, localBrowser, skipVerify, err := resolveAuthLoginInput(app, "token-123", "", true, true, true)
	if err != nil {
		t.Fatalf("resolveAuthLoginInput() error = %v", err)
	}
	if token != "token-123" || !skipOpen || !localBrowser || !skipVerify {
		t.Fatalf("resolved values = %q %v %v %v", token, skipOpen, localBrowser, skipVerify)
	}
}

func TestWebAuthTokenURLDerivesPublicHostFromAPIBaseURL(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		input  string
		output string
	}{
		{name: "production api host", input: "https://api.chill.institute", output: "https://chill.institute/auth/cli-token"},
		{name: "production api host with port", input: "https://api.chill.institute:8443", output: "https://chill.institute:8443/auth/cli-token"},
		{name: "staging api host", input: "https://staging-api.chill.institute", output: "https://staging.chill.institute/auth/cli-token"},
		{name: "staging api host with port", input: "https://staging-api.chill.institute:8443", output: "https://staging.chill.institute:8443/auth/cli-token"},
		{name: "production web host", input: "https://chill.institute", output: "https://chill.institute/auth/cli-token"},
		{name: "staging web host", input: "https://staging.chill.institute", output: "https://staging.chill.institute/auth/cli-token"},
		{name: "localhost", input: "http://localhost:8080", output: "http://localhost:8080/auth/cli-token"},
		{name: "dev api host with port", input: "https://api.chill.test:4443", output: "https://chill.test:4443/auth/cli-token"},
		{name: "dev web host with port", input: "https://chill.test:4443", output: "https://chill.test:4443/auth/cli-token"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := webAuthTokenURL(tc.input)
			if err != nil {
				t.Fatalf("webAuthTokenURL() error = %v", err)
			}
			if got != tc.output {
				t.Fatalf("webAuthTokenURL() = %q, want %q", got, tc.output)
			}
		})
	}
}

func TestAuthLoginDryRunOutputsWebTokenLoginURL(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.json")
	store, err := config.NewStore(configPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Save(config.Config{APIBaseURL: "https://staging-api.chill.institute:8443"}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	stdout := &bytes.Buffer{}
	app := &appContext{
		opts:   &appOptions{configPath: configPath, output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &strings.Builder{},
	}

	command := newAuthLoginCommand(app)
	command.SetArgs([]string{"--dry-run"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output struct {
		Request map[string]any `json:"request"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("output json decode error: %v", err)
	}
	if output.Request["mode"] != "web_token" {
		t.Fatalf("request.mode = %v, want web_token", output.Request["mode"])
	}
	if output.Request["login_url"] != "https://staging.chill.institute:8443/auth/cli-token" {
		t.Fatalf("request.login_url = %v", output.Request["login_url"])
	}
}

func TestWebAuthTokenURLRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	for _, input := range []string{"", "api.chill.institute", "ftp://api.chill.institute", "http://api.localhost:3000", "https://api.chill.institute/path"} {
		input := input
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			if _, err := webAuthTokenURL(input); err == nil {
				t.Fatalf("webAuthTokenURL(%q) error = nil, want error", input)
			}
		})
	}
}
