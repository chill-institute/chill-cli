package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/chill-institute/chill-cli/v2/pkg/chill"
	"github.com/chill-institute/chill-cli/v2/pkg/rpc"
	"github.com/spf13/cobra"
)

const (
	addTransferURLDescription         = "magnet link or HTTP(S) URL"
	addTransferURLFlagDescription     = addTransferURLDescription + " to add as transfer"
	addTransferMovieSourceDescription = "optional movie catalog source: imdb-moviemeter|imdb-top-250|yts|rotten-tomatoes|trakt"
	addTransferTVSourceDescription    = "optional TV catalog source: all-providers|netflix|hbo-max|apple-tv-plus|prime-video|disney-plus|hulu|paramount-plus|amc-plus|peacock"
)

func newAddTransferCommand(app *appContext) *cobra.Command {
	var transferURL string
	var rawRequest string
	var movieSource string
	var tvSource string
	var dryRun bool

	command := &cobra.Command{
		Use:   "add-transfer",
		Short: "Add a transfer through chill.institute",
		Example: strings.TrimSpace(`
chilly add-transfer --url "magnet:?xt=urn:btih:..."
chilly add-transfer --url "magnet:?xt=urn:btih:..." --movie-source trakt
printf '{"url":"magnet:?xt=urn:btih:..."}' | chilly add-transfer --json @- --output json
chilly add-transfer --url "magnet:?xt=urn:btih:..." --dry-run --output json
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAddTransfer(app, "add-transfer", transferURL, rawRequest, movieSource, tvSource, dryRun)
		},
	}

	command.Flags().StringVar(&transferURL, "url", "", addTransferURLFlagDescription)
	command.Flags().StringVar(&rawRequest, "json", "", "raw JSON request body, or @- to read it from stdin")
	command.Flags().StringVar(&movieSource, "movie-source", "", addTransferMovieSourceDescription)
	command.Flags().StringVar(&tvSource, "tv-source", "", addTransferTVSourceDescription)
	command.Flags().BoolVar(&dryRun, "dry-run", false, "validate input and print the request without executing it")
	return command
}

func runAddTransfer(app *appContext, commandID string, transferURL string, rawRequest string, movieSource string, tvSource string, dryRun bool) error {
	request, err := buildAddTransferRequest(app, transferURL, rawRequest, movieSource, tvSource)
	if err != nil {
		return err
	}
	if dryRun {
		return app.writeDryRunPreview(commandID, procedureUserAddTransfer, rpc.AuthUser, request)
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
		procedureUserAddTransfer,
		request,
		rpc.AuthUser,
		token,
	)
	if err != nil {
		return fmt.Errorf("add transfer: %w", err)
	}
	return app.writeSelectedResponseBodyWithRenderer(response.Body, nil, renderTransferPretty)
}

func buildAddTransferRequest(app *appContext, transferURL string, rawRequest string, movieSource string, tvSource string) (map[string]any, error) {
	trimmedURL := strings.TrimSpace(transferURL)
	trimmedRequest := strings.TrimSpace(rawRequest)
	trimmedMovieSource := strings.TrimSpace(movieSource)
	trimmedTVSource := strings.TrimSpace(tvSource)

	if trimmedURL != "" && trimmedRequest != "" {
		return nil, usageError("ambiguous_transfer_input", "use either --url or --json, not both")
	}
	if trimmedRequest != "" && (trimmedMovieSource != "" || trimmedTVSource != "") {
		return nil, usageError("ambiguous_transfer_input", "source flags cannot be combined with --json")
	}
	if trimmedMovieSource != "" && trimmedTVSource != "" {
		return nil, usageError("ambiguous_transfer_source", "use either --movie-source or --tv-source, not both")
	}

	if trimmedRequest != "" {
		request, err := app.decodeJSONObjectFlag(rawRequest, "--json")
		if err != nil {
			return nil, err
		}
		urlValue, ok := request["url"].(string)
		if !ok {
			return nil, usageError("invalid_json_payload", "--json payload must include a string url field")
		}
		normalizedURL, err := normalizeTransferURL(urlValue)
		if err != nil {
			return nil, err
		}
		request["url"] = normalizedURL
		return request, nil
	}

	normalizedURL, err := normalizeTransferURL(transferURL)
	if err != nil {
		return nil, err
	}
	request := map[string]any{"url": normalizedURL}
	if trimmedMovieSource != "" {
		source, err := normalizeAddTransferMovieSource(trimmedMovieSource)
		if err != nil {
			return nil, err
		}
		request["catalogOrigin"] = map[string]any{"moviesSource": source}
	}
	if trimmedTVSource != "" {
		source, err := normalizeTVShowsSourceValue(trimmedTVSource, false)
		if err != nil {
			return nil, err
		}
		request["catalogOrigin"] = map[string]any{"tvShowsSource": source}
	}
	return request, nil
}

func normalizeAddTransferMovieSource(raw string) (string, error) {
	value, err := chill.NormalizeMovieSource(raw)
	return value, wrapValidationError(err)
}

func normalizeTransferURL(raw string) (string, error) {
	value, err := chill.NormalizeTransferURL(raw)
	return value, wrapValidationError(err)
}
