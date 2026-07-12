package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/chill-institute/chill-cli/internal/rpc"
)

func TestCLIErrorErrorUsesMessage(t *testing.T) {
	t.Parallel()

	err := &cliError{Message: "boom"}
	if err.Error() != "boom" {
		t.Fatalf("Error() = %q", err.Error())
	}
}

func TestCLIErrorErrorFallsBackToWrappedErrorAndDefault(t *testing.T) {
	t.Parallel()

	wrapped := &cliError{Err: errors.New("wrapped")}
	if wrapped.Error() != "wrapped" {
		t.Fatalf("Error() = %q, want wrapped error", wrapped.Error())
	}

	empty := &cliError{}
	if empty.Error() != "cli error" {
		t.Fatalf("Error() = %q, want default message", empty.Error())
	}
}

func TestLooksLikeUsageError(t *testing.T) {
	t.Parallel()

	if !looksLikeUsageError(usageError("unknown_flag", "unknown flag: --bad")) {
		t.Fatal("expected usage-like error")
	}
	if !looksLikeUsageError(usageError("unknown_shorthand", "unknown shorthand flag: '1' in -1")) {
		t.Fatal("expected shorthand flag error to be usage-like")
	}
}

func TestClassifyErrorMapsAPIAndFallbackCases(t *testing.T) {
	t.Parallel()

	authClassified := classifyError(rpc.APIError{Code: "invalid_auth_token", Message: "nope", StatusCode: 401, RequestID: "req-1"})
	if authClassified.Kind != errorKindAuth || authClassified.Code != "invalid_auth_token" || authClassified.RequestID != "req-1" {
		t.Fatalf("auth classified = %#v", authClassified)
	}

	apiClassified := classifyError(rpc.APIError{Message: "boom", StatusCode: 500})
	if apiClassified.Kind != errorKindAPI || apiClassified.Code != "api_error" {
		t.Fatalf("api classified = %#v", apiClassified)
	}

	usageClassified := classifyError(errors.New("unknown command \"wat\""))
	if usageClassified.Kind != errorKindUsage {
		t.Fatalf("usage classified = %#v", usageClassified)
	}

	internalClassified := classifyError(errors.New("boom"))
	if internalClassified.Kind != errorKindInternal || internalClassified.Code != "internal_error" {
		t.Fatalf("internal classified = %#v", internalClassified)
	}
}

func TestWriteErrorPrefersJSONWhenRequested(t *testing.T) {
	t.Parallel()

	stderr := &bytes.Buffer{}
	app := &appContext{
		opts:   &appOptions{output: outputJSON},
		stderr: stderr,
	}

	writeError(app, rpc.APIError{Code: "nope", Message: "boom", StatusCode: 418, RequestID: "req-1"})
	if got := stderr.String(); got == "" || got[0] != '{' {
		t.Fatalf("stderr = %q, want json envelope", got)
	}
}

func TestWriteErrorEscapesControlsInPrettyMode(t *testing.T) {
	t.Parallel()

	stderr := &bytes.Buffer{}
	app := &appContext{
		opts:   &appOptions{output: outputPretty},
		stderr: stderr,
	}

	writeError(app, rpc.APIError{Message: "upstream\x1b[2J\nspoofed", StatusCode: 500})
	got := stderr.String()
	if strings.Contains(got, "\x1b") || strings.Contains(got, "\nspoofed") {
		t.Fatalf("stderr contains raw terminal control: %q", got)
	}
	if !strings.Contains(got, `upstream\x1b[2J\nspoofed`) {
		t.Fatalf("stderr = %q, want escaped controls", got)
	}
}

func TestWriteErrorPreservesControlTextInJSONData(t *testing.T) {
	t.Parallel()

	stderr := &bytes.Buffer{}
	app := &appContext{
		opts:   &appOptions{output: outputJSON},
		stderr: stderr,
	}

	writeError(app, rpc.APIError{Message: "upstream\x1b[2J\nspoofed", StatusCode: 500})
	if bytes.ContainsRune(stderr.Bytes(), '\x1b') {
		t.Fatalf("json stderr contains raw escape: %q", stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stderr.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	message, ok := payload["message"].(string)
	if !ok || !strings.Contains(message, "upstream\x1b[2J\nspoofed") {
		t.Fatalf("message = %#v, want original semantic text", payload["message"])
	}
}

func TestExitCodeForErrorMapsKinds(t *testing.T) {
	t.Parallel()

	if got := exitCodeForError(usageError("bad", "bad")); got != int(exitCodeUsage) {
		t.Fatalf("usage exit code = %d", got)
	}
	if got := exitCodeForError(authError("bad", "bad")); got != int(exitCodeAuth) {
		t.Fatalf("auth exit code = %d", got)
	}
	if got := exitCodeForError(rpc.APIError{Code: "nope", Message: "boom", StatusCode: 500}); got != int(exitCodeAPI) {
		t.Fatalf("api exit code = %d", got)
	}
	if got := exitCodeForError(errors.New("boom")); got != int(exitCodeInternal) {
		t.Fatalf("internal exit code = %d", got)
	}
}
