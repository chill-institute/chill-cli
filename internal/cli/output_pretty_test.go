package cli

import (
	"strings"
	"testing"
)

func TestRenderWhoamiPretty(t *testing.T) {
	t.Parallel()

	rendered, ok, err := renderWhoamiPretty(map[string]any{
		"username": "sample-user",
		"email":    "user@example.test",
		"user_id":  "123",
		"plan":     "pro",
	})
	if err != nil {
		t.Fatalf("renderWhoamiPretty() error = %v", err)
	}
	if !ok {
		t.Fatal("renderWhoamiPretty() ok = false, want true")
	}
	for _, fragment := range []string{"Username: sample-user", "Email: user@example.test", "User ID: 123", "Plan: pro"} {
		if !strings.Contains(stripANSI(rendered), fragment) {
			t.Fatalf("rendered = %q, want fragment %q", rendered, fragment)
		}
	}
}

func TestPrettyRenderersEscapeUntrustedControls(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		render func(any) (string, bool, error)
		value  any
		want   string
	}{
		{
			name:   "search result",
			render: renderSearchPretty,
			value: map[string]any{
				"query": "dune\nspoofed",
				"results": []any{map[string]any{
					"title": "Dune\x1b]52;c;payload\a",
				}},
			},
			want: `Dune\x1b]52;c;payload\a`,
		},
		{
			name:   "settings key and value",
			render: renderUserSettingsPretty,
			value: map[string]any{
				"catalog\x1b[2J": "movies\nspoofed",
			},
			want: `catalog\x1b[2J: movies\nspoofed`,
		},
		{
			name:   "transfer message",
			render: renderTransferPretty,
			value: map[string]any{
				"statusMessage": "queued\rreplaced",
			},
			want: `queued\rreplaced`,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			rendered, ok, err := testCase.render(testCase.value)
			if err != nil {
				t.Fatalf("render() error = %v", err)
			}
			if !ok {
				t.Fatal("render() ok = false, want true")
			}
			visible := stripANSI(rendered)
			if strings.ContainsAny(visible, "\x1b\r\a") {
				t.Fatalf("rendered contains raw terminal control: %q", visible)
			}
			if !strings.Contains(visible, testCase.want) {
				t.Fatalf("rendered = %q, want %q", visible, testCase.want)
			}
		})
	}
}

func TestRenderSearchPretty(t *testing.T) {
	t.Parallel()

	results := make([]any, 0, 11)
	for index := range 11 {
		results = append(results, map[string]any{
			"title":       "Result",
			"indexerName": "YTS",
			"size":        "1.4 GB",
			"seeders":     float64(index + 1),
			"peers":       float64(index + 2),
			"magnetLink":  "magnet:?xt=urn:btih:test",
		})
	}

	rendered, ok, err := renderSearchPretty(map[string]any{
		"query":   "dune",
		"results": results,
	})
	if err != nil {
		t.Fatalf("renderSearchPretty() error = %v", err)
	}
	if !ok {
		t.Fatal("renderSearchPretty() ok = false, want true")
	}
	for _, fragment := range []string{
		"Results: 11",
		"Query: dune",
		"1. Result",
		"Indexer: YTS",
		"Seeds: 1",
		"Peers: 2",
		"... 1 more results omitted.",
	} {
		if !strings.Contains(stripANSI(rendered), fragment) {
			t.Fatalf("rendered = %q, want fragment %q", rendered, fragment)
		}
	}
}

func TestRenderSearchPrettyEmptyResults(t *testing.T) {
	t.Parallel()

	rendered, ok, err := renderSearchPretty(map[string]any{
		"results": []any{},
	})
	if err != nil {
		t.Fatalf("renderSearchPretty() error = %v", err)
	}
	if !ok {
		t.Fatal("renderSearchPretty() ok = false, want true")
	}
	if !strings.Contains(rendered, "No results.") {
		t.Fatalf("rendered = %q", rendered)
	}
}

func TestRenderMoviesPretty(t *testing.T) {
	t.Parallel()

	rendered, ok, err := renderMoviesPretty(map[string]any{
		"movies": []any{
			map[string]any{
				"title":      "Dune",
				"year":       "2021",
				"imdbRating": float64(8),
				"tmdbRating": float64(7.5),
			},
		},
	})
	if err != nil {
		t.Fatalf("renderMoviesPretty() error = %v", err)
	}
	if !ok {
		t.Fatal("renderMoviesPretty() ok = false, want true")
	}
	for _, fragment := range []string{"Movies: 1", "1. Dune (2021)", "IMDb: 8", "TMDb: 7.5"} {
		if !strings.Contains(stripANSI(rendered), fragment) {
			t.Fatalf("rendered = %q, want fragment %q", rendered, fragment)
		}
	}
}

func TestRenderMoviesPrettyEmpty(t *testing.T) {
	t.Parallel()

	rendered, ok, err := renderMoviesPretty(map[string]any{"movies": []any{}})
	if err != nil {
		t.Fatalf("renderMoviesPretty(empty) error = %v", err)
	}
	if !ok {
		t.Fatal("renderMoviesPretty(empty) ok = false, want true")
	}
	if !strings.Contains(rendered, "No movies yet.") {
		t.Fatalf("rendered = %q", rendered)
	}
}

func TestRenderTVShowsPretty(t *testing.T) {
	t.Parallel()

	rendered, ok, err := renderTVShowsPretty(map[string]any{
		"shows": []any{
			map[string]any{
				"title":    "The Pitt",
				"year":     "2025",
				"imdbId":   "tt31938062",
				"rating":   float64(8.8),
				"status":   "TV_SHOW_STATUS_RETURNING",
				"networks": []any{"HBO Max"},
			},
		},
	})
	if err != nil {
		t.Fatalf("renderTVShowsPretty() error = %v", err)
	}
	if !ok {
		t.Fatal("renderTVShowsPretty() ok = false, want true")
	}
	for _, fragment := range []string{"TV Shows: 1", "1. The Pitt (2025)", "IMDb ID: tt31938062", "Networks: HBO Max"} {
		if !strings.Contains(stripANSI(rendered), fragment) {
			t.Fatalf("rendered = %q, want fragment %q", rendered, fragment)
		}
	}
}

func TestRenderTVShowDetailPretty(t *testing.T) {
	t.Parallel()

	rendered, ok, err := renderTVShowDetailPretty(map[string]any{
		"show": map[string]any{
			"title":       "The Pitt",
			"year":        "2025",
			"imdbId":      "tt31938062",
			"rating":      float64(8.8),
			"status":      "TV_SHOW_STATUS_RETURNING",
			"seasonCount": float64(1),
			"networks":    []any{"HBO Max"},
			"genres":      []any{"Drama"},
			"overview":    "A tense hospital drama",
		},
		"seasons": []any{
			map[string]any{"seasonNumber": float64(1), "name": "Season 1", "episodeCount": float64(10)},
		},
	})
	if err != nil {
		t.Fatalf("renderTVShowDetailPretty() error = %v", err)
	}
	if !ok {
		t.Fatal("renderTVShowDetailPretty() ok = false, want true")
	}
	for _, fragment := range []string{"The Pitt (2025)", "IMDb ID: tt31938062", "Networks: HBO Max", "Seasons: 1"} {
		if !strings.Contains(stripANSI(rendered), fragment) {
			t.Fatalf("rendered = %q, want fragment %q", rendered, fragment)
		}
	}
}

func TestRenderTVShowSeasonPretty(t *testing.T) {
	t.Parallel()

	rendered, ok, err := renderTVShowSeasonPretty(map[string]any{
		"season": map[string]any{
			"name":         "Season 1",
			"seasonNumber": float64(1),
			"episodeCount": float64(2),
		},
		"episodes": []any{
			map[string]any{"episodeNumber": float64(1), "name": "Pilot", "runtime": float64(55)},
			map[string]any{"episodeNumber": float64(2), "name": "Second", "runtime": float64(54)},
		},
	})
	if err != nil {
		t.Fatalf("renderTVShowSeasonPretty() error = %v", err)
	}
	if !ok {
		t.Fatal("renderTVShowSeasonPretty() ok = false, want true")
	}
	for _, fragment := range []string{"Season 1", "Episodes: 2", "1. E1 Pilot", "2. E2 Second"} {
		if !strings.Contains(stripANSI(rendered), fragment) {
			t.Fatalf("rendered = %q, want fragment %q", rendered, fragment)
		}
	}
}

func TestRenderTVShowEpisodeDownloadPretty(t *testing.T) {
	t.Parallel()

	rendered, ok, err := renderTVShowEpisodeDownloadPretty(map[string]any{
		"searchQuery": "The Pitt S01E01",
		"download": map[string]any{
			"title":      "The.Pitt.S01E01.1080p.WEB-DL",
			"indexer":    "knaben",
			"resolution": "1080p",
		},
	})
	if err != nil {
		t.Fatalf("renderTVShowEpisodeDownloadPretty() error = %v", err)
	}
	if !ok {
		t.Fatal("renderTVShowEpisodeDownloadPretty() ok = false, want true")
	}
	for _, fragment := range []string{"Search Query: The Pitt S01E01", "Title: The.Pitt.S01E01.1080p.WEB-DL", "Indexer: knaben"} {
		if !strings.Contains(stripANSI(rendered), fragment) {
			t.Fatalf("rendered = %q, want fragment %q", rendered, fragment)
		}
	}
}

func TestRenderTVShowEpisodeDownloadPrettyEmpty(t *testing.T) {
	t.Parallel()

	rendered, ok, err := renderTVShowEpisodeDownloadPretty(map[string]any{
		"searchQuery": "The Pitt S01E02",
	})
	if err != nil {
		t.Fatalf("renderTVShowEpisodeDownloadPretty(empty) error = %v", err)
	}
	if !ok {
		t.Fatal("renderTVShowEpisodeDownloadPretty(empty) ok = false, want true")
	}
	if !strings.Contains(rendered, "No matching episode download found.") {
		t.Fatalf("rendered = %q", rendered)
	}
}

func TestRenderTVShowSeasonDownloadsPretty(t *testing.T) {
	t.Parallel()

	rendered, ok, err := renderTVShowSeasonDownloadsPretty(map[string]any{
		"seasonSearchQuery": "The Pitt S01",
		"seasonPack": map[string]any{
			"title":   "The.Pitt.S01.COMPLETE.1080p.WEB-DL",
			"indexer": "knaben",
		},
		"episodes": []any{
			map[string]any{
				"episodeNumber": float64(1),
				"searchQuery":   "The Pitt S01E01",
				"download": map[string]any{
					"title": "The.Pitt.S01E01.1080p.WEB-DL",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("renderTVShowSeasonDownloadsPretty() error = %v", err)
	}
	if !ok {
		t.Fatal("renderTVShowSeasonDownloadsPretty() ok = false, want true")
	}
	for _, fragment := range []string{"Season Search Query: The Pitt S01", "Season Pack", "Episode Results: 1", "Episode 1"} {
		if !strings.Contains(stripANSI(rendered), fragment) {
			t.Fatalf("rendered = %q, want fragment %q", rendered, fragment)
		}
	}
}

func TestRenderUserIndexersPretty(t *testing.T) {
	t.Parallel()

	rendered, ok, err := renderUserIndexersPretty(map[string]any{
		"indexers": []any{
			map[string]any{
				"id":      "yts",
				"name":    "YTS",
				"enabled": true,
				"status":  "INDEXER_STATUS_READY",
			},
			map[string]any{
				"id":      "tbp",
				"name":    "The Pirate Bay",
				"enabled": true,
				"status":  "INDEXER_STATUS_DEGRADED",
			},
		},
	})
	if err != nil {
		t.Fatalf("renderUserIndexersPretty() error = %v", err)
	}
	if !ok {
		t.Fatal("renderUserIndexersPretty() ok = false, want true")
	}
	for _, fragment := range []string{"Indexers: 2", "1. YTS [yts]", "Enabled: enabled", "Indexer Status: ready", "2. The Pirate Bay [tbp]", "Indexer Status: degraded"} {
		if !strings.Contains(stripANSI(rendered), fragment) {
			t.Fatalf("rendered = %q, want fragment %q", rendered, fragment)
		}
	}
}

func TestRenderUserSettingsPretty(t *testing.T) {
	t.Parallel()

	rendered, ok, err := renderUserSettingsPretty(map[string]any{
		"zeta":  true,
		"alpha": "value",
	})
	if err != nil {
		t.Fatalf("renderUserSettingsPretty() error = %v", err)
	}
	if !ok {
		t.Fatal("renderUserSettingsPretty() ok = false, want true")
	}
	stripped := stripANSI(rendered)
	if !strings.Contains(stripped, "User Settings") || !strings.Contains(stripped, "alpha: value") || !strings.Contains(stripped, "zeta: true") {
		t.Fatalf("rendered = %q", rendered)
	}
	if strings.Index(stripped, "alpha: value") > strings.Index(stripped, "zeta: true") {
		t.Fatalf("rendered = %q, want sorted keys", rendered)
	}
}

func TestRenderUserSettingsPrettyNestedDomains(t *testing.T) {
	t.Parallel()

	rendered, ok, err := renderUserSettingsPretty(map[string]any{
		"search": map[string]any{
			"sortBy":             "SORT_BY_SEEDERS",
			"filterNastyResults": true,
		},
		"download": map[string]any{
			"folderId": "42",
		},
	})
	if err != nil {
		t.Fatalf("renderUserSettingsPretty() error = %v", err)
	}
	if !ok {
		t.Fatal("renderUserSettingsPretty() ok = false, want true")
	}
	stripped := stripANSI(rendered)
	for _, fragment := range []string{"User Settings", "download:", "folderId: 42", "search:", "filterNastyResults: true", "sortBy: SORT_BY_SEEDERS"} {
		if !strings.Contains(stripped, fragment) {
			t.Fatalf("rendered = %q, want fragment %q", rendered, fragment)
		}
	}
}

func TestRenderDownloadFolderPretty(t *testing.T) {
	t.Parallel()

	rendered, ok, err := renderDownloadFolderPretty(map[string]any{
		"folder": map[string]any{
			"name":      "Movies",
			"id":        "42",
			"file_type": "FOLDER",
			"is_shared": true,
		},
	})
	if err != nil {
		t.Fatalf("renderDownloadFolderPretty() error = %v", err)
	}
	if !ok {
		t.Fatal("renderDownloadFolderPretty() ok = false, want true")
	}
	for _, fragment := range []string{"Download Folder", "Name: Movies", "ID: 42", "Type: FOLDER", "Shared: true"} {
		if !strings.Contains(stripANSI(rendered), fragment) {
			t.Fatalf("rendered = %q, want fragment %q", rendered, fragment)
		}
	}
}

func TestRenderFolderPretty(t *testing.T) {
	t.Parallel()

	rendered, ok, err := renderFolderPretty(map[string]any{
		"parent": map[string]any{
			"name":     "Movies",
			"id":       "42",
			"fileType": "FOLDER",
		},
		"files": []any{
			map[string]any{
				"name":      "Dune.mkv",
				"id":        "99",
				"file_type": "VIDEO",
			},
		},
	})
	if err != nil {
		t.Fatalf("renderFolderPretty() error = %v", err)
	}
	if !ok {
		t.Fatal("renderFolderPretty() ok = false, want true")
	}
	for _, fragment := range []string{"Folder", "Name: Movies", "Children: 1", "1. Dune.mkv [VIDEO]", "ID: 99"} {
		if !strings.Contains(stripANSI(rendered), fragment) {
			t.Fatalf("rendered = %q, want fragment %q", rendered, fragment)
		}
	}
}

func TestRenderTransferPretty(t *testing.T) {
	t.Parallel()

	rendered, ok, err := renderTransferPretty(map[string]any{
		"status": "queued",
		"transfer": map[string]any{
			"id":                   "42",
			"name":                 "Dune",
			"status":               "COMPLETED",
			"percentDone":          float64(100),
			"isFinished":           true,
			"statusMessage":        "done",
			"errorMessage":         "none",
			"fileUrl":              "https://put.io/files/42",
			"fileId":               "file-42",
			"estimatedTimeSeconds": float64(12),
		},
	})
	if err != nil {
		t.Fatalf("renderTransferPretty() error = %v", err)
	}
	if !ok {
		t.Fatal("renderTransferPretty() ok = false, want true")
	}
	for _, fragment := range []string{
		"Transfer",
		"Request Status: queued",
		"ID: 42",
		"Name: Dune",
		"Status: COMPLETED",
		"Progress: 100%",
		"Finished: true",
		"Message: done",
		"Error: none",
		"File URL: https://put.io/files/42",
		"File ID: file-42",
		"ETA Seconds: 12",
	} {
		if !strings.Contains(stripANSI(rendered), fragment) {
			t.Fatalf("rendered = %q, want fragment %q", rendered, fragment)
		}
	}
}

func TestRenderDoctorPretty(t *testing.T) {
	t.Parallel()

	rendered, ok, err := renderDoctorPretty(map[string]any{
		"status": "ok",
		"build": map[string]any{
			"version":    "v1.0.0",
			"commit":     "abc1234",
			"build_date": "2026-03-17",
			"dev":        true,
		},
		"config": map[string]any{
			"profile": "dev",
			"path":    "/tmp/config.json",
			"exists":  true,
		},
		"api": map[string]any{
			"base_url": "https://api.chill.institute",
		},
		"auth": map[string]any{
			"configured": true,
			"status":     "ok",
			"request_id": "req-123",
			"code":       "ok",
			"message":    "verified",
			"user": map[string]any{
				"username": "sample-user",
				"email":    "user@example.test",
				"user_id":  "123",
				"plan":     "pro",
			},
		},
	})
	if err != nil {
		t.Fatalf("renderDoctorPretty() error = %v", err)
	}
	if !ok {
		t.Fatal("renderDoctorPretty() ok = false, want true")
	}
	for _, fragment := range []string{
		"Doctor",
		"Status: ok",
		"Build",
		"Version: v1.0.0",
		"Config",
		"Profile: dev",
		"API",
		"Base URL: https://api.chill.institute",
		"Auth",
		"Configured: true",
		"Username: sample-user",
	} {
		if !strings.Contains(stripANSI(rendered), fragment) {
			t.Fatalf("rendered = %q, want fragment %q", rendered, fragment)
		}
	}
}

func TestPrettyHelpers(t *testing.T) {
	t.Parallel()

	payload := map[string]any{
		"name":        " Dune ",
		"count":       float64(7),
		"fraction":    float64(7.5),
		"enabled":     true,
		"is_shared":   false,
		"created_at":  "2026-03-17",
		"nested_name": "nested",
	}

	if status := prettyIndexerStatus("INDEXER_STATUS_READY"); status != "ready" {
		t.Fatalf("prettyIndexerStatus() = %q", status)
	}
	if status := prettyIndexerStatus("INDEXER_STATUS_DEGRADED"); status != "degraded" {
		t.Fatalf("prettyIndexerStatus(degraded) = %q", status)
	}
	if status := prettyIndexerStatus(""); status != "" {
		t.Fatalf("prettyIndexerStatus(empty) = %q", status)
	}
	if got, ok := stringValue(payload, "name"); !ok || got != "Dune" {
		t.Fatalf("stringValue(name) = %q, %t", got, ok)
	}
	if got, ok := stringValue(payload, "count"); !ok || got != "7" {
		t.Fatalf("stringValue(count) = %q, %t", got, ok)
	}
	if got, ok := stringValue(payload, "enabled"); !ok || got != "true" {
		t.Fatalf("stringValue(enabled) = %q, %t", got, ok)
	}
	if got := formatNumeric(7.5); got != "7.5" {
		t.Fatalf("formatNumeric() = %q", got)
	}
	if got := prettyValue([]any{"a", float64(2), true}); got != "a, 2, true" {
		t.Fatalf("prettyValue(slice) = %q", got)
	}

	lines := []string{}
	lines = appendIfString(lines, "Name", payload, "name")
	lines = appendDoctorLine(lines, "Enabled", payload, "enabled")
	lines = appendDetailLine(lines, "Created", payload, "created_at")
	if len(lines) != 3 {
		t.Fatalf("lines = %#v", lines)
	}

	fileLines := prettyUserFileLines(map[string]any{
		"name":       "Movies",
		"id":         "42",
		"file_type":  "FOLDER",
		"created_at": "2026-03-17",
		"is_shared":  false,
	})
	if len(fileLines) != 5 {
		t.Fatalf("prettyUserFileLines() = %#v", fileLines)
	}
	if got := firstPresent(map[string]any{"other": 1, "id": "42"}, "id", "other"); got != "42" {
		t.Fatalf("firstPresent() = %#v", got)
	}
	if got := min(2, 3); got != 2 {
		t.Fatalf("min() = %d", got)
	}
}
