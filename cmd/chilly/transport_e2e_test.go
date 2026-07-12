package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type cliErrorEnvelope struct {
	Code       string `json:"code"`
	Kind       string `json:"kind"`
	Message    string `json:"message"`
	RequestID  string `json:"request_id"`
	StatusCode int    `json:"status_code"`
}

func TestCLITransportHardeningEndToEnd(t *testing.T) {
	binary := buildCLIBinary(t)

	t.Run("rejects cross-origin authenticated redirect", func(t *testing.T) {
		redirected := make(chan struct{}, 1)
		destination := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			redirected <- struct{}{}
			writer.WriteHeader(http.StatusNoContent)
		}))
		defer destination.Close()

		authHeaders := make(chan string, 1)
		source := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			authHeaders <- request.Header.Get("Authorization")
			writer.Header().Set("Location", destination.URL+"/redirected")
			writer.WriteHeader(http.StatusTemporaryRedirect)
		}))
		defer source.Close()

		configPath := filepath.Join(t.TempDir(), "config.json")
		configureCLIAuth(t, binary, configPath, source.URL)
		stdout, stderr, exitCode := runCLIBinary(t, binary,
			"--config", configPath,
			"--api-url", source.URL,
			"--output", "json",
			"whoami",
		)
		if exitCode != 5 {
			t.Fatalf("exit code = %d, want 5; stderr = %s", exitCode, stderr)
		}
		if strings.TrimSpace(stdout) != "" {
			t.Fatalf("stdout = %q, want empty", stdout)
		}
		if got := <-authHeaders; got != "Bearer test-token" {
			t.Fatalf("Authorization = %q", got)
		}
		select {
		case <-redirected:
			t.Fatal("cross-origin redirect target received a request")
		default:
		}

		envelope := decodeCLIError(t, stderr)
		if envelope.Kind != "internal" || envelope.Code != "internal_error" || !strings.Contains(envelope.Message, "refusing redirect that changes API origin") {
			t.Fatalf("error envelope = %#v", envelope)
		}
	})

	t.Run("classifies oversized auth response", func(t *testing.T) {
		authHeaders := make(chan string, 1)
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			authHeaders <- request.Header.Get("Authorization")
			writer.Header().Set("X-Request-Id", "req-e2e")
			writer.WriteHeader(http.StatusUnauthorized)
			_, _ = writer.Write([]byte(strings.Repeat("x", (64<<10)+1)))
		}))
		defer server.Close()

		configPath := filepath.Join(t.TempDir(), "config.json")
		configureCLIAuth(t, binary, configPath, server.URL)
		stdout, stderr, exitCode := runCLIBinary(t, binary,
			"--config", configPath,
			"--api-url", server.URL,
			"--output", "json",
			"whoami",
		)
		if exitCode != 3 {
			t.Fatalf("exit code = %d, want 3; stderr = %s", exitCode, stderr)
		}
		if strings.TrimSpace(stdout) != "" {
			t.Fatalf("stdout = %q, want empty", stdout)
		}
		if got := <-authHeaders; got != "Bearer test-token" {
			t.Fatalf("Authorization = %q", got)
		}

		envelope := decodeCLIError(t, stderr)
		if envelope.Kind != "auth" || envelope.Code != "api_error" || envelope.StatusCode != http.StatusUnauthorized || envelope.RequestID != "req-e2e" {
			t.Fatalf("error envelope = %#v", envelope)
		}
		if !strings.Contains(envelope.Message, "maximum 65536 bytes") || strings.Contains(stderr, strings.Repeat("x", 16)) {
			t.Fatalf("unsafe or unhelpful error message = %q", envelope.Message)
		}
	})

	t.Run("accepts success body above error limit", func(t *testing.T) {
		padding := strings.Repeat("y", 70<<10)
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(writer).Encode(map[string]any{"user_id": "user-e2e", "padding": padding})
		}))
		defer server.Close()

		configPath := filepath.Join(t.TempDir(), "config.json")
		configureCLIAuth(t, binary, configPath, server.URL)
		stdout, stderr, exitCode := runCLIBinary(t, binary,
			"--config", configPath,
			"--api-url", server.URL,
			"--output", "json",
			"whoami",
		)
		if exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr = %s", exitCode, stderr)
		}
		if strings.TrimSpace(stderr) != "" {
			t.Fatalf("stderr = %q, want empty", stderr)
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
			t.Fatalf("decode stdout: %v; stdout = %q", err, stdout)
		}
		if payload["user_id"] != "user-e2e" || payload["padding"] != padding {
			t.Fatalf("response payload did not survive CLI round trip")
		}
	})
}

func buildCLIBinary(t *testing.T) string {
	t.Helper()
	binaryName := "chilly"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binary := filepath.Join(t.TempDir(), binaryName)
	command := exec.Command("go", "build", "-o", binary, ".")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	return binary
}

func configureCLIAuth(t *testing.T, binary string, configPath string, apiURL string) {
	t.Helper()
	stdout, stderr, exitCode := runCLIBinary(t, binary,
		"--config", configPath,
		"--api-url", apiURL,
		"--output", "json",
		"auth", "login",
		"--token", "test-token",
		"--skip-verify",
	)
	if exitCode != 0 {
		t.Fatalf("configure auth exit code = %d; stdout = %s; stderr = %s", exitCode, stdout, stderr)
	}
}

func runCLIBinary(t *testing.T, binary string, args ...string) (string, string, int) {
	t.Helper()
	command := exec.Command(binary, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	if err == nil {
		return stdout.String(), stderr.String(), 0
	}
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		t.Fatalf("run CLI: %v", err)
	}
	return stdout.String(), stderr.String(), exitError.ExitCode()
}

func decodeCLIError(t *testing.T, stderr string) cliErrorEnvelope {
	t.Helper()
	var envelope cliErrorEnvelope
	if err := json.Unmarshal([]byte(stderr), &envelope); err != nil {
		t.Fatalf("decode error envelope: %v; stderr = %q", err, stderr)
	}
	return envelope
}
