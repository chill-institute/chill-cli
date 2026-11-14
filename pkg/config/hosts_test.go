package config

import (
	"slices"
	"testing"
)

func TestAPIBaseURLsOrder(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		cfg  Config
		want []string
	}{
		{name: "default", cfg: Default(), want: []string{"https://api.chill.institute", "https://api.binge.institute"}},
		{name: "empty", cfg: Config{}, want: []string{"https://api.chill.institute", "https://api.binge.institute"}},
		{name: "remembered official host first", cfg: Config{APIBaseURL: "https://api.chill.institute", LastWorkingAPIBaseURL: "https://api.binge.institute"}, want: []string{"https://api.binge.institute", "https://api.chill.institute"}},
		{name: "custom single host", cfg: Config{APIBaseURL: "https://api.example.test"}, want: []string{"https://api.example.test"}},
		{name: "custom single host ignores remembered official host", cfg: Config{APIBaseURL: "https://api.example.test", LastWorkingAPIBaseURL: "https://api.binge.institute"}, want: []string{"https://api.example.test"}},
		{name: "custom list", cfg: Config{APIBaseURL: "https://a.example.test, https://b.example.test/"}, want: []string{"https://a.example.test", "https://b.example.test"}},
		{name: "custom list remembered", cfg: Config{APIBaseURL: "https://a.example.test,https://b.example.test,https://c.example.test", LastWorkingAPIBaseURL: "https://c.example.test"}, want: []string{"https://c.example.test", "https://a.example.test", "https://b.example.test"}},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.cfg.APIBaseURLs(); !slices.Equal(got, tc.want) {
				t.Fatalf("APIBaseURLs() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestWithLastWorkingAPIBaseURL(t *testing.T) {
	t.Parallel()

	remembered := Default().WithLastWorkingAPIBaseURL("https://api.binge.institute")
	if remembered.LastWorkingAPIBaseURL != "https://api.binge.institute" {
		t.Fatalf("LastWorkingAPIBaseURL = %q", remembered.LastWorkingAPIBaseURL)
	}
	if got := remembered.WithLastWorkingAPIBaseURL("https://api.chill.institute").LastWorkingAPIBaseURL; got != "" {
		t.Fatalf("primary host should clear the remembered host, got %q", got)
	}
	if got := remembered.WithLastWorkingAPIBaseURL("https://api.example.test").LastWorkingAPIBaseURL; got != "https://api.binge.institute" {
		t.Fatalf("unconfigured host should not be remembered, got %q", got)
	}
}
