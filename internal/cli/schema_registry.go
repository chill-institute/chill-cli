package cli

import (
	"sort"
	"strings"
)

type schemaInput struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

type schemaInputMode struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Inputs      []string `json:"inputs"`
	Required    []string `json:"required"`
}

type schemaOutput struct {
	JSON  bool   `json:"json"`
	Human bool   `json:"human"`
	Type  string `json:"type,omitempty"`
}

type schemaEntry struct {
	ID              string            `json:"id"`
	Kind            string            `json:"kind"`
	Summary         string            `json:"summary"`
	AliasFor        string            `json:"alias_for,omitempty"`
	AuthMode        string            `json:"auth_mode"`
	Mutates         bool              `json:"mutates"`
	SupportsDryRun  bool              `json:"supports_dry_run"`
	SupportsFields  bool              `json:"supports_fields"`
	LinkedProcedure string            `json:"linked_procedure,omitempty"`
	Inputs          []schemaInput     `json:"inputs,omitempty"`
	InputModes      []schemaInputMode `json:"input_modes,omitempty"`
	Output          schemaOutput      `json:"output"`
}

type schemaField struct {
	Name        string `json:"name"`
	JSONName    string `json:"json_name"`
	Type        string `json:"type"`
	Repeated    bool   `json:"repeated,omitempty"`
	Optional    bool   `json:"optional,omitempty"`
	Description string `json:"description,omitempty"`
}

type schemaType struct {
	ID      string        `json:"id"`
	Kind    string        `json:"kind"`
	Summary string        `json:"summary"`
	Fields  []schemaField `json:"fields"`
}

const (
	typeSearchResponse = "chill.v4.SearchResponse"
	typeSearchResult   = "chill.v4.SearchResult"
	typeReleaseInfo    = "chill.v4.ReleaseInfo"
)

var commonCommandInputs = []schemaInput{
	{Name: "api-url", Type: "string", Description: "override API base URL"},
	{Name: "config", Type: "string", Description: "config file path"},
	{Name: "profile", Type: "string", Description: "config profile to use"},
	{Name: "output", Type: "string", Description: "output mode: pretty|json|ndjson"},
	{Name: "describe", Type: "boolean", Description: "print command metadata and exit"},
}

var commandSchemaRegistry = map[string]schemaEntry{
	"chilly": {
		ID:       "chilly",
		Kind:     "command",
		Summary:  "chill.institute CLI for humans and agents",
		AuthMode: string(rpcAuthNone),
		Output:   schemaOutput{JSON: true, Human: true},
		Inputs:   cloneInputs(commonCommandInputs),
	},
	"add-transfer": {
		ID:              "add-transfer",
		Kind:            "command",
		Summary:         "Add a transfer through chill.institute",
		AuthMode:        string(rpcAuthUser),
		Mutates:         true,
		SupportsDryRun:  true,
		LinkedProcedure: procedureUserAddTransfer,
		Output:          schemaOutput{JSON: true, Human: true, Type: "chill.v4.AddTransferResponse"},
		Inputs: appendInputs(
			schemaInput{Name: "url", Type: "string", Description: addTransferURLFlagDescription + "; use either --url or --json"},
			schemaInput{Name: "json", Type: "string", Description: "raw JSON request body, or @- to read it from stdin"},
			schemaInput{Name: "dry-run", Type: "boolean", Description: "validate input and print the request without executing it"},
		),
		InputModes: addTransferInputModes(),
	},
	"get-transfer": {
		ID:              "get-transfer",
		Kind:            "command",
		Summary:         "Show one transfer through chill.institute",
		AuthMode:        string(rpcAuthUser),
		SupportsFields:  true,
		LinkedProcedure: procedureUserGetTransfer,
		Output:          schemaOutput{JSON: true, Human: true, Type: "chill.v4.GetTransferResponse"},
		Inputs: appendInputs(
			schemaInput{Name: "id", Type: "integer", Required: true, Description: "transfer id"},
			schemaInput{Name: "fields", Type: "string", Description: "comma-separated field paths to include in the output"},
		),
	},
	"auth": {
		ID:       "auth",
		Kind:     "command",
		Summary:  "Sign in and manage local auth tokens",
		AuthMode: string(rpcAuthNone),
		Output:   schemaOutput{JSON: true, Human: true},
		Inputs:   cloneInputs(commonCommandInputs),
	},
	"auth login": {
		ID:             "auth login",
		Kind:           "command",
		Summary:        "Sign in through the hosted web token flow or store a setup token",
		AuthMode:       string(rpcAuthNone),
		Mutates:        true,
		SupportsDryRun: true,
		Output:         schemaOutput{JSON: true, Human: true},
		Inputs: appendInputs(
			schemaInput{Name: "token", Type: "string", Description: "setup token to store directly"},
			schemaInput{Name: "json", Type: "string", Description: "raw JSON request body, or @- to read it from stdin"},
			schemaInput{Name: "no-browser", Type: "boolean", Description: "with --local-browser, print the localhost callback URL instead of opening a browser automatically"},
			schemaInput{Name: "local-browser", Type: "boolean", Description: "use the localhost callback flow instead of the hosted web token flow"},
			schemaInput{Name: "skip-verify", Type: "boolean", Description: "skip token verification call"},
			schemaInput{Name: "dry-run", Type: "boolean", Description: "preview the auth action without verifying or saving credentials"},
		),
	},
	"auth logout": {
		ID:             "auth logout",
		Kind:           "command",
		Summary:        "Clear the stored auth token from local config",
		AuthMode:       string(rpcAuthNone),
		Mutates:        true,
		SupportsDryRun: true,
		Output:         schemaOutput{JSON: true, Human: true},
		Inputs: appendInputs(
			schemaInput{Name: "json", Type: "string", Description: "raw JSON request body, or @- to read it from stdin"},
			schemaInput{Name: "dry-run", Type: "boolean", Description: "validate input and print the local config change without saving it"},
		),
	},
	"doctor": {
		ID:             "doctor",
		Kind:           "command",
		Summary:        "Inspect CLI health, config, and auth status",
		AuthMode:       string(rpcAuthNone),
		SupportsFields: true,
		Output:         schemaOutput{JSON: true, Human: true},
		Inputs: appendInputs(
			schemaInput{Name: "offline", Type: "boolean", Description: "skip auth verification and only inspect local state"},
			schemaInput{Name: "fields", Type: "string", Description: "comma-separated field paths to include in the output"},
		),
	},
	"completion": {
		ID:       "completion",
		Kind:     "command",
		Summary:  "Generate shell completion scripts",
		AuthMode: string(rpcAuthNone),
		Output:   schemaOutput{JSON: false, Human: true},
		Inputs: appendInputs(
			schemaInput{Name: "shell", Type: "string", Required: true, Description: "shell name: bash, zsh, fish, or powershell"},
		),
	},
	"movies": {
		ID:              "movies",
		Kind:            "command",
		Summary:         "List movies using your profile settings",
		AuthMode:        string(rpcAuthUser),
		SupportsFields:  true,
		LinkedProcedure: procedureUserGetMovies,
		Output:          schemaOutput{JSON: true, Human: true, Type: "chill.v4.GetMoviesResponse"},
		Inputs: appendInputs(
			schemaInput{Name: "fields", Type: "string", Description: "comma-separated field paths to include in the output"},
		),
	},
	"tv-shows": {
		ID:              "tv-shows",
		Kind:            "command",
		Summary:         "List TV shows from a TV provider source",
		AuthMode:        string(rpcAuthUser),
		SupportsFields:  true,
		LinkedProcedure: procedureUserGetTVShows,
		Output:          schemaOutput{JSON: true, Human: true, Type: "chill.v4.GetTVShowsResponse"},
		Inputs: appendInputs(
			schemaInput{Name: "source", Type: "string", Description: tvShowsSourceFlagDescription},
			schemaInput{Name: "fields", Type: "string", Description: "comma-separated field paths to include in the output"},
		),
	},
	"tv-shows detail": {
		ID:              "tv-shows detail",
		Kind:            "command",
		Summary:         "Show TV show detail by IMDb id",
		AuthMode:        string(rpcAuthUser),
		SupportsFields:  true,
		LinkedProcedure: procedureUserGetTVShowDetail,
		Output:          schemaOutput{JSON: true, Human: true, Type: "chill.v4.GetTVShowDetailResponse"},
		Inputs: appendInputs(
			schemaInput{Name: "imdb-id", Type: "string", Required: true, Description: "IMDb id such as tt0944947"},
			schemaInput{Name: "fields", Type: "string", Description: "comma-separated field paths to include in the output"},
		),
	},
	"tv-shows season": {
		ID:              "tv-shows season",
		Kind:            "command",
		Summary:         "Show one TV show season by IMDb id",
		AuthMode:        string(rpcAuthUser),
		SupportsFields:  true,
		LinkedProcedure: procedureUserGetTVShowSeason,
		Output:          schemaOutput{JSON: true, Human: true, Type: "chill.v4.GetTVShowSeasonResponse"},
		Inputs: appendInputs(
			schemaInput{Name: "imdb-id", Type: "string", Required: true, Description: "IMDb id such as tt0944947"},
			schemaInput{Name: "season-number", Type: "integer", Required: true, Description: "season number"},
			schemaInput{Name: "fields", Type: "string", Description: "comma-separated field paths to include in the output"},
		),
	},
	"tv-shows episode-download": {
		ID:              "tv-shows episode-download",
		Kind:            "command",
		Summary:         "Find one TV episode download by IMDb id",
		AuthMode:        string(rpcAuthUser),
		SupportsFields:  true,
		LinkedProcedure: procedureUserGetTVShowEpisodeDownload,
		Output:          schemaOutput{JSON: true, Human: true, Type: "chill.v4.GetTVShowEpisodeDownloadResponse"},
		Inputs: appendInputs(
			schemaInput{Name: "imdb-id", Type: "string", Required: true, Description: "IMDb id such as tt0944947"},
			schemaInput{Name: "season-number", Type: "integer", Required: true, Description: "season number"},
			schemaInput{Name: "episode-number", Type: "integer", Required: true, Description: "episode number"},
			schemaInput{Name: "fields", Type: "string", Description: "comma-separated field paths to include in the output"},
		),
	},
	"tv-shows season-downloads": {
		ID:              "tv-shows season-downloads",
		Kind:            "command",
		Summary:         "Find season and episode downloads for one TV season by IMDb id",
		AuthMode:        string(rpcAuthUser),
		SupportsFields:  true,
		LinkedProcedure: procedureUserGetTVShowSeasonDownloads,
		Output:          schemaOutput{JSON: true, Human: true, Type: "chill.v4.GetTVShowSeasonDownloadsResponse"},
		Inputs: appendInputs(
			schemaInput{Name: "imdb-id", Type: "string", Required: true, Description: "IMDb id such as tt0944947"},
			schemaInput{Name: "season-number", Type: "integer", Required: true, Description: "season number"},
			schemaInput{Name: "fields", Type: "string", Description: "comma-separated field paths to include in the output"},
		),
	},
	"schema": {
		ID:             "schema",
		Kind:           "command",
		Summary:        "Inspect command and procedure contracts",
		AuthMode:       string(rpcAuthNone),
		SupportsFields: true,
		Output:         schemaOutput{JSON: true, Human: true},
		Inputs: appendInputs(
			schemaInput{Name: "fields", Type: "string", Description: "comma-separated field paths to include in the output"},
		),
	},
	"schema command": {
		ID:             "schema command",
		Kind:           "command",
		Summary:        "Show metadata for one CLI command",
		AuthMode:       string(rpcAuthNone),
		SupportsFields: true,
		Output:         schemaOutput{JSON: true, Human: true},
		Inputs: appendInputs(
			schemaInput{Name: "name", Type: "string", Required: true, Description: "command id such as search or settings get"},
			schemaInput{Name: "fields", Type: "string", Description: "comma-separated field paths to include in the output"},
		),
	},
	"schema procedure": {
		ID:             "schema procedure",
		Kind:           "command",
		Summary:        "Show metadata for one backend procedure",
		AuthMode:       string(rpcAuthNone),
		SupportsFields: true,
		Output:         schemaOutput{JSON: true, Human: true},
		Inputs: appendInputs(
			schemaInput{Name: "procedure", Type: "string", Required: true, Description: "procedure id such as chill.v4.UserService/Search"},
			schemaInput{Name: "fields", Type: "string", Description: "comma-separated field paths to include in the output"},
		),
	},
	"schema type": {
		ID:             "schema type",
		Kind:           "command",
		Summary:        "Show metadata for one output type",
		AuthMode:       string(rpcAuthNone),
		SupportsFields: true,
		Output:         schemaOutput{JSON: true, Human: true, Type: "schema.Type"},
		Inputs: appendInputs(
			schemaInput{Name: "name", Type: "string", Required: true, Description: "type id such as chill.v4.ReleaseInfo"},
			schemaInput{Name: "fields", Type: "string", Description: "comma-separated field paths to include in the output"},
		),
	},
	"search": {
		ID:              "search",
		Kind:            "command",
		Summary:         "Search using your saved profile settings",
		AuthMode:        string(rpcAuthUser),
		SupportsFields:  true,
		LinkedProcedure: procedureUserSearch,
		Output:          schemaOutput{JSON: true, Human: true, Type: typeSearchResponse},
		Inputs: appendInputs(
			schemaInput{Name: "query", Type: "string", Required: true, Description: "search query"},
			schemaInput{Name: "indexer-id", Type: "string", Description: "optional indexer id; prefer one indexer at a time for agent workflows"},
			schemaInput{Name: "fields", Type: "string", Description: "comma-separated field paths to include in the output"},
		),
	},
	"self-update": {
		ID:             "self-update",
		Kind:           "command",
		Summary:        "Check for or install released CLI updates",
		AuthMode:       string(rpcAuthNone),
		Mutates:        true,
		SupportsDryRun: true,
		Output:         schemaOutput{JSON: true, Human: true},
		Inputs: appendInputs(
			schemaInput{Name: "check", Type: "boolean", Description: "check for a newer release without installing it"},
			schemaInput{Name: "version", Type: "string", Description: "specific release tag to install"},
			schemaInput{Name: "json", Type: "string", Description: "raw JSON request body, or @- to read it from stdin"},
			schemaInput{Name: "dry-run", Type: "boolean", Description: "preview update resolution without replacing the current executable"},
		),
	},
	"settings": {
		ID:       "settings",
		Kind:     "command",
		Summary:  "Manage local CLI config",
		AuthMode: string(rpcAuthNone),
		Output:   schemaOutput{JSON: true, Human: true},
		Inputs:   cloneInputs(commonCommandInputs),
	},
	"settings get": {
		ID:             "settings get",
		Kind:           "command",
		Summary:        "Show one local CLI setting",
		AuthMode:       string(rpcAuthNone),
		SupportsFields: true,
		Output:         schemaOutput{JSON: true, Human: true},
		Inputs: appendInputs(
			schemaInput{Name: "key", Type: "string", Required: true, Description: "settings key such as api-base-url"},
			schemaInput{Name: "fields", Type: "string", Description: "comma-separated field paths to include in the output"},
		),
	},
	"settings path": {
		ID:             "settings path",
		Kind:           "command",
		Summary:        "Show local config file path",
		AuthMode:       string(rpcAuthNone),
		SupportsFields: true,
		Output:         schemaOutput{JSON: true, Human: true},
		Inputs: appendInputs(
			schemaInput{Name: "fields", Type: "string", Description: "comma-separated field paths to include in the output"},
		),
	},
	"settings set": {
		ID:             "settings set",
		Kind:           "command",
		Summary:        "Set one local CLI setting",
		AuthMode:       string(rpcAuthNone),
		Mutates:        true,
		SupportsDryRun: true,
		Output:         schemaOutput{JSON: true, Human: true},
		Inputs: appendInputs(
			schemaInput{Name: "key", Type: "string", Description: "settings key such as api-base-url; use with value or use --json"},
			schemaInput{Name: "value", Type: "string", Description: "next value for the selected settings key; use with key or use --json"},
			schemaInput{Name: "json", Type: "string", Description: "raw JSON request body, or @- to read it from stdin"},
			schemaInput{Name: "dry-run", Type: "boolean", Description: "validate input and print the local config change without saving it"},
		),
		InputModes: settingsSetInputModes(),
	},
	"settings show": {
		ID:             "settings show",
		Kind:           "command",
		Summary:        "Show local CLI config (auth token redacted)",
		AuthMode:       string(rpcAuthNone),
		SupportsFields: true,
		Output:         schemaOutput{JSON: true, Human: true},
		Inputs: appendInputs(
			schemaInput{Name: "fields", Type: "string", Description: "comma-separated field paths to include in the output"},
		),
	},
	"user": {
		ID:       "user",
		Kind:     "command",
		Summary:  "Run user account commands through chill.institute",
		AuthMode: string(rpcAuthUser),
		Output:   schemaOutput{JSON: true, Human: true},
		Inputs:   cloneInputs(commonCommandInputs),
	},
	"user indexers": {
		ID:              "user indexers",
		Kind:            "command",
		Summary:         "List user indexers",
		AuthMode:        string(rpcAuthUser),
		SupportsFields:  true,
		LinkedProcedure: procedureUserGetIndexers,
		Output:          schemaOutput{JSON: true, Human: true, Type: "chill.v4.UserGetIndexersResponse"},
		Inputs: appendInputs(
			schemaInput{Name: "fields", Type: "string", Description: "comma-separated field paths to include in the output"},
		),
	},
	"user download-folder": {
		ID:              "user download-folder",
		Kind:            "command",
		Summary:         "Show your current download folder",
		AuthMode:        string(rpcAuthUser),
		SupportsFields:  true,
		LinkedProcedure: procedureUserGetDownloadFolder,
		Output:          schemaOutput{JSON: true, Human: true, Type: "chill.v4.GetDownloadFolderResponse"},
		Inputs: appendInputs(
			schemaInput{Name: "fields", Type: "string", Description: "comma-separated field paths to include in the output"},
		),
	},
	"user download-folder set": {
		ID:              "user download-folder set",
		Kind:            "command",
		Summary:         "Set the current download folder",
		AuthMode:        string(rpcAuthUser),
		Mutates:         true,
		SupportsDryRun:  true,
		LinkedProcedure: procedureUserSaveUserSettings,
		Output:          schemaOutput{JSON: true, Human: true, Type: "chill.v4.UserSettings"},
		Inputs: appendInputs(
			schemaInput{Name: "id", Type: "integer", Required: true, Description: "folder id"},
			schemaInput{Name: "json", Type: "string", Description: "raw JSON request body, bare settings object JSON, or @- to read from stdin"},
			schemaInput{Name: "dry-run", Type: "boolean", Description: "validate input and print the request or patch without executing it"},
		),
	},
	"user download-folder clear": {
		ID:              "user download-folder clear",
		Kind:            "command",
		Summary:         "Clear the current download folder setting",
		AuthMode:        string(rpcAuthUser),
		Mutates:         true,
		SupportsDryRun:  true,
		LinkedProcedure: procedureUserSaveUserSettings,
		Output:          schemaOutput{JSON: true, Human: true, Type: "chill.v4.UserSettings"},
		Inputs: appendInputs(
			schemaInput{Name: "json", Type: "string", Description: "raw JSON request body, bare settings object JSON, or @- to read from stdin"},
			schemaInput{Name: "dry-run", Type: "boolean", Description: "validate input and print the request or patch without executing it"},
		),
	},
	"user folder": {
		ID:       "user folder",
		Kind:     "command",
		Summary:  "Inspect folders",
		AuthMode: string(rpcAuthUser),
		Output:   schemaOutput{JSON: true, Human: true},
		Inputs:   cloneInputs(commonCommandInputs),
	},
	"user folder get": {
		ID:              "user folder get",
		Kind:            "command",
		Summary:         "Get one folder and its children",
		AuthMode:        string(rpcAuthUser),
		SupportsFields:  true,
		LinkedProcedure: procedureUserGetFolder,
		Output:          schemaOutput{JSON: true, Human: true, Type: "chill.v4.GetFolderResponse"},
		Inputs: appendInputs(
			schemaInput{Name: "id", Type: "integer", Required: true, Description: "folder id"},
			schemaInput{Name: "fields", Type: "string", Description: "comma-separated field paths to include in the output"},
		),
	},
	"user profile": {
		ID:              "user profile",
		Kind:            "command",
		Summary:         "Show authenticated profile (alias for whoami)",
		AliasFor:        "whoami",
		AuthMode:        string(rpcAuthUser),
		SupportsFields:  true,
		LinkedProcedure: procedureUserGetUserProfile,
		Output:          schemaOutput{JSON: true, Human: true, Type: "chill.v4.UserProfile"},
		Inputs: appendInputs(
			schemaInput{Name: "fields", Type: "string", Description: "comma-separated field paths to include in the output"},
		),
	},
	"user search": {
		ID:              "user search",
		Kind:            "command",
		Summary:         "Search using your saved profile settings",
		AliasFor:        "search",
		AuthMode:        string(rpcAuthUser),
		SupportsFields:  true,
		LinkedProcedure: procedureUserSearch,
		Output:          schemaOutput{JSON: true, Human: true, Type: typeSearchResponse},
		Inputs: appendInputs(
			schemaInput{Name: "query", Type: "string", Required: true, Description: "search query"},
			schemaInput{Name: "indexer-id", Type: "string", Description: "optional indexer id"},
			schemaInput{Name: "fields", Type: "string", Description: "comma-separated field paths to include in the output"},
		),
	},
	"user settings": {
		ID:       "user settings",
		Kind:     "command",
		Summary:  "Read and update hosted user settings",
		AuthMode: string(rpcAuthUser),
		Output:   schemaOutput{JSON: true, Human: true},
		Inputs:   cloneInputs(commonCommandInputs),
	},
	"user settings get": {
		ID:              "user settings get",
		Kind:            "command",
		Summary:         "Show hosted user settings",
		AuthMode:        string(rpcAuthUser),
		SupportsFields:  true,
		LinkedProcedure: procedureUserGetUserSettings,
		Output:          schemaOutput{JSON: true, Human: true, Type: "chill.v4.UserSettings"},
		Inputs: appendInputs(
			schemaInput{Name: "fields", Type: "string", Description: "comma-separated field paths to include in the output"},
		),
	},
	"user settings set": {
		ID:              "user settings set",
		Kind:            "command",
		Summary:         "Save full user settings JSON payload or patch one supported field",
		AuthMode:        string(rpcAuthUser),
		Mutates:         true,
		SupportsDryRun:  true,
		LinkedProcedure: procedureUserSaveUserSettings,
		Output:          schemaOutput{JSON: true, Human: true, Type: "chill.v4.UserSettings"},
		Inputs: append(appendInputs(
			schemaInput{Name: "json", Type: "string", Description: "raw JSON request body, bare settings object JSON, or @- to read from stdin"},
			schemaInput{Name: "field", Type: "string", Description: "supported settings field to patch"},
			schemaInput{Name: "value", Type: "string", Description: "normalized patch value for the selected field"},
			schemaInput{Name: "dry-run", Type: "boolean", Description: "validate input and print the request or patch without executing it"},
		), supportedUserSettingsPatchInputs()...),
	},
	"user movies": {
		ID:              "user movies",
		Kind:            "command",
		Summary:         "List movies using your profile settings",
		AliasFor:        "movies",
		AuthMode:        string(rpcAuthUser),
		SupportsFields:  true,
		LinkedProcedure: procedureUserGetMovies,
		Output:          schemaOutput{JSON: true, Human: true, Type: "chill.v4.GetMoviesResponse"},
		Inputs: appendInputs(
			schemaInput{Name: "fields", Type: "string", Description: "comma-separated field paths to include in the output"},
		),
	},
	"user tv-shows": {
		ID:              "user tv-shows",
		Kind:            "command",
		Summary:         "List TV shows from a TV provider source",
		AliasFor:        "tv-shows",
		AuthMode:        string(rpcAuthUser),
		SupportsFields:  true,
		LinkedProcedure: procedureUserGetTVShows,
		Output:          schemaOutput{JSON: true, Human: true, Type: "chill.v4.GetTVShowsResponse"},
		Inputs: appendInputs(
			schemaInput{Name: "source", Type: "string", Description: tvShowsSourceFlagDescription},
			schemaInput{Name: "fields", Type: "string", Description: "comma-separated field paths to include in the output"},
		),
	},
	"user tv-shows detail": {
		ID:              "user tv-shows detail",
		Kind:            "command",
		Summary:         "Show TV show detail by IMDb id",
		AliasFor:        "tv-shows detail",
		AuthMode:        string(rpcAuthUser),
		SupportsFields:  true,
		LinkedProcedure: procedureUserGetTVShowDetail,
		Output:          schemaOutput{JSON: true, Human: true, Type: "chill.v4.GetTVShowDetailResponse"},
		Inputs: appendInputs(
			schemaInput{Name: "imdb-id", Type: "string", Required: true, Description: "IMDb id such as tt0944947"},
			schemaInput{Name: "fields", Type: "string", Description: "comma-separated field paths to include in the output"},
		),
	},
	"user tv-shows season": {
		ID:              "user tv-shows season",
		Kind:            "command",
		Summary:         "Show one TV show season by IMDb id",
		AliasFor:        "tv-shows season",
		AuthMode:        string(rpcAuthUser),
		SupportsFields:  true,
		LinkedProcedure: procedureUserGetTVShowSeason,
		Output:          schemaOutput{JSON: true, Human: true, Type: "chill.v4.GetTVShowSeasonResponse"},
		Inputs: appendInputs(
			schemaInput{Name: "imdb-id", Type: "string", Required: true, Description: "IMDb id such as tt0944947"},
			schemaInput{Name: "season-number", Type: "integer", Required: true, Description: "season number"},
			schemaInput{Name: "fields", Type: "string", Description: "comma-separated field paths to include in the output"},
		),
	},
	"user tv-shows episode-download": {
		ID:              "user tv-shows episode-download",
		Kind:            "command",
		Summary:         "Find one TV episode download by IMDb id",
		AliasFor:        "tv-shows episode-download",
		AuthMode:        string(rpcAuthUser),
		SupportsFields:  true,
		LinkedProcedure: procedureUserGetTVShowEpisodeDownload,
		Output:          schemaOutput{JSON: true, Human: true, Type: "chill.v4.GetTVShowEpisodeDownloadResponse"},
		Inputs: appendInputs(
			schemaInput{Name: "imdb-id", Type: "string", Required: true, Description: "IMDb id such as tt0944947"},
			schemaInput{Name: "season-number", Type: "integer", Required: true, Description: "season number"},
			schemaInput{Name: "episode-number", Type: "integer", Required: true, Description: "episode number"},
			schemaInput{Name: "fields", Type: "string", Description: "comma-separated field paths to include in the output"},
		),
	},
	"user tv-shows season-downloads": {
		ID:              "user tv-shows season-downloads",
		Kind:            "command",
		Summary:         "Find season and episode downloads for one TV season by IMDb id",
		AliasFor:        "tv-shows season-downloads",
		AuthMode:        string(rpcAuthUser),
		SupportsFields:  true,
		LinkedProcedure: procedureUserGetTVShowSeasonDownloads,
		Output:          schemaOutput{JSON: true, Human: true, Type: "chill.v4.GetTVShowSeasonDownloadsResponse"},
		Inputs: appendInputs(
			schemaInput{Name: "imdb-id", Type: "string", Required: true, Description: "IMDb id such as tt0944947"},
			schemaInput{Name: "season-number", Type: "integer", Required: true, Description: "season number"},
			schemaInput{Name: "fields", Type: "string", Description: "comma-separated field paths to include in the output"},
		),
	},
	"user transfer": {
		ID:       "user transfer",
		Kind:     "command",
		Summary:  "Transfer commands",
		AuthMode: string(rpcAuthUser),
		Output:   schemaOutput{JSON: true, Human: true},
		Inputs:   cloneInputs(commonCommandInputs),
	},
	"user transfer add": {
		ID:              "user transfer add",
		Kind:            "command",
		Summary:         "Add a transfer through chill.institute",
		AliasFor:        "add-transfer",
		AuthMode:        string(rpcAuthUser),
		Mutates:         true,
		SupportsDryRun:  true,
		LinkedProcedure: procedureUserAddTransfer,
		Output:          schemaOutput{JSON: true, Human: true, Type: "chill.v4.AddTransferResponse"},
		Inputs: appendInputs(
			schemaInput{Name: "url", Type: "string", Description: addTransferURLDescription + "; use either --url or --json"},
			schemaInput{Name: "json", Type: "string", Description: "raw JSON request body, or @- to read it from stdin"},
			schemaInput{Name: "dry-run", Type: "boolean", Description: "validate input and print the request without executing it"},
		),
		InputModes: addTransferInputModes(),
	},
	"user transfer get": {
		ID:              "user transfer get",
		Kind:            "command",
		Summary:         "Show one transfer through chill.institute",
		AliasFor:        "get-transfer",
		AuthMode:        string(rpcAuthUser),
		SupportsFields:  true,
		LinkedProcedure: procedureUserGetTransfer,
		Output:          schemaOutput{JSON: true, Human: true, Type: "chill.v4.GetTransferResponse"},
		Inputs: appendInputs(
			schemaInput{Name: "id", Type: "integer", Required: true, Description: "transfer id"},
			schemaInput{Name: "fields", Type: "string", Description: "comma-separated field paths to include in the output"},
		),
	},
	"whoami": {
		ID:              "whoami",
		Kind:            "command",
		Summary:         "Show authenticated user profile",
		AuthMode:        string(rpcAuthUser),
		SupportsFields:  true,
		LinkedProcedure: procedureUserGetUserProfile,
		Output:          schemaOutput{JSON: true, Human: true, Type: "chill.v4.UserProfile"},
		Inputs: appendInputs(
			schemaInput{Name: "fields", Type: "string", Description: "comma-separated field paths to include in the output"},
		),
	},
	"version": {
		ID:             "version",
		Kind:           "command",
		Summary:        "Show CLI build info",
		AuthMode:       string(rpcAuthNone),
		SupportsFields: true,
		Output:         schemaOutput{JSON: true, Human: true},
		Inputs: appendInputs(
			schemaInput{Name: "fields", Type: "string", Description: "comma-separated field paths to include in the output"},
		),
	},
}

var procedureSchemaRegistry = map[string]schemaEntry{
	procedureUserAddTransfer: {
		ID:             procedureUserAddTransfer,
		Kind:           "procedure",
		Summary:        "Add transfer to put.io",
		AuthMode:       string(rpcAuthUser),
		Mutates:        true,
		SupportsDryRun: true,
		Output:         schemaOutput{JSON: true, Type: "chill.v4.AddTransferResponse"},
		Inputs: []schemaInput{
			{Name: "url", Type: "string", Required: true, Description: addTransferURLFlagDescription},
		},
	},
	procedureUserGetIndexers: {
		ID:       procedureUserGetIndexers,
		Kind:     "procedure",
		Summary:  "List user indexers",
		AuthMode: string(rpcAuthUser),
		Output:   schemaOutput{JSON: true, Type: "chill.v4.UserGetIndexersResponse"},
	},
	procedureUserGetTransfer: {
		ID:       procedureUserGetTransfer,
		Kind:     "procedure",
		Summary:  "Fetch one transfer",
		AuthMode: string(rpcAuthUser),
		Output:   schemaOutput{JSON: true, Type: "chill.v4.GetTransferResponse"},
		Inputs: []schemaInput{
			{Name: "id", Type: "integer", Required: true, Description: "transfer id"},
		},
	},
	procedureUserGetDownloadFolder: {
		ID:       procedureUserGetDownloadFolder,
		Kind:     "procedure",
		Summary:  "Fetch the current download folder",
		AuthMode: string(rpcAuthUser),
		Output:   schemaOutput{JSON: true, Type: "chill.v4.GetDownloadFolderResponse"},
	},
	procedureUserGetFolder: {
		ID:       procedureUserGetFolder,
		Kind:     "procedure",
		Summary:  "Fetch one folder and its children",
		AuthMode: string(rpcAuthUser),
		Output:   schemaOutput{JSON: true, Type: "chill.v4.GetFolderResponse"},
		Inputs: []schemaInput{
			{Name: "id", Type: "integer", Required: true, Description: "folder id"},
		},
	},
	procedureUserGetMovies: {
		ID:       procedureUserGetMovies,
		Kind:     "procedure",
		Summary:  "List movies for the current user",
		AuthMode: string(rpcAuthUser),
		Output:   schemaOutput{JSON: true, Type: "chill.v4.GetMoviesResponse"},
	},
	procedureUserGetTVShows: {
		ID:       procedureUserGetTVShows,
		Kind:     "procedure",
		Summary:  "List TV shows for the current user",
		AuthMode: string(rpcAuthUser),
		Output:   schemaOutput{JSON: true, Type: "chill.v4.GetTVShowsResponse"},
		Inputs: []schemaInput{
			{Name: "source", Type: "string", Description: "optional TVShowsSource enum override; unspecified uses TV_SHOWS_SOURCE_ALL_PROVIDERS"},
		},
	},
	procedureUserGetTVShowDetail: {
		ID:       procedureUserGetTVShowDetail,
		Kind:     "procedure",
		Summary:  "Fetch TV show detail by IMDb id",
		AuthMode: string(rpcAuthUser),
		Output:   schemaOutput{JSON: true, Type: "chill.v4.GetTVShowDetailResponse"},
		Inputs: []schemaInput{
			{Name: "imdb_id", Type: "string", Required: true, Description: "IMDb id such as tt0944947"},
		},
	},
	procedureUserGetTVShowSeason: {
		ID:       procedureUserGetTVShowSeason,
		Kind:     "procedure",
		Summary:  "Fetch one TV show season by IMDb id",
		AuthMode: string(rpcAuthUser),
		Output:   schemaOutput{JSON: true, Type: "chill.v4.GetTVShowSeasonResponse"},
		Inputs: []schemaInput{
			{Name: "imdb_id", Type: "string", Required: true, Description: "IMDb id such as tt0944947"},
			{Name: "season_number", Type: "integer", Required: true, Description: "season number"},
		},
	},
	procedureUserGetTVShowEpisodeDownload: {
		ID:       procedureUserGetTVShowEpisodeDownload,
		Kind:     "procedure",
		Summary:  "Find one TV episode download by IMDb id",
		AuthMode: string(rpcAuthUser),
		Output:   schemaOutput{JSON: true, Type: "chill.v4.GetTVShowEpisodeDownloadResponse"},
		Inputs: []schemaInput{
			{Name: "imdb_id", Type: "string", Required: true, Description: "IMDb id such as tt0944947"},
			{Name: "season_number", Type: "integer", Required: true, Description: "season number"},
			{Name: "episode_number", Type: "integer", Required: true, Description: "episode number"},
		},
	},
	procedureUserGetTVShowSeasonDownloads: {
		ID:       procedureUserGetTVShowSeasonDownloads,
		Kind:     "procedure",
		Summary:  "Find season and episode downloads for one TV season by IMDb id",
		AuthMode: string(rpcAuthUser),
		Output:   schemaOutput{JSON: true, Type: "chill.v4.GetTVShowSeasonDownloadsResponse"},
		Inputs: []schemaInput{
			{Name: "imdb_id", Type: "string", Required: true, Description: "IMDb id such as tt0944947"},
			{Name: "season_number", Type: "integer", Required: true, Description: "season number"},
		},
	},
	procedureUserGetUserProfile: {
		ID:       procedureUserGetUserProfile,
		Kind:     "procedure",
		Summary:  "Fetch the authenticated user profile",
		AuthMode: string(rpcAuthUser),
		Output:   schemaOutput{JSON: true, Type: "chill.v4.UserProfile"},
	},
	procedureUserGetUserSettings: {
		ID:       procedureUserGetUserSettings,
		Kind:     "procedure",
		Summary:  "Fetch user settings",
		AuthMode: string(rpcAuthUser),
		Output:   schemaOutput{JSON: true, Type: "chill.v4.UserSettings"},
	},
	procedureUserSaveUserSettings: {
		ID:             procedureUserSaveUserSettings,
		Kind:           "procedure",
		Summary:        "Save user settings",
		AuthMode:       string(rpcAuthUser),
		Mutates:        true,
		SupportsDryRun: true,
		Output:         schemaOutput{JSON: true, Type: "chill.v4.UserSettings"},
		Inputs: []schemaInput{
			{Name: "settings", Type: "object", Required: true, Description: "full user settings object"},
		},
	},
	procedureUserSearch: {
		ID:       procedureUserSearch,
		Kind:     "procedure",
		Summary:  "Search using the current user profile settings",
		AuthMode: string(rpcAuthUser),
		Output:   schemaOutput{JSON: true, Type: typeSearchResponse},
		Inputs: []schemaInput{
			{Name: "query", Type: "string", Required: true, Description: "search query"},
			{Name: "indexer_id", Type: "string", Description: "optional indexer id"},
		},
	},
}

var typeSchemaRegistry = map[string]schemaType{
	"schema.Field": {
		ID:      "schema.Field",
		Kind:    "type",
		Summary: "One field in local CLI schema type metadata",
		Fields: []schemaField{
			schemaFieldFor("name", "string"),
			{Name: "json_name", JSONName: "json_name", Type: "string"},
			schemaFieldFor("type", "string"),
			optionalSchemaField("repeated", "boolean"),
			optionalSchemaField("optional", "boolean"),
			optionalSchemaField("description", "string"),
		},
	},
	"schema.Type": {
		ID:      "schema.Type",
		Kind:    "type",
		Summary: "Local CLI schema type metadata returned by schema type",
		Fields: []schemaField{
			schemaFieldFor("id", "string"),
			schemaFieldFor("kind", "string"),
			schemaFieldFor("summary", "string"),
			repeatedSchemaField("fields", "schema.Field"),
		},
	},
	"chill.v4.AddTransferResponse": {
		ID:      "chill.v4.AddTransferResponse",
		Kind:    "type",
		Summary: "Transfer creation response",
		Fields: []schemaField{
			schemaFieldFor("status", "string"),
			schemaFieldFor("transfer", "chill.v4.Transfer"),
		},
	},
	"chill.v4.CatalogSettings": {
		ID:      "chill.v4.CatalogSettings",
		Kind:    "type",
		Summary: "User catalog source preferences",
		Fields: []schemaField{
			schemaFieldFor("movies_source", "string"),
			schemaFieldFor("tv_shows_source", "string"),
		},
	},
	"chill.v4.DownloadSettings": {
		ID:      "chill.v4.DownloadSettings",
		Kind:    "type",
		Summary: "User download destination preferences",
		Fields: []schemaField{
			optionalSchemaField("folder_id", "integer"),
		},
	},
	"chill.v4.GetDownloadFolderResponse": {
		ID:      "chill.v4.GetDownloadFolderResponse",
		Kind:    "type",
		Summary: "Current download folder response",
		Fields: []schemaField{
			schemaFieldFor("folder", "chill.v4.UserFile"),
		},
	},
	"chill.v4.GetFolderResponse": {
		ID:      "chill.v4.GetFolderResponse",
		Kind:    "type",
		Summary: "Folder listing response",
		Fields: []schemaField{
			schemaFieldFor("parent", "chill.v4.UserFile"),
			repeatedSchemaField("files", "chill.v4.UserFile"),
		},
	},
	"chill.v4.GetMoviesResponse": {
		ID:      "chill.v4.GetMoviesResponse",
		Kind:    "type",
		Summary: "Movie catalog response",
		Fields: []schemaField{
			schemaFieldFor("source", "string"),
			repeatedSchemaField("movies", "chill.v4.Movie"),
			schemaFieldFor("rss_feed_url", "string"),
		},
	},
	"chill.v4.GetTransferResponse": {
		ID:      "chill.v4.GetTransferResponse",
		Kind:    "type",
		Summary: "Single transfer response",
		Fields: []schemaField{
			schemaFieldFor("transfer", "chill.v4.Transfer"),
		},
	},
	"chill.v4.GetTVShowDetailResponse": {
		ID:      "chill.v4.GetTVShowDetailResponse",
		Kind:    "type",
		Summary: "TV show detail response",
		Fields: []schemaField{
			schemaFieldFor("show", "chill.v4.TVShowDetail"),
			repeatedSchemaField("seasons", "chill.v4.TVShowSeason"),
		},
	},
	"chill.v4.GetTVShowEpisodeDownloadResponse": {
		ID:      "chill.v4.GetTVShowEpisodeDownloadResponse",
		Kind:    "type",
		Summary: "TV episode download response",
		Fields: []schemaField{
			optionalSchemaField("download", "chill.v4.TVShowDownload"),
			schemaFieldFor("search_query", "string"),
		},
	},
	"chill.v4.GetTVShowsResponse": {
		ID:      "chill.v4.GetTVShowsResponse",
		Kind:    "type",
		Summary: "TV show catalog response",
		Fields: []schemaField{
			schemaFieldFor("source", "string"),
			repeatedSchemaField("shows", "chill.v4.TVShow"),
		},
	},
	"chill.v4.GetTVShowSeasonDownloadsResponse": {
		ID:      "chill.v4.GetTVShowSeasonDownloadsResponse",
		Kind:    "type",
		Summary: "TV season downloads response",
		Fields: []schemaField{
			optionalSchemaField("season_pack", "chill.v4.TVShowDownload"),
			repeatedSchemaField("episodes", "chill.v4.TVShowEpisodeDownloadResult"),
			schemaFieldFor("season_search_query", "string"),
		},
	},
	"chill.v4.GetTVShowSeasonResponse": {
		ID:      "chill.v4.GetTVShowSeasonResponse",
		Kind:    "type",
		Summary: "TV show season response",
		Fields: []schemaField{
			schemaFieldFor("imdb_id", "string"),
			schemaFieldFor("season_number", "integer"),
			schemaFieldFor("season", "chill.v4.TVShowSeason"),
			repeatedSchemaField("episodes", "chill.v4.TVShowEpisode"),
		},
	},
	"chill.v4.Movie": {
		ID:      "chill.v4.Movie",
		Kind:    "type",
		Summary: "Movie catalog item",
		Fields: []schemaField{
			schemaFieldFor("id", "string"),
			schemaFieldFor("title", "string"),
			schemaFieldFor("year", "integer"),
			schemaFieldFor("source", "string"),
			schemaFieldFor("title_pretty", "string"),
			schemaFieldFor("link", "string"),
			schemaFieldFor("peers", "integer"),
			schemaFieldFor("seeders", "integer"),
			schemaFieldFor("size", "integer"),
			schemaFieldFor("uploaded_at", "string"),
			schemaFieldFor("poster_url", "string"),
			schemaFieldFor("rating", "number"),
			schemaFieldFor("external_url", "string"),
			schemaFieldFor("backdrop_url", "string"),
			schemaFieldFor("overview", "string"),
			repeatedSchemaField("genres", "string"),
		},
	},
	"chill.v4.SearchSettings": {
		ID:      "chill.v4.SearchSettings",
		Kind:    "type",
		Summary: "User search preferences",
		Fields: []schemaField{
			repeatedSchemaField("codec_filters", "string"),
			repeatedSchemaField("disabled_indexer_ids", "string"),
			schemaFieldFor("filter_nasty_results", "boolean"),
			schemaFieldFor("filter_results_with_no_seeders", "boolean"),
			repeatedSchemaField("other_filters", "string"),
			schemaFieldFor("remember_quick_filters", "boolean"),
			repeatedSchemaField("resolution_filters", "string"),
			schemaFieldFor("search_result_display_behavior", "string"),
			schemaFieldFor("search_result_title_behavior", "string"),
			schemaFieldFor("sort_by", "string"),
			schemaFieldFor("sort_direction", "string"),
		},
	},
	"chill.v4.TVShow": {
		ID:      "chill.v4.TVShow",
		Kind:    "type",
		Summary: "TV show catalog item",
		Fields: []schemaField{
			schemaFieldFor("imdb_id", "string"),
			schemaFieldFor("title", "string"),
			schemaFieldFor("year", "integer"),
			schemaFieldFor("source", "string"),
			schemaFieldFor("poster_url", "string"),
			schemaFieldFor("rating", "number"),
			schemaFieldFor("overview", "string"),
			schemaFieldFor("external_url", "string"),
			schemaFieldFor("season_count", "integer"),
			schemaFieldFor("status", "string"),
			repeatedSchemaField("networks", "string"),
		},
	},
	"chill.v4.TVShowDetail": {
		ID:      "chill.v4.TVShowDetail",
		Kind:    "type",
		Summary: "TV show detail item",
		Fields: []schemaField{
			schemaFieldFor("imdb_id", "string"),
			schemaFieldFor("title", "string"),
			schemaFieldFor("year", "integer"),
			schemaFieldFor("poster_url", "string"),
			schemaFieldFor("backdrop_url", "string"),
			schemaFieldFor("rating", "number"),
			schemaFieldFor("overview", "string"),
			schemaFieldFor("external_url", "string"),
			schemaFieldFor("season_count", "integer"),
			schemaFieldFor("status", "string"),
			repeatedSchemaField("networks", "string"),
			repeatedSchemaField("genres", "string"),
		},
	},
	"chill.v4.TVShowDownload": {
		ID:      "chill.v4.TVShowDownload",
		Kind:    "type",
		Summary: "TV show download search match",
		Fields: []schemaField{
			schemaFieldFor("title", "string"),
			schemaFieldFor("link", "string"),
			schemaFieldFor("size", "integer"),
			schemaFieldFor("seeders", "integer"),
			schemaFieldFor("resolution", "string"),
			schemaFieldFor("codec", "string"),
			schemaFieldFor("quality", "string"),
			schemaFieldFor("indexer", "string"),
			schemaFieldFor("season_number", "integer"),
			optionalSchemaField("episode_number", "integer"),
		},
	},
	"chill.v4.TVShowEpisode": {
		ID:      "chill.v4.TVShowEpisode",
		Kind:    "type",
		Summary: "TV show episode item",
		Fields: []schemaField{
			schemaFieldFor("season_number", "integer"),
			schemaFieldFor("episode_number", "integer"),
			schemaFieldFor("name", "string"),
			schemaFieldFor("overview", "string"),
			schemaFieldFor("air_date", "string"),
			schemaFieldFor("runtime", "integer"),
			schemaFieldFor("still_url", "string"),
			schemaFieldFor("rating", "number"),
		},
	},
	"chill.v4.TVShowEpisodeDownloadResult": {
		ID:      "chill.v4.TVShowEpisodeDownloadResult",
		Kind:    "type",
		Summary: "Per-episode TV download result",
		Fields: []schemaField{
			schemaFieldFor("episode_number", "integer"),
			optionalSchemaField("download", "chill.v4.TVShowDownload"),
			schemaFieldFor("search_query", "string"),
		},
	},
	"chill.v4.TVShowSeason": {
		ID:      "chill.v4.TVShowSeason",
		Kind:    "type",
		Summary: "TV show season item",
		Fields: []schemaField{
			schemaFieldFor("season_number", "integer"),
			schemaFieldFor("name", "string"),
			schemaFieldFor("episode_count", "integer"),
			schemaFieldFor("air_date", "string"),
			schemaFieldFor("poster_url", "string"),
		},
	},
	"chill.v4.Transfer": {
		ID:      "chill.v4.Transfer",
		Kind:    "type",
		Summary: "Transfer status and progress",
		Fields: []schemaField{
			schemaFieldFor("id", "integer"),
			schemaFieldFor("name", "string"),
			schemaFieldFor("status", "string"),
			schemaFieldFor("percent_done", "integer"),
			schemaFieldFor("status_message", "string"),
			schemaFieldFor("error_message", "string"),
			schemaFieldFor("size", "integer"),
			schemaFieldFor("downloaded", "integer"),
			schemaFieldFor("uploaded", "integer"),
			schemaFieldFor("download_speed", "integer"),
			schemaFieldFor("upload_speed", "integer"),
			schemaFieldFor("peers_connected", "integer"),
			schemaFieldFor("peers_sending_to_us", "integer"),
			schemaFieldFor("peers_getting_from_us", "integer"),
			schemaFieldFor("estimated_time_seconds", "integer"),
			optionalSchemaField("file_id", "integer"),
			optionalSchemaField("file_url", "string"),
			optionalSchemaField("save_parent_id", "integer"),
			schemaFieldFor("source", "string"),
			optionalSchemaField("created_at", "string"),
			optionalSchemaField("finished_at", "string"),
			schemaFieldFor("is_finished", "boolean"),
		},
	},
	"chill.v4.UserFile": {
		ID:      "chill.v4.UserFile",
		Kind:    "type",
		Summary: "User file or folder",
		Fields: []schemaField{
			schemaFieldFor("id", "integer"),
			schemaFieldFor("name", "string"),
			schemaFieldFor("file_type", "string"),
			schemaFieldFor("is_shared", "boolean"),
			schemaFieldFor("created_at", "string"),
		},
	},
	"chill.v4.UserGetIndexersResponse": {
		ID:      "chill.v4.UserGetIndexersResponse",
		Kind:    "type",
		Summary: "User indexer listing response",
		Fields: []schemaField{
			repeatedSchemaField("indexers", "chill.v4.UserIndexer"),
		},
	},
	"chill.v4.UserIndexer": {
		ID:      "chill.v4.UserIndexer",
		Kind:    "type",
		Summary: "User indexer configuration and health",
		Fields: []schemaField{
			schemaFieldFor("id", "string"),
			schemaFieldFor("name", "string"),
			schemaFieldFor("enabled", "boolean"),
			repeatedSchemaField("tags", "string"),
			optionalSchemaField("status", "string"),
		},
	},
	"chill.v4.UserProfile": {
		ID:      "chill.v4.UserProfile",
		Kind:    "type",
		Summary: "Authenticated user profile",
		Fields: []schemaField{
			schemaFieldFor("user_id", "string"),
			schemaFieldFor("username", "string"),
			schemaFieldFor("avatar_url", "string"),
			schemaFieldFor("email", "string"),
		},
	},
	"chill.v4.UserSettings": {
		ID:      "chill.v4.UserSettings",
		Kind:    "type",
		Summary: "User settings grouped by search, catalog, and download preferences",
		Fields: []schemaField{
			schemaFieldFor("search", "chill.v4.SearchSettings"),
			schemaFieldFor("catalog", "chill.v4.CatalogSettings"),
			schemaFieldFor("download", "chill.v4.DownloadSettings"),
		},
	},
	typeReleaseInfo: {
		ID:      typeReleaseInfo,
		Kind:    "type",
		Summary: "Parsed release metadata attached to search results when the API can infer it from torrent names",
		Fields: []schemaField{
			schemaFieldFor("title", "string"),
			optionalSchemaField("year", "integer"),
			optionalSchemaField("season", "integer"),
			optionalSchemaField("episode", "integer"),
			optionalSchemaField("episode_end", "integer"),
			optionalSchemaField("part", "integer"),
			schemaFieldFor("resolution", "string"),
			schemaFieldFor("quality", "string"),
			schemaFieldFor("source", "string"),
			schemaFieldFor("codec", "string"),
			schemaFieldFor("hdr", "string"),
			schemaFieldFor("audio", "string"),
			schemaFieldFor("group", "string"),
			schemaFieldFor("container", "string"),
			schemaFieldFor("language", "string"),
			schemaFieldFor("region", "string"),
			schemaFieldFor("size", "string"),
			schemaFieldFor("bit_depth", "string"),
			schemaFieldFor("edition", "string"),
			schemaFieldFor("extended", "boolean"),
			schemaFieldFor("hardcoded", "boolean"),
			schemaFieldFor("proper", "boolean"),
			schemaFieldFor("repack", "boolean"),
			schemaFieldFor("remastered", "boolean"),
			schemaFieldFor("complete", "boolean"),
			schemaFieldFor("three_d", "boolean"),
			schemaFieldFor("imax", "boolean"),
			schemaFieldFor("unrated", "boolean"),
			schemaFieldFor("widescreen", "boolean"),
			schemaFieldFor("excess", "string"),
		},
	},
	typeSearchResponse: {
		ID:      typeSearchResponse,
		Kind:    "type",
		Summary: "Search response returned by user search procedures",
		Fields: []schemaField{
			schemaFieldFor("query", "string"),
			repeatedSchemaField("results", typeSearchResult),
		},
	},
	typeSearchResult: {
		ID:      typeSearchResult,
		Kind:    "type",
		Summary: "One torrent search result",
		Fields: []schemaField{
			schemaFieldFor("id", "string"),
			schemaFieldFor("title", "string"),
			schemaFieldFor("indexer", "string"),
			schemaFieldFor("link", "string"),
			optionalSchemaField("imdb_id", "string"),
			schemaFieldFor("peers", "integer"),
			schemaFieldFor("seeders", "integer"),
			schemaFieldFor("size", "integer"),
			schemaFieldFor("source", "string"),
			schemaFieldFor("uploaded_at", "string"),
			optionalSchemaField("release_info", typeReleaseInfo),
		},
	},
}

func listCommandSchemas() []schemaEntry {
	return sortedSchemaEntries(commandSchemaRegistry)
}

func listProcedureSchemas() []schemaEntry {
	return sortedSchemaEntries(procedureSchemaRegistry)
}

func listTypeSchemas() []schemaType {
	entries := make([]schemaType, 0, len(typeSchemaRegistry))
	for _, entry := range typeSchemaRegistry {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ID < entries[j].ID
	})
	return entries
}

func lookupCommandSchema(id string) (schemaEntry, bool) {
	entry, ok := commandSchemaRegistry[strings.TrimSpace(id)]
	return entry, ok
}

func lookupProcedureSchema(id string) (schemaEntry, bool) {
	entry, ok := procedureSchemaRegistry[strings.TrimSpace(id)]
	return entry, ok
}

func lookupTypeSchema(id string) (schemaType, bool) {
	entry, ok := typeSchemaRegistry[strings.TrimSpace(id)]
	return entry, ok
}

func schemaCommandID(commandPath string) string {
	trimmed := strings.TrimSpace(commandPath)
	if trimmed == "" || trimmed == "chilly" {
		return "chilly"
	}
	return strings.TrimSpace(strings.TrimPrefix(trimmed, "chilly "))
}

func sortedSchemaEntries(registry map[string]schemaEntry) []schemaEntry {
	entries := make([]schemaEntry, 0, len(registry))
	for _, entry := range registry {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ID < entries[j].ID
	})
	return entries
}

func schemaFieldFor(name string, fieldType string) schemaField {
	return schemaField{
		Name:     name,
		JSONName: snakeToLowerCamel(name),
		Type:     fieldType,
	}
}

func optionalSchemaField(name string, fieldType string) schemaField {
	field := schemaFieldFor(name, fieldType)
	field.Optional = true
	return field
}

func repeatedSchemaField(name string, fieldType string) schemaField {
	field := schemaFieldFor(name, fieldType)
	field.Repeated = true
	return field
}

func appendInputs(extra ...schemaInput) []schemaInput {
	inputs := cloneInputs(commonCommandInputs)
	inputs = append(inputs, extra...)
	return inputs
}

func addTransferInputModes() []schemaInputMode {
	return []schemaInputMode{
		schemaInputModeForRequiredInputs(
			"url",
			"Build the request from --url.",
			"url",
		),
		schemaInputModeForRequiredInputs(
			"json",
			"Use a raw JSON request body from --json.",
			"json",
		),
	}
}

func settingsSetInputModes() []schemaInputMode {
	return []schemaInputMode{
		schemaInputModeForRequiredInputs(
			"key-value",
			"Set one setting from positional key and value arguments.",
			"key",
			"value",
		),
		schemaInputModeForRequiredInputs(
			"json",
			"Set one setting from a raw JSON request body.",
			"json",
		),
	}
}

func schemaInputModeForRequiredInputs(name string, description string, inputs ...string) schemaInputMode {
	return schemaInputMode{
		Name:        name,
		Description: description,
		Inputs:      cloneStrings(inputs),
		Required:    cloneStrings(inputs),
	}
}

func cloneInputs(inputs []schemaInput) []schemaInput {
	if len(inputs) == 0 {
		return nil
	}
	cloned := make([]schemaInput, len(inputs))
	copy(cloned, inputs)
	return cloned
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}

type metadataAuthMode string

const (
	rpcAuthNone metadataAuthMode = "none"
	rpcAuthUser metadataAuthMode = "user"
)
