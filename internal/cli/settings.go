package cli

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/chill-institute/chill-cli/internal/config"
	"github.com/spf13/cobra"
)

const redactedToken = "[redacted]"

func newSettingsCommand(app *appContext) *cobra.Command {
	command := &cobra.Command{
		Use:   "settings",
		Short: "Manage local CLI config",
		Example: strings.TrimSpace(`
chilly settings show
chilly settings get api-base-url
chilly settings set api-base-url https://api.chill.institute --dry-run --output json
`),
	}

	command.AddCommand(newSettingsPathCommand(app))
	command.AddCommand(newSettingsShowCommand(app))
	command.AddCommand(newSettingsGetCommand(app))
	command.AddCommand(newSettingsSetCommand(app))
	return command
}

func newSettingsPathCommand(app *appContext) *cobra.Command {
	var fields string
	command := &cobra.Command{
		Use:   "path",
		Short: "Show local config file path",
		RunE: func(cmd *cobra.Command, args []string) error {
			selection, err := parseFieldSelection(fields)
			if err != nil {
				return err
			}
			store, err := app.configStore()
			if err != nil {
				return err
			}
			profile, err := app.activeProfile()
			if err != nil {
				return err
			}
			return app.writeAnyWithRenderer(map[string]any{
				"path":    store.Path(),
				"profile": profile,
			}, selection, nil)
		},
	}
	command.Flags().StringVar(&fields, "fields", "", "comma-separated field paths to include in the output")
	return command
}

func newSettingsShowCommand(app *appContext) *cobra.Command {
	var fields string
	command := &cobra.Command{
		Use:   "show",
		Short: "Show local CLI config (auth token redacted)",
		RunE: func(cmd *cobra.Command, args []string) error {
			selection, err := parseFieldSelection(fields)
			if err != nil {
				return err
			}
			cfg, err := loadStoredCLISettings(app)
			if err != nil {
				return err
			}
			profile, err := app.activeProfile()
			if err != nil {
				return err
			}
			authToken := ""
			if strings.TrimSpace(cfg.AuthToken) != "" {
				authToken = redactedToken
			}
			return app.writeAnyWithRenderer(map[string]any{
				"profile":      profile,
				"api_base_url": cfg.APIBaseURL,
				"auth_token":   authToken,
			}, selection, nil)
		},
	}
	command.Flags().StringVar(&fields, "fields", "", "comma-separated field paths to include in the output")
	return command
}

func newSettingsGetCommand(app *appContext) *cobra.Command {
	var fields string
	command := &cobra.Command{
		Use:   "get <key>",
		Short: "Show one local CLI setting",
		Args:  allowDescribeArgs(cobra.ExactArgs(1)),
		RunE: func(cmd *cobra.Command, args []string) error {
			selection, err := parseFieldSelection(fields)
			if err != nil {
				return err
			}
			key, err := normalizeSettingsKey(args[0])
			if err != nil {
				return err
			}

			cfg, err := loadStoredCLISettings(app)
			if err != nil {
				return err
			}

			switch key {
			case "api-base-url":
				return app.writeAnyWithRenderer(map[string]any{
					"key":   key,
					"value": cfg.APIBaseURL,
				}, selection, nil)
			default:
				return fmt.Errorf("unsupported settings key %q", key)
			}
		},
	}
	command.Flags().StringVar(&fields, "fields", "", "comma-separated field paths to include in the output")
	return command
}

func newSettingsSetCommand(app *appContext) *cobra.Command {
	var dryRun bool
	var rawRequest string

	command := &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set one local CLI setting",
		Example: strings.TrimSpace(`
chilly settings set api-base-url https://api.chill.institute
chilly settings set api-base-url https://api.chill.institute --dry-run --output json
printf '{"key":"api-base-url","value":"https://api.chill.institute"}' | chilly settings set --json @- --dry-run --output json
`),
		Args: allowDescribeArgs(cobra.MaximumNArgs(2)),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, nextValue, err := resolveSettingsSetInput(app, args, rawRequest)
			if err != nil {
				return err
			}

			cfg, err := loadStoredCLISettings(app)
			if err != nil {
				return err
			}

			request := map[string]any{
				"key": key,
			}

			switch key {
			case "api-base-url":
				request["value"] = nextValue
				if dryRun {
					return app.writeLocalDryRunPreview("settings set", request)
				}
				cfg.APIBaseURL = nextValue
			default:
				return fmt.Errorf("unsupported settings key %q", key)
			}

			if err := app.saveConfig(cfg); err != nil {
				return err
			}

			return app.writeJSONPayload(map[string]any{
				"status": "ok",
				"key":    key,
				"value":  cfg.APIBaseURL,
			})
		},
	}

	command.Flags().StringVar(&rawRequest, "json", "", "raw JSON request body, or @- to read it from stdin")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "validate input and print the local config change without saving it")
	return command
}

func resolveSettingsSetInput(app *appContext, args []string, rawRequest string) (string, string, error) {
	trimmedRequest := strings.TrimSpace(rawRequest)
	if trimmedRequest == "" {
		if len(args) != 2 {
			return "", "", usageError("missing_settings_update", "provide either --json or <key> <value>")
		}
		key, err := normalizeSettingsKey(args[0])
		if err != nil {
			return "", "", err
		}
		nextValue, err := normalizeSettingsValue(key, args[1])
		if err != nil {
			return "", "", err
		}
		return key, nextValue, nil
	}
	if len(args) != 0 {
		return "", "", usageError("ambiguous_settings_update", "use either --json or <key> <value>, not both")
	}

	payload, err := app.decodeJSONObjectFlag(rawRequest, "--json")
	if err != nil {
		return "", "", err
	}
	rawKey, ok := payload["key"].(string)
	if !ok {
		return "", "", usageError("invalid_json_payload", "--json payload must include a string key field")
	}
	rawValue, ok := payload["value"].(string)
	if !ok {
		return "", "", usageError("invalid_json_payload", "--json payload must include a string value field")
	}
	key, err := normalizeSettingsKey(rawKey)
	if err != nil {
		return "", "", err
	}
	nextValue, err := normalizeSettingsValue(key, rawValue)
	if err != nil {
		return "", "", err
	}
	return key, nextValue, nil
}

func normalizeSettingsValue(key string, raw string) (string, error) {
	switch key {
	case "api-base-url":
		return normalizeAPIBaseURL(raw)
	default:
		return "", fmt.Errorf("unsupported settings key %q", key)
	}
}

func loadStoredCLISettings(app *appContext) (config.Config, error) {
	store, err := app.configStore()
	if err != nil {
		return config.Config{}, err
	}
	cfg, err := store.Load()
	if err != nil {
		return config.Config{}, err
	}
	return cfg.Normalized(), nil
}

func normalizeSettingsKey(raw string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch normalized {
	case "api-base-url", "api_base_url":
		return "api-base-url", nil
	default:
		return "", usageError("unsupported_settings_key", "unsupported settings key %q (supported: api-base-url)", raw)
	}
}

func normalizeAPIBaseURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", usageError("empty_api_base_url", "api-base-url cannot be empty")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", usageError("invalid_api_base_url", "parse api-base-url: %v", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", usageError("invalid_api_base_url", "api-base-url must include scheme and host")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", usageError("invalid_api_base_url", "api-base-url must start with http:// or https://")
	}
	if parsed.Scheme == "http" && !isLoopbackHost(parsed.Hostname()) {
		return "", usageError("invalid_api_base_url", "api-base-url must use https except for loopback development hosts")
	}
	if parsed.User != nil {
		return "", usageError("invalid_api_base_url", "api-base-url must not include user info")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", usageError("invalid_api_base_url", "api-base-url must not include query or fragment data")
	}
	if strings.TrimSpace(parsed.Path) != "" && parsed.Path != "/" {
		return "", usageError("invalid_api_base_url", "api-base-url must not include a path")
	}

	return strings.TrimRight(trimmed, "/"), nil
}

func isLoopbackHost(raw string) bool {
	host := strings.ToLower(strings.TrimSpace(raw))
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
