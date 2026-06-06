package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/chill-institute/chill-cli/internal/rpc"
	"github.com/spf13/cobra"
)

const tvShowsSourceFlagDescription = "TV source override: all-providers, netflix, hbo-max, apple-tv-plus, prime-video, disney-plus, hulu, paramount-plus, amc-plus, or peacock"

func newTVShowsCommand(app *appContext) *cobra.Command {
	return newTVShowsReadCommand(
		app,
		"tv-shows",
		"List TV shows from a TV provider source",
		"",
		strings.TrimSpace(`
chilly tv-shows
chilly tv-shows --source all-providers --fields source,shows.title --output json
chilly tv-shows --source hulu --output ndjson
chilly tv-shows --fields shows.title --output json
chilly tv-shows detail tt0944947
chilly tv-shows season tt0944947 1
chilly tv-shows season-downloads tt0944947 1 --output json
`),
	)
}

func newUserTVShowsCommand(app *appContext) *cobra.Command {
	return newTVShowsReadCommand(
		app,
		"tv-shows",
		"List TV shows from a TV provider source",
		"Alias for the top-level tv-shows command.",
		strings.TrimSpace(`
chilly user tv-shows
chilly user tv-shows --source peacock --fields shows.title --output json
chilly user tv-shows detail tt0944947
chilly user tv-shows season-downloads tt0944947 1 --output json
`),
	)
}

func newTVShowsReadCommand(app *appContext, use, short, long, example string) *cobra.Command {
	var fields string
	var source string

	command := &cobra.Command{
		Use:     use,
		Short:   short,
		Long:    long,
		Example: example,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTVShows(app, fields, source)
		},
	}

	command.Flags().StringVar(&fields, "fields", "", "comma-separated field paths to include in the output")
	command.Flags().StringVar(
		&source,
		"source",
		"",
		tvShowsSourceFlagDescription,
	)
	command.AddCommand(newTVShowDetailCommand(app))
	command.AddCommand(newTVShowSeasonCommand(app))
	command.AddCommand(newTVShowEpisodeDownloadCommand(app))
	command.AddCommand(newTVShowSeasonDownloadsCommand(app))
	return command
}

func newTVShowDetailCommand(app *appContext) *cobra.Command {
	var fields string

	command := &cobra.Command{
		Use:   "detail <imdb-id>",
		Short: "Show TV show detail by IMDb id",
		Example: strings.TrimSpace(`
chilly tv-shows detail tt0944947
chilly tv-shows detail tt0944947 --fields show.title,seasons.seasonNumber --output json
`),
		Args: allowDescribeArgs(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			imdbID, err := normalizeIMDbID(args[0])
			if err != nil {
				return err
			}
			selection, err := parseFieldSelection(fields)
			if err != nil {
				return err
			}
			return runUserRPCWithRenderer(
				app,
				procedureUserGetTVShowDetail,
				map[string]any{"imdbId": imdbID},
				selection,
				renderTVShowDetailPretty,
			)
		},
	}

	command.Flags().StringVar(&fields, "fields", "", "comma-separated field paths to include in the output")
	return command
}

func newTVShowSeasonCommand(app *appContext) *cobra.Command {
	var fields string

	command := &cobra.Command{
		Use:   "season <imdb-id> <season-number>",
		Short: "Show one TV show season by IMDb id",
		Example: strings.TrimSpace(`
chilly tv-shows season tt0944947 1
chilly tv-shows season tt0944947 1 --fields episodes.name --output json
`),
		Args: allowDescribeArgs(cobra.ExactArgs(2)),
		RunE: func(cmd *cobra.Command, args []string) error {
			imdbID, err := normalizeIMDbID(args[0])
			if err != nil {
				return err
			}
			seasonNumber, err := normalizeEpisodeOrdinal(args[1], "season")
			if err != nil {
				return err
			}
			selection, err := parseFieldSelection(fields)
			if err != nil {
				return err
			}
			return runUserRPCWithRenderer(
				app,
				procedureUserGetTVShowSeason,
				map[string]any{
					"imdbId":       imdbID,
					"seasonNumber": seasonNumber,
				},
				selection,
				renderTVShowSeasonPretty,
			)
		},
	}

	command.Flags().StringVar(&fields, "fields", "", "comma-separated field paths to include in the output")
	return command
}

func newTVShowEpisodeDownloadCommand(app *appContext) *cobra.Command {
	var fields string

	command := &cobra.Command{
		Use:   "episode-download <imdb-id> <season-number> <episode-number>",
		Short: "Find one TV episode download by IMDb id",
		Example: strings.TrimSpace(`
chilly tv-shows episode-download tt0944947 1 1
chilly tv-shows episode-download tt0944947 1 1 --fields download.title,searchQuery --output json
`),
		Args: allowDescribeArgs(cobra.ExactArgs(3)),
		RunE: func(cmd *cobra.Command, args []string) error {
			imdbID, err := normalizeIMDbID(args[0])
			if err != nil {
				return err
			}
			seasonNumber, err := normalizeEpisodeOrdinal(args[1], "season")
			if err != nil {
				return err
			}
			episodeNumber, err := normalizeEpisodeOrdinal(args[2], "episode")
			if err != nil {
				return err
			}
			selection, err := parseFieldSelection(fields)
			if err != nil {
				return err
			}
			return runUserRPCWithRenderer(
				app,
				procedureUserGetTVShowEpisodeDownload,
				map[string]any{
					"imdbId":        imdbID,
					"seasonNumber":  seasonNumber,
					"episodeNumber": episodeNumber,
				},
				selection,
				renderTVShowEpisodeDownloadPretty,
			)
		},
	}

	command.Flags().StringVar(&fields, "fields", "", "comma-separated field paths to include in the output")
	return command
}

func newTVShowSeasonDownloadsCommand(app *appContext) *cobra.Command {
	var fields string

	command := &cobra.Command{
		Use:   "season-downloads <imdb-id> <season-number>",
		Short: "Find season and episode downloads for one TV season by IMDb id",
		Example: strings.TrimSpace(`
chilly tv-shows season-downloads tt0944947 1
chilly tv-shows season-downloads tt0944947 1 --fields seasonPack.title,episodes.download.title --output json
`),
		Args: allowDescribeArgs(cobra.ExactArgs(2)),
		RunE: func(cmd *cobra.Command, args []string) error {
			imdbID, err := normalizeIMDbID(args[0])
			if err != nil {
				return err
			}
			seasonNumber, err := normalizeEpisodeOrdinal(args[1], "season")
			if err != nil {
				return err
			}
			selection, err := parseFieldSelection(fields)
			if err != nil {
				return err
			}
			return runUserRPCWithRenderer(
				app,
				procedureUserGetTVShowSeasonDownloads,
				map[string]any{
					"imdbId":       imdbID,
					"seasonNumber": seasonNumber,
				},
				selection,
				renderTVShowSeasonDownloadsPretty,
			)
		},
	}

	command.Flags().StringVar(&fields, "fields", "", "comma-separated field paths to include in the output")
	return command
}

func runTVShows(app *appContext, fields string, source string) error {
	selection, err := parseFieldSelection(fields)
	if err != nil {
		return err
	}
	normalizedSource, err := normalizeTVShowsSource(source)
	if err != nil {
		return err
	}

	cfg, err := app.loadConfig()
	if err != nil {
		return err
	}
	token, err := app.userToken(cfg)
	if err != nil {
		return err
	}

	response, err := app.callRPC(
		context.Background(),
		cfg,
		procedureUserGetTVShows,
		tvShowsRequestBody(normalizedSource),
		rpc.AuthUser,
		token,
	)
	if err != nil {
		return fmt.Errorf("list tv shows: %w", err)
	}
	return app.writeSelectedResponseBodyWithRenderer(response.Body, selection, renderTVShowsPretty)
}

func tvShowsRequestBody(source string) map[string]any {
	if source == "" {
		return map[string]any{}
	}
	return map[string]any{"source": source}
}

func normalizeTVShowsSource(raw string) (string, error) {
	return normalizeTVShowsSourceValue(raw, true)
}

func normalizeTVShowsSourcePatchValue(raw string) (any, error) {
	return normalizeTVShowsSourceValue(raw, false)
}

func normalizeTVShowsSourceValue(raw string, allowEmpty bool) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		if allowEmpty {
			return "", nil
		}
		return "", usageError("invalid_tv_shows_source", "TV shows source is required")
	}
	if strings.IndexFunc(trimmed, unicode.IsControl) >= 0 {
		return "", usageError("invalid_tv_shows_source", "TV shows source must not contain control characters")
	}
	if strings.Contains(trimmed, "..") || strings.Contains(trimmed, "%") || strings.ContainsAny(trimmed, "/?#") {
		return "", usageError("invalid_tv_shows_source", "TV shows source must be one of the documented source names")
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
		return "", usageError("invalid_tv_shows_source", "unknown TV shows source %q", raw)
	}
}

func normalizeIMDbID(raw string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return "", usageError("missing_imdb_id", "IMDb id is required")
	}
	if strings.IndexFunc(trimmed, unicode.IsControl) >= 0 {
		return "", usageError("invalid_imdb_id", "IMDb id must not contain control characters")
	}
	if strings.Contains(trimmed, "..") {
		return "", usageError("invalid_imdb_id", "IMDb id must not contain traversal segments")
	}
	if strings.ContainsAny(trimmed, "/?#") {
		return "", usageError("invalid_imdb_id", "IMDb id must not contain path, query, or fragment characters")
	}
	if strings.Contains(trimmed, "%") {
		return "", usageError("invalid_imdb_id", "IMDb id must not contain percent-encoded characters")
	}
	if !strings.HasPrefix(trimmed, "tt") {
		return "", usageError("invalid_imdb_id", "IMDb id must start with tt")
	}
	digits := strings.TrimPrefix(trimmed, "tt")
	if len(digits) < 7 || len(digits) > 12 {
		return "", usageError("invalid_imdb_id", "IMDb id must include 7 to 12 digits")
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return "", usageError("invalid_imdb_id", "IMDb id must contain only digits after tt")
		}
	}
	return trimmed, nil
}

func normalizeEpisodeOrdinal(raw string, kind string) (int32, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, usageError("missing_"+kind+"_number", "%s number is required", kind)
	}

	value, err := strconv.ParseInt(trimmed, 10, 32)
	if err != nil {
		return 0, usageError("invalid_"+kind+"_number", "%s number must be an integer", kind)
	}
	if value <= 0 {
		return 0, usageError("invalid_"+kind+"_number", "%s number must be positive", kind)
	}
	return int32(value), nil
}
