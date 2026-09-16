package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/chill-institute/chill-cli/v2/pkg/chill"
	"github.com/chill-institute/chill-cli/v2/pkg/rpc"
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
	value, err := chill.NormalizeTVShowsSource(raw, allowEmpty)
	return value, wrapValidationError(err)
}

func normalizeIMDbID(raw string) (string, error) {
	value, err := chill.NormalizeIMDbID(raw)
	return value, wrapValidationError(err)
}

func normalizeEpisodeOrdinal(raw string, kind string) (int32, error) {
	value, err := chill.NormalizeEpisodeOrdinal(raw, kind)
	return value, wrapValidationError(err)
}
