package cli

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chill-institute/chill-cli/internal/update"
	"github.com/chill-institute/chill-cli/pkg/config"
)

func TestResolveSettingsSetInput(t *testing.T) {
	t.Parallel()

	app := &appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: &strings.Builder{},
		stderr: &strings.Builder{},
	}

	key, value, err := resolveSettingsSetInput(app, []string{"api-base-url", "https://api.chill.institute"}, "")
	if err != nil {
		t.Fatalf("resolveSettingsSetInput(flags) error = %v", err)
	}
	if key != "api-base-url" || value != "https://api.chill.institute" {
		t.Fatalf("resolveSettingsSetInput(flags) = %q, %q", key, value)
	}

	key, value, err = resolveSettingsSetInput(app, nil, `{"key":"api_base_url","value":"https://api.chill.institute/"}`)
	if err != nil {
		t.Fatalf("resolveSettingsSetInput(json) error = %v", err)
	}
	if key != "api-base-url" || value != "https://api.chill.institute" {
		t.Fatalf("resolveSettingsSetInput(json) = %q, %q", key, value)
	}

	testCases := []struct {
		name       string
		args       []string
		rawRequest string
	}{
		{name: "missing args", args: []string{"api-base-url"}},
		{name: "ambiguous input", args: []string{"api-base-url", "https://api.chill.institute"}, rawRequest: `{"key":"api-base-url","value":"https://api.chill.institute"}`},
		{name: "json missing key", rawRequest: `{"value":"https://api.chill.institute"}`},
		{name: "json missing value", rawRequest: `{"key":"api-base-url"}`},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, _, err := resolveSettingsSetInput(app, tc.args, tc.rawRequest); err == nil {
				t.Fatalf("resolveSettingsSetInput(%s) error = nil, want error", tc.name)
			}
		})
	}
}

func TestNormalizeSettingsValueAndLoadStoredCLISettings(t *testing.T) {
	t.Parallel()

	if _, err := normalizeSettingsValue("api-base-url", " "); err == nil {
		t.Fatal("normalizeSettingsValue(empty api-base-url) error = nil, want error")
	}
	if _, err := normalizeSettingsValue("nope", "value"); err == nil {
		t.Fatal("normalizeSettingsValue(unsupported) error = nil, want error")
	}

	configPath := filepath.Join(t.TempDir(), "config.json")
	app := &appContext{opts: &appOptions{configPath: configPath, output: outputJSON}}
	if err := app.saveConfig(config.Config{APIBaseURL: " https://api.chill.institute ", AuthToken: " token-123 "}); err != nil {
		t.Fatalf("saveConfig() error = %v", err)
	}

	cfg, err := loadStoredCLISettings(app)
	if err != nil {
		t.Fatalf("loadStoredCLISettings() error = %v", err)
	}
	if cfg.APIBaseURL != "https://api.chill.institute" || cfg.AuthToken != "token-123" {
		t.Fatalf("cfg = %#v", cfg)
	}
}

func TestDecodeJSONObjectFlagRejectsBadPayloads(t *testing.T) {
	t.Parallel()

	app := &appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader("[]"),
		stdout: &strings.Builder{},
		stderr: &strings.Builder{},
	}

	if _, err := app.decodeJSONObjectFlag("@-", "--json"); err == nil {
		t.Fatal("decodeJSONObjectFlag(array) error = nil, want object error")
	}
	if _, err := app.decodeJSONObjectFlag("{", "--json"); err == nil {
		t.Fatal("decodeJSONObjectFlag(invalid json) error = nil, want decode error")
	}
}

func TestReadJSONFlagRejectsEmptyAndReadErrors(t *testing.T) {
	t.Parallel()

	app := &appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  errReader{},
		stdout: &strings.Builder{},
		stderr: &strings.Builder{},
	}

	if _, err := app.readJSONFlag("", "--json"); err == nil {
		t.Fatal("readJSONFlag(empty) error = nil, want error")
	}
	if _, err := app.readJSONFlag("@-", "--json"); err == nil {
		t.Fatal("readJSONFlag(read error) error = nil, want error")
	}
}

func TestResolveSelfUpdateInput(t *testing.T) {
	t.Parallel()

	app := &appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: &strings.Builder{},
		stderr: &strings.Builder{},
	}

	version, checkOnly, err := resolveSelfUpdateInput(app, "v1.2.3", "", false)
	if err != nil {
		t.Fatalf("resolveSelfUpdateInput(flags) error = %v", err)
	}
	if version != "v1.2.3" || checkOnly {
		t.Fatalf("resolveSelfUpdateInput(flags) = %q, %v", version, checkOnly)
	}

	version, checkOnly, err = resolveSelfUpdateInput(app, "", `{"version":" v1.2.3 ","check":true}`, false)
	if err != nil {
		t.Fatalf("resolveSelfUpdateInput(json) error = %v", err)
	}
	if version != "v1.2.3" || !checkOnly {
		t.Fatalf("resolveSelfUpdateInput(json) = %q, %v", version, checkOnly)
	}

	for _, rawRequest := range []string{`{"version":true}`, `{"check":"yes"}`} {
		rawRequest := rawRequest
		t.Run(rawRequest, func(t *testing.T) {
			t.Parallel()

			if _, _, err := resolveSelfUpdateInput(app, "", rawRequest, false); err == nil {
				t.Fatalf("resolveSelfUpdateInput(%s) error = nil, want error", rawRequest)
			}
		})
	}

	if _, _, err := resolveSelfUpdateInput(app, "v1.2.3", `{"check":true}`, false); err == nil {
		t.Fatal("resolveSelfUpdateInput(ambiguous) error = nil, want error")
	}
}

func TestResolveRelease(t *testing.T) {
	t.Parallel()

	service := &stubReleaseService{
		latestRelease: update.Release{TagName: "v2.0.0"},
		tagRelease:    update.Release{TagName: "v1.2.3"},
	}

	release, err := resolveRelease(context.Background(), service, "")
	if err != nil {
		t.Fatalf("resolveRelease(latest) error = %v", err)
	}
	if release.TagName != "v2.0.0" {
		t.Fatalf("latest release = %#v", release)
	}

	release, err = resolveRelease(context.Background(), service, "1.2.3")
	if err != nil {
		t.Fatalf("resolveRelease(tag) error = %v", err)
	}
	if release.TagName != "v1.2.3" {
		t.Fatalf("tag release = %#v", release)
	}

	if _, err := resolveRelease(context.Background(), service, "../v1.2.3"); err == nil {
		t.Fatal("resolveRelease(invalid version) error = nil, want error")
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, errors.New("boom")
}
