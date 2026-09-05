//go:build integration

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLiveAuthAndReadSurfaces(t *testing.T) {
	apiURL := strings.TrimSpace(os.Getenv("CHILLY_TEST_API_URL"))
	token := strings.TrimSpace(os.Getenv("CHILLY_TEST_TOKEN"))
	if apiURL == "" || token == "" {
		t.Skip("CHILLY_TEST_API_URL and CHILLY_TEST_TOKEN are required")
	}

	configPath := filepath.Join(t.TempDir(), "config.json")

	loginOutput := runLiveCommand(t, "auth login", []string{
		"--config", configPath,
		"--api-url", apiURL,
		"auth", "login",
		"--token", token,
		"--output", "json",
	})
	if loginOutput["status"] != "ok" || loginOutput["saved"] != true {
		t.Fatalf("login output = %#v", loginOutput)
	}

	whoamiOutput := runLiveCommand(t, "whoami", []string{
		"--config", configPath,
		"whoami",
		"--output", "json",
	})
	username, _ := whoamiOutput["username"].(string)
	if strings.TrimSpace(username) == "" {
		t.Fatalf("whoami output = %#v", whoamiOutput)
	}

	doctorOutput := runLiveCommand(t, "doctor", []string{
		"--config", configPath,
		"doctor",
		"--output", "json",
	})
	if doctorOutput["status"] != "ok" {
		t.Fatalf("doctor output = %#v", doctorOutput)
	}

	searchOutput := runLiveCommand(t, "search", []string{
		"--config", configPath,
		"search",
		"--query", "dune",
		"--fields", "results.title",
		"--output", "json",
	})
	results, ok := searchOutput["results"].([]any)
	if !ok || len(results) == 0 {
		t.Fatalf("search output = %#v", searchOutput)
	}
}
