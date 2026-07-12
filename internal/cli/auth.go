package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/chill-institute/chill-cli/internal/config"
	"github.com/chill-institute/chill-cli/internal/rpc"
	"github.com/spf13/cobra"
)

func newAuthCommand(app *appContext) *cobra.Command {
	command := &cobra.Command{
		Use:   "auth",
		Short: "Sign in and manage local auth tokens",
		Example: strings.TrimSpace(`
chilly auth login
chilly auth login --token "token-from-setup"
chilly auth logout --dry-run --output json
`),
	}

	command.AddCommand(newAuthLoginCommand(app))
	command.AddCommand(newAuthLogoutCommand(app))
	return command
}

func newAuthLoginCommand(app *appContext) *cobra.Command {
	var token string
	var rawRequest string
	var skipOpen bool
	var localBrowser bool
	var skipVerify bool
	var dryRun bool

	command := &cobra.Command{
		Use:   "login",
		Short: "Sign in by pasting a setup token from the hosted web flow",
		Long: strings.TrimSpace(`
Sign in by opening the hosted web token page, copying the setup token,
and pasting it into a hidden terminal prompt. Use --token only when
automation cannot provide controlled stdin; command arguments may be
visible in shell history or process inspection. Use
--local-browser only when you explicitly want the localhost
callback flow with optional browser auto-open.
`),
		Example: strings.TrimSpace(`
chilly auth login
chilly auth login --token "token-from-setup"
printf '{"token":"token-from-setup","skip_verify":true}' | chilly auth login --json @- --dry-run --output json
chilly auth login --local-browser
chilly auth login --local-browser --no-browser
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := app.loadConfig()
			if err != nil {
				return err
			}

			resolvedToken, resolvedSkipOpen, resolvedLocalBrowser, resolvedSkipVerify, err := resolveAuthLoginInput(app, token, rawRequest, skipOpen, localBrowser, skipVerify)
			if err != nil {
				return err
			}
			if dryRun {
				request := map[string]any{
					"api_base_url": cfg.APIBaseURL,
					"skip_verify":  resolvedSkipVerify,
					"no_browser":   resolvedSkipOpen,
				}
				if strings.TrimSpace(resolvedToken) != "" {
					request["mode"] = "token"
					request["token_provided"] = true
				} else if resolvedLocalBrowser {
					request["mode"] = "local_browser"
				} else {
					loginURL, err := webAuthTokenURL(cfg.APIBaseURL)
					if err != nil {
						return err
					}
					request["mode"] = "web_token"
					request["login_url"] = loginURL
				}
				return app.writeLocalDryRunPreview("auth login", request)
			}
			if resolvedToken == "" {
				if resolvedLocalBrowser {
					resolvedToken, err = app.loginWithBrowser(context.Background(), cfg, resolvedSkipOpen)
				} else {
					resolvedToken, err = app.loginWithWebToken(cfg, resolvedSkipOpen)
				}
				if err != nil {
					return err
				}
			}

			if resolvedToken == "" {
				return usageError("empty_token", "token cannot be empty")
			}

			var verifyResponse rpc.CallResponse
			if !resolvedSkipVerify {
				verifyResponse, err = app.callRPC(
					context.Background(),
					cfg,
					procedureUserGetUserProfile,
					map[string]any{},
					rpc.AuthUser,
					resolvedToken,
				)
				if err != nil {
					return fmt.Errorf("verify auth token: %w", err)
				}
			}

			cfg.AuthToken = resolvedToken
			if err := app.saveConfig(cfg); err != nil {
				return fmt.Errorf("persist auth token: %w", err)
			}

			if resolvedSkipVerify {
				return app.writeJSONPayload(map[string]any{"status": "ok", "saved": true})
			}
			return app.writeJSONPayload(map[string]any{
				"status":     "ok",
				"saved":      true,
				"request_id": verifyResponse.RequestID,
				"user":       jsonMessage(verifyResponse.Body),
			})
		},
	}

	command.Flags().StringVar(&token, "token", "", "setup token to store (non-interactive)")
	command.Flags().StringVar(&rawRequest, "json", "", "raw JSON request body, or @- to read it from stdin")
	command.Flags().BoolVar(&skipOpen, "no-browser", false, "with --local-browser, print the login URL instead of opening a browser automatically")
	command.Flags().BoolVar(&localBrowser, "local-browser", false, "use the localhost callback flow instead of the hosted web token flow")
	command.Flags().BoolVar(&skipVerify, "skip-verify", false, "skip token verification call")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "preview the auth action without verifying or saving credentials")
	return command
}

func resolveAuthLoginInput(app *appContext, token string, rawRequest string, skipOpen bool, localBrowser bool, skipVerify bool) (string, bool, bool, bool, error) {
	trimmedToken := strings.TrimSpace(token)
	trimmedRequest := strings.TrimSpace(rawRequest)

	if trimmedRequest == "" {
		return trimmedToken, skipOpen, localBrowser, skipVerify, nil
	}
	if trimmedToken != "" || skipOpen || localBrowser || skipVerify {
		return "", false, false, false, usageError("ambiguous_auth_login_input", "use either flags or --json for auth login input, not both")
	}

	payload, err := app.decodeJSONObjectFlag(rawRequest, "--json")
	if err != nil {
		return "", false, false, false, err
	}

	resolvedSkipOpen := false
	if value, ok := payload["no_browser"]; ok {
		typed, ok := value.(bool)
		if !ok {
			return "", false, false, false, usageError("invalid_json_payload", "--json payload field no_browser must be a boolean")
		}
		resolvedSkipOpen = typed
	}

	resolvedLocalBrowser := false
	if value, ok := payload["local_browser"]; ok {
		typed, ok := value.(bool)
		if !ok {
			return "", false, false, false, usageError("invalid_json_payload", "--json payload field local_browser must be a boolean")
		}
		resolvedLocalBrowser = typed
	}

	resolvedSkipVerify := false
	if value, ok := payload["skip_verify"]; ok {
		typed, ok := value.(bool)
		if !ok {
			return "", false, false, false, usageError("invalid_json_payload", "--json payload field skip_verify must be a boolean")
		}
		resolvedSkipVerify = typed
	}

	resolvedToken := ""
	if value, ok := payload["token"]; ok {
		typed, ok := value.(string)
		if !ok {
			return "", false, false, false, usageError("invalid_json_payload", "--json payload field token must be a string")
		}
		resolvedToken = strings.TrimSpace(typed)
	}

	return resolvedToken, resolvedSkipOpen, resolvedLocalBrowser, resolvedSkipVerify, nil
}

func (app *appContext) loginWithBrowser(ctx context.Context, cfg config.Config, skipOpen bool) (string, error) {
	flow, err := newLoopbackAuthFlow(cfg.APIBaseURL)
	if err != nil {
		return "", err
	}
	defer func() {
		if shutdownErr := flow.shutdown(); shutdownErr != nil {
			_, _ = fmt.Fprintf(app.stderr, "Unable to stop local auth server cleanly: %v\n", shutdownErr)
		}
	}()

	noticeWriter := app.stdout
	if wantsJSONOutput(app.opts.output) {
		noticeWriter = app.stderr
	}
	if _, err := fmt.Fprintf(noticeWriter, "Open this URL to authenticate:\n%s\n\n", flow.loginURL); err != nil {
		return "", err
	}

	errCh := make(chan error, 1)
	flow.start(errCh)

	if !skipOpen {
		if err := app.openURL(flow.loginURL); err != nil {
			_, _ = fmt.Fprintf(app.stderr, "Unable to open browser automatically: %v\n", err)
		}
	}

	timeout := app.authFlowTimeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	token, err := flow.waitForToken(waitCtx, errCh)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(token), nil
}

func (app *appContext) loginWithWebToken(cfg config.Config, skipOpen bool) (string, error) {
	loginURL, err := webAuthTokenURL(cfg.APIBaseURL)
	if err != nil {
		return "", err
	}

	noticeWriter := app.stdout
	if wantsJSONOutput(app.opts.output) {
		noticeWriter = app.stderr
	}
	if _, err := fmt.Fprintf(noticeWriter, "1. Open this URL in a signed-in browser:\n%s\n\n2. Copy the setup token.\n3. Paste it below.\n\n", loginURL); err != nil {
		return "", err
	}

	if app == nil || app.isInputTerminal == nil || !app.isInputTerminal(app.stdin) {
		return "", usageError("interactive_auth_requires_terminal", "auth login requires a terminal for token entry; use --token or --json for non-interactive auth")
	}

	token, err := app.readSecretLine("Paste setup token: ")
	if err != nil {
		return "", fmt.Errorf("read setup token: %w", err)
	}
	if strings.TrimSpace(token) == "" {
		return "", usageError("empty_token", "token cannot be empty")
	}
	return strings.TrimSpace(token), nil
}

func webAuthTokenURL(apiBaseURL string) (string, error) {
	normalizedAPIBaseURL, err := normalizeAPIBaseURL(apiBaseURL)
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(normalizedAPIBaseURL)
	if err != nil {
		return "", fmt.Errorf("parse api base url: %w", err)
	}

	host := webAuthHost(parsed.Hostname())
	if port := parsed.Port(); port != "" {
		host = net.JoinHostPort(host, port)
	}

	return (&url.URL{
		Scheme: parsed.Scheme,
		Host:   host,
		Path:   "/auth/cli-token",
	}).String(), nil
}

func webAuthHost(apiHost string) string {
	switch apiHost {
	case "api.chill.institute":
		return "chill.institute"
	case "staging-api.chill.institute":
		return "staging.chill.institute"
	default:
		return strings.TrimPrefix(apiHost, "api.")
	}
}

func newAuthLogoutCommand(app *appContext) *cobra.Command {
	var dryRun bool
	var rawRequest string

	command := &cobra.Command{
		Use:   "logout",
		Short: "Clear the stored auth token from local config",
		Example: strings.TrimSpace(`
chilly auth logout
chilly auth logout --dry-run --output json
chilly auth logout --json '{"clear_auth_token":true}' --output json
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := app.loadConfig()
			if err != nil {
				return err
			}
			request, err := resolveAuthLogoutInput(app, rawRequest, strings.TrimSpace(cfg.AuthToken) != "")
			if err != nil {
				return err
			}
			if dryRun {
				return app.writeLocalDryRunPreview("auth logout", request)
			}
			cfg.AuthToken = ""
			if err := app.saveConfig(cfg); err != nil {
				return fmt.Errorf("persist config: %w", err)
			}
			return app.writeJSONPayload(map[string]any{"status": "ok", "logged_out": true})
		},
	}

	command.Flags().StringVar(&rawRequest, "json", "", "raw JSON request body, or @- to read it from stdin")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "preview the local auth token removal without saving it")
	return command
}

func resolveAuthLogoutInput(app *appContext, rawRequest string, hadAuthToken bool) (map[string]any, error) {
	request := map[string]any{
		"clear_auth_token": true,
		"had_auth_token":   hadAuthToken,
	}

	if strings.TrimSpace(rawRequest) == "" {
		return request, nil
	}

	payload, err := app.decodeJSONObjectFlag(rawRequest, "--json")
	if err != nil {
		return nil, err
	}
	if value, ok := payload["clear_auth_token"]; ok {
		typed, ok := value.(bool)
		if !ok || !typed {
			return nil, usageError("invalid_json_payload", "--json payload field clear_auth_token must be true")
		}
	}
	return request, nil
}

func jsonMessage(raw []byte) any {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return map[string]any{"raw": string(raw)}
	}
	return value
}
