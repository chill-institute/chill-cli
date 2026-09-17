package chill

import (
	"errors"
	"testing"
)

func TestUserSettingsPatchNormalizesAliasesAndValues(t *testing.T) {
	t.Parallel()
	cases := []struct {
		field, value, wantField string
		want                    any
	}{
		{"filter-nasty-results", "true", "search.filterNastyResults", true},
		{"search.sort-by", "Uploaded-At", "search.sortBy", "SORT_BY_UPLOADED_AT"},
		{"tv-shows-source", "paramount plus", "catalog.tvShowsSource", "TV_SHOWS_SOURCE_PARAMOUNT_PLUS"},
		{"movies-source", "IMDb Top 250", "catalog.moviesSource", "MOVIES_SOURCE_IMDB_TOP_250"},
		{"catalog.sort", "release-date-desc", "catalog.sort", "CATALOG_SORT_RELEASE_DATE_DESC"},
		{"download.folder-id", "42", "download.folderId", "42"},
		{"download.folderId", "null", "download.folderId", nil},
	}
	for _, tc := range cases {
		patch, err := NormalizeUserSettingsPatch(tc.field, tc.value)
		if err != nil {
			t.Fatalf("%s: %v", tc.field, err)
		}
		if patch.Field != tc.wantField || patch.Value != tc.want {
			t.Fatalf("%s: patch = %#v", tc.field, patch)
		}
	}
	for _, tc := range [][2]string{{"nope", "1"}, {"catalog.sort", "unspecified"}, {"filter-nasty-results", "maybe"}, {"download.folder-id", "-1"}} {
		_, err := NormalizeUserSettingsPatch(tc[0], tc[1])
		var validation *ValidationError
		if !errors.As(err, &validation) {
			t.Fatalf("%v: error = %v, want ValidationError", tc, err)
		}
	}
	if len(UserSettingsFields()) != 11 {
		t.Fatalf("field count = %d", len(UserSettingsFields()))
	}
}

func TestApplyUserSettingsPatchCopiesDomainsOnly(t *testing.T) {
	t.Parallel()
	source := map[string]any{
		"search":   map[string]any{"sortBy": "SORT_BY_TITLE", "disabledIndexerIds": []any{"a"}},
		"catalog":  map[string]any{"moviesSource": "MOVIES_SOURCE_YTS"},
		"download": map[string]any{"folderId": "1"},
		"extra":    "dropped",
	}
	patched := ApplyUserSettingsPatch(source, UserSettingsPatch{Field: "catalog.sort", Value: "CATALOG_SORT_POPULARITY"})
	if _, ok := patched["extra"]; ok {
		t.Fatal("unknown top-level field survived")
	}
	if patched["catalog"].(map[string]any)["sort"] != "CATALOG_SORT_POPULARITY" {
		t.Fatalf("patched = %#v", patched)
	}
	if _, ok := source["catalog"].(map[string]any)["sort"]; ok {
		t.Fatal("source mutated")
	}
	patched["search"].(map[string]any)["disabledIndexerIds"].([]any)[0] = "b"
	if source["search"].(map[string]any)["disabledIndexerIds"].([]any)[0] != "a" {
		t.Fatal("array aliased between source and clone")
	}
}

func TestIDAndTVRequestBuilders(t *testing.T) {
	t.Parallel()
	if id, err := NormalizeFolderID(" 0 "); err != nil || id != 0 {
		t.Fatalf("folder id = %d, %v", id, err)
	}
	if _, err := NormalizeFolderID("-1"); err == nil {
		t.Fatal("negative folder id accepted")
	}
	if _, err := NormalizeTransferID("0"); err == nil {
		t.Fatal("zero transfer id accepted")
	}
	body, err := TVShowEpisodeRequest("TT0944947", "1", "3")
	if err != nil {
		t.Fatal(err)
	}
	if body["imdbId"] != "tt0944947" || body["seasonNumber"] != int32(1) || body["episodeNumber"] != int32(3) {
		t.Fatalf("body = %#v", body)
	}
	if _, err := TVShowSeasonRequest("tt0944947", "0"); err == nil {
		t.Fatal("zero season accepted")
	}
	if _, err := TVShowDetailRequest("nm1"); err == nil {
		t.Fatal("bad imdb accepted")
	}
}
