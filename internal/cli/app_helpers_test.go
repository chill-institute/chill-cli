package cli

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chill-institute/chill-cli/internal/config"
)

func TestNewAppContextDefaults(t *testing.T) {
	t.Parallel()

	opts := &appOptions{output: outputJSON}
	app := newAppContext(opts)

	if app.opts != opts {
		t.Fatal("newAppContext() did not keep options pointer")
	}
	if app.stdin != os.Stdin || app.stdout != os.Stdout || app.stderr != os.Stderr {
		t.Fatal("newAppContext() did not wire stdio defaults")
	}
	if app.openURL == nil || app.readSecret == nil || app.isTerminal == nil || app.isInputTerminal == nil || app.newTicker == nil {
		t.Fatal("newAppContext() left helper hooks nil")
	}

	ticker := app.newTicker(time.Millisecond)
	if ticker.C() == nil {
		t.Fatal("ticker.C() = nil")
	}
	ticker.Stop()
}

func TestConfigStoreUsesResolvedProfilePath(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	app := &appContext{opts: &appOptions{profile: "dev", output: outputJSON}}
	store, err := app.configStore()
	if err != nil {
		t.Fatalf("configStore() error = %v", err)
	}

	want := filepath.Join(configHome, "chilly", "profiles", "dev", "config.json")
	if store.Path() != want {
		t.Fatalf("store.Path() = %q, want %q", store.Path(), want)
	}
}

func TestConfigStoreUsesExplicitPath(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "custom.json")
	app := &appContext{opts: &appOptions{configPath: configPath, output: outputJSON}}
	store, err := app.configStore()
	if err != nil {
		t.Fatalf("configStore() error = %v", err)
	}
	if store.Path() != configPath {
		t.Fatalf("store.Path() = %q, want %q", store.Path(), configPath)
	}
}

func TestLoadConfigAppliesAPIOverride(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.json")
	app := &appContext{opts: &appOptions{configPath: configPath, output: outputJSON}}
	if err := app.saveConfig(config.Config{APIBaseURL: "https://api.old.example", AuthToken: " token-123 "}); err != nil {
		t.Fatalf("saveConfig() error = %v", err)
	}

	app.opts.apiURL = "https://api.chill.institute/"
	cfg, err := app.loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if cfg.APIBaseURL != "https://api.chill.institute" {
		t.Fatalf("cfg.APIBaseURL = %q, want %q", cfg.APIBaseURL, "https://api.chill.institute")
	}
	if cfg.AuthToken != "token-123" {
		t.Fatalf("cfg.AuthToken = %q, want %q", cfg.AuthToken, "token-123")
	}
}

func TestLoadConfigRejectsUnsafeAPIOverride(t *testing.T) {
	t.Parallel()

	app := &appContext{opts: &appOptions{configPath: filepath.Join(t.TempDir(), "config.json"), apiURL: "https://api.chill.institute/v4", output: outputJSON}}
	if _, err := app.loadConfig(); err == nil {
		t.Fatal("loadConfig() error = nil, want invalid api url error")
	}
}

func TestReadLinePropagatesPromptWriteError(t *testing.T) {
	t.Parallel()

	boom := errors.New("boom")
	app := &appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader("token\n"),
		stdout: &bytes.Buffer{},
		stderr: failingWriter{err: boom},
	}

	if _, err := app.readLine("token: "); !errors.Is(err, boom) {
		t.Fatalf("readLine() error = %v, want %v", err, boom)
	}
}

func TestWriterIsTerminalRejectsNonFileWriters(t *testing.T) {
	t.Parallel()

	if writerIsTerminal(&bytes.Buffer{}) {
		t.Fatal("writerIsTerminal(buffer) = true, want false")
	}
	if writerIsTerminal(failingWriter{}) {
		t.Fatal("writerIsTerminal(failingWriter) = true, want false")
	}
}

func TestReaderIsTerminalRejectsNonFileReaders(t *testing.T) {
	t.Parallel()

	if readerIsTerminal(strings.NewReader("token")) {
		t.Fatal("readerIsTerminal(strings.NewReader) = true, want false")
	}
}

func TestFileIsTerminalRejectsNilAndRegularFiles(t *testing.T) {
	t.Parallel()

	if fileIsTerminal(nil) {
		t.Fatal("fileIsTerminal(nil) = true, want false")
	}

	file, err := os.CreateTemp(t.TempDir(), "terminal-check")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}
	t.Cleanup(func() { _ = file.Close() })

	if fileIsTerminal(file) {
		t.Fatal("fileIsTerminal(tempfile) = true, want false")
	}
}

type failingWriter struct {
	err error
}

func (writer failingWriter) Write([]byte) (int, error) {
	if writer.err == nil {
		writer.err = io.ErrClosedPipe
	}
	return 0, writer.err
}
