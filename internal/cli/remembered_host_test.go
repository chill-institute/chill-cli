package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chill-institute/chill-cli/v2/pkg/config"
)

func TestRememberedHostValidation(t *testing.T) {
	for _, raw := range []string{"https://other.example", "not a URL", "https://user:secret@api.binge.institute", "https://api.binge.institute/path", "http://api.binge.institute", "https://api.binge.institute?q=x", "https://api.binge.institute#x", "https://api.binge.institute/\n"} {
		t.Run(raw, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			store, _ := config.NewStore(path)
			if err := store.Save(config.Default()); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path+".host", []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			cfg, err := store.Load()
			if err != nil {
				t.Fatal(err)
			}
			want := ""
			if raw == "https://api.binge.institute/\n" {
				want = "https://api.binge.institute"
			}
			if cfg.LastWorkingAPIBaseURL != want {
				t.Fatalf("remembered=%q", cfg.LastWorkingAPIBaseURL)
			}
			if err := store.RememberWorkingAPIBaseURL("invalid"); err == nil {
				t.Fatal("accepted invalid host")
			}
			after, err := os.ReadFile(path + ".host")
			if err != nil || string(after) != raw {
				t.Fatalf("hint changed: %q %v", after, err)
			}
		})
	}
}

func TestRememberingSameHostDoesNotWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	store, _ := config.NewStore(path)
	if err := store.Save(config.Default()); err != nil {
		t.Fatal(err)
	}
	if err := store.RememberWorkingAPIBaseURL("https://api.binge.institute"); err != nil {
		t.Fatal(err)
	}
	stamp := time.Unix(123456789, 0)
	if err := os.Chtimes(path+".host", stamp, stamp); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path + ".host")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RememberWorkingAPIBaseURL("https://api.binge.institute/"); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path + ".host")
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) || !after.ModTime().Equal(stamp) {
		t.Fatal("unchanged hint rewritten")
	}
}

func TestSaveDropsLegacyHostAndResetsChangedList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	store, _ := config.NewStore(path)
	if err := os.WriteFile(path, []byte(`{"api_base_url":"https://api.chill.institute","last_working_api_base_url":"https://api.binge.institute"}`), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LastWorkingAPIBaseURL != "" {
		t.Fatal("legacy preference must be ignored")
	}
	if err := store.RememberWorkingAPIBaseURL("https://api.binge.institute"); err != nil {
		t.Fatal(err)
	}
	cfg, err = store.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.APIBaseURL = "https://api.binge.institute,https://new.example"
	if err := store.Save(cfg); err != nil {
		t.Fatal(err)
	}
	cfg, err = store.Load()
	if err != nil || cfg.LastWorkingAPIBaseURL != "" {
		t.Fatalf("config=%+v err=%v", cfg, err)
	}
	data, err := os.ReadFile(path)
	if err != nil || strings.Contains(string(data), "last_working") {
		t.Fatalf("config=%s err=%v", data, err)
	}
}
