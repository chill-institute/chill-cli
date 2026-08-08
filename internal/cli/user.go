package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/chill-institute/chill-cli/internal/rpc"
	"github.com/spf13/cobra"
)

func newUserCommand(app *appContext) *cobra.Command {
	command := &cobra.Command{
		Use:   "user",
		Short: "Run user account commands through chill.institute",
		Example: strings.TrimSpace(`
chilly user profile
chilly user settings get --output json
chilly user download-folder
`),
	}

	var profileFields string
	profileCommand := &cobra.Command{
		Use:   "profile",
		Short: "Show authenticated profile (alias for whoami)",
		Long:  "Alias for the top-level whoami command.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWhoami(app, profileFields)
		},
	}
	profileCommand.Flags().StringVar(&profileFields, "fields", "", "comma-separated field paths to include in the output")
	command.AddCommand(profileCommand)

	var indexerFields string
	indexersCommand := &cobra.Command{
		Use:   "indexers",
		Short: "List user indexers",
		RunE: func(cmd *cobra.Command, args []string) error {
			selection, err := parseFieldSelection(indexerFields)
			if err != nil {
				return err
			}
			return runUserRPCWithRenderer(app, procedureUserGetIndexers, map[string]any{}, selection, renderUserIndexersPretty)
		},
	}
	indexersCommand.Flags().StringVar(&indexerFields, "fields", "", "comma-separated field paths to include in the output")
	command.AddCommand(indexersCommand)

	var query string
	var indexerID string
	var searchFields string
	searchCommand := &cobra.Command{
		Use:   "search",
		Short: "Search using your saved profile settings",
		Long:  "Alias for the top-level search command under the user namespace.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSearch(app, query, indexerID, searchFields)
		},
	}
	searchCommand.Flags().StringVar(&query, "query", "", "search query")
	searchCommand.Flags().StringVar(&indexerID, "indexer-id", "", "optional indexer id")
	searchCommand.Flags().StringVar(&searchFields, "fields", "", "comma-separated field paths to include in the output")
	command.AddCommand(searchCommand)

	command.AddCommand(newUserMoviesCommand(app))
	command.AddCommand(newUserTVShowsCommand(app))

	command.AddCommand(newUserSettingsCommand(app))
	command.AddCommand(newUserTransferCommand(app))
	command.AddCommand(newUserDownloadFolderCommand(app))
	command.AddCommand(newUserFolderCommand(app))

	return command
}

func newUserSettingsCommand(app *appContext) *cobra.Command {
	settingsCommand := &cobra.Command{
		Use:   "settings",
		Short: "Read and update hosted user settings",
	}

	var getFields string
	getCommand := &cobra.Command{
		Use:   "get",
		Short: "Show hosted user settings",
		RunE: func(cmd *cobra.Command, args []string) error {
			selection, err := parseFieldSelection(getFields)
			if err != nil {
				return err
			}
			return runUserRPCWithFields(app, procedureUserGetUserSettings, map[string]any{}, selection)
		},
	}
	getCommand.Flags().StringVar(&getFields, "fields", "", "comma-separated field paths to include in the output")
	settingsCommand.AddCommand(getCommand)

	var rawSettings string
	var dryRun bool
	setCommand := &cobra.Command{
		Use:   "set [field] [value]",
		Short: "Save full user settings JSON payload or patch one setting",
		Long: strings.TrimSpace(`
Save user settings in one of two modes:

- full replacement with --json
- single-field patching with <field> <value>

` + supportedUserSettingsPatchHelp()),
		Example: strings.TrimSpace(`
chilly user settings set filter-nasty-results true
chilly user settings set sort-by title --dry-run --output json
chilly user settings set --json '{"search":{"filterNastyResults":true,"filterResultsWithNoSeeders":false,"rememberQuickFilters":false,"disabledIndexerIds":[],"resolutionFilters":[],"codecFilters":[],"otherFilters":[],"sortBy":"SORT_BY_SEEDERS","sortDirection":"SORT_DIRECTION_DESC","searchResultDisplayBehavior":"SEARCH_RESULT_DISPLAY_BEHAVIOR_FASTEST","searchResultTitleBehavior":"SEARCH_RESULT_TITLE_BEHAVIOR_TEXT"},"catalog":{"moviesSource":"MOVIES_SOURCE_YTS","tvShowsSource":"TV_SHOWS_SOURCE_ALL_PROVIDERS"},"download":{"folderId":42}}'
printf '{"settings":{"search":{"filterNastyResults":true,"filterResultsWithNoSeeders":false,"rememberQuickFilters":false,"disabledIndexerIds":[],"resolutionFilters":[],"codecFilters":[],"otherFilters":[],"sortBy":"SORT_BY_SEEDERS","sortDirection":"SORT_DIRECTION_DESC","searchResultDisplayBehavior":"SEARCH_RESULT_DISPLAY_BEHAVIOR_FASTEST","searchResultTitleBehavior":"SEARCH_RESULT_TITLE_BEHAVIOR_TEXT"},"catalog":{"moviesSource":"MOVIES_SOURCE_YTS","tvShowsSource":"TV_SHOWS_SOURCE_ALL_PROVIDERS"},"download":{"folderId":42}}}' | chilly user settings set --json @- --output json
`),
		Args: allowDescribeArgs(cobra.MaximumNArgs(2)),
		RunE: func(cmd *cobra.Command, args []string) error {
			trimmed := strings.TrimSpace(rawSettings)
			if trimmed != "" {
				if len(args) != 0 {
					return usageError("ambiguous_user_settings_update", "use either --json or <field> <value>, not both")
				}

				request, err := decodeUserSettingsRequest(app, rawSettings)
				if err != nil {
					return err
				}
				if dryRun {
					return app.writeDryRunPreview("user settings set", procedureUserSaveUserSettings, rpc.AuthUser, request)
				}
				return runUserRPC(app, procedureUserSaveUserSettings, request)
			}

			if len(args) != 2 {
				return usageError("missing_user_settings_update", "provide either --json or <field> <value>")
			}

			patch, err := normalizeUserSettingsPatch(args[0], args[1])
			if err != nil {
				return err
			}
			return runUserSettingsPatch(app, "user settings set", patch, dryRun)
		},
	}
	setCommand.Flags().StringVar(&rawSettings, "json", "", "raw JSON request body, bare settings object JSON, or @- to read from stdin")
	setCommand.Flags().BoolVar(&dryRun, "dry-run", false, "validate input and print the request or patch without executing it")
	settingsCommand.AddCommand(setCommand)

	return settingsCommand
}

func newUserTransferCommand(app *appContext) *cobra.Command {
	transferCommand := &cobra.Command{
		Use:   "transfer",
		Short: "Transfer commands",
	}

	var transferURL string
	var rawRequest string
	var movieSource string
	var tvSource string
	var dryRun bool
	addCommand := &cobra.Command{
		Use:   "add",
		Short: "Add a transfer through chill.institute",
		Long:  "Alias for the top-level add-transfer command under the user namespace.",
		Example: strings.TrimSpace(`
chilly user transfer add --url "magnet:?xt=urn:btih:..." --tv-source netflix
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAddTransfer(app, "user transfer add", transferURL, rawRequest, movieSource, tvSource, dryRun)
		},
	}
	addCommand.Flags().StringVar(&transferURL, "url", "", addTransferURLDescription)
	addCommand.Flags().StringVar(&rawRequest, "json", "", "raw JSON request body, or @- to read it from stdin")
	addCommand.Flags().StringVar(&movieSource, "movie-source", "", addTransferMovieSourceDescription)
	addCommand.Flags().StringVar(&tvSource, "tv-source", "", addTransferTVSourceDescription)
	addCommand.Flags().BoolVar(&dryRun, "dry-run", false, "validate input and print the request without executing it")
	transferCommand.AddCommand(addCommand)

	var fields string
	getCommand := &cobra.Command{
		Use:   "get <id>",
		Short: "Show one transfer through chill.institute",
		Long:  "Alias for the top-level get-transfer command under the user namespace.",
		Args:  allowDescribeArgs(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := normalizeTransferID(args[0])
			if err != nil {
				return err
			}
			selection, err := parseFieldSelection(fields)
			if err != nil {
				return err
			}
			return runUserRPCWithRenderer(app, procedureUserGetTransfer, map[string]any{"id": id}, selection, renderTransferPretty)
		},
	}
	getCommand.Flags().StringVar(&fields, "fields", "", "comma-separated field paths to include in the output")
	transferCommand.AddCommand(getCommand)

	return transferCommand
}

func newUserDownloadFolderCommand(app *appContext) *cobra.Command {
	var fields string
	command := &cobra.Command{
		Use:   "download-folder",
		Short: "Show your current download folder",
		Example: strings.TrimSpace(`
chilly user download-folder
chilly user download-folder --fields folder.id,folder.name --output json
chilly user download-folder set 42 --dry-run --output json
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			selection, err := parseFieldSelection(fields)
			if err != nil {
				return err
			}
			return runUserRPCWithRenderer(app, procedureUserGetDownloadFolder, map[string]any{}, selection, renderDownloadFolderPretty)
		},
	}
	command.Flags().StringVar(&fields, "fields", "", "comma-separated field paths to include in the output")

	var dryRun bool
	var rawRequest string
	setCommand := &cobra.Command{
		Use:   "set [id]",
		Short: "Set the current download folder",
		Example: strings.TrimSpace(`
chilly user download-folder set 42
chilly user download-folder set 42 --dry-run --output json
printf '{"download":{"folderId":42}}' | chilly user download-folder set --json @- --dry-run --output json
`),
		Args: allowDescribeArgs(cobra.MaximumNArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(rawRequest) != "" {
				if len(args) != 0 {
					return usageError("ambiguous_download_folder_update", "use either --json or <id>, not both")
				}
				request, err := resolveDownloadFolderSetRequest(app, rawRequest)
				if err != nil {
					return err
				}
				return runUserSettingsPatch(app, "user download-folder set", request, dryRun)
			}
			if len(args) != 1 {
				return usageError("missing_folder_id", "folder id is required")
			}
			id, err := normalizeFolderID(args[0])
			if err != nil {
				return err
			}
			return runUserSettingsPatch(app, "user download-folder set", userSettingsPatch{
				Field: "download.folderId",
				Value: strconv.FormatInt(id, 10),
			}, dryRun)
		},
	}
	setCommand.Flags().StringVar(&rawRequest, "json", "", "raw JSON request body, bare settings object JSON, or @- to read from stdin")
	setCommand.Flags().BoolVar(&dryRun, "dry-run", false, "validate input and print the request or patch without executing it")
	command.AddCommand(setCommand)

	var clearDryRun bool
	var clearRawRequest string
	clearCommand := &cobra.Command{
		Use:   "clear",
		Short: "Clear the current download folder setting",
		Example: strings.TrimSpace(`
chilly user download-folder clear
chilly user download-folder clear --dry-run --output json
printf '{"settings":{"download":{"folderId":null}}}' | chilly user download-folder clear --json @- --dry-run --output json
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(clearRawRequest) != "" {
				request, err := resolveDownloadFolderClearRequest(app, clearRawRequest)
				if err != nil {
					return err
				}
				return runUserSettingsPatch(app, "user download-folder clear", request, clearDryRun)
			}
			return runUserSettingsPatch(app, "user download-folder clear", userSettingsPatch{
				Field: "download.folderId",
				Value: nil,
			}, clearDryRun)
		},
	}
	clearCommand.Flags().StringVar(&clearRawRequest, "json", "", "raw JSON request body, bare settings object JSON, or @- to read from stdin")
	clearCommand.Flags().BoolVar(&clearDryRun, "dry-run", false, "validate input and print the request or patch without executing it")
	command.AddCommand(clearCommand)

	return command
}

func newUserFolderCommand(app *appContext) *cobra.Command {
	command := &cobra.Command{
		Use:   "folder",
		Short: "Inspect folders",
	}

	var fields string
	getCommand := &cobra.Command{
		Use:   "get <id>",
		Short: "Get one folder and its children",
		Example: strings.TrimSpace(`
chilly user folder get 0
chilly user folder get 42 --output json
chilly user folder get 42 --fields parent.name,files.name --output json
`),
		Args: allowDescribeArgs(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := normalizeFolderID(args[0])
			if err != nil {
				return err
			}
			selection, err := parseFieldSelection(fields)
			if err != nil {
				return err
			}
			return runUserRPCWithRenderer(app, procedureUserGetFolder, map[string]any{"id": id}, selection, renderFolderPretty)
		},
	}
	getCommand.Flags().StringVar(&fields, "fields", "", "comma-separated field paths to include in the output")

	command.AddCommand(getCommand)
	return command
}

func runUserRPC(app *appContext, procedure string, body any) error {
	return runUserRPCWithRenderer(app, procedure, body, nil, nil)
}

func decodeUserSettingsRequest(app *appContext, rawSettings string) (map[string]any, error) {
	payload, err := app.decodeJSONObjectFlag(rawSettings, "--json")
	if err != nil {
		return nil, err
	}
	request := payload
	if settings, ok := payload["settings"]; ok {
		settingsObject, ok := settings.(map[string]any)
		if !ok {
			return nil, usageError("invalid_json_payload", "--json payload field settings must be a JSON object")
		}
		normalized, err := normalizeUserSettingsJSONObject(settingsObject, true)
		if err != nil {
			return nil, err
		}
		request["settings"] = normalized
	} else {
		normalized, err := normalizeUserSettingsJSONObject(payload, true)
		if err != nil {
			return nil, err
		}
		request = map[string]any{"settings": normalized}
	}
	return request, nil
}

func resolveDownloadFolderSetRequest(app *appContext, rawRequest string) (userSettingsPatch, error) {
	return resolveDownloadFolderRequest(app, rawRequest, false)
}

func resolveDownloadFolderClearRequest(app *appContext, rawRequest string) (userSettingsPatch, error) {
	patch, err := resolveDownloadFolderRequest(app, rawRequest, true)
	if err != nil {
		return userSettingsPatch{}, err
	}
	if patch.Value != nil {
		return userSettingsPatch{}, usageError("invalid_json_payload", "--json payload for download-folder clear must set download.folderId to null")
	}
	return patch, nil
}

func resolveDownloadFolderRequest(app *appContext, rawRequest string, allowNull bool) (userSettingsPatch, error) {
	settings, err := decodeDownloadFolderSettingsObject(app, rawRequest)
	if err != nil {
		return userSettingsPatch{}, err
	}
	rawValue, ok := downloadFolderJSONValue(settings)
	if !ok {
		return userSettingsPatch{}, usageError("invalid_json_payload", "--json payload must include settings.download.folderId")
	}
	normalizedValue, err := normalizeDownloadFolderJSONValue(rawValue, allowNull)
	if err != nil {
		return userSettingsPatch{}, err
	}
	return userSettingsPatch{
		Field: "download.folderId",
		Value: normalizedValue,
	}, nil
}

func decodeDownloadFolderSettingsObject(app *appContext, rawRequest string) (map[string]any, error) {
	payload, err := app.decodeJSONObjectFlag(rawRequest, "--json")
	if err != nil {
		return nil, err
	}
	if settings, ok := payload["settings"]; ok {
		settingsObject, ok := settings.(map[string]any)
		if !ok {
			return nil, usageError("invalid_json_payload", "--json payload field settings must be a JSON object")
		}
		return normalizeUserSettingsJSONObject(settingsObject, false)
	}
	return normalizeUserSettingsJSONObject(payload, false)
}

func normalizeUserSettingsJSONObject(settings map[string]any, requireAllDomains bool) (map[string]any, error) {
	normalized := cloneJSONObject(settings)
	allowedDomains := map[string]struct{}{
		"catalog":  {},
		"download": {},
		"search":   {},
	}
	if requireAllDomains {
		for field := range normalized {
			if _, ok := allowedDomains[field]; !ok {
				return nil, usageError("invalid_json_payload", "unsupported settings field %q", field)
			}
		}
	}
	if download, ok := normalized["download"].(map[string]any); ok {
		if rawFolderID, ok := download["folderId"]; ok {
			folderID, err := normalizeDownloadFolderJSONValue(rawFolderID, true)
			if err != nil {
				return nil, err
			}
			download["folderId"] = folderID
		}
	}
	if !requireAllDomains {
		return normalized, nil
	}
	for _, domain := range []string{"search", "catalog", "download"} {
		if _, ok := normalized[domain].(map[string]any); !ok {
			return nil, usageError("invalid_json_payload", "--json settings must include object settings.%s", domain)
		}
	}
	return normalized, nil
}

func downloadFolderJSONValue(settings map[string]any) (any, bool) {
	if download, ok := settings["download"].(map[string]any); ok {
		value, found := download["folderId"]
		return value, found
	}
	return nil, false
}

func normalizeDownloadFolderJSONValue(value any, allowNull bool) (any, error) {
	switch typed := value.(type) {
	case nil:
		if !allowNull {
			return nil, usageError("invalid_json_payload", "download.folderId must be an integer")
		}
		return nil, nil
	case string:
		id, err := normalizeFolderID(typed)
		if err != nil {
			return nil, err
		}
		return strconv.FormatInt(id, 10), nil
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) || typed != math.Trunc(typed) || typed < 0 || typed > math.MaxInt64 {
			return nil, usageError("invalid_json_payload", "download.folderId must be a non-negative integer")
		}
		return strconv.FormatInt(int64(typed), 10), nil
	default:
		if allowNull {
			return nil, usageError("invalid_json_payload", "download.folderId must be a non-negative integer or null")
		}
		return nil, usageError("invalid_json_payload", "download.folderId must be a non-negative integer")
	}
}

func loadCurrentUserSettings(app *appContext) (map[string]any, error) {
	cfg, err := app.loadConfig()
	if err != nil {
		return nil, err
	}
	token, err := app.userToken(cfg)
	if err != nil {
		return nil, err
	}

	response, err := app.callRPC(
		context.Background(),
		cfg,
		procedureUserGetUserSettings,
		map[string]any{},
		rpc.AuthUser,
		token,
	)
	if err != nil {
		return nil, fmt.Errorf("load current user settings: %w", err)
	}

	var settings map[string]any
	if err := json.Unmarshal(response.Body, &settings); err != nil {
		return nil, wrapInternalError("user_settings_decode_failed", "decode current user settings", err)
	}
	if settings == nil {
		settings = map[string]any{}
	}
	return settings, nil
}

func runUserSettingsPatch(app *appContext, commandID string, patch userSettingsPatch, dryRun bool) error {
	if dryRun {
		return app.writeDryRunPreview(commandID, procedureUserSaveUserSettings, rpc.AuthUser, map[string]any{
			"patch": patch,
		})
	}

	currentSettings, err := loadCurrentUserSettings(app)
	if err != nil {
		return err
	}
	request := map[string]any{"settings": applyUserSettingsPatch(currentSettings, patch)}
	return runUserRPC(app, procedureUserSaveUserSettings, request)
}

func normalizeFolderID(raw string) (int64, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, usageError("missing_folder_id", "folder id is required")
	}

	value, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return 0, usageError("invalid_folder_id", "folder id must be an integer")
	}
	if value < 0 {
		return 0, usageError("invalid_folder_id", "folder id must be zero or positive")
	}
	return value, nil
}

func normalizeTransferID(raw string) (int64, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, usageError("missing_transfer_id", "transfer id is required")
	}

	value, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return 0, usageError("invalid_transfer_id", "transfer id must be an integer")
	}
	if value <= 0 {
		return 0, usageError("invalid_transfer_id", "transfer id must be positive")
	}
	return value, nil
}

func runUserRPCWithFields(app *appContext, procedure string, body any, selection *fieldSelection) error {
	return runUserRPCWithRenderer(app, procedure, body, selection, nil)
}

func runUserRPCWithRenderer(app *appContext, procedure string, body any, selection *fieldSelection, renderer prettyRenderer) error {
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
		procedure,
		body,
		rpc.AuthUser,
		token,
	)
	if err != nil {
		return fmt.Errorf("user rpc call: %w", err)
	}
	if renderer == nil {
		renderer = prettyRendererForProcedure(procedure)
	}
	return app.writeSelectedResponseBodyWithRenderer(response.Body, selection, renderer)
}

func prettyRendererForProcedure(procedure string) prettyRenderer {
	switch procedure {
	case procedureUserGetIndexers:
		return renderUserIndexersPretty
	case procedureUserGetUserProfile:
		return renderWhoamiPretty
	case procedureUserGetUserSettings:
		return renderUserSettingsPretty
	case procedureUserSearch:
		return renderSearchPretty
	case procedureUserGetMovies:
		return renderMoviesPretty
	case procedureUserGetTVShows:
		return renderTVShowsPretty
	case procedureUserGetTVShowDetail:
		return renderTVShowDetailPretty
	case procedureUserGetTVShowSeason:
		return renderTVShowSeasonPretty
	case procedureUserGetTVShowEpisodeDownload:
		return renderTVShowEpisodeDownloadPretty
	case procedureUserGetTVShowSeasonDownloads:
		return renderTVShowSeasonDownloadsPretty
	case procedureUserAddTransfer, procedureUserGetTransfer:
		return renderTransferPretty
	default:
		return nil
	}
}
