package chill

import (
	"fmt"
	"strconv"
	"strings"
)

// UserSettingsPatch is one dotted-path change to hosted user settings, for
// example {Field: "catalog.sort", Value: "CATALOG_SORT_POPULARITY"}.
type UserSettingsPatch struct {
	Field string `json:"field"`
	Value any    `json:"value"`
}

// UserSettingsField describes one patchable hosted user setting.
type UserSettingsField struct {
	// Path is the JSON path inside the settings object.
	Path []string
	// Aliases are accepted human spellings such as "movies-source".
	Aliases []string
	// ValueType is boolean, enum, or integer-or-null.
	ValueType string
	// Description lists accepted values for enums.
	Description string
	normalize   func(string) (any, error)
}

// Name is the dotted JSON path, for example "catalog.moviesSource".
func (field UserSettingsField) Name() string {
	return strings.Join(field.Path, ".")
}

var userSettingsFields = []UserSettingsField{
	{Aliases: []string{"filter-nasty-results", "search.filter-nasty-results"}, Path: []string{"search", "filterNastyResults"}, ValueType: "boolean", Description: "whether nasty results should be filtered", normalize: NormalizeBooleanValue},
	{Aliases: []string{"filter-results-with-no-seeders", "search.filter-results-with-no-seeders"}, Path: []string{"search", "filterResultsWithNoSeeders"}, ValueType: "boolean", Description: "whether results with no seeders should be filtered", normalize: NormalizeBooleanValue},
	{Aliases: []string{"remember-quick-filters", "search.remember-quick-filters"}, Path: []string{"search", "rememberQuickFilters"}, ValueType: "boolean", Description: "whether quick filters should be remembered", normalize: NormalizeBooleanValue},
	{Aliases: []string{"download.folder-id"}, Path: []string{"download", "folderId"}, ValueType: "integer-or-null", Description: "download folder id, or null to clear it", normalize: NormalizeNullableNonNegativeInt64Value},
	{Aliases: []string{"sort-by", "search.sort-by"}, Path: []string{"search", "sortBy"}, ValueType: "enum", Description: "one of: title, seeders, size, uploaded-at, source", normalize: NormalizeEnumValue(map[string]string{
		"title": "SORT_BY_TITLE", "seeders": "SORT_BY_SEEDERS", "size": "SORT_BY_SIZE", "uploaded-at": "SORT_BY_UPLOADED_AT", "uploaded_at": "SORT_BY_UPLOADED_AT", "source": "SORT_BY_SOURCE",
	})},
	{Aliases: []string{"sort-direction", "search.sort-direction"}, Path: []string{"search", "sortDirection"}, ValueType: "enum", Description: "one of: asc, desc", normalize: NormalizeEnumValue(map[string]string{
		"asc": "SORT_DIRECTION_ASC", "desc": "SORT_DIRECTION_DESC",
	})},
	{Aliases: []string{"search-result-display-behavior", "search.search-result-display-behavior"}, Path: []string{"search", "searchResultDisplayBehavior"}, ValueType: "enum", Description: "one of: all, fastest", normalize: NormalizeEnumValue(map[string]string{
		"all": "SEARCH_RESULT_DISPLAY_BEHAVIOR_ALL", "fastest": "SEARCH_RESULT_DISPLAY_BEHAVIOR_FASTEST",
	})},
	{Aliases: []string{"search-result-title-behavior", "search.search-result-title-behavior"}, Path: []string{"search", "searchResultTitleBehavior"}, ValueType: "enum", Description: "one of: link, text", normalize: NormalizeEnumValue(map[string]string{
		"link": "SEARCH_RESULT_TITLE_BEHAVIOR_LINK", "text": "SEARCH_RESULT_TITLE_BEHAVIOR_TEXT",
	})},
	{Aliases: []string{"movies-source", "catalog.movies-source"}, Path: []string{"catalog", "moviesSource"}, ValueType: "enum", Description: "one of: imdb-moviemeter, imdb-top-250, yts, rotten-tomatoes, trakt", normalize: func(raw string) (any, error) {
		return NormalizeMovieSource(raw)
	}},
	{Aliases: []string{"tv-shows-source", "catalog.tv-shows-source"}, Path: []string{"catalog", "tvShowsSource"}, ValueType: "enum", Description: "one of: all-providers, netflix, hbo-max, apple-tv-plus, prime-video, disney-plus, hulu, paramount-plus, amc-plus, peacock", normalize: func(raw string) (any, error) {
		return NormalizeTVShowsSource(raw, false)
	}},
	{Aliases: []string{"catalog-sort", "catalog.sort"}, Path: []string{"catalog", "sort"}, ValueType: "enum", Description: "shared movies, TV shows, and providers ordering; one of: popularity, rating-desc, rating-asc, release-date-desc, release-date-asc", normalize: NormalizeEnumValue(map[string]string{
		"popularity": "CATALOG_SORT_POPULARITY", "rating-desc": "CATALOG_SORT_RATING_DESC", "rating_desc": "CATALOG_SORT_RATING_DESC", "rating-asc": "CATALOG_SORT_RATING_ASC", "rating_asc": "CATALOG_SORT_RATING_ASC", "release-date-desc": "CATALOG_SORT_RELEASE_DATE_DESC", "release_date_desc": "CATALOG_SORT_RELEASE_DATE_DESC", "release-date-asc": "CATALOG_SORT_RELEASE_DATE_ASC", "release_date_asc": "CATALOG_SORT_RELEASE_DATE_ASC",
	})},
}

// UserSettingsFields returns the patchable settings in documentation order.
func UserSettingsFields() []UserSettingsField {
	out := make([]UserSettingsField, len(userSettingsFields))
	copy(out, userSettingsFields)
	return out
}

// LookupUserSettingsField resolves a dotted path or alias, case-insensitively.
func LookupUserSettingsField(raw string) (UserSettingsField, bool) {
	trimmed := strings.TrimSpace(raw)
	for _, field := range userSettingsFields {
		if strings.EqualFold(trimmed, field.Name()) {
			return field, true
		}
		for _, alias := range field.Aliases {
			if strings.EqualFold(trimmed, alias) {
				return field, true
			}
		}
	}
	return UserSettingsField{}, false
}

// NormalizeUserSettingsPatch validates one field and value into a patch.
func NormalizeUserSettingsPatch(field string, value string) (UserSettingsPatch, error) {
	spec, ok := LookupUserSettingsField(field)
	if !ok {
		return UserSettingsPatch{}, invalid("unsupported_user_settings_field", "unsupported user settings field %q", field)
	}
	normalized, err := spec.normalize(value)
	if err != nil {
		return UserSettingsPatch{}, err
	}
	return UserSettingsPatch{Field: spec.Name(), Value: normalized}, nil
}

// ApplyUserSettingsPatch returns the search, catalog, and download domains of
// settings with the patch applied. The input is not modified.
func ApplyUserSettingsPatch(settings map[string]any, patch UserSettingsPatch) map[string]any {
	cloned := CloneUserSettingsDomains(settings)
	SetNestedJSONObjectValue(cloned, strings.Split(patch.Field, "."), patch.Value)
	return cloned
}

// NormalizeBooleanValue parses a boolean setting value.
func NormalizeBooleanValue(raw string) (any, error) {
	parsed, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return nil, invalid("invalid_user_settings_value", "expected boolean value, got %q", raw)
	}
	return parsed, nil
}

// NormalizeNullableNonNegativeInt64Value parses an id or null/none, returning
// the id as a decimal string to match the API's int64 JSON encoding.
func NormalizeNullableNonNegativeInt64Value(raw string) (any, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, invalid("invalid_user_settings_value", "expected non-negative integer or null, got empty value")
	}
	if strings.EqualFold(trimmed, "null") || strings.EqualFold(trimmed, "none") {
		return nil, nil
	}
	parsed, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil || parsed < 0 {
		return nil, invalid("invalid_user_settings_value", "expected non-negative integer or null, got %q", raw)
	}
	return strconv.FormatInt(parsed, 10), nil
}

// NormalizeEnumValue maps lowercase aliases to enum values.
func NormalizeEnumValue(values map[string]string) func(string) (any, error) {
	return func(raw string) (any, error) {
		trimmed := strings.TrimSpace(strings.ToLower(raw))
		if normalized, ok := values[trimmed]; ok {
			return normalized, nil
		}
		return nil, invalid("invalid_user_settings_value", "unsupported value %q", raw)
	}
}

// NormalizeFolderID parses a zero-or-positive put.io folder id.
func NormalizeFolderID(raw string) (int64, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, invalid("missing_folder_id", "folder id is required")
	}
	value, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return 0, invalid("invalid_folder_id", "folder id must be an integer")
	}
	if value < 0 {
		return 0, invalid("invalid_folder_id", "folder id must be zero or positive")
	}
	return value, nil
}

// NormalizeTransferID parses a positive put.io transfer id.
func NormalizeTransferID(raw string) (int64, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, invalid("missing_transfer_id", "transfer id is required")
	}
	value, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return 0, invalid("invalid_transfer_id", "transfer id must be an integer")
	}
	if value <= 0 {
		return 0, invalid("invalid_transfer_id", "transfer id must be positive")
	}
	return value, nil
}

// CloneJSONObject deep-copies decoded JSON objects and arrays.
func CloneJSONObject(source map[string]any) map[string]any {
	if len(source) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		switch typed := value.(type) {
		case map[string]any:
			cloned[key] = CloneJSONObject(typed)
		case []any:
			next := make([]any, len(typed))
			copy(next, typed)
			cloned[key] = next
		default:
			cloned[key] = value
		}
	}
	return cloned
}

// CloneUserSettingsDomains copies only the search, catalog, and download
// objects, dropping unknown top-level fields before a save.
func CloneUserSettingsDomains(settings map[string]any) map[string]any {
	cloned := map[string]any{}
	for _, domain := range []string{"search", "catalog", "download"} {
		if value, ok := settings[domain].(map[string]any); ok {
			cloned[domain] = CloneJSONObject(value)
		}
	}
	return cloned
}

// SetNestedJSONObjectValue sets value at path, creating objects as needed.
func SetNestedJSONObjectValue(target map[string]any, path []string, value any) {
	if len(path) == 0 {
		return
	}
	if len(path) == 1 {
		target[path[0]] = value
		return
	}
	next, ok := target[path[0]].(map[string]any)
	if !ok {
		next = map[string]any{}
		target[path[0]] = next
	}
	SetNestedJSONObjectValue(next, path[1:], value)
}

// TVShowDetailRequest builds the GetTVShowDetail body.
func TVShowDetailRequest(imdbID string) (map[string]any, error) {
	id, err := NormalizeIMDbID(imdbID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"imdbId": id}, nil
}

// TVShowSeasonRequest builds the GetTVShowSeason and GetTVShowSeasonDownloads body.
func TVShowSeasonRequest(imdbID string, season string) (map[string]any, error) {
	id, err := NormalizeIMDbID(imdbID)
	if err != nil {
		return nil, err
	}
	seasonNumber, err := NormalizeEpisodeOrdinal(season, "season")
	if err != nil {
		return nil, err
	}
	return map[string]any{"imdbId": id, "seasonNumber": seasonNumber}, nil
}

// TVShowEpisodeRequest builds the GetTVShowEpisodeDownload body.
func TVShowEpisodeRequest(imdbID string, season string, episode string) (map[string]any, error) {
	body, err := TVShowSeasonRequest(imdbID, season)
	if err != nil {
		return nil, err
	}
	episodeNumber, err := NormalizeEpisodeOrdinal(episode, "episode")
	if err != nil {
		return nil, err
	}
	body["episodeNumber"] = episodeNumber
	return body, nil
}

// String returns the field list for help text.
func (field UserSettingsField) String() string {
	return fmt.Sprintf("%s (%s): %s", field.Name(), field.ValueType, field.Description)
}
