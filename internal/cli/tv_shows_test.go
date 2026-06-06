package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/chill-institute/chill-cli/internal/config"
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

func TestNormalizeTVShowsSource(t *testing.T) {
	t.Parallel()

	for raw, expected := range map[string]string{
		"all-providers":             "TV_SHOWS_SOURCE_ALL_PROVIDERS",
		"TV_SHOWS_SOURCE_HULU":      "TV_SHOWS_SOURCE_HULU",
		"paramount_plus":            "TV_SHOWS_SOURCE_PARAMOUNT_PLUS",
		"amc":                       "TV_SHOWS_SOURCE_AMC_PLUS",
		"peacock":                   "TV_SHOWS_SOURCE_PEACOCK",
		"  amazon-prime-video \t\n": "TV_SHOWS_SOURCE_PRIME_VIDEO",
	} {
		value, err := normalizeTVShowsSource(raw)
		if err != nil {
			t.Fatalf("normalizeTVShowsSource(%q) error = %v", raw, err)
		}
		if value != expected {
			t.Fatalf("normalizeTVShowsSource(%q) = %q, want %q", raw, value, expected)
		}
	}

	if _, err := normalizeTVShowsSource("hulu?source=netflix"); err == nil {
		t.Fatal("normalizeTVShowsSource(unsafe) error = nil, want error")
	}
}
