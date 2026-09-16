package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/chill-institute/chill-cli/pkg/rpc"
	"github.com/spf13/cobra"
)

func newWhoamiCommand(app *appContext) *cobra.Command {
	var fields string

	command := &cobra.Command{
		Use:   "whoami",
		Short: "Show authenticated user profile",
		Example: strings.TrimSpace(`
chilly whoami
chilly whoami --fields username,email --output json
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWhoami(app, fields)
		},
	}

	command.Flags().StringVar(&fields, "fields", "", "comma-separated field paths to include in the output")
	return command
}

func runWhoami(app *appContext, fields string) error {
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
		procedureUserGetUserProfile,
		map[string]any{},
		rpc.AuthUser,
		token,
	)
	if err != nil {
		return fmt.Errorf("get user profile: %w", err)
	}
	return app.writeSelectedResponseBodyWithRenderer(response.Body, selection, renderWhoamiPretty)
}
