package cli

import (
	"testing"
)

func TestNormalizeUserSettingsPatch(t *testing.T) {
	t.Parallel()

	patch, err := normalizeUserSettingsPatch("filter-nasty-results", "true")
	if err != nil {
		t.Fatalf("normalizeUserSettingsPatch() error = %v", err)
	}
	if patch.Field != "search.filterNastyResults" || patch.Value != true {
		t.Fatalf("patch = %#v", patch)
	}

	enumPatch, err := normalizeUserSettingsPatch("sort-by", "uploaded-at")
	if err != nil {
		t.Fatalf("normalizeUserSettingsPatch(enum) error = %v", err)
	}
	if enumPatch.Value != "SORT_BY_UPLOADED_AT" {
		t.Fatalf("enumPatch = %#v", enumPatch)
	}

	tvSourcePatch, err := normalizeUserSettingsPatch("tv-shows-source", "paramount-plus")
	if err != nil {
		t.Fatalf("normalizeUserSettingsPatch(tv source) error = %v", err)
	}
	if tvSourcePatch.Field != "catalog.tvShowsSource" ||
		tvSourcePatch.Value != "TV_SHOWS_SOURCE_PARAMOUNT_PLUS" {
		t.Fatalf("tvSourcePatch = %#v", tvSourcePatch)
	}

	downloadPatch, err := normalizeUserSettingsPatch("download.folderId", "42")
	if err != nil {
		t.Fatalf("normalizeUserSettingsPatch(download) error = %v", err)
	}
	if downloadPatch.Field != "download.folderId" || downloadPatch.Value != "42" {
		t.Fatalf("downloadPatch = %#v", downloadPatch)
	}

	if _, err := normalizeUserSettingsPatch("download.folderId", "-1"); err == nil {
		t.Fatal("normalizeUserSettingsPatch(download negative) error = nil, want invalid value")
	}
	if _, err := normalizeUserSettingsPatch("download-folder-id", "42"); err == nil {
		t.Fatal("normalizeUserSettingsPatch(download-folder-id) error = nil, want unsupported legacy field")
	}
	if _, err := normalizeUserSettingsPatch("missing-field", "x"); err == nil {
		t.Fatal("normalizeUserSettingsPatch() error = nil, want unsupported field")
	}
}

func TestNormalizeNullableNonNegativeInt64Value(t *testing.T) {
	t.Parallel()

	if value, err := normalizeNullableNonNegativeInt64Value("42"); err != nil || value != "42" {
		t.Fatalf("normalizeNullableNonNegativeInt64Value(42) = %#v, %v", value, err)
	}
	if value, err := normalizeNullableNonNegativeInt64Value("null"); err != nil || value != nil {
		t.Fatalf("normalizeNullableNonNegativeInt64Value(null) = %#v, %v", value, err)
	}
	if value, err := normalizeNullableNonNegativeInt64Value("none"); err != nil || value != nil {
		t.Fatalf("normalizeNullableNonNegativeInt64Value(none) = %#v, %v", value, err)
	}
	if _, err := normalizeNullableNonNegativeInt64Value(""); err == nil {
		t.Fatal("normalizeNullableNonNegativeInt64Value(empty) error = nil, want error")
	}
	if _, err := normalizeNullableNonNegativeInt64Value("nope"); err == nil {
		t.Fatal("normalizeNullableNonNegativeInt64Value(nope) error = nil, want error")
	}
	if _, err := normalizeNullableNonNegativeInt64Value("-1"); err == nil {
		t.Fatal("normalizeNullableNonNegativeInt64Value(-1) error = nil, want error")
	}
}

func TestNormalizeEnumValue(t *testing.T) {
	t.Parallel()

	normalize := normalizeEnumValue(map[string]string{"asc": "ASC"})
	if value, err := normalize("ASC"); err != nil || value != "ASC" {
		t.Fatalf("normalizeEnumValue() = %#v, %v", value, err)
	}
	if _, err := normalize("desc"); err == nil {
		t.Fatal("normalizeEnumValue(desc) error = nil, want error")
	}
}

func TestApplyUserSettingsPatchAndCloneJSONObject(t *testing.T) {
	t.Parallel()

	source := map[string]any{
		"unrelated": "value",
		"search": map[string]any{
			"sortBy": "SORT_BY_SEEDERS",
		},
		"settings": map[string]any{
			"nested": "value",
		},
		"items": []any{"a", "b"},
	}

	cloned := cloneJSONObject(source)
	clonedSettings := cloned["settings"].(map[string]any)
	clonedSettings["nested"] = "changed"
	if source["settings"].(map[string]any)["nested"] != "value" {
		t.Fatalf("source mutated = %#v", source)
	}

	patched := applyUserSettingsPatch(source, userSettingsPatch{
		Field: "search.filterNastyResults",
		Value: true,
	})
	search := patched["search"].(map[string]any)
	if search["filterNastyResults"] != true {
		t.Fatalf("patched = %#v", patched)
	}
	if search["sortBy"] != "SORT_BY_SEEDERS" {
		t.Fatalf("patched.search.sortBy = %v, want canonical nested value", search["sortBy"])
	}
	if _, ok := patched["unrelated"]; ok {
		t.Fatalf("patched kept non-domain field: %#v", patched)
	}
}
