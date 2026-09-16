package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	appDirName      = "chilly"
	profilesDirName = "profiles"
	configFileName  = "config.json"
	defaultAPIBase  = "https://api.chill.institute"
	defaultProfile  = "default"
	devProfile      = "dev"
	envProfile      = "CHILLY_PROFILE"
	directoryPerm   = 0o700
	configFilePerm  = 0o600
	tempFilePattern = "config-*.tmp"
)

var userConfigDir = os.UserConfigDir
var validProfilePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

type Config struct {
	APIBaseURL string `json:"api_base_url"`
	AuthToken  string `json:"auth_token,omitempty"`
}

func DefaultPath(profile string) (string, error) {
	baseDir := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if baseDir == "" {
		var err error
		baseDir, err = userConfigDir()
		if err != nil {
			return "", fmt.Errorf("resolve config base dir: %w", err)
		}
	}
	if strings.TrimSpace(baseDir) == "" {
		return "", errors.New("empty config base dir")
	}

	normalizedProfile, err := NormalizeProfile(profile)
	if err != nil {
		return "", err
	}
	if normalizedProfile == defaultProfile {
		return filepath.Join(baseDir, appDirName, configFileName), nil
	}
	return filepath.Join(baseDir, appDirName, profilesDirName, normalizedProfile, configFileName), nil
}

func ResolveProfile(raw string, isDev bool) (string, error) {
	profile := strings.TrimSpace(raw)
	if profile == "" {
		profile = strings.TrimSpace(os.Getenv(envProfile))
	}
	if profile == "" {
		if isDev {
			profile = devProfile
		} else {
			profile = defaultProfile
		}
	}
	return NormalizeProfile(profile)
}

func NormalizeProfile(raw string) (string, error) {
	profile := strings.ToLower(strings.TrimSpace(raw))
	if profile == "" {
		return defaultProfile, nil
	}
	if !validProfilePattern.MatchString(profile) {
		return "", fmt.Errorf("invalid profile %q", raw)
	}
	return profile, nil
}

func DefaultProfile() string {
	return defaultProfile
}

func Default() Config {
	return Config{APIBaseURL: defaultAPIBase}
}

func (cfg Config) Normalized() Config {
	normalized := cfg
	normalized.APIBaseURL = strings.TrimSpace(normalized.APIBaseURL)
	normalized.AuthToken = strings.TrimSpace(normalized.AuthToken)
	if normalized.APIBaseURL == "" {
		normalized.APIBaseURL = defaultAPIBase
	}
	return normalized
}

type Store struct {
	path string
}

func NewStore(path string) (*Store, error) {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		defaultPath, err := DefaultPath(defaultProfile)
		if err != nil {
			return nil, err
		}
		trimmedPath = defaultPath
	}
	return &Store{path: trimmedPath}, nil
}

func (store Store) Path() string {
	return store.path
}

func (store Store) Load() (Config, error) {
	content, err := os.ReadFile(store.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Default(), nil
		}
		return Config{}, fmt.Errorf("read config %q: %w", store.path, err)
	}

	var cfg Config
	if err := json.Unmarshal(content, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %q: %w", store.path, err)
	}
	return cfg.Normalized(), nil
}

func (store Store) Save(cfg Config) error {
	normalized := cfg.Normalized()

	if err := os.MkdirAll(filepath.Dir(store.path), directoryPerm); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	payload, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	payload = append(payload, '\n')

	tmpFile, err := os.CreateTemp(filepath.Dir(store.path), tempFilePattern)
	if err != nil {
		return fmt.Errorf("create temp config file: %w", err)
	}
	tmpPath := tmpFile.Name()
	cleanup := func() {
		_ = os.Remove(tmpPath)
	}

	if err := tmpFile.Chmod(configFilePerm); err != nil {
		_ = tmpFile.Close()
		cleanup()
		return fmt.Errorf("chmod temp config file: %w", err)
	}

	if _, err := tmpFile.Write(payload); err != nil {
		_ = tmpFile.Close()
		cleanup()
		return fmt.Errorf("write temp config file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp config file: %w", err)
	}

	if err := os.Rename(tmpPath, store.path); err != nil {
		cleanup()
		return fmt.Errorf("replace config file: %w", err)
	}

	if err := os.Chmod(store.path, configFilePerm); err != nil {
		return fmt.Errorf("chmod config file: %w", err)
	}

	return nil
}
