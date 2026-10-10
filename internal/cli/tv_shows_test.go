package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/chill-institute/chill-cli/v2/pkg/config"
)

func TestRunTVShowsRejectsBadFieldsAndMissingAuth(t *testing.T) {
	t.Parallel()

	app := &appContext{
		opts:   &appOptions{configPath: filepath.Join(t.TempDir(), "config.json"), output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: &strings.Builder{},
		stderr: &strings.Builder{},
	}

	if err := runTVShows(app, "[", ""); err == nil {
		t.Fatal("runTVShows(invalid fields) error = nil, want error")
	}

	if err := app.saveConfig(config.Config{APIBaseURL: "https://api.chill.institute"}); err != nil {
		t.Fatalf("saveConfig() error = %v", err)
	}
	if err := runTVShows(app, "", ""); err == nil {
		t.Fatal("runTVShows(missing auth) error = nil, want error")
	}
}
