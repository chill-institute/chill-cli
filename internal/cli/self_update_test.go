package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chill-institute/chill-cli/v2/internal/buildinfo"
	"github.com/chill-institute/chill-cli/v2/internal/update"
)

type stubReleaseService struct {
	latestRelease update.Release
	tagRelease    update.Release
	downloads     map[string][]byte
}

func (service *stubReleaseService) Latest(context.Context) (update.Release, error) {
	return service.latestRelease, nil
}

func (service *stubReleaseService) ByTag(context.Context, string) (update.Release, error) {
	return service.tagRelease, nil
}

func (service *stubReleaseService) Download(_ context.Context, rawURL string) ([]byte, error) {
	if service.downloads == nil {
		return nil, errors.New("unexpected download")
	}
	payload, ok := service.downloads[strings.TrimSpace(rawURL)]
	if !ok {
		return nil, errors.New("unexpected download")
	}
	return payload, nil
}

func TestSelfUpdateCheckReportsReleaseWithoutInstalling(t *testing.T) {
	restoreBuildInfo := currentBuildInfo
	restoreReleaseService := newReleaseService
	restoreGOOS := currentRuntimeGOOS
	restoreGOARCH := currentRuntimeGOARCH
	currentBuildInfo = func() buildinfo.Info { return buildinfo.Info{Version: "v1.0.0"} }
	newReleaseService = func() releaseService {
		return &stubReleaseService{
			latestRelease: update.Release{TagName: "v1.2.0"},
		}
	}
	currentRuntimeGOOS = "darwin"
	currentRuntimeGOARCH = "arm64"
	t.Cleanup(func() {
		currentBuildInfo = restoreBuildInfo
		newReleaseService = restoreReleaseService
		currentRuntimeGOOS = restoreGOOS
		currentRuntimeGOARCH = restoreGOARCH
	})

	stdout := &bytes.Buffer{}
	command := newSelfUpdateCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"--check"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if output["latest_version"] != "v1.2.0" {
		t.Fatalf("latest_version = %v", output["latest_version"])
	}
	if output["up_to_date"] != false {
		t.Fatalf("up_to_date = %v", output["up_to_date"])
	}
}

func TestSelfUpdateInstallsOnlyVerifiedAsset(t *testing.T) {
	restoreBuildInfo := currentBuildInfo
	restoreReleaseService := newReleaseService
	restoreGOOS := currentRuntimeGOOS
	restoreGOARCH := currentRuntimeGOARCH
	restoreExecutable := currentExecutable
	currentBuildInfo = func() buildinfo.Info { return buildinfo.Info{Version: "v1.0.0"} }
	currentRuntimeGOOS = "darwin"
	currentRuntimeGOARCH = "arm64"

	executablePath := filepath.Join(t.TempDir(), "chilly")
	if err := os.WriteFile(executablePath, []byte("old-binary"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	currentExecutable = func() (string, error) { return executablePath, nil }

	archive := mustTarGZBinary(t, "new-binary")
	assetURL := "https://example.invalid/chilly.tar.gz"
	checksumURL := "https://example.invalid/checksums.txt"
	checksumPayload := []byte(fmt.Sprintf("%x  %s\n", sha256.Sum256(archive), "chilly_1.2.0_darwin_arm64.tar.gz"))

	newReleaseService = func() releaseService {
		return &stubReleaseService{
			latestRelease: update.Release{
				TagName: "v1.2.0",
				Assets: []update.ReleaseAsset{
					{Name: "checksums.txt", BrowserDownloadURL: checksumURL},
					{Name: "chilly_1.2.0_darwin_arm64.tar.gz", BrowserDownloadURL: assetURL},
				},
			},
			downloads: map[string][]byte{
				checksumURL: checksumPayload,
				assetURL:    archive,
			},
		}
	}

	t.Cleanup(func() {
		currentBuildInfo = restoreBuildInfo
		newReleaseService = restoreReleaseService
		currentRuntimeGOOS = restoreGOOS
		currentRuntimeGOARCH = restoreGOARCH
		currentExecutable = restoreExecutable
	})

	stdout := &bytes.Buffer{}
	command := newSelfUpdateCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs(nil)
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	installed, err := os.ReadFile(executablePath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(installed) != "new-binary" {
		t.Fatalf("installed binary = %q", string(installed))
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if output["installed_version"] != "v1.2.0" {
		t.Fatalf("installed_version = %v", output["installed_version"])
	}
}

func TestSelfUpdateRejectsChecksumMismatch(t *testing.T) {
	restoreBuildInfo := currentBuildInfo
	restoreReleaseService := newReleaseService
	restoreGOOS := currentRuntimeGOOS
	restoreGOARCH := currentRuntimeGOARCH
	restoreExecutable := currentExecutable
	currentBuildInfo = func() buildinfo.Info { return buildinfo.Info{Version: "v1.0.0"} }
	currentRuntimeGOOS = "darwin"
	currentRuntimeGOARCH = "arm64"

	executablePath := filepath.Join(t.TempDir(), "chilly")
	if err := os.WriteFile(executablePath, []byte("old-binary"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	currentExecutable = func() (string, error) { return executablePath, nil }

	archive := mustTarGZBinary(t, "new-binary")
	assetURL := "https://example.invalid/chilly.tar.gz"
	checksumURL := "https://example.invalid/checksums.txt"

	newReleaseService = func() releaseService {
		return &stubReleaseService{
			latestRelease: update.Release{
				TagName: "v1.2.0",
				Assets: []update.ReleaseAsset{
					{Name: "checksums.txt", BrowserDownloadURL: checksumURL},
					{Name: "chilly_1.2.0_darwin_arm64.tar.gz", BrowserDownloadURL: assetURL},
				},
			},
			downloads: map[string][]byte{
				checksumURL: []byte("deadbeef  chilly_1.2.0_darwin_arm64.tar.gz\n"),
				assetURL:    archive,
			},
		}
	}

	t.Cleanup(func() {
		currentBuildInfo = restoreBuildInfo
		newReleaseService = restoreReleaseService
		currentRuntimeGOOS = restoreGOOS
		currentRuntimeGOARCH = restoreGOARCH
		currentExecutable = restoreExecutable
	})

	command := newSelfUpdateCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
	})
	command.SetArgs(nil)
	err := command.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want checksum failure")
	}
	if !strings.Contains(err.Error(), "verify release asset checksum") {
		t.Fatalf("error = %v", err)
	}

	installed, readErr := os.ReadFile(executablePath)
	if readErr != nil {
		t.Fatalf("ReadFile() error = %v", readErr)
	}
	if string(installed) != "old-binary" {
		t.Fatalf("installed binary = %q", string(installed))
	}
}

func TestSelfUpdateRejectsInvalidVersionFlag(t *testing.T) {
	restoreReleaseService := newReleaseService
	newReleaseService = func() releaseService { return &stubReleaseService{} }
	t.Cleanup(func() { newReleaseService = restoreReleaseService })

	command := newSelfUpdateCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"--version", "../v1.2.3"})
	err := command.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want invalid version")
	}
	var classified *cliError
	if !errors.As(err, &classified) {
		t.Fatalf("error = %T, want *cliError", err)
	}
	if classified.Kind != errorKindUsage {
		t.Fatalf("kind = %q", classified.Kind)
	}
	if classified.Code != "invalid_release_version" {
		t.Fatalf("code = %q", classified.Code)
	}
}

func TestSelfUpdateDryRunReturnsPreviewWithoutReplacingExecutable(t *testing.T) {
	restoreBuildInfo := currentBuildInfo
	restoreReleaseService := newReleaseService
	restoreGOOS := currentRuntimeGOOS
	restoreGOARCH := currentRuntimeGOARCH
	restoreExecutable := currentExecutable
	currentBuildInfo = func() buildinfo.Info { return buildinfo.Info{Version: "v1.0.0"} }
	currentRuntimeGOOS = "darwin"
	currentRuntimeGOARCH = "arm64"

	executablePath := filepath.Join(t.TempDir(), "chilly")
	if err := os.WriteFile(executablePath, []byte("old-binary"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	currentExecutable = func() (string, error) { return executablePath, nil }

	newReleaseService = func() releaseService {
		return &stubReleaseService{
			latestRelease: update.Release{
				TagName: "v1.2.0",
				Assets: []update.ReleaseAsset{
					{Name: "checksums.txt", BrowserDownloadURL: "https://example.invalid/checksums.txt"},
					{Name: "chilly_1.2.0_darwin_arm64.tar.gz", BrowserDownloadURL: "https://example.invalid/chilly.tar.gz"},
				},
			},
		}
	}

	t.Cleanup(func() {
		currentBuildInfo = restoreBuildInfo
		newReleaseService = restoreReleaseService
		currentRuntimeGOOS = restoreGOOS
		currentRuntimeGOARCH = restoreGOARCH
		currentExecutable = restoreExecutable
	})

	stdout := &bytes.Buffer{}
	command := newSelfUpdateCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"--dry-run"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	installed, err := os.ReadFile(executablePath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(installed) != "old-binary" {
		t.Fatalf("installed binary = %q", string(installed))
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if output["dry_run"] != true {
		t.Fatalf("dry_run = %v, want true", output["dry_run"])
	}
	if output["target_version"] != "v1.2.0" {
		t.Fatalf("target_version = %v, want %q", output["target_version"], "v1.2.0")
	}
	if output["asset"] != "chilly_1.2.0_darwin_arm64.tar.gz" {
		t.Fatalf("asset = %v", output["asset"])
	}
}

func TestSelfUpdateAcceptsJSONInput(t *testing.T) {
	restoreBuildInfo := currentBuildInfo
	restoreReleaseService := newReleaseService
	restoreGOOS := currentRuntimeGOOS
	restoreGOARCH := currentRuntimeGOARCH
	currentBuildInfo = func() buildinfo.Info { return buildinfo.Info{Version: "v1.0.0"} }
	currentRuntimeGOOS = "darwin"
	currentRuntimeGOARCH = "arm64"
	newReleaseService = func() releaseService {
		return &stubReleaseService{
			latestRelease: update.Release{TagName: "v1.2.0"},
		}
	}
	t.Cleanup(func() {
		currentBuildInfo = restoreBuildInfo
		newReleaseService = restoreReleaseService
		currentRuntimeGOOS = restoreGOOS
		currentRuntimeGOARCH = restoreGOARCH
	})

	stdout := &bytes.Buffer{}
	command := newSelfUpdateCommand(&appContext{
		opts:   &appOptions{output: outputJSON},
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
	})
	command.SetArgs([]string{"--json", `{"check":true}`})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if output["checked"] != true {
		t.Fatalf("checked = %v, want true", output["checked"])
	}
	if output["latest_version"] != "v1.2.0" {
		t.Fatalf("latest_version = %v, want %q", output["latest_version"], "v1.2.0")
	}
}

func mustTarGZBinary(t *testing.T, payload string) []byte {
	t.Helper()

	var archive bytes.Buffer
	gzipWriter := gzip.NewWriter(&archive)
	tarWriter := tar.NewWriter(gzipWriter)

	content := []byte(payload)
	header := &tar.Header{
		Name: "chilly",
		Mode: 0o755,
		Size: int64(len(content)),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatalf("WriteHeader() error = %v", err)
	}
	if _, err := tarWriter.Write(content); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("Close() tar error = %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("Close() gzip error = %v", err)
	}
	return archive.Bytes()
}
