package chill

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode"
)

// ValidationError reports rejected local input. Code is a stable snake_case
// identifier; Message is safe to show to users.
type ValidationError struct {
	Code    string
	Message string
}

func (err *ValidationError) Error() string {
	return err.Message
}

func invalid(code string, format string, args ...any) error {
	return &ValidationError{Code: code, Message: fmt.Sprintf(format, args...)}
}

// NormalizeIndexerID validates an opaque search indexer ID.
func NormalizeIndexerID(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", invalid("missing_indexer_id", "indexer id cannot be empty")
	}
	if strings.IndexFunc(trimmed, unicode.IsControl) >= 0 {
		return "", invalid("invalid_indexer_id", "indexer id must not contain control characters")
	}
	if strings.Contains(trimmed, "..") {
		return "", invalid("invalid_indexer_id", "indexer id must not contain traversal segments")
	}
	if strings.ContainsAny(trimmed, `/\?#`) {
		return "", invalid("invalid_indexer_id", "indexer id must not contain path, query, or fragment characters")
	}
	if strings.Contains(trimmed, "%") {
		return "", invalid("invalid_indexer_id", "indexer id must not contain percent-encoded characters")
	}
	return trimmed, nil
}

// NormalizeTransferURL accepts a magnet link or an absolute http(s) URL.
func NormalizeTransferURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", invalid("missing_url", "url is required")
	}
	if strings.IndexFunc(trimmed, unicode.IsControl) >= 0 {
		return "", invalid("invalid_url", "url must not contain control characters")
	}
	if strings.IndexFunc(trimmed, unicode.IsSpace) >= 0 {
		return "", invalid("invalid_url", "url must not contain unescaped whitespace")
	}
	if strings.HasPrefix(strings.ToLower(trimmed), "magnet:?") {
		return trimmed, nil
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", invalid("invalid_url", "parse url: %v", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", invalid("invalid_url", "url must be a magnet link or start with http:// or https://")
	}
	if parsed.Hostname() == "" {
		return "", invalid("invalid_url", "url must include a host")
	}
	return trimmed, nil
}

// NormalizeMovieSource maps a documented movie source alias to its enum value.
func NormalizeMovieSource(raw string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.ReplaceAll(normalized, " ", "-")
	values := map[string]string{
		"imdb-moviemeter": "MOVIES_SOURCE_IMDB_MOVIEMETER",
		"imdb/moviemeter": "MOVIES_SOURCE_IMDB_MOVIEMETER",
		"imdb-top-250":    "MOVIES_SOURCE_IMDB_TOP_250",
		"imdb/top-250":    "MOVIES_SOURCE_IMDB_TOP_250",
		"yts":             "MOVIES_SOURCE_YTS",
		"rotten-tomatoes": "MOVIES_SOURCE_ROTTEN_TOMATOES",
		"rottentomatoes":  "MOVIES_SOURCE_ROTTEN_TOMATOES",
		"trakt":           "MOVIES_SOURCE_TRAKT",
	}
	if source, ok := values[normalized]; ok {
		return source, nil
	}
	return "", invalid("invalid_movies_source", "movie source must be one of the documented source names")
}

// NormalizeTVShowsSource maps a documented TV provider alias to its enum value.
// With allowEmpty, blank input returns "" so callers can omit the field.
func NormalizeTVShowsSource(raw string, allowEmpty bool) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		if allowEmpty {
			return "", nil
		}
		return "", invalid("invalid_tv_shows_source", "TV shows source is required")
	}
	if strings.IndexFunc(trimmed, unicode.IsControl) >= 0 {
		return "", invalid("invalid_tv_shows_source", "TV shows source must not contain control characters")
	}
	if strings.Contains(trimmed, "..") || strings.Contains(trimmed, "%") || strings.ContainsAny(trimmed, "/?#") {
		return "", invalid("invalid_tv_shows_source", "TV shows source must be one of the documented source names")
	}

	switch strings.ToUpper(trimmed) {
	case "TV_SHOWS_SOURCE_ALL_PROVIDERS",
		"TV_SHOWS_SOURCE_NETFLIX",
		"TV_SHOWS_SOURCE_HBO_MAX",
		"TV_SHOWS_SOURCE_APPLE_TV_PLUS",
		"TV_SHOWS_SOURCE_PRIME_VIDEO",
		"TV_SHOWS_SOURCE_DISNEY_PLUS",
		"TV_SHOWS_SOURCE_HULU",
		"TV_SHOWS_SOURCE_PARAMOUNT_PLUS",
		"TV_SHOWS_SOURCE_AMC_PLUS",
		"TV_SHOWS_SOURCE_PEACOCK":
		return strings.ToUpper(trimmed), nil
	}

	normalized := strings.ToLower(trimmed)
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.ReplaceAll(normalized, " ", "-")
	switch normalized {
	case "all", "all-providers", "all-provider", "providers":
		return "TV_SHOWS_SOURCE_ALL_PROVIDERS", nil
	case "netflix":
		return "TV_SHOWS_SOURCE_NETFLIX", nil
	case "hbo-max", "hbomax":
		return "TV_SHOWS_SOURCE_HBO_MAX", nil
	case "apple-tv-plus", "apple-tv", "appletv-plus", "appletv":
		return "TV_SHOWS_SOURCE_APPLE_TV_PLUS", nil
	case "prime-video", "prime", "amazon-prime", "amazon-prime-video":
		return "TV_SHOWS_SOURCE_PRIME_VIDEO", nil
	case "disney-plus", "disney":
		return "TV_SHOWS_SOURCE_DISNEY_PLUS", nil
	case "hulu":
		return "TV_SHOWS_SOURCE_HULU", nil
	case "paramount-plus", "paramount":
		return "TV_SHOWS_SOURCE_PARAMOUNT_PLUS", nil
	case "amc-plus", "amc":
		return "TV_SHOWS_SOURCE_AMC_PLUS", nil
	case "peacock":
		return "TV_SHOWS_SOURCE_PEACOCK", nil
	default:
		return "", invalid("invalid_tv_shows_source", "unknown TV shows source %q", raw)
	}
}

// NormalizeIMDbID validates a lowercase tt-prefixed IMDb title ID.
func NormalizeIMDbID(raw string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return "", invalid("missing_imdb_id", "IMDb id is required")
	}
	if strings.IndexFunc(trimmed, unicode.IsControl) >= 0 {
		return "", invalid("invalid_imdb_id", "IMDb id must not contain control characters")
	}
	if strings.Contains(trimmed, "..") {
		return "", invalid("invalid_imdb_id", "IMDb id must not contain traversal segments")
	}
	if strings.ContainsAny(trimmed, "/?#") {
		return "", invalid("invalid_imdb_id", "IMDb id must not contain path, query, or fragment characters")
	}
	if strings.Contains(trimmed, "%") {
		return "", invalid("invalid_imdb_id", "IMDb id must not contain percent-encoded characters")
	}
	if !strings.HasPrefix(trimmed, "tt") {
		return "", invalid("invalid_imdb_id", "IMDb id must start with tt")
	}
	digits := strings.TrimPrefix(trimmed, "tt")
	if len(digits) < 7 || len(digits) > 12 {
		return "", invalid("invalid_imdb_id", "IMDb id must include 7 to 12 digits")
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return "", invalid("invalid_imdb_id", "IMDb id must contain only digits after tt")
		}
	}
	return trimmed, nil
}

// NormalizeEpisodeOrdinal parses a positive season or episode number. kind
// names the field in codes and messages, for example "season".
func NormalizeEpisodeOrdinal(raw string, kind string) (int32, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, invalid("missing_"+kind+"_number", "%s number is required", kind)
	}

	value, err := strconv.ParseInt(trimmed, 10, 32)
	if err != nil {
		return 0, invalid("invalid_"+kind+"_number", "%s number must be an integer", kind)
	}
	if value <= 0 {
		return 0, invalid("invalid_"+kind+"_number", "%s number must be positive", kind)
	}
	return int32(value), nil
}
