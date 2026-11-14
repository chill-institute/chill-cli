package config

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestLegacyHostHintIsIgnoredAndPreserved(t *testing.T) {
	for _, directory := range []bool{false, true} {
		name := "stale file"
		if directory {
			name = "unreadable directory"
		}
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			hint := path + ".host"
			content := []byte("https://second.example\n")
			if directory {
				if err := os.Mkdir(hint, 0700); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(hint, content, 0600); err != nil {
				t.Fatal(err)
			}
			store, err := NewStore(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := store.Save(Config{APIBaseURL: "https://first.example,https://second.example"}); err != nil {
				t.Fatal(err)
			}
			cfg, err := store.Load()
			if err != nil || !slices.Equal(cfg.APIBaseURLs(), []string{"https://first.example", "https://second.example"}) {
				t.Fatalf("config=%+v error=%v", cfg, err)
			}
			if directory {
				info, err := os.Stat(hint)
				if err != nil || !info.IsDir() {
					t.Fatalf("legacy directory changed: %v", err)
				}
			} else {
				got, err := os.ReadFile(hint)
				if err != nil || string(got) != string(content) {
					t.Fatalf("legacy file changed: %q %v", got, err)
				}
			}
		})
	}
}

func TestSaveOverwritesCorruptConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{broken"), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	want := Config{APIBaseURL: "https://custom.example", AuthToken: "new-token"}
	if err := store.Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load()
	if err != nil || got != want {
		t.Fatalf("config=%+v error=%v", got, err)
	}
}
