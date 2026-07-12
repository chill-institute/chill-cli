package cli

import (
	"strings"
	"testing"
)

func TestNormalizeDownloadFolderJSONValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		value     any
		allowNull bool
		want      any
		wantErr   bool
	}{
		{name: "string", value: "42", want: "42"},
		{name: "float", value: float64(42), want: "42"},
		{name: "null allowed", value: nil, allowNull: true, want: nil},
		{name: "negative", value: float64(-1), wantErr: true},
		{name: "fraction", value: float64(1.5), wantErr: true},
		{name: "null disallowed", value: nil, wantErr: true},
		{name: "unsupported type", value: true, allowNull: true, wantErr: true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalizeDownloadFolderJSONValue(tc.value, tc.allowNull)
			if tc.wantErr {
				if err == nil {
					t.Fatal("normalizeDownloadFolderJSONValue() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeDownloadFolderJSONValue() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("normalizeDownloadFolderJSONValue() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestResolveDownloadFolderRequests(t *testing.T) {
	t.Parallel()

	app := &appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: &strings.Builder{},
		stderr: &strings.Builder{},
	}

	setRequest, err := resolveDownloadFolderSetRequest(app, `{"download":{"folderId":42}}`)
	if err != nil {
		t.Fatalf("resolveDownloadFolderSetRequest() error = %v", err)
	}
	if setRequest.Field != "download.folderId" || setRequest.Value != "42" {
		t.Fatalf("setRequest = %#v", setRequest)
	}

	nestedSetRequest, err := resolveDownloadFolderSetRequest(app, `{"settings":{"download":{"folderId":43}}}`)
	if err != nil {
		t.Fatalf("resolveDownloadFolderSetRequest(nested) error = %v", err)
	}
	if nestedSetRequest.Field != "download.folderId" || nestedSetRequest.Value != "43" {
		t.Fatalf("nestedSetRequest = %#v", nestedSetRequest)
	}

	clearRequest, err := resolveDownloadFolderClearRequest(app, `{"settings":{"download":{"folderId":null}}}`)
	if err != nil {
		t.Fatalf("resolveDownloadFolderClearRequest() error = %v", err)
	}
	if clearRequest.Field != "download.folderId" || clearRequest.Value != nil {
		t.Fatalf("clearRequest = %#v", clearRequest)
	}

	if _, err := resolveDownloadFolderRequest(app, `{}`, false); err == nil {
		t.Fatal("resolveDownloadFolderRequest() error = nil, want missing field error")
	}
	if _, err := resolveDownloadFolderClearRequest(app, `{"settings":{"download":{"folderId":42}}}`); err == nil {
		t.Fatal("resolveDownloadFolderClearRequest() error = nil, want null requirement error")
	}
}

func TestNormalizeUserSettingsJSONObjectRequiresDomainObjects(t *testing.T) {
	t.Parallel()

	if _, err := normalizeUserSettingsJSONObject(map[string]any{
		"search":   map[string]any{},
		"download": map[string]any{},
	}, true); err == nil {
		t.Fatal("normalizeUserSettingsJSONObject(missing catalog) error = nil, want required domains error")
	}
	if _, err := normalizeUserSettingsJSONObject(map[string]any{
		"search":   map[string]any{},
		"catalog":  "MOVIES_SOURCE_YTS",
		"download": map[string]any{},
	}, true); err == nil {
		t.Fatal("normalizeUserSettingsJSONObject(non-object catalog) error = nil, want required domains error")
	}
	if _, err := normalizeUserSettingsJSONObject(map[string]any{
		"search":     map[string]any{},
		"catalog":    map[string]any{},
		"download":   map[string]any{},
		"unexpected": "42",
	}, true); err == nil {
		t.Fatal("normalizeUserSettingsJSONObject(extra top-level field) error = nil, want unsupported field error")
	}
}

func TestDecodeUserSettingsRequestRequiresNestedDomainSettings(t *testing.T) {
	t.Parallel()

	app := &appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: &strings.Builder{},
		stderr: &strings.Builder{},
	}

	request, err := decodeUserSettingsRequest(app, `{"search":{"filterNastyResults":true,"filterResultsWithNoSeeders":false,"rememberQuickFilters":false,"disabledIndexerIds":[],"resolutionFilters":[],"codecFilters":[],"otherFilters":[],"sortBy":"SORT_BY_TITLE","sortDirection":"SORT_DIRECTION_DESC","searchResultDisplayBehavior":"SEARCH_RESULT_DISPLAY_BEHAVIOR_FASTEST","searchResultTitleBehavior":"SEARCH_RESULT_TITLE_BEHAVIOR_TEXT"},"catalog":{"moviesSource":"MOVIES_SOURCE_YTS","tvShowsSource":"TV_SHOWS_SOURCE_HBO_MAX"},"download":{"folderId":42}}`)
	if err != nil {
		t.Fatalf("decodeUserSettingsRequest() error = %v", err)
	}
	settings := request["settings"].(map[string]any)
	if settings["search"].(map[string]any)["sortBy"] != "SORT_BY_TITLE" {
		t.Fatalf("settings = %#v", settings)
	}
	if settings["download"].(map[string]any)["folderId"] != "42" {
		t.Fatalf("settings = %#v", settings)
	}
	if settings["catalog"].(map[string]any)["moviesSource"] != "MOVIES_SOURCE_YTS" {
		t.Fatalf("settings = %#v", settings)
	}
	if settings["catalog"].(map[string]any)["tvShowsSource"] != "TV_SHOWS_SOURCE_HBO_MAX" {
		t.Fatalf("settings = %#v", settings)
	}

	wrapped, err := decodeUserSettingsRequest(app, `{"settings":{"search":{"filterNastyResults":true,"filterResultsWithNoSeeders":false,"rememberQuickFilters":false,"disabledIndexerIds":[],"resolutionFilters":[],"codecFilters":[],"otherFilters":[],"sortBy":"SORT_BY_SEEDERS","sortDirection":"SORT_DIRECTION_DESC","searchResultDisplayBehavior":"SEARCH_RESULT_DISPLAY_BEHAVIOR_FASTEST","searchResultTitleBehavior":"SEARCH_RESULT_TITLE_BEHAVIOR_TEXT"},"catalog":{"moviesSource":"MOVIES_SOURCE_YTS","tvShowsSource":"TV_SHOWS_SOURCE_NETFLIX"},"download":{"folderId":42}}}`)
	if err != nil {
		t.Fatalf("decodeUserSettingsRequest(wrapped) error = %v", err)
	}
	wrappedSettings := wrapped["settings"].(map[string]any)
	if wrappedSettings["search"].(map[string]any)["filterNastyResults"] != true {
		t.Fatalf("wrappedSettings = %#v", wrappedSettings)
	}

	if _, err := decodeUserSettingsRequest(app, `{"settings":{"search":{"filterNastyResults":true}}}`); err == nil {
		t.Fatal("decodeUserSettingsRequest(partial) error = nil, want required domains error")
	}
	if _, err := decodeUserSettingsRequest(app, `{"settings":{"search":{},"catalog":{},"download":{},"unexpected":true}}`); err == nil {
		t.Fatal("decodeUserSettingsRequest(extra top-level field) error = nil, want unsupported field error")
	}
}

func TestNormalizeIDsAndRendererMapping(t *testing.T) {
	t.Parallel()

	if _, err := normalizeFolderID("-1"); err == nil {
		t.Fatal("normalizeFolderID(-1) error = nil, want error")
	}
	if value, err := normalizeFolderID("42"); err != nil || value != 42 {
		t.Fatalf("normalizeFolderID(42) = %d, %v", value, err)
	}
	if _, err := normalizeTransferID("0"); err == nil {
		t.Fatal("normalizeTransferID(0) error = nil, want error")
	}
	if value, err := normalizeTransferID("42"); err != nil || value != 42 {
		t.Fatalf("normalizeTransferID(42) = %d, %v", value, err)
	}

	if prettyRendererForProcedure(procedureUserGetIndexers) == nil {
		t.Fatal("prettyRendererForProcedure(indexers) = nil")
	}
	if prettyRendererForProcedure(procedureUserGetMovies) == nil {
		t.Fatal("prettyRendererForProcedure(movies) = nil")
	}
	if prettyRendererForProcedure(procedureUserGetTVShows) == nil {
		t.Fatal("prettyRendererForProcedure(tv shows) = nil")
	}
	if prettyRendererForProcedure(procedureUserGetTVShowDetail) == nil {
		t.Fatal("prettyRendererForProcedure(tv show detail) = nil")
	}
	if prettyRendererForProcedure(procedureUserGetTVShowSeason) == nil {
		t.Fatal("prettyRendererForProcedure(tv show season) = nil")
	}
	if prettyRendererForProcedure(procedureUserGetTVShowEpisodeDownload) == nil {
		t.Fatal("prettyRendererForProcedure(tv show episode download) = nil")
	}
	if prettyRendererForProcedure(procedureUserGetTVShowSeasonDownloads) == nil {
		t.Fatal("prettyRendererForProcedure(tv show season downloads) = nil")
	}
	if prettyRendererForProcedure("unknown") != nil {
		t.Fatal("prettyRendererForProcedure(unknown) != nil")
	}
}

func TestNormalizeIMDbID(t *testing.T) {
	t.Parallel()

	if got, err := normalizeIMDbID(" tt0944947 "); err != nil || got != "tt0944947" {
		t.Fatalf("normalizeIMDbID() = %q, %v", got, err)
	}

	for _, raw := range []string{"", "bad", "ttabc", "tt123", "tt0944947?x=1", "tt0944947%2f"} {
		raw := raw
		t.Run(raw, func(t *testing.T) {
			t.Parallel()

			if _, err := normalizeIMDbID(raw); err == nil {
				t.Fatalf("normalizeIMDbID(%q) error = nil, want error", raw)
			}
		})
	}
}

func TestNormalizeEpisodeOrdinal(t *testing.T) {
	t.Parallel()

	if got, err := normalizeEpisodeOrdinal("1", "season"); err != nil || got != 1 {
		t.Fatalf("normalizeEpisodeOrdinal() = %d, %v", got, err)
	}
	if _, err := normalizeEpisodeOrdinal("0", "season"); err == nil {
		t.Fatal("normalizeEpisodeOrdinal(0) error = nil, want error")
	}
	if _, err := normalizeEpisodeOrdinal("abc", "episode"); err == nil {
		t.Fatal("normalizeEpisodeOrdinal(abc) error = nil, want error")
	}
}

func TestSettingsValidationHelpers(t *testing.T) {
	t.Parallel()

	validURLs := map[string]string{
		"https://api.chill.institute/": "https://api.chill.institute",
		"https://api.example.test":     "https://api.example.test",
		"http://localhost:8080":        "http://localhost:8080",
		"http://api.localhost:3000":    "http://api.localhost:3000",
		"http://127.0.0.1:8080":        "http://127.0.0.1:8080",
		"http://[::1]:8080":            "http://[::1]:8080",
	}
	for raw, want := range validURLs {
		raw := raw
		want := want
		t.Run(raw, func(t *testing.T) {
			t.Parallel()
			got, err := normalizeAPIBaseURL(raw)
			if err != nil || got != want {
				t.Fatalf("normalizeAPIBaseURL(%q) = %q, %v; want %q", raw, got, err, want)
			}
		})
	}

	invalidURLs := []string{
		"https://api.chill.institute/path",
		"https://user@api.chill.institute",
		"http://example.test",
		"http://localhost.example.test",
		"http://notlocalhost",
	}
	for _, raw := range invalidURLs {
		raw := raw
		t.Run(raw, func(t *testing.T) {
			t.Parallel()
			if _, err := normalizeAPIBaseURL(raw); err == nil {
				t.Fatalf("normalizeAPIBaseURL(%q) error = nil, want error", raw)
			}
		})
	}

	if _, err := normalizeSettingsKey("api_base_url"); err != nil {
		t.Fatalf("normalizeSettingsKey(api_base_url) error = %v", err)
	}
	if _, err := normalizeSettingsKey("missing"); err == nil {
		t.Fatal("normalizeSettingsKey(missing) error = nil, want error")
	}
}

func TestNormalizeIndexerID(t *testing.T) {
	t.Parallel()

	if got, err := normalizeIndexerID(" yts "); err != nil || got != "yts" {
		t.Fatalf("normalizeIndexerID() = %q, %v", got, err)
	}

	for _, raw := range []string{"", "bad\x00id", "../yts", "bad/id", "bad?id", "bad%2Fyts"} {
		raw := raw
		t.Run(raw, func(t *testing.T) {
			t.Parallel()

			if _, err := normalizeIndexerID(raw); err == nil {
				t.Fatalf("normalizeIndexerID(%q) error = nil, want error", raw)
			}
		})
	}
}

func TestNormalizeTransferURL(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		raw  string
		want string
	}{
		{name: "magnet", raw: " magnet:?xt=urn:btih:test ", want: "magnet:?xt=urn:btih:test"},
		{name: "http", raw: " http://example.test/file.torrent ", want: "http://example.test/file.torrent"},
		{name: "https", raw: " https://example.test/file.torrent ", want: "https://example.test/file.torrent"},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got, err := normalizeTransferURL(tc.raw); err != nil || got != tc.want {
				t.Fatalf("normalizeTransferURL() = %q, %v; want %q, nil", got, err, tc.want)
			}
		})
	}

	if _, err := normalizeTransferURL(""); err == nil {
		t.Fatal("normalizeTransferURL(empty) error = nil, want error")
	}

	for _, raw := range []string{
		"ftp://example.test/file.torrent",
		"not-a-url",
		"bad\x00url",
		"https://:443/file.torrent",
		"https://example.test/file name.torrent",
		"magnet:?xt=urn:btih:test&dn=My Movie",
	} {
		raw := raw
		t.Run(raw, func(t *testing.T) {
			t.Parallel()

			if _, err := normalizeTransferURL(raw); err == nil {
				t.Fatalf("normalizeTransferURL(%q) error = nil, want error", raw)
			}
		})
	}
}
