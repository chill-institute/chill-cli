package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/chill-institute/chill-cli/v2/internal/buildinfo"
)

func runVersionCommand(t *testing.T, info buildinfo.Info, output string, args ...string) string {
	t.Helper()
	restore := currentBuildInfo
	currentBuildInfo = func() buildinfo.Info { return info }
	t.Cleanup(func() { currentBuildInfo = restore })

	stdout := &bytes.Buffer{}
	command := newVersionCommand(&appContext{
		opts:   &appOptions{output: output},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs(args)
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	return stdout.String()
}

func TestVersionCommandOutputsBuildInfo(t *testing.T) {
	info := buildinfo.Info{Version: "v1.2.3", Commit: "abc1234", BuildDate: "2026-03-15T00:00:00Z"}
	for _, output := range []string{outputJSON, outputNDJSON} {
		t.Run(output, func(t *testing.T) {
			stdout := runVersionCommand(t, info, output)
			var got map[string]any
			if err := json.Unmarshal([]byte(stdout), &got); err != nil {
				t.Fatalf("json.Unmarshal() error = %v, stdout = %q", err, stdout)
			}
			want := map[string]any{"name": "chilly", "version": "v1.2.3", "commit": "abc1234", "build_date": "2026-03-15T00:00:00Z"}
			for key, value := range want {
				if got[key] != value {
					t.Fatalf("%s = %v, want %v", key, got[key], value)
				}
			}
		})
	}
}

func TestVersionCommandOutputsPrettyVersionLine(t *testing.T) {
	info := buildinfo.Info{Version: "0.1.5", Commit: "dacd5f16ad68251e65c87a0295a1992b12f00335", BuildDate: "2026-03-15T00:00:00Z"}
	if got := strings.TrimSpace(runVersionCommand(t, info, outputPretty)); got != "0.1.5 (dacd5f1)" {
		t.Fatalf("stdout = %q", got)
	}
}

func TestVersionCommandFiltersFields(t *testing.T) {
	info := buildinfo.Info{Version: "v1.2.3", Commit: "abc1234", BuildDate: "2026-03-15T00:00:00Z"}
	stdout := runVersionCommand(t, info, outputPretty, "--fields", "version")

	var got map[string]any
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(got) != 1 || got["version"] != "v1.2.3" {
		t.Fatalf("output = %#v, want only version", got)
	}
}
