package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runLiveCommand(t *testing.T, label string, args []string) map[string]any {
	t.Helper()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	exitCode := Run(args, strings.NewReader(""), stdout, stderr)
	if exitCode != int(exitCodeSuccess) {
		t.Fatalf("%s failed: exitCode = %d (%s)", label, exitCode, liveFailureContext(exitCode))
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("%s returned invalid JSON: %v", label, err)
	}
	return output
}

func liveFailureContext(code int) string {
	switch exitCode(code) {
	case exitCodeUsage:
		return "invalid command input"
	case exitCodeAuth:
		return "authentication failed"
	case exitCodeAPI:
		return "API request failed"
	default:
		return "command execution failed"
	}
}

func TestLiveLoginFailureDiagnostic(t *testing.T) {
	const marker = "synthetic-login-token-marker"
	if os.Getenv("CHILLY_TEST_DIAGNOSTIC_CHILD") == "1" {
		runLiveCommand(t, "auth login", []string{
			"--config", filepath.Join(t.TempDir(), "config.json"),
			"--api-url", "http://127.0.0.1:1",
			"auth", "login", "--token", marker, "--output", "json",
		})
		return
	}
	command := exec.Command(os.Args[0], "-test.run=^TestLiveLoginFailureDiagnostic$")
	command.Env = append(os.Environ(), "CHILLY_TEST_DIAGNOSTIC_CHILD=1")
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatal("expected synthetic login to fail")
	}
	if bytes.Contains(output, []byte(marker)) {
		t.Fatal("failure diagnostic contains the synthetic token")
	}
	for _, want := range []string{"auth login failed", "exitCode = 5", "command execution failed"} {
		if !bytes.Contains(output, []byte(want)) {
			t.Fatalf("missing safe diagnostic context %q", want)
		}
	}
}
