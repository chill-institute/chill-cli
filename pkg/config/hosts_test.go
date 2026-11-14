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
		{name: "custom single host", cfg: Config{APIBaseURL: "https://api.example.test"}, want: []string{"https://api.example.test"}},
		{name: "custom list", cfg: Config{APIBaseURL: "https://a.example.test, https://b.example.test/"}, want: []string{"https://a.example.test", "https://b.example.test"}},
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
