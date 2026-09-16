package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/chill-institute/chill-cli/v2/pkg/rpc"
	"github.com/spf13/cobra"
)

func newMoviesCommand(app *appContext) *cobra.Command {
	return newMoviesReadCommand(
		app,
		"movies",
		"List movies using your profile settings",
		"",
		strings.TrimSpace(`
chilly movies
chilly movies --fields movies.title --output json
`),
	)
}

func newUserMoviesCommand(app *appContext) *cobra.Command {
	return newMoviesReadCommand(
		app,
		"movies",
		"List movies using your profile settings",
		"Alias for the top-level movies command.",
		strings.TrimSpace(`
chilly user movies
chilly user movies --fields movies.title --output json
`),
	)
}

func newMoviesReadCommand(app *appContext, use, short, long, example string) *cobra.Command {
	var fields string

	command := &cobra.Command{
		Use:     use,
		Short:   short,
		Long:    long,
		Example: example,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMovies(app, fields)
		},
	}

	command.Flags().StringVar(&fields, "fields", "", "comma-separated field paths to include in the output")
	return command
}

func runMovies(app *appContext, fields string) error {
	selection, err := parseFieldSelection(fields)
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
		procedureUserGetMovies,
		map[string]any{},
		rpc.AuthUser,
		token,
	)
	if err != nil {
		return fmt.Errorf("list movies: %w", err)
	}
	return app.writeSelectedResponseBodyWithRenderer(response.Body, selection, renderMoviesPretty)
}
