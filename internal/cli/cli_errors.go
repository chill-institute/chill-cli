package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/chill-institute/chill-cli/internal/rpc"
)

type exitCode int

const (
	exitCodeSuccess  exitCode = 0
	exitCodeUsage    exitCode = 2
	exitCodeAuth     exitCode = 3
	exitCodeAPI      exitCode = 4
	exitCodeInternal exitCode = 5
)

type errorKind string

const (
	errorKindUsage    errorKind = "usage"
	errorKindAuth     errorKind = "auth"
	errorKindAPI      errorKind = "api"
	errorKindInternal errorKind = "internal"
)

type cliError struct {
	Kind       errorKind
	Code       string
	Message    string
	RequestID  string
	StatusCode int
	Err        error
}

func (err *cliError) Error() string {
	if strings.TrimSpace(err.Message) != "" {
		return err.Message
	}
	if err.Err != nil {
		return err.Err.Error()
	}
	return "cli error"
}

func (err *cliError) Unwrap() error {
	return err.Err
}

func usageError(code, format string, args ...any) error {
	return &cliError{Kind: errorKindUsage, Code: code, Message: fmt.Sprintf(format, args...)}
}

func authError(code, format string, args ...any) error {
	return &cliError{Kind: errorKindAuth, Code: code, Message: fmt.Sprintf(format, args...)}
}

func wrapInternalError(code, message string, err error) error {
	if err == nil {
		return nil
	}
	return &cliError{Kind: errorKindInternal, Code: code, Message: message, Err: err}
}

func wrapUsageError(code string, err error) error {
	if err == nil {
		return nil
	}
	return &cliError{Kind: errorKindUsage, Code: code, Message: err.Error(), Err: err}
}

func classifyError(err error) *cliError {
	if err == nil {
		return nil
	}

	var typed *cliError
	if errors.As(err, &typed) {
		return typed
	}

	var apiErr rpc.APIError
	if errors.As(err, &apiErr) {
		kind := errorKindAPI
		if apiErr.StatusCode == 401 || apiErr.StatusCode == 403 || strings.TrimSpace(apiErr.Code) == "invalid_auth_token" {
			kind = errorKindAuth
		}
		code := strings.TrimSpace(apiErr.Code)
		if code == "" {
			code = "api_error"
		}
		return &cliError{
			Kind:       kind,
			Code:       code,
			Message:    err.Error(),
			RequestID:  apiErr.RequestID,
			StatusCode: apiErr.StatusCode,
			Err:        err,
		}
	}

	if looksLikeUsageError(err) {
		return &cliError{Kind: errorKindUsage, Code: "usage_error", Message: err.Error(), Err: err}
	}

	return &cliError{Kind: errorKindInternal, Code: "internal_error", Message: err.Error(), Err: err}
}

func looksLikeUsageError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(message, "unknown command") ||
		strings.Contains(message, "unknown flag") ||
		strings.Contains(message, "unknown shorthand flag") ||
		strings.Contains(message, "unknown help topic") ||
		strings.Contains(message, "requires at least") ||
		strings.Contains(message, "accepts ") ||
		strings.Contains(message, "flag needs an argument")
}

func exitCodeForError(err error) int {
	classified := classifyError(err)
	switch classified.Kind {
	case errorKindUsage:
		return int(exitCodeUsage)
	case errorKindAuth:
		return int(exitCodeAuth)
	case errorKindAPI:
		return int(exitCodeAPI)
	default:
		return int(exitCodeInternal)
	}
}

func writeError(app *appContext, err error) {
	classified := classifyError(err)
	if app == nil || app.stderr == nil {
		return
	}
	if wantsJSONOutput(app.opts.output) {
		payload := map[string]any{
			"code":    classified.Code,
			"message": classified.Message,
			"kind":    classified.Kind,
		}
		if strings.TrimSpace(classified.RequestID) != "" {
			payload["request_id"] = classified.RequestID
		}
		if classified.StatusCode > 0 {
			payload["status_code"] = classified.StatusCode
		}
		if encoded, marshalErr := json.Marshal(payload); marshalErr == nil {
			_, _ = fmt.Fprintln(app.stderr, string(encoded))
			return
		}
	}
	_, _ = fmt.Fprintln(app.stderr, sanitizeTerminalText(classified.Message))
}

func wantsJSONOutput(value string) bool {
	trimmed := strings.TrimSpace(value)
	return strings.EqualFold(trimmed, outputJSON) || strings.EqualFold(trimmed, outputNDJSON)
}

func runCommand(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	opts := &appOptions{output: outputPretty}
	app := newAppContext(opts)
	if stdin != nil {
		app.stdin = stdin
	}
	if stdout != nil {
		app.stdout = stdout
	}
	if stderr != nil {
		app.stderr = stderr
	}

	command := newRootCommand(app)
	seedOutputModeForEarlyErrors(app, args)
	command.SetArgs(args)
	if err := command.Execute(); err != nil {
		writeError(app, err)
		return exitCodeForError(err)
	}
	return int(exitCodeSuccess)
}

func seedOutputModeForEarlyErrors(app *appContext, args []string) {
	if app == nil || app.opts == nil {
		return
	}
	for index, arg := range args {
		if arg == "--output" && index+1 < len(args) {
			app.opts.output = args[index+1]
			return
		}
		if value, ok := strings.CutPrefix(arg, "--output="); ok {
			app.opts.output = value
			return
		}
	}
	if app.isTerminal != nil && !app.isTerminal(app.stdout) {
		app.opts.output = outputJSON
	}
}
