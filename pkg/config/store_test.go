package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultPathUsesXDGConfigHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/chilly-xdg")

	path, err := DefaultPath(defaultProfile)
	if err != nil {
		t.Fatalf("DefaultPath() error = %v", err)
	}

	want := filepath.Join("/tmp/chilly-xdg", appDirName, configFileName)
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}

func TestDefaultPathFallsBackToUserConfigDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")

	originalUserConfigDir := userConfigDir
	userConfigDir = func() (string, error) { return "/tmp/chilly-user-config", nil }
	t.Cleanup(func() { userConfigDir = originalUserConfigDir })

	path, err := DefaultPath(defaultProfile)
	if err != nil {
		t.Fatalf("DefaultPath() error = %v", err)
	}

	want := filepath.Join("/tmp/chilly-user-config", appDirName, configFileName)
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}

func TestDefaultPathPropagatesUserConfigDirErrors(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")

	originalUserConfigDir := userConfigDir
	userConfigDir = func() (string, error) { return "", os.ErrPermission }
	t.Cleanup(func() { userConfigDir = originalUserConfigDir })

	if _, err := DefaultPath(defaultProfile); err == nil {
		t.Fatal("DefaultPath() error = nil, want user config dir error")
	}
}

func TestDefaultPathRejectsEmptyBaseDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")

	originalUserConfigDir := userConfigDir
	userConfigDir = func() (string, error) { return " ", nil }
	t.Cleanup(func() { userConfigDir = originalUserConfigDir })

	if _, err := DefaultPath(defaultProfile); err == nil {
		t.Fatal("DefaultPath() error = nil, want empty base dir error")
	}
}

func TestLoadMissingConfigReturnsDefaults(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.APIBaseURL != defaultAPIBase {
		t.Fatalf("APIBaseURL = %q, want %q", cfg.APIBaseURL, defaultAPIBase)
	}
	if cfg.AuthToken != "" {
		t.Fatalf("AuthToken = %q, want empty", cfg.AuthToken)
	}
}

func TestLoadRejectsInvalidJSON(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte("{"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store, err := NewStore(configPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	if _, err := store.Load(); err == nil {
		t.Fatal("Load() error = nil, want parse error")
	}
}

func TestLoadPropagatesReadErrors(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config-dir")
	if err := os.Mkdir(configPath, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	store, err := NewStore(configPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	if _, err := store.Load(); err == nil {
		t.Fatal("Load() error = nil, want read error")
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "nested", "cfg.json")
	store, err := NewStore(configPath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	saved := Config{
		APIBaseURL: " https://chill.example ",
		AuthToken:  " token-123 ",
	}
	if err := store.Save(saved); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.APIBaseURL != "https://chill.example" {
		t.Fatalf("APIBaseURL = %q, want %q", loaded.APIBaseURL, "https://chill.example")
	}
	if loaded.AuthToken != "token-123" {
		t.Fatalf("AuthToken = %q, want %q", loaded.AuthToken, "token-123")
	}

	stat, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if stat.Mode().Perm() != configFilePerm {
		t.Fatalf("perm = %o, want %o", stat.Mode().Perm(), configFilePerm)
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(raw), "\n") {
		t.Fatal("expected saved config to be pretty-printed")
	}
}

func TestNewStoreUsesDefaultPathWhenEmpty(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/chilly-store-default")

	store, err := NewStore(" ")
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}

	want := filepath.Join("/tmp/chilly-store-default", appDirName, configFileName)
	if store.Path() != want {
		t.Fatalf("Path() = %q, want %q", store.Path(), want)
	}
}

func TestNewStorePropagatesDefaultPathErrors(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")

	originalUserConfigDir := userConfigDir
	userConfigDir = func() (string, error) { return "", os.ErrPermission }
	t.Cleanup(func() { userConfigDir = originalUserConfigDir })

	if _, err := NewStore(" "); err == nil {
		t.Fatal("NewStore() error = nil, want default path error")
	}
}

func TestDefaultPathUsesProfilesSubdirectoryForNamedProfiles(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/chilly-xdg")

	path, err := DefaultPath("dev")
	if err != nil {
		t.Fatalf("DefaultPath() error = %v", err)
	}

	want := filepath.Join("/tmp/chilly-xdg", appDirName, profilesDirName, "dev", configFileName)
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}

func TestResolveProfileUsesDevDefaultForDevBuilds(t *testing.T) {
	t.Setenv(envProfile, "")

	profile, err := ResolveProfile("", true)
	if err != nil {
		t.Fatalf("ResolveProfile() error = %v", err)
	}
	if profile != devProfile {
		t.Fatalf("profile = %q, want %q", profile, devProfile)
	}
}

func TestResolveProfileUsesEnvironmentOverride(t *testing.T) {
	t.Setenv(envProfile, "production")

	profile, err := ResolveProfile("", false)
	if err != nil {
		t.Fatalf("ResolveProfile() error = %v", err)
	}
	if profile != "production" {
		t.Fatalf("profile = %q, want %q", profile, "production")
	}
}

func TestResolveProfileUsesDefaultProfileForReleaseBuilds(t *testing.T) {
	t.Setenv(envProfile, "")

	profile, err := ResolveProfile("", false)
	if err != nil {
		t.Fatalf("ResolveProfile() error = %v", err)
	}
	if profile != defaultProfile {
		t.Fatalf("profile = %q, want %q", profile, defaultProfile)
	}
}

func TestResolveProfilePropagatesInvalidEnvironmentOverride(t *testing.T) {
	t.Setenv(envProfile, "../prod")

	if _, err := ResolveProfile("", false); err == nil {
		t.Fatal("ResolveProfile() error = nil, want invalid env profile error")
	}
}

func TestNormalizeProfileNormalizesCaseAndEmpty(t *testing.T) {
	if profile, err := NormalizeProfile(" Production "); err != nil || profile != "production" {
		t.Fatalf("NormalizeProfile() = %q, %v", profile, err)
	}
	if profile, err := NormalizeProfile(" "); err != nil || profile != defaultProfile {
		t.Fatalf("NormalizeProfile() empty = %q, %v", profile, err)
	}
}

func TestNormalizeProfileRejectsUnsafeValues(t *testing.T) {
	if _, err := NormalizeProfile("../prod"); err == nil {
		t.Fatal("expected invalid profile error")
	}
}

func TestDefaultProfileReturnsDefaultProfileName(t *testing.T) {
	if DefaultProfile() != defaultProfile {
		t.Fatalf("DefaultProfile() = %q, want %q", DefaultProfile(), defaultProfile)
	}
}

func TestDefaultReturnsDefaultConfig(t *testing.T) {
	cfg := Default()
	if cfg.APIBaseURL != defaultAPIBase {
		t.Fatalf("Default().APIBaseURL = %q, want %q", cfg.APIBaseURL, defaultAPIBase)
	}
	if cfg.AuthToken != "" {
		t.Fatalf("Default().AuthToken = %q, want empty", cfg.AuthToken)
	}
}

func TestConfigNormalizedTrimsAndBackfillsDefaults(t *testing.T) {
	cfg := Config{APIBaseURL: " ", AuthToken: " token-123 "}
	normalized := cfg.Normalized()
	if normalized.APIBaseURL != defaultAPIBase {
		t.Fatalf("Normalized().APIBaseURL = %q, want %q", normalized.APIBaseURL, defaultAPIBase)
	}
	if normalized.AuthToken != "token-123" {
		t.Fatalf("Normalized().AuthToken = %q, want %q", normalized.AuthToken, "token-123")
	}
}

func TestSaveRejectsUnwritableTargets(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "parent-file")
	if err := os.WriteFile(parent, []byte("not-a-dir"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store, err := NewStore(filepath.Join(parent, "config.json"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Save(Default()); err == nil {
		t.Fatal("Save() error = nil, want mkdir error")
	}

	replacePath := filepath.Join(t.TempDir(), "config-target")
	if err := os.Mkdir(replacePath, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	store, err = NewStore(replacePath)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Save(Default()); err == nil {
		t.Fatal("Save() error = nil, want replace error")
	}
}
