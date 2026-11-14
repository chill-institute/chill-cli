package cli

import (
	"fmt"
	"sort"
	"strings"
)

const maxPrettyListItems = 10

func renderWhoamiPretty(value any) (string, bool, error) {
	profile, ok := value.(map[string]any)
	if !ok {
		return "", false, nil
	}

	lines := make([]string, 0, 6)
	lines = appendIfString(lines, "Username", profile, "username")
	lines = appendIfString(lines, "Email", profile, "email")
	lines = appendIfString(lines, "User ID", profile, "userId", "user_id")
	lines = appendIfString(lines, "Plan", profile, "plan")

	if len(lines) == 0 {
		return "", false, nil
	}
	return strings.Join(lines, "\n"), true, nil
}

func renderSearchPretty(value any) (string, bool, error) {
	payload, ok := value.(map[string]any)
	if !ok {
		return "", false, nil
	}

	results, ok := payload["results"].([]any)
	if !ok {
		return "", false, nil
	}

	lines := []string{
		bold(fmt.Sprintf("Results: %d", len(results))),
	}
	if query, ok := stringValue(payload, "query"); ok {
		lines = append(lines, fmt.Sprintf("%s %s", dim("Query:"), query))
	}

	if len(results) == 0 {
		lines = append(lines, "", "No results.")
		return strings.Join(lines, "\n"), true, nil
	}

	for index, item := range results[:min(len(results), maxPrettyListItems)] {
		result, ok := item.(map[string]any)
		if !ok {
			continue
		}

		title := firstString(result, "title", "name")
		if title == "" {
			title = fmt.Sprintf("Result %d", index+1)
		}

		lines = append(lines, "")
		lines = append(lines, bold(fmt.Sprintf("%d. %s", index+1, title)))
		lines = appendDetailLine(lines, "Indexer", result, "indexerName", "indexer_name")
		lines = appendDetailLine(lines, "Size", result, "size")
		lines = appendDetailLine(lines, "Seeds", result, "seeders", "seeds")
		lines = appendDetailLine(lines, "Peers", result, "peers", "leechers")
		lines = appendDetailLine(lines, "Magnet", result, "magnetLink", "magnet_link")
	}
	if len(results) > maxPrettyListItems {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("... %d more results omitted. Use --output json or --fields for the full payload.", len(results)-maxPrettyListItems))
	}

	return strings.Join(lines, "\n"), true, nil
}

func renderMoviesPretty(value any) (string, bool, error) {
	payload, ok := value.(map[string]any)
	if !ok {
		return "", false, nil
	}

	movies, ok := payload["movies"].([]any)
	if !ok {
		return "", false, nil
	}

	lines := []string{
		bold(fmt.Sprintf("Movies: %d", len(movies))),
	}
	if len(movies) == 0 {
		lines = append(lines, "", "No movies yet.")
		return strings.Join(lines, "\n"), true, nil
	}

	for index, item := range movies[:min(len(movies), maxPrettyListItems)] {
		movie, ok := item.(map[string]any)
		if !ok {
			continue
		}

		title := firstString(movie, "title", "name")
		if title == "" {
			title = fmt.Sprintf("Movie %d", index+1)
		}

		year := firstString(movie, "year")
		if year != "" {
			title = fmt.Sprintf("%s (%s)", title, year)
		}

		lines = append(lines, bold(fmt.Sprintf("%d. %s", index+1, title)))
		lines = appendDetailLine(lines, "IMDb", movie, "imdbRating", "imdb_rating")
		lines = appendDetailLine(lines, "TMDb", movie, "tmdbRating", "tmdb_rating")
	}
	if len(movies) > maxPrettyListItems {
		lines = append(lines, fmt.Sprintf("... %d more movies omitted. Use --output json or --fields for the full payload.", len(movies)-maxPrettyListItems))
	}

	return strings.Join(lines, "\n"), true, nil
}

func renderTVShowsPretty(value any) (string, bool, error) {
	payload, ok := value.(map[string]any)
	if !ok {
		return "", false, nil
	}

	shows, ok := payload["shows"].([]any)
	if !ok {
		return "", false, nil
	}

	lines := []string{
		bold(fmt.Sprintf("TV Shows: %d", len(shows))),
	}
	if len(shows) == 0 {
		lines = append(lines, "", "No TV shows yet.")
		return strings.Join(lines, "\n"), true, nil
	}

	for index, item := range shows[:min(len(shows), maxPrettyListItems)] {
		show, ok := item.(map[string]any)
		if !ok {
			continue
		}

		title := firstString(show, "title", "name")
		if title == "" {
			title = fmt.Sprintf("TV Show %d", index+1)
		}
		if year := firstString(show, "year"); year != "" {
			title = fmt.Sprintf("%s (%s)", title, year)
		}

		lines = append(lines, bold(fmt.Sprintf("%d. %s", index+1, title)))
		lines = appendDetailLine(lines, "IMDb ID", show, "imdbId", "imdb_id")
		lines = appendDetailLine(lines, "Rating", show, "rating")
		lines = appendDetailLine(lines, "Status", show, "status")
		lines = appendJoinedListLine(lines, "Networks", show, "networks")
	}
	if len(shows) > maxPrettyListItems {
		lines = append(lines, fmt.Sprintf("... %d more TV shows omitted. Use --output json or --fields for the full payload.", len(shows)-maxPrettyListItems))
	}

	return strings.Join(lines, "\n"), true, nil
}

func renderTVShowDetailPretty(value any) (string, bool, error) {
	payload, ok := value.(map[string]any)
	if !ok {
		return "", false, nil
	}
	show, ok := payload["show"].(map[string]any)
	if !ok {
		return "", false, nil
	}
	seasons, _ := payload["seasons"].([]any)

	title := firstString(show, "title", "name")
	if title == "" {
		title = "TV Show"
	}
	if year := firstString(show, "year"); year != "" {
		title = fmt.Sprintf("%s (%s)", title, year)
	}

	lines := []string{title}
	lines = appendDetailLine(lines, "IMDb ID", show, "imdbId", "imdb_id")
	lines = appendDetailLine(lines, "Rating", show, "rating")
	lines = appendDetailLine(lines, "Status", show, "status")
	lines = appendDetailLine(lines, "Season Count", show, "seasonCount", "season_count")
	lines = appendJoinedListLine(lines, "Networks", show, "networks")
	lines = appendJoinedListLine(lines, "Genres", show, "genres")
	if overview := firstString(show, "overview"); overview != "" {
		lines = append(lines, "Overview: "+overview)
	}
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Seasons: %d", len(seasons)))
	for index, item := range seasons[:min(len(seasons), maxPrettyListItems)] {
		season, ok := item.(map[string]any)
		if !ok {
			continue
		}

		number := firstString(season, "seasonNumber", "season_number")
		name := firstString(season, "name")
		if name == "" {
			name = fmt.Sprintf("Season %s", number)
		}
		lines = append(lines, fmt.Sprintf("%d. %s", index+1, name))
		lines = appendDetailLine(lines, "Season Number", season, "seasonNumber", "season_number")
		lines = appendDetailLine(lines, "Episodes", season, "episodeCount", "episode_count")
		lines = appendDetailLine(lines, "Air Date", season, "airDate", "air_date")
	}
	if len(seasons) > maxPrettyListItems {
		lines = append(lines, fmt.Sprintf("... %d more seasons omitted. Use --output json or --fields for the full payload.", len(seasons)-maxPrettyListItems))
	}

	return strings.Join(lines, "\n"), true, nil
}

func renderTVShowSeasonPretty(value any) (string, bool, error) {
	payload, ok := value.(map[string]any)
	if !ok {
		return "", false, nil
	}
	season, ok := payload["season"].(map[string]any)
	if !ok {
		return "", false, nil
	}
	episodes, _ := payload["episodes"].([]any)

	name := firstString(season, "name")
	if name == "" {
		name = "Season"
	}
	lines := []string{name}
	lines = appendDetailLine(lines, "Season Number", season, "seasonNumber", "season_number")
	lines = appendDetailLine(lines, "Episode Count", season, "episodeCount", "episode_count")
	lines = appendDetailLine(lines, "Air Date", season, "airDate", "air_date")
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Episodes: %d", len(episodes)))

	for index, item := range episodes[:min(len(episodes), maxPrettyListItems)] {
		episode, ok := item.(map[string]any)
		if !ok {
			continue
		}
		number := firstString(episode, "episodeNumber", "episode_number")
		title := firstString(episode, "name")
		if title == "" {
			title = fmt.Sprintf("Episode %s", number)
		}
		lines = append(lines, fmt.Sprintf("%d. E%s %s", index+1, number, title))
		lines = appendDetailLine(lines, "Air Date", episode, "airDate", "air_date")
		lines = appendDetailLine(lines, "Runtime", episode, "runtime")
		lines = appendDetailLine(lines, "Rating", episode, "rating")
	}
	if len(episodes) > maxPrettyListItems {
		lines = append(lines, fmt.Sprintf("... %d more episodes omitted. Use --output json or --fields for the full payload.", len(episodes)-maxPrettyListItems))
	}

	return strings.Join(lines, "\n"), true, nil
}

func renderTVShowEpisodeDownloadPretty(value any) (string, bool, error) {
	payload, ok := value.(map[string]any)
	if !ok {
		return "", false, nil
	}

	lines := []string{}
	if query := firstString(payload, "searchQuery", "search_query"); query != "" {
		lines = append(lines, "Search Query: "+query)
	}

	download, ok := payload["download"].(map[string]any)
	if !ok {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, "No matching episode download found.")
		return strings.Join(lines, "\n"), true, nil
	}

	if len(lines) > 0 {
		lines = append(lines, "")
	}
	lines = append(lines, prettyTVShowDownloadLines(download)...)
	return strings.Join(lines, "\n"), true, nil
}

func renderTVShowSeasonDownloadsPretty(value any) (string, bool, error) {
	payload, ok := value.(map[string]any)
	if !ok {
		return "", false, nil
	}

	lines := []string{}
	if query := firstString(payload, "seasonSearchQuery", "season_search_query"); query != "" {
		lines = append(lines, "Season Search Query: "+query)
	}

	if seasonPack, ok := payload["seasonPack"].(map[string]any); ok {
		lines = append(lines, "")
		lines = append(lines, "Season Pack")
		lines = append(lines, prettyTVShowDownloadLinesWithIndent(nil, seasonPack, "   ")...)
	}

	episodes, _ := payload["episodes"].([]any)
	if len(lines) > 0 {
		lines = append(lines, "")
	}
	lines = append(lines, fmt.Sprintf("Episode Results: %d", len(episodes)))
	for index, item := range episodes[:min(len(episodes), maxPrettyListItems)] {
		result, ok := item.(map[string]any)
		if !ok {
			continue
		}
		number := firstString(result, "episodeNumber", "episode_number")
		lines = append(lines, fmt.Sprintf("%d. Episode %s", index+1, number))
		lines = appendDetailLine(lines, "Search Query", result, "searchQuery", "search_query")
		if download, ok := result["download"].(map[string]any); ok {
			lines = append(lines, prettyTVShowDownloadLinesWithIndent(nil, download, "   ")...)
		} else {
			lines = append(lines, "   Download: none")
		}
	}
	if len(episodes) > maxPrettyListItems {
		lines = append(lines, fmt.Sprintf("... %d more episode results omitted. Use --output json or --fields for the full payload.", len(episodes)-maxPrettyListItems))
	}

	return strings.Join(lines, "\n"), true, nil
}

func renderUserIndexersPretty(value any) (string, bool, error) {
	payload, ok := value.(map[string]any)
	if !ok {
		return "", false, nil
	}

	indexers, ok := payload["indexers"].([]any)
	if !ok {
		return "", false, nil
	}

	lines := []string{
		bold(fmt.Sprintf("Indexers: %d", len(indexers))),
	}
	if len(indexers) == 0 {
		lines = append(lines, "", "No indexers configured.")
		return strings.Join(lines, "\n"), true, nil
	}

	for index, item := range indexers[:min(len(indexers), maxPrettyListItems)] {
		indexer, ok := item.(map[string]any)
		if !ok {
			continue
		}

		name := firstString(indexer, "name")
		if name == "" {
			name = firstString(indexer, "id")
		}
		if name == "" {
			name = fmt.Sprintf("Indexer %d", index+1)
		}

		enabledStatus := "disabled"
		if enabled, ok := indexer["enabled"].(bool); ok && enabled {
			enabledStatus = "enabled"
		}

		line := fmt.Sprintf("%d. %s", index+1, name)
		if id := firstString(indexer, "id"); id != "" && id != name {
			line = fmt.Sprintf("%s [%s]", line, id)
		}
		lines = append(lines, line)
		lines = append(lines, fmt.Sprintf("   Enabled: %s", enabledStatus))
		if status := prettyIndexerStatus(firstString(indexer, "status")); status != "" {
			lines = append(lines, fmt.Sprintf("   Indexer Status: %s", status))
		}
	}
	if len(indexers) > maxPrettyListItems {
		lines = append(lines, fmt.Sprintf("... %d more indexers omitted. Use --output json for the full payload.", len(indexers)-maxPrettyListItems))
	}

	return strings.Join(lines, "\n"), true, nil
}

func prettyIndexerStatus(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	trimmed = strings.TrimPrefix(trimmed, "INDEXER_STATUS_")
	if trimmed == "" {
		return ""
	}

	return strings.ToLower(trimmed)
}

func renderUserSettingsPretty(value any) (string, bool, error) {
	settings, ok := value.(map[string]any)
	if !ok {
		return "", false, nil
	}
	if len(settings) == 0 {
		return "User settings are empty.", true, nil
	}

	keys := make([]string, 0, len(settings))
	for key := range settings {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	lines := []string{bold("User Settings")}
	lines = appendUserSettingsLines(lines, settings, "")
	return strings.Join(lines, "\n"), true, nil
}

func appendUserSettingsLines(lines []string, settings map[string]any, indent string) []string {
	keys := make([]string, 0, len(settings))
	for key := range settings {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := settings[key]
		if nested, ok := value.(map[string]any); ok {
			lines = append(lines, fmt.Sprintf("%s%s", indent, dim(sanitizeTerminalText(key)+":")))
			lines = appendUserSettingsLines(lines, nested, indent+"  ")
			continue
		}
		lines = append(lines, fmt.Sprintf("%s%s %s", indent, dim(sanitizeTerminalText(key)+":"), prettyValue(value)))
	}
	return lines
}

func renderDownloadFolderPretty(value any) (string, bool, error) {
	payload, ok := value.(map[string]any)
	if !ok {
		return "", false, nil
	}
	folder, ok := payload["folder"].(map[string]any)
	if !ok {
		return "", false, nil
	}

	lines := []string{bold("Download Folder")}
	lines = append(lines, prettyUserFileLines(folder)...)
	return strings.Join(lines, "\n"), true, nil
}

func renderFolderPretty(value any) (string, bool, error) {
	payload, ok := value.(map[string]any)
	if !ok {
		return "", false, nil
	}

	parent, ok := payload["parent"].(map[string]any)
	if !ok {
		return "", false, nil
	}
	files, ok := payload["files"].([]any)
	if !ok {
		return "", false, nil
	}

	lines := []string{bold("Folder")}
	lines = append(lines, prettyUserFileLines(parent)...)
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Children: %d", len(files)))

	for index, item := range files[:min(len(files), maxPrettyListItems)] {
		file, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name := firstString(file, "name")
		if name == "" {
			name = fmt.Sprintf("Item %d", index+1)
		}
		fileType := firstString(file, "fileType", "file_type")
		if fileType == "" {
			fileType = "unknown"
		}
		lines = append(lines, fmt.Sprintf("%d. %s [%s]", index+1, name, fileType))
		if id := firstString(file, "id"); id != "" {
			lines = append(lines, fmt.Sprintf("   ID: %s", id))
		}
	}
	if len(files) > maxPrettyListItems {
		lines = append(lines, fmt.Sprintf("... %d more items omitted. Use --output json for the full payload.", len(files)-maxPrettyListItems))
	}

	return strings.Join(lines, "\n"), true, nil
}

func renderTransferPretty(value any) (string, bool, error) {
	payload, ok := value.(map[string]any)
	if !ok {
		return "", false, nil
	}

	transfer := payload
	if nested, ok := payload["transfer"].(map[string]any); ok {
		transfer = nested
	}
	if len(transfer) == 0 {
		return "", false, nil
	}

	lines := []string{bold("Transfer")}
	if status, ok := stringValue(payload, "status"); ok && payload["transfer"] != nil {
		lines = append(lines, fmt.Sprintf("Request Status: %s", status))
	}
	if id := prettyValue(firstPresent(transfer, "id")); id != "" && id != "<nil>" {
		lines = append(lines, fmt.Sprintf("ID: %s", id))
	}
	if name, ok := stringValue(transfer, "name"); ok {
		lines = append(lines, fmt.Sprintf("Name: %s", name))
	}
	if status := prettyValue(firstPresent(transfer, "status")); status != "" && status != "<nil>" {
		lines = append(lines, fmt.Sprintf("Status: %s", status))
	}
	if progress := prettyValue(firstPresent(transfer, "percentDone", "percent_done")); progress != "" && progress != "<nil>" {
		lines = append(lines, fmt.Sprintf("Progress: %s%%", progress))
	}
	if finished, ok := transfer["isFinished"].(bool); ok {
		lines = append(lines, fmt.Sprintf("Finished: %t", finished))
	} else if finished, ok := transfer["is_finished"].(bool); ok {
		lines = append(lines, fmt.Sprintf("Finished: %t", finished))
	}
	if message := firstString(transfer, "statusMessage", "status_message"); message != "" {
		lines = append(lines, fmt.Sprintf("Message: %s", message))
	}
	if message := firstString(transfer, "errorMessage", "error_message"); message != "" {
		lines = append(lines, fmt.Sprintf("Error: %s", message))
	}
	if fileURL := firstString(transfer, "fileUrl", "file_url"); fileURL != "" {
		lines = append(lines, fmt.Sprintf("File URL: %s", fileURL))
	}
	if fileID := prettyValue(firstPresent(transfer, "fileId", "file_id")); fileID != "" && fileID != "<nil>" {
		lines = append(lines, fmt.Sprintf("File ID: %s", fileID))
	}
	if eta := prettyValue(firstPresent(transfer, "estimatedTimeSeconds", "estimated_time_seconds")); eta != "" && eta != "<nil>" && eta != "0" {
		lines = append(lines, fmt.Sprintf("ETA Seconds: %s", eta))
	}

	return strings.Join(lines, "\n"), true, nil
}

func renderDoctorPretty(value any) (string, bool, error) {
	payload, ok := value.(map[string]any)
	if !ok {
		return "", false, nil
	}

	lines := []string{bold("Doctor")}
	lines = appendDoctorLine(lines, "Status", payload, "status")

	if build, ok := payload["build"].(map[string]any); ok {
		lines = append(lines, "")
		lines = append(lines, bold("Build"))
		lines = appendDoctorLine(lines, "Version", build, "version")
		lines = appendDoctorLine(lines, "Commit", build, "commit")
		lines = appendDoctorLine(lines, "Build Date", build, "build_date")
		lines = appendDoctorLine(lines, "Dev Build", build, "dev")
	}

	if config, ok := payload["config"].(map[string]any); ok {
		lines = append(lines, "")
		lines = append(lines, bold("Config"))
		lines = appendDoctorLine(lines, "Profile", config, "profile")
		lines = appendDoctorLine(lines, "Path", config, "path")
		lines = appendDoctorLine(lines, "Exists", config, "exists")
	}

	if api, ok := payload["api"].(map[string]any); ok {
		lines = append(lines, "")
		lines = append(lines, bold("API"))
		lines = appendDoctorLine(lines, "Base URL", api, "base_url")
	}

	if auth, ok := payload["auth"].(map[string]any); ok {
		lines = append(lines, "")
		lines = append(lines, bold("Auth"))
		lines = appendDoctorLine(lines, "Configured", auth, "configured")
		lines = appendDoctorLine(lines, "Status", auth, "status")
		lines = appendDoctorLine(lines, "Request ID", auth, "request_id")
		lines = appendDoctorLine(lines, "Code", auth, "code")
		lines = appendDoctorLine(lines, "Message", auth, "message")

		if user, ok := auth["user"].(map[string]any); ok {
			lines = appendDoctorLine(lines, "Username", user, "username")
			lines = appendDoctorLine(lines, "Email", user, "email")
			lines = appendDoctorLine(lines, "User ID", user, "userId", "user_id")
			lines = appendDoctorLine(lines, "Plan", user, "plan")
		}
	}

	return strings.Join(lines, "\n"), true, nil
}

func appendIfString(lines []string, label string, payload map[string]any, keys ...string) []string {
	if value := firstString(payload, keys...); value != "" {
		return append(lines, fmt.Sprintf("%s %s", dim(label+":"), value))
	}
	return lines
}

func appendDoctorLine(lines []string, label string, payload map[string]any, keys ...string) []string {
	for _, key := range keys {
		if value, ok := payload[key]; ok {
			lines = append(lines, fmt.Sprintf("%s %s", dim(label+":"), prettyValue(value)))
			return lines
		}
	}
	return lines
}

func appendDetailLine(lines []string, label string, payload map[string]any, keys ...string) []string {
	return appendDetailLineWithIndent(lines, "   ", label, payload, keys...)
}

func appendDetailLineWithIndent(lines []string, indent string, label string, payload map[string]any, keys ...string) []string {
	if value := firstString(payload, keys...); value != "" {
		return append(lines, fmt.Sprintf("%s%s %s", indent, dim(label+":"), value))
	}
	return lines
}

func appendJoinedListLine(lines []string, label string, payload map[string]any, keys ...string) []string {
	for _, key := range keys {
		values, ok := payload[key].([]any)
		if !ok || len(values) == 0 {
			continue
		}

		items := make([]string, 0, len(values))
		for _, value := range values {
			rendered := prettyValue(value)
			if rendered != "" && rendered != "null" {
				items = append(items, rendered)
			}
		}
		if len(items) == 0 {
			return lines
		}
		return append(lines, fmt.Sprintf("   %s: %s", label, strings.Join(items, ", ")))
	}
	return lines
}

func firstString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := stringValue(payload, key); ok {
			return value
		}
	}
	return ""
}

func stringValue(payload map[string]any, key string) (string, bool) {
	value, ok := payload[key]
	if !ok || value == nil {
		return "", false
	}

	switch typed := value.(type) {
	case string:
		trimmed := sanitizeTerminalText(strings.TrimSpace(typed))
		return trimmed, trimmed != ""
	case float64:
		return formatNumeric(typed), true
	case bool:
		if typed {
			return "true", true
		}
		return "false", true
	default:
		return "", false
	}
}

func formatNumeric(value float64) string {
	if value == float64(int64(value)) {
		return fmt.Sprintf("%d", int64(value))
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", value), "0"), ".")
}

func prettyValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return "null"
	case string:
		return sanitizeTerminalText(typed)
	case bool:
		if typed {
			return "true"
		}
		return "false"
	case float64:
		return formatNumeric(typed)
	case []any:
		if len(typed) == 0 {
			return "[]"
		}
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			items = append(items, prettyValue(item))
		}
		return strings.Join(items, ", ")
	default:
		return sanitizeTerminalText(fmt.Sprintf("%v", value))
	}
}

func prettyUserFileLines(file map[string]any) []string {
	lines := make([]string, 0, 5)
	if name := firstString(file, "name"); name != "" {
		lines = append(lines, fmt.Sprintf("Name: %s", name))
	}
	if id := firstString(file, "id"); id != "" {
		lines = append(lines, fmt.Sprintf("ID: %s", id))
	}
	if fileType := firstString(file, "fileType", "file_type"); fileType != "" {
		lines = append(lines, fmt.Sprintf("Type: %s", fileType))
	}
	if createdAt := firstString(file, "createdAt", "created_at"); createdAt != "" {
		lines = append(lines, fmt.Sprintf("Created: %s", createdAt))
	}
	if shared, ok := file["isShared"].(bool); ok {
		lines = append(lines, fmt.Sprintf("Shared: %t", shared))
	} else if shared, ok := file["is_shared"].(bool); ok {
		lines = append(lines, fmt.Sprintf("Shared: %t", shared))
	}
	return lines
}

func prettyTVShowDownloadLines(download map[string]any) []string {
	return prettyTVShowDownloadLinesWithIndent(nil, download, "")
}

func prettyTVShowDownloadLinesWithIndent(lines []string, download map[string]any, indent string) []string {
	title := firstString(download, "title")
	if title == "" {
		title = "Download"
	}
	lines = append(lines, indent+"Title: "+title)
	lines = appendDetailLineWithIndent(lines, indent+"   ", "Indexer", download, "indexer")
	lines = appendDetailLineWithIndent(lines, indent+"   ", "Resolution", download, "resolution")
	lines = appendDetailLineWithIndent(lines, indent+"   ", "Quality", download, "quality")
	lines = appendDetailLineWithIndent(lines, indent+"   ", "Codec", download, "codec")
	lines = appendDetailLineWithIndent(lines, indent+"   ", "Seeders", download, "seeders")
	lines = appendDetailLineWithIndent(lines, indent+"   ", "Size", download, "size")
	lines = appendDetailLineWithIndent(lines, indent+"   ", "Link", download, "link")
	return lines
}

func firstPresent(payload map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := payload[key]; ok {
			return value
		}
	}
	return nil
}

func min(left int, right int) int {
	if left < right {
		return left
	}
	return right
}
