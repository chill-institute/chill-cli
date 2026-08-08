package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chill-institute/chill-cli/internal/buildinfo"
	"github.com/chill-institute/chill-cli/internal/config"
)

func TestRunAddTransferDryRunSkipsAuthAndReturnsPreview(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.json")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := Run([]string{
		"--config", configPath,
		"add-transfer",
		"--url", "magnet:?xt=urn:btih:dryrun",
		"--dry-run",
		"--output", "json",
	}, strings.NewReader(""), stdout, stderr)
	if exitCode != int(exitCodeSuccess) {
		t.Fatalf("exitCode = %d, want %d", exitCode, exitCodeSuccess)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal(stdout) error = %v", err)
	}
	if output["dry_run"] != true {
		t.Fatalf("dry_run = %v, want true", output["dry_run"])
	}
	if output["command"] != "add-transfer" {
		t.Fatalf("command = %v, want %q", output["command"], "add-transfer")
	}
	if output["procedure"] != procedureUserAddTransfer {
		t.Fatalf("procedure = %v, want %q", output["procedure"], procedureUserAddTransfer)
	}

	request, ok := output["request"].(map[string]any)
	if !ok {
		t.Fatalf("request = %#v, want object", output["request"])
	}
	if request["url"] != "magnet:?xt=urn:btih:dryrun" {
		t.Fatalf("request.url = %v, want %q", request["url"], "magnet:?xt=urn:btih:dryrun")
	}
	if _, ok := request["catalogOrigin"]; ok {
		t.Fatalf("request.catalogOrigin = %#v, want omitted", request["catalogOrigin"])
	}
}

func TestRunAddTransferDryRunIncludesCatalogOrigin(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	exitCode := Run([]string{
		"--config", filepath.Join(t.TempDir(), "config.json"),
		"add-transfer",
		"--url", "magnet:?xt=urn:btih:dryrun",
		"--movie-source", "trakt",
		"--dry-run",
		"--output", "json",
	}, strings.NewReader(""), stdout, &bytes.Buffer{})
	if exitCode != int(exitCodeSuccess) {
		t.Fatalf("exitCode = %d, want %d", exitCode, exitCodeSuccess)
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal(stdout) error = %v", err)
	}
	request, ok := output["request"].(map[string]any)
	if !ok {
		t.Fatalf("request = %#v, want object", output["request"])
	}
	catalogOrigin, ok := request["catalogOrigin"].(map[string]any)
	if !ok || catalogOrigin["moviesSource"] != "MOVIES_SOURCE_TRAKT" {
		t.Fatalf("catalogOrigin = %#v", request["catalogOrigin"])
	}
}

func TestRunAddTransferDryRunAcceptsJSONPayloadFromStdin(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.json")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := Run([]string{
		"--config", configPath,
		"add-transfer",
		"--json", "@-",
		"--dry-run",
		"--output", "json",
	}, strings.NewReader(`{"url":"magnet:?xt=urn:btih:stdin","catalogOrigin":{"moviesSource":"MOVIES_SOURCE_TRAKT"}}`), stdout, stderr)
	if exitCode != int(exitCodeSuccess) {
		t.Fatalf("exitCode = %d, want %d", exitCode, exitCodeSuccess)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal(stdout) error = %v", err)
	}
	request, ok := output["request"].(map[string]any)
	if !ok {
		t.Fatalf("request = %#v, want object", output["request"])
	}
	if request["url"] != "magnet:?xt=urn:btih:stdin" {
		t.Fatalf("request.url = %v, want %q", request["url"], "magnet:?xt=urn:btih:stdin")
	}
	catalogOrigin, ok := request["catalogOrigin"].(map[string]any)
	if !ok || catalogOrigin["moviesSource"] != "MOVIES_SOURCE_TRAKT" {
		t.Fatalf("catalogOrigin = %#v", request["catalogOrigin"])
	}
}

func TestRunAddTransferDryRunRejectsInvalidURLsWithJSONEnvelope(t *testing.T) {
	t.Parallel()

	for _, rawURL := range []string{
		"ftp://example.test/file.torrent",
		"not-a-url",
		"https://exa\x00mple.test/file.torrent",
		"https://example.test/file name.torrent",
		"magnet:?xt=urn:btih:test&dn=My Movie",
	} {
		rawURL := rawURL
		t.Run(rawURL, func(t *testing.T) {
			t.Parallel()

			configPath := filepath.Join(t.TempDir(), "config.json")
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			exitCode := Run([]string{
				"--config", configPath,
				"add-transfer",
				"--url", rawURL,
				"--dry-run",
				"--output", "json",
			}, strings.NewReader(""), stdout, stderr)
			if exitCode != int(exitCodeUsage) {
				t.Fatalf("exitCode = %d, want %d", exitCode, exitCodeUsage)
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}

			var output map[string]any
			if err := json.Unmarshal(stderr.Bytes(), &output); err != nil {
				t.Fatalf("json.Unmarshal(stderr) error = %v", err)
			}
			if output["kind"] != string(errorKindUsage) {
				t.Fatalf("kind = %v, want %q", output["kind"], errorKindUsage)
			}
			if output["code"] != "invalid_url" {
				t.Fatalf("code = %v, want %q", output["code"], "invalid_url")
			}
		})
	}
}

func TestRunUserSettingsSetDryRunSkipsAuthAndReturnsPreview(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.json")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := Run([]string{
		"--config", configPath,
		"user", "settings", "set",
		"--json", `{"search":{"filterNastyResults":true,"filterResultsWithNoSeeders":false,"rememberQuickFilters":false,"disabledIndexerIds":[],"resolutionFilters":[],"codecFilters":[],"otherFilters":[],"sortBy":"SORT_BY_SEEDERS","sortDirection":"SORT_DIRECTION_DESC","searchResultDisplayBehavior":"SEARCH_RESULT_DISPLAY_BEHAVIOR_FASTEST","searchResultTitleBehavior":"SEARCH_RESULT_TITLE_BEHAVIOR_TEXT"},"catalog":{"moviesSource":"MOVIES_SOURCE_YTS","tvShowsSource":"TV_SHOWS_SOURCE_NETFLIX"},"download":{"folderId":42}}`,
		"--dry-run",
		"--output", "json",
	}, strings.NewReader(""), stdout, stderr)
	if exitCode != int(exitCodeSuccess) {
		t.Fatalf("exitCode = %d, want %d", exitCode, exitCodeSuccess)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal(stdout) error = %v", err)
	}
	if output["dry_run"] != true {
		t.Fatalf("dry_run = %v, want true", output["dry_run"])
	}
	if output["command"] != "user settings set" {
		t.Fatalf("command = %v, want %q", output["command"], "user settings set")
	}
	if output["procedure"] != procedureUserSaveUserSettings {
		t.Fatalf("procedure = %v, want %q", output["procedure"], procedureUserSaveUserSettings)
	}

	request, ok := output["request"].(map[string]any)
	if !ok {
		t.Fatalf("request = %#v, want object", output["request"])
	}
	settings, ok := request["settings"].(map[string]any)
	if !ok {
		t.Fatalf("request.settings = %#v, want object", request["settings"])
	}
	search, ok := settings["search"].(map[string]any)
	if !ok {
		t.Fatalf("request.settings.search = %#v, want object", settings["search"])
	}
	if search["filterNastyResults"] != true {
		t.Fatalf("request.settings.search.filterNastyResults = %v, want true", search["filterNastyResults"])
	}
	if _, ok := settings["catalog"].(map[string]any); !ok {
		t.Fatalf("request.settings.catalog = %#v, want object", settings["catalog"])
	}
	if _, ok := settings["download"].(map[string]any); !ok {
		t.Fatalf("request.settings.download = %#v, want object", settings["download"])
	}
}

func TestRunUserSettingsSetDryRunAcceptsFullRequestJSONFromStdin(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.json")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := Run([]string{
		"--config", configPath,
		"user", "settings", "set",
		"--json", "@-",
		"--dry-run",
		"--output", "json",
	}, strings.NewReader(`{"settings":{"search":{"filterNastyResults":true,"filterResultsWithNoSeeders":false,"rememberQuickFilters":false,"disabledIndexerIds":[],"resolutionFilters":[],"codecFilters":[],"otherFilters":[],"sortBy":"SORT_BY_SEEDERS","sortDirection":"SORT_DIRECTION_DESC","searchResultDisplayBehavior":"SEARCH_RESULT_DISPLAY_BEHAVIOR_FASTEST","searchResultTitleBehavior":"SEARCH_RESULT_TITLE_BEHAVIOR_TEXT"},"catalog":{"moviesSource":"MOVIES_SOURCE_YTS","tvShowsSource":"TV_SHOWS_SOURCE_NETFLIX"},"download":{"folderId":42}}}`), stdout, stderr)
	if exitCode != int(exitCodeSuccess) {
		t.Fatalf("exitCode = %d, want %d", exitCode, exitCodeSuccess)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal(stdout) error = %v", err)
	}
	request, ok := output["request"].(map[string]any)
	if !ok {
		t.Fatalf("request = %#v, want object", output["request"])
	}
	settings, ok := request["settings"].(map[string]any)
	if !ok {
		t.Fatalf("request.settings = %#v, want object", request["settings"])
	}
	search, ok := settings["search"].(map[string]any)
	if !ok {
		t.Fatalf("request.settings.search = %#v, want object", settings["search"])
	}
	if search["filterNastyResults"] != true {
		t.Fatalf("request.settings.search.filterNastyResults = %v, want true", search["filterNastyResults"])
	}
	if _, ok := settings["catalog"].(map[string]any); !ok {
		t.Fatalf("request.settings.catalog = %#v, want object", settings["catalog"])
	}
	if _, ok := settings["download"].(map[string]any); !ok {
		t.Fatalf("request.settings.download = %#v, want object", settings["download"])
	}
}

func TestRunUserSettingsSetPatchDryRunSkipsAuthAndReturnsPreview(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.json")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := Run([]string{
		"--config", configPath,
		"user", "settings", "set",
		"filter-nasty-results", "true",
		"--dry-run",
		"--output", "json",
	}, strings.NewReader(""), stdout, stderr)
	if exitCode != int(exitCodeSuccess) {
		t.Fatalf("exitCode = %d, want %d", exitCode, exitCodeSuccess)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal(stdout) error = %v", err)
	}
	patch, ok := output["request"].(map[string]any)["patch"].(map[string]any)
	if !ok {
		t.Fatalf("request.patch = %#v, want object", output["request"])
	}
	if patch["field"] != "search.filterNastyResults" {
		t.Fatalf("patch.field = %v, want %q", patch["field"], "search.filterNastyResults")
	}
	if patch["value"] != true {
		t.Fatalf("patch.value = %v, want true", patch["value"])
	}
}

func TestRunSettingsSetDryRunReturnsLocalPreview(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.json")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := Run([]string{
		"--config", configPath,
		"settings", "set",
		"api-base-url", "https://api.chill.test",
		"--dry-run",
		"--output", "json",
	}, strings.NewReader(""), stdout, stderr)
	if exitCode != int(exitCodeSuccess) {
		t.Fatalf("exitCode = %d, want %d", exitCode, exitCodeSuccess)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal(stdout) error = %v", err)
	}
	if output["dry_run"] != true {
		t.Fatalf("dry_run = %v, want true", output["dry_run"])
	}
	if output["command"] != "settings set" {
		t.Fatalf("command = %v, want %q", output["command"], "settings set")
	}
	if _, ok := output["procedure"]; ok {
		t.Fatalf("procedure = %#v, want omitted for local preview", output["procedure"])
	}

	request, ok := output["request"].(map[string]any)
	if !ok {
		t.Fatalf("request = %#v, want object", output["request"])
	}
	if request["key"] != "api-base-url" {
		t.Fatalf("request.key = %v, want %q", request["key"], "api-base-url")
	}
	if request["value"] != "https://api.chill.test" {
		t.Fatalf("request.value = %v, want %q", request["value"], "https://api.chill.test")
	}
}

func TestRunSettingsSetDryRunAcceptsJSONPayloadFromStdin(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.json")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := Run([]string{
		"--config", configPath,
		"settings", "set",
		"--json", "@-",
		"--dry-run",
		"--output", "json",
	}, strings.NewReader(`{"key":"api-base-url","value":"https://api.chill.test"}`), stdout, stderr)
	if exitCode != int(exitCodeSuccess) {
		t.Fatalf("exitCode = %d, want %d", exitCode, exitCodeSuccess)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal(stdout) error = %v", err)
	}
	request, ok := output["request"].(map[string]any)
	if !ok {
		t.Fatalf("request = %#v, want object", output["request"])
	}
	if request["key"] != "api-base-url" {
		t.Fatalf("request.key = %v, want %q", request["key"], "api-base-url")
	}
	if request["value"] != "https://api.chill.test" {
		t.Fatalf("request.value = %v, want %q", request["value"], "https://api.chill.test")
	}
}

func TestRunAuthLoginDryRunAcceptsJSONPayloadFromStdin(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.json")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := Run([]string{
		"--config", configPath,
		"auth", "login",
		"--json", "@-",
		"--dry-run",
		"--output", "json",
	}, strings.NewReader(`{"token":"setup-token","skip_verify":true}`), stdout, stderr)
	if exitCode != int(exitCodeSuccess) {
		t.Fatalf("exitCode = %d, want %d", exitCode, exitCodeSuccess)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal(stdout) error = %v", err)
	}
	if output["command"] != "auth login" {
		t.Fatalf("command = %v, want %q", output["command"], "auth login")
	}
	request, ok := output["request"].(map[string]any)
	if !ok {
		t.Fatalf("request = %#v, want object", output["request"])
	}
	if request["mode"] != "token" {
		t.Fatalf("request.mode = %v, want %q", request["mode"], "token")
	}
	if request["token_provided"] != true {
		t.Fatalf("request.token_provided = %v, want true", request["token_provided"])
	}
	if request["skip_verify"] != true {
		t.Fatalf("request.skip_verify = %v, want true", request["skip_verify"])
	}
}

func TestRunAuthLogoutDryRunReturnsLocalPreview(t *testing.T) {
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
	stderr := &bytes.Buffer{}
	exitCode := Run([]string{
		"--config", configPath,
		"auth", "logout",
		"--dry-run",
		"--output", "json",
	}, strings.NewReader(""), stdout, stderr)
	if exitCode != int(exitCodeSuccess) {
		t.Fatalf("exitCode = %d, want %d", exitCode, exitCodeSuccess)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal(stdout) error = %v", err)
	}
	if output["command"] != "auth logout" {
		t.Fatalf("command = %v, want %q", output["command"], "auth logout")
	}
	request, ok := output["request"].(map[string]any)
	if !ok {
		t.Fatalf("request = %#v, want object", output["request"])
	}
	if request["clear_auth_token"] != true {
		t.Fatalf("request.clear_auth_token = %v, want true", request["clear_auth_token"])
	}
	if request["had_auth_token"] != true {
		t.Fatalf("request.had_auth_token = %v, want true", request["had_auth_token"])
	}
}

func TestRunUserDownloadFolderSetDryRunAcceptsJSONPayloadFromStdin(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.json")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := Run([]string{
		"--config", configPath,
		"user", "download-folder", "set",
		"--json", "@-",
		"--dry-run",
		"--output", "json",
	}, strings.NewReader(`{"download":{"folderId":42}}`), stdout, stderr)
	if exitCode != int(exitCodeSuccess) {
		t.Fatalf("exitCode = %d, want %d", exitCode, exitCodeSuccess)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal(stdout) error = %v", err)
	}
	request, ok := output["request"].(map[string]any)
	if !ok {
		t.Fatalf("request = %#v, want object", output["request"])
	}
	patch, ok := request["patch"].(map[string]any)
	if !ok {
		t.Fatalf("request.patch = %#v, want object", request["patch"])
	}
	if patch["field"] != "download.folderId" || patch["value"] != "42" {
		t.Fatalf("patch = %#v", patch)
	}
}

func TestRunUserDownloadFolderClearDryRunAcceptsJSONPayloadFromStdin(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.json")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := Run([]string{
		"--config", configPath,
		"user", "download-folder", "clear",
		"--json", "@-",
		"--dry-run",
		"--output", "json",
	}, strings.NewReader(`{"settings":{"download":{"folderId":null}}}`), stdout, stderr)
	if exitCode != int(exitCodeSuccess) {
		t.Fatalf("exitCode = %d, want %d", exitCode, exitCodeSuccess)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal(stdout) error = %v", err)
	}
	request, ok := output["request"].(map[string]any)
	if !ok {
		t.Fatalf("request = %#v, want object", output["request"])
	}
	patch, ok := request["patch"].(map[string]any)
	if !ok {
		t.Fatalf("request.patch = %#v, want object", request["patch"])
	}
	if patch["field"] != "download.folderId" || patch["value"] != nil {
		t.Fatalf("patch = %#v", patch)
	}
}

func TestRunSettingsPathUsesDevProfileByDefaultForDevBuilds(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	original := currentBuildInfo
	currentBuildInfo = func() buildinfo.Info {
		return buildinfo.Info{Version: "dev", Commit: "test", BuildDate: "test"}
	}
	t.Cleanup(func() { currentBuildInfo = original })

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := Run([]string{"settings", "path", "--output", "json"}, strings.NewReader(""), stdout, stderr)
	if exitCode != int(exitCodeSuccess) {
		t.Fatalf("exitCode = %d, want %d", exitCode, exitCodeSuccess)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal(stdout) error = %v", err)
	}

	wantPath := filepath.Join(configHome, "chilly", "profiles", "dev", "config.json")
	if output["path"] != wantPath {
		t.Fatalf("path = %v, want %q", output["path"], wantPath)
	}
	if output["profile"] != "dev" {
		t.Fatalf("profile = %v, want %q", output["profile"], "dev")
	}
}

func TestRunSettingsPathRejectsInvalidFields(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := Run([]string{"settings", "path", "--fields", "path..value", "--output", "json"}, strings.NewReader(""), stdout, stderr)
	if exitCode != int(exitCodeUsage) {
		t.Fatalf("exitCode = %d, want %d", exitCode, exitCodeUsage)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}

	var output map[string]any
	if err := json.Unmarshal(stderr.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal(stderr) error = %v", err)
	}
	if output["code"] != "invalid_fields" {
		t.Fatalf("code = %v, want %q", output["code"], "invalid_fields")
	}
}

func TestRunRejectsInvalidAPIURLOverride(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := Run([]string{
		"--api-url", "https://api.chill.institute/v4",
		"version",
		"--output", "json",
	}, strings.NewReader(""), stdout, stderr)
	if exitCode != int(exitCodeUsage) {
		t.Fatalf("exitCode = %d, want %d", exitCode, exitCodeUsage)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}

	var output map[string]any
	if err := json.Unmarshal(stderr.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal(stderr) error = %v", err)
	}
	if output["code"] != "invalid_api_base_url" {
		t.Fatalf("code = %v, want %q", output["code"], "invalid_api_base_url")
	}
}

func TestRunReturnsUsageExitCodeAndJSONErrorEnvelope(t *testing.T) {
	t.Parallel()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := Run([]string{"search", "--output", "json"}, strings.NewReader(""), stdout, stderr)
	if exitCode != int(exitCodeUsage) {
		t.Fatalf("exitCode = %d, want %d", exitCode, exitCodeUsage)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}

	var output map[string]any
	if err := json.Unmarshal(stderr.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal(stderr) error = %v", err)
	}
	if output["kind"] != "usage" {
		t.Fatalf("kind = %v, want %q", output["kind"], "usage")
	}
	if output["code"] != "missing_query" {
		t.Fatalf("code = %v, want %q", output["code"], "missing_query")
	}
}

func TestRunReturnsCommandJSONUsageErrorsForInvalidPositionalIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		args      []string
		wantCode  string
		wantError string
	}{
		{
			name:      "negative transfer id",
			args:      []string{"get-transfer", "-1"},
			wantCode:  "invalid_transfer_id",
			wantError: "transfer id must be positive",
		},
		{
			name:      "malformed transfer id",
			args:      []string{"get-transfer", "nope"},
			wantCode:  "invalid_transfer_id",
			wantError: "transfer id must be an integer",
		},
		{
			name:      "negative folder id",
			args:      []string{"user", "folder", "get", "-1"},
			wantCode:  "invalid_folder_id",
			wantError: "folder id must be zero or positive",
		},
		{
			name:      "malformed folder id",
			args:      []string{"user", "folder", "get", "nope"},
			wantCode:  "invalid_folder_id",
			wantError: "folder id must be an integer",
		},
		{
			name:      "negative season number",
			args:      []string{"tv-shows", "season", "tt0944947", "-1"},
			wantCode:  "invalid_season_number",
			wantError: "season number must be positive",
		},
		{
			name:      "negative imdb id",
			args:      []string{"tv-shows", "season", "-1", "1"},
			wantCode:  "invalid_imdb_id",
			wantError: "IMDb id must start with tt",
		},
		{
			name:      "negative detail imdb id",
			args:      []string{"tv-shows", "detail", "-1"},
			wantCode:  "invalid_imdb_id",
			wantError: "IMDb id must start with tt",
		},
		{
			name:      "negative episode number",
			args:      []string{"tv-shows", "episode-download", "tt0944947", "1", "-1"},
			wantCode:  "invalid_episode_number",
			wantError: "episode number must be positive",
		},
		{
			name:      "malformed season number",
			args:      []string{"tv-shows", "season", "tt0944947", "nope"},
			wantCode:  "invalid_season_number",
			wantError: "season number must be an integer",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			args := append(append([]string{}, tc.args...), "--output", "json")
			exitCode := Run(args, strings.NewReader(""), stdout, stderr)
			if exitCode != int(exitCodeUsage) {
				t.Fatalf("exitCode = %d, want %d; stderr = %q", exitCode, exitCodeUsage, stderr.String())
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}

			var output map[string]any
			if err := json.Unmarshal(stderr.Bytes(), &output); err != nil {
				t.Fatalf("json.Unmarshal(stderr) error = %v; stderr = %q", err, stderr.String())
			}
			if output["kind"] != string(errorKindUsage) {
				t.Fatalf("kind = %v, want %q", output["kind"], errorKindUsage)
			}
			if output["code"] != tc.wantCode {
				t.Fatalf("code = %v, want %q", output["code"], tc.wantCode)
			}
			if output["message"] != tc.wantError {
				t.Fatalf("message = %v, want %q", output["message"], tc.wantError)
			}
		})
	}
}

func TestRunReturnsAuthExitCodeAndJSONErrorEnvelope(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.json")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := Run([]string{"--config", configPath, "whoami", "--output", "json"}, strings.NewReader(""), stdout, stderr)
	if exitCode != int(exitCodeAuth) {
		t.Fatalf("exitCode = %d, want %d", exitCode, exitCodeAuth)
	}

	var output map[string]any
	if err := json.Unmarshal(stderr.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal(stderr) error = %v", err)
	}
	if output["kind"] != "auth" {
		t.Fatalf("kind = %v, want %q", output["kind"], "auth")
	}
	if output["code"] != "missing_auth_token" {
		t.Fatalf("code = %v, want %q", output["code"], "missing_auth_token")
	}
}

func TestRunReturnsAPIExitCodeAndJSONErrorEnvelope(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte(`{"code":"search_failed","message":"search backend failed","request_id":"req-500"}`))
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
	stderr := &bytes.Buffer{}
	exitCode := Run([]string{"--config", configPath, "search", "--query", "dune", "--output", "json"}, strings.NewReader(""), stdout, stderr)
	if exitCode != int(exitCodeAPI) {
		t.Fatalf("exitCode = %d, want %d", exitCode, exitCodeAPI)
	}

	var output map[string]any
	if err := json.Unmarshal(stderr.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal(stderr) error = %v", err)
	}
	if output["kind"] != "api" {
		t.Fatalf("kind = %v, want %q", output["kind"], "api")
	}
	if output["code"] != "search_failed" {
		t.Fatalf("code = %v, want %q", output["code"], "search_failed")
	}
	if output["request_id"] != "req-500" {
		t.Fatalf("request_id = %v, want %q", output["request_id"], "req-500")
	}
	if output["status_code"] != float64(http.StatusInternalServerError) {
		t.Fatalf("status_code = %v, want %d", output["status_code"], http.StatusInternalServerError)
	}
}

func TestRunReturnsInternalExitCodeAndJSONErrorEnvelope(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte("{invalid"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := Run([]string{"--config", configPath, "whoami", "--output", "json"}, strings.NewReader(""), stdout, stderr)
	if exitCode != int(exitCodeInternal) {
		t.Fatalf("exitCode = %d, want %d", exitCode, exitCodeInternal)
	}

	var output map[string]any
	if err := json.Unmarshal(stderr.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal(stderr) error = %v", err)
	}
	if output["kind"] != "internal" {
		t.Fatalf("kind = %v, want %q", output["kind"], "internal")
	}
	if output["code"] != "config_load_failed" {
		t.Fatalf("code = %v, want %q", output["code"], "config_load_failed")
	}
}
