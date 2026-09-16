package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/chill-institute/chill-cli/v2/pkg/chill"
	"github.com/chill-institute/chill-cli/v2/pkg/rpc"
	"github.com/spf13/cobra"
)

func newSearchCommand(app *appContext) *cobra.Command {
	var query string
	var indexerID string
	var fields string

	command := &cobra.Command{
		Use:   "search",
		Short: "Search using your saved profile settings",
		Example: strings.TrimSpace(`
chilly search --query "dune"
chilly search --query "dune" --indexer-id yts --output json
chilly search --query "dune" --fields results.title --output json
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearch(app, query, indexerID, fields)
		},
	}

	command.Flags().StringVar(&query, "query", "", "search query")
	command.Flags().StringVar(&indexerID, "indexer-id", "", "optional indexer id; prefer one indexer at a time for agent workflows")
	command.Flags().StringVar(&fields, "fields", "", "comma-separated field paths to include in the output")
	return command
}

func runSearch(app *appContext, query string, indexerID string, fields string) error {
	trimmedQuery := strings.TrimSpace(query)
	if trimmedQuery == "" {
		return usageError("missing_query", "--query is required")
	}
	selection, err := parseFieldSelection(fields)
	if err != nil {
		return err
	}
	normalizedIndexerID := ""
	if trimmedIndexer := strings.TrimSpace(indexerID); trimmedIndexer != "" {
		normalizedIndexerID, err = normalizeIndexerID(trimmedIndexer)
		if err != nil {
			return err
		}
	}

	cfg, err := app.loadConfig()
	if err != nil {
		return err
	}
	token, err := app.userToken(cfg)
	if err != nil {
		return err
	}

	payload := map[string]any{"query": trimmedQuery}
	if normalizedIndexerID != "" {
		payload["indexer_id"] = normalizedIndexerID
	}

	response, err := app.callRPC(
		context.Background(),
		cfg,
		procedureUserSearch,
		payload,
		rpc.AuthUser,
		token,
	)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}
	return app.writeSelectedResponseBodyWithRenderer(response.Body, selection, renderSearchPretty)
}

func normalizeIndexerID(raw string) (string, error) {
	value, err := chill.NormalizeIndexerID(raw)
	return value, wrapValidationError(err)
}
