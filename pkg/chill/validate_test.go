package chill

import (
	"errors"
	"testing"
)

func TestValidatorsRejectHostileInput(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		run  func() error
		code string
	}{
		{"indexer traversal", func() error { _, err := NormalizeIndexerID("../yts"); return err }, "invalid_indexer_id"},
		{"indexer encoded", func() error { _, err := NormalizeIndexerID("yts%2f"); return err }, "invalid_indexer_id"},
		{"indexer empty", func() error { _, err := NormalizeIndexerID("  "); return err }, "missing_indexer_id"},
		{"indexer control", func() error { _, err := NormalizeIndexerID("bad\x00id"); return err }, "invalid_indexer_id"},
		{"indexer path", func() error { _, err := NormalizeIndexerID("bad/id"); return err }, "invalid_indexer_id"},
		{"indexer query", func() error { _, err := NormalizeIndexerID("bad?id"); return err }, "invalid_indexer_id"},
		{"url scheme", func() error { _, err := NormalizeTransferURL("ftp://example.com/a"); return err }, "invalid_url"},
		{"url no scheme", func() error { _, err := NormalizeTransferURL("not-a-url"); return err }, "invalid_url"},
		{"url control", func() error { _, err := NormalizeTransferURL("bad\x00url"); return err }, "invalid_url"},
		{"url no host", func() error { _, err := NormalizeTransferURL("https://:443/file.torrent"); return err }, "invalid_url"},
		{"url whitespace", func() error { _, err := NormalizeTransferURL("http://example.com/a b"); return err }, "invalid_url"},
		{"magnet whitespace", func() error { _, err := NormalizeTransferURL("magnet:?xt=urn:btih:test&dn=My Movie"); return err }, "invalid_url"},
		{"url empty", func() error { _, err := NormalizeTransferURL(""); return err }, "missing_url"},
		{"movie source", func() error { _, err := NormalizeMovieSource("letterboxd"); return err }, "invalid_movies_source"},
		{"tv source required", func() error { _, err := NormalizeTVShowsSource("", false); return err }, "invalid_tv_shows_source"},
		{"tv source unknown", func() error { _, err := NormalizeTVShowsSource("tubi", true); return err }, "invalid_tv_shows_source"},
		{"tv source query", func() error { _, err := NormalizeTVShowsSource("hulu?source=netflix", true); return err }, "invalid_tv_shows_source"},
		{"imdb missing", func() error { _, err := NormalizeIMDbID(""); return err }, "missing_imdb_id"},
		{"imdb prefix", func() error { _, err := NormalizeIMDbID("nm0000001"); return err }, "invalid_imdb_id"},
		{"imdb short", func() error { _, err := NormalizeIMDbID("tt123"); return err }, "invalid_imdb_id"},
		{"imdb letters", func() error { _, err := NormalizeIMDbID("ttabc"); return err }, "invalid_imdb_id"},
		{"imdb query", func() error { _, err := NormalizeIMDbID("tt0944947?x=1"); return err }, "invalid_imdb_id"},
		{"imdb encoded", func() error { _, err := NormalizeIMDbID("tt0944947%2f"); return err }, "invalid_imdb_id"},
		{"ordinal zero", func() error { _, err := NormalizeEpisodeOrdinal("0", "season"); return err }, "invalid_season_number"},
		{"ordinal not integer", func() error { _, err := NormalizeEpisodeOrdinal("abc", "episode"); return err }, "invalid_episode_number"},
		{"ordinal missing", func() error { _, err := NormalizeEpisodeOrdinal("", "episode"); return err }, "missing_episode_number"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.run()
			var validation *ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("error = %v, want *ValidationError", err)
			}
			if validation.Code != tc.code {
				t.Fatalf("code = %q, want %q", validation.Code, tc.code)
			}
		})
	}
}

func TestValidatorsNormalizeAcceptedInput(t *testing.T) {
	t.Parallel()
	check := func(got string, err error, want string) {
		t.Helper()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	}
	got, err := NormalizeIndexerID(" yts ")
	check(got, err, "yts")
	got, err = NormalizeTransferURL(" MAGNET:?xt=urn:btih:abc ")
	check(got, err, "MAGNET:?xt=urn:btih:abc")
	got, err = NormalizeTransferURL("https://example.com/file.torrent")
	check(got, err, "https://example.com/file.torrent")
	got, err = NormalizeTransferURL(" http://example.com/file.torrent ")
	check(got, err, "http://example.com/file.torrent")
	got, err = NormalizeMovieSource("IMDb Top 250")
	check(got, err, "MOVIES_SOURCE_IMDB_TOP_250")
	for raw, want := range map[string]string{
		"Apple TV":                  "TV_SHOWS_SOURCE_APPLE_TV_PLUS",
		"all-providers":             "TV_SHOWS_SOURCE_ALL_PROVIDERS",
		"TV_SHOWS_SOURCE_HULU":      "TV_SHOWS_SOURCE_HULU",
		"paramount_plus":            "TV_SHOWS_SOURCE_PARAMOUNT_PLUS",
		"amc":                       "TV_SHOWS_SOURCE_AMC_PLUS",
		"peacock":                   "TV_SHOWS_SOURCE_PEACOCK",
		"  amazon-prime-video \t\n": "TV_SHOWS_SOURCE_PRIME_VIDEO",
	} {
		got, err = NormalizeTVShowsSource(raw, false)
		check(got, err, want)
	}
	got, err = NormalizeTVShowsSource("", true)
	check(got, err, "")
	got, err = NormalizeIMDbID(" TT0111161 ")
	check(got, err, "tt0111161")
	ordinal, err := NormalizeEpisodeOrdinal(" 7 ", "episode")
	if err != nil || ordinal != 7 {
		t.Fatalf("ordinal = %d, %v; want 7", ordinal, err)
	}
}
