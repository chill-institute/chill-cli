package cli

import (
	"fmt"
	"strconv"
	"strings"
)

type userSettingsPatch struct {
	Field string `json:"field"`
	Value any    `json:"value"`
}

type userSettingsPatchSpec struct {
	aliases     []string
	path        []string
	valueType   string
	description string
	normalize   func(string) (any, error)
}

var userSettingsPatchSpecs = []userSettingsPatchSpec{
	{
		aliases:     []string{"filter-nasty-results", "search.filter-nasty-results"},
		path:        []string{"search", "filterNastyResults"},
		valueType:   "boolean",
		description: "whether nasty results should be filtered",
		normalize:   normalizeBooleanValue,
	},
	{
		aliases:     []string{"filter-results-with-no-seeders", "search.filter-results-with-no-seeders"},
		path:        []string{"search", "filterResultsWithNoSeeders"},
		valueType:   "boolean",
		description: "whether results with no seeders should be filtered",
		normalize:   normalizeBooleanValue,
	},
	{
		aliases:     []string{"remember-quick-filters", "search.remember-quick-filters"},
		path:        []string{"search", "rememberQuickFilters"},
		valueType:   "boolean",
		description: "whether quick filters should be remembered",
		normalize:   normalizeBooleanValue,
	},
	{
		aliases:     []string{"download.folder-id"},
		path:        []string{"download", "folderId"},
		valueType:   "integer-or-null",
		description: "download folder id, or null to clear it",
		normalize:   normalizeNullableNonNegativeInt64Value,
	},
	{
		aliases:     []string{"sort-by", "search.sort-by"},
		path:        []string{"search", "sortBy"},
		valueType:   "enum",
		description: "one of: title, seeders, size, uploaded-at, source",
		normalize: normalizeEnumValue(map[string]string{
			"title":       "SORT_BY_TITLE",
			"seeders":     "SORT_BY_SEEDERS",
			"size":        "SORT_BY_SIZE",
			"uploaded-at": "SORT_BY_UPLOADED_AT",
			"uploaded_at": "SORT_BY_UPLOADED_AT",
			"source":      "SORT_BY_SOURCE",
		}),
	},
	{
		aliases:     []string{"sort-direction", "search.sort-direction"},
		path:        []string{"search", "sortDirection"},
		valueType:   "enum",
		description: "one of: asc, desc",
		normalize: normalizeEnumValue(map[string]string{
			"asc":  "SORT_DIRECTION_ASC",
			"desc": "SORT_DIRECTION_DESC",
		}),
	},
	{
		aliases:     []string{"search-result-display-behavior", "search.search-result-display-behavior"},
		path:        []string{"search", "searchResultDisplayBehavior"},
		valueType:   "enum",
		description: "one of: all, fastest",
		normalize: normalizeEnumValue(map[string]string{
			"all":     "SEARCH_RESULT_DISPLAY_BEHAVIOR_ALL",
			"fastest": "SEARCH_RESULT_DISPLAY_BEHAVIOR_FASTEST",
		}),
	},
	{
		aliases:     []string{"search-result-title-behavior", "search.search-result-title-behavior"},
		path:        []string{"search", "searchResultTitleBehavior"},
		valueType:   "enum",
		description: "one of: link, text",
		normalize: normalizeEnumValue(map[string]string{
			"link": "SEARCH_RESULT_TITLE_BEHAVIOR_LINK",
			"text": "SEARCH_RESULT_TITLE_BEHAVIOR_TEXT",
		}),
	},
	{
		aliases:     []string{"movies-source", "catalog.movies-source"},
		path:        []string{"catalog", "moviesSource"},
		valueType:   "enum",
		description: "one of: imdb-moviemeter, imdb-top-250, yts, rotten-tomatoes, trakt",
		normalize: normalizeEnumValue(map[string]string{
			"imdb-moviemeter": "MOVIES_SOURCE_IMDB_MOVIEMETER",
			"imdb/moviemeter": "MOVIES_SOURCE_IMDB_MOVIEMETER",
			"imdb-top-250":    "MOVIES_SOURCE_IMDB_TOP_250",
			"imdb/top-250":    "MOVIES_SOURCE_IMDB_TOP_250",
			"yts":             "MOVIES_SOURCE_YTS",
			"rotten-tomatoes": "MOVIES_SOURCE_ROTTEN_TOMATOES",
			"rotten_tomatoes": "MOVIES_SOURCE_ROTTEN_TOMATOES",
			"rottentomatoes":  "MOVIES_SOURCE_ROTTEN_TOMATOES",
			"trakt":           "MOVIES_SOURCE_TRAKT",
		}),
	},
	{
		aliases:     []string{"tv-shows-source", "catalog.tv-shows-source"},
		path:        []string{"catalog", "tvShowsSource"},
		valueType:   "enum",
		description: "one of: all-providers, netflix, hbo-max, apple-tv-plus, prime-video, disney-plus, hulu, paramount-plus, amc-plus, peacock",
		normalize:   normalizeTVShowsSourcePatchValue,
	},
}

func normalizeUserSettingsPatch(field string, value string) (userSettingsPatch, error) {
	spec, ok := userSettingsPatchSpecForField(field)
	if !ok {
		return userSettingsPatch{}, usageError("unsupported_user_settings_field", "unsupported user settings field %q", field)
	}

	normalizedValue, err := spec.normalize(value)
	if err != nil {
		return userSettingsPatch{}, err
	}
	return userSettingsPatch{
		Field: strings.Join(spec.path, "."),
		Value: normalizedValue,
	}, nil
}

func applyUserSettingsPatch(settings map[string]any, patch userSettingsPatch) map[string]any {
	cloned := cloneUserSettingsDomains(settings)
	setNestedJSONObjectValue(cloned, strings.Split(patch.Field, "."), patch.Value)
	return cloned
}

func supportedUserSettingsPatchInputs() []schemaInput {
	inputs := make([]schemaInput, 0, len(userSettingsPatchSpecs)*2)
	for _, spec := range userSettingsPatchSpecs {
		inputs = append(inputs,
			schemaInput{
				Name:        fmt.Sprintf("field:%s", strings.Join(spec.path, ".")),
				Type:        spec.valueType,
				Description: spec.description,
			},
		)
	}
	return inputs
}

func supportedUserSettingsPatchHelp() string {
	lines := make([]string, 0, len(userSettingsPatchSpecs)+1)
	lines = append(lines, "Supported patch fields:")
	for _, spec := range userSettingsPatchSpecs {
		lines = append(lines, fmt.Sprintf("  - %s (%s): %s", strings.Join(spec.path, "."), spec.valueType, spec.description))
	}
	return strings.Join(lines, "\n")
}

func userSettingsPatchSpecForField(raw string) (userSettingsPatchSpec, bool) {
	trimmed := strings.TrimSpace(raw)
	for _, spec := range userSettingsPatchSpecs {
		if strings.EqualFold(trimmed, strings.Join(spec.path, ".")) {
			return spec, true
		}
		for _, alias := range spec.aliases {
			if strings.EqualFold(trimmed, alias) {
				return spec, true
			}
		}
	}
	return userSettingsPatchSpec{}, false
}

func normalizeBooleanValue(raw string) (any, error) {
	parsed, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return nil, usageError("invalid_user_settings_value", "expected boolean value, got %q", raw)
	}
	return parsed, nil
}

func normalizeNullableNonNegativeInt64Value(raw string) (any, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, usageError("invalid_user_settings_value", "expected non-negative integer or null, got empty value")
	}
	if strings.EqualFold(trimmed, "null") || strings.EqualFold(trimmed, "none") {
		return nil, nil
	}

	parsed, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil || parsed < 0 {
		return nil, usageError("invalid_user_settings_value", "expected non-negative integer or null, got %q", raw)
	}
	return strconv.FormatInt(parsed, 10), nil
}

func normalizeEnumValue(values map[string]string) func(string) (any, error) {
	return func(raw string) (any, error) {
		trimmed := strings.TrimSpace(strings.ToLower(raw))
		if normalized, ok := values[trimmed]; ok {
			return normalized, nil
		}
		return nil, usageError("invalid_user_settings_value", "unsupported value %q", raw)
	}
}

func cloneJSONObject(source map[string]any) map[string]any {
	if len(source) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		switch typed := value.(type) {
		case map[string]any:
			cloned[key] = cloneJSONObject(typed)
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

func cloneUserSettingsDomains(settings map[string]any) map[string]any {
	cloned := map[string]any{}
	for _, domain := range []string{"search", "catalog", "download"} {
		value, ok := settings[domain].(map[string]any)
		if ok {
			cloned[domain] = cloneJSONObject(value)
		}
	}
	return cloned
}

func setNestedJSONObjectValue(target map[string]any, path []string, value any) {
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
	setNestedJSONObjectValue(next, path[1:], value)
}
