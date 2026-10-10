package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestFindAssetRejectsUnsupportedTarget(t *testing.T) {
	t.Parallel()

	release := Release{
		TagName: "v1.2.3",
		Assets:  []ReleaseAsset{{Name: "chilly_1.2.3_plan9_amd64.tar.gz"}},
	}
	if _, err := FindAsset(release, "plan9", "amd64"); err == nil {
		t.Fatal("FindAsset() error = nil, want unsupported os")
	}
}

func TestNewClientDefaults(t *testing.T) {
	t.Parallel()

	client := NewClient(nil)
	if client.httpClient == nil || client.httpClient.Timeout == 0 {
		t.Fatalf("httpClient = %#v, want default timeout", client.httpClient)
	}
}

func TestFindAsset(t *testing.T) {
	t.Parallel()

	release := Release{
		TagName: "v1.2.3",
		Assets: []ReleaseAsset{
			{Name: "chilly_1.2.3_darwin_arm64.tar.gz", BrowserDownloadURL: "https://example.invalid/chilly.tgz"},
		},
	}

	asset, err := FindAsset(release, "darwin", "arm64")
	if err != nil {
		t.Fatalf("FindAsset() error = %v", err)
	}
	if asset.BrowserDownloadURL == "" {
		t.Fatal("expected asset URL")
	}
}

func TestFindChecksumAsset(t *testing.T) {
	t.Parallel()

	release := Release{
		TagName: "v1.2.3",
		Assets: []ReleaseAsset{
			{Name: "checksums.txt", BrowserDownloadURL: "https://example.invalid/checksums.txt"},
		},
	}

	asset, err := FindChecksumAsset(release)
	if err != nil {
		t.Fatalf("FindChecksumAsset() error = %v", err)
	}
	if asset.BrowserDownloadURL == "" {
		t.Fatal("expected checksum asset URL")
	}
}

func TestFindAssetAndChecksumAssetRejectMissingAssets(t *testing.T) {
	t.Parallel()

	release := Release{TagName: "v1.2.3"}
	if _, err := FindAsset(release, "darwin", "arm64"); err == nil {
		t.Fatal("FindAsset() error = nil, want missing asset")
	}
	if _, err := FindChecksumAsset(release); err == nil {
		t.Fatal("FindChecksumAsset() error = nil, want missing checksum asset")
	}
}

func TestVerifyAssetChecksum(t *testing.T) {
	t.Parallel()

	payload := []byte("archive-bytes")
	checksums := []byte(fmt.Sprintf("%x  chilly_1.2.3_darwin_arm64.tar.gz\n", sha256.Sum256(payload)))
	if err := VerifyAssetChecksum("chilly_1.2.3_darwin_arm64.tar.gz", payload, checksums); err != nil {
		t.Fatalf("VerifyAssetChecksum() error = %v", err)
	}
}

func TestVerifyAssetChecksumMismatch(t *testing.T) {
	t.Parallel()

	payload := []byte("archive-bytes")
	checksums := []byte("deadbeef  chilly_1.2.3_darwin_arm64.tar.gz\n")
	if err := VerifyAssetChecksum("chilly_1.2.3_darwin_arm64.tar.gz", payload, checksums); err == nil {
		t.Fatal("VerifyAssetChecksum() error = nil, want mismatch")
	}
}

func TestExtractBinaryTarGZ(t *testing.T) {
	t.Parallel()

	var archive bytes.Buffer
	gzipWriter := gzip.NewWriter(&archive)
	tarWriter := tar.NewWriter(gzipWriter)

	payload := []byte("binary-data")
	header := &tar.Header{
		Name: "chilly",
		Mode: 0o755,
		Size: int64(len(payload)),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatalf("WriteHeader() error = %v", err)
	}
	if _, err := tarWriter.Write(payload); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("Close() tar error = %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("Close() gzip error = %v", err)
	}

	extracted, err := ExtractBinary(archive.Bytes(), "darwin")
	if err != nil {
		t.Fatalf("ExtractBinary() error = %v", err)
	}
	if string(extracted) != "binary-data" {
		t.Fatalf("ExtractBinary() = %q", string(extracted))
	}
}

func TestExtractBinaryZip(t *testing.T) {
	t.Parallel()

	var archive bytes.Buffer
	zipWriter := zip.NewWriter(&archive)
	fileWriter, err := zipWriter.Create("chilly.exe")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := fileWriter.Write([]byte("windows-binary")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	extracted, err := ExtractBinary(archive.Bytes(), "windows")
	if err != nil {
		t.Fatalf("ExtractBinary() error = %v", err)
	}
	if string(extracted) != "windows-binary" {
		t.Fatalf("ExtractBinary() = %q", string(extracted))
	}
}

func TestReplaceExecutable(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "chilly")
	if err := os.WriteFile(path, []byte("old"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := ReplaceExecutable(path, []byte("new"), 0o755); err != nil {
		t.Fatalf("ReplaceExecutable() error = %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(content) != "new" {
		t.Fatalf("content = %q", string(content))
	}
}

func TestReplaceExecutableUsesDefaultMode(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "chilly")
	if err := os.WriteFile(path, []byte("old"), 0o700); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := ReplaceExecutable(path, []byte("new"), 0); err != nil {
		t.Fatalf("ReplaceExecutable() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("perm = %o, want %o", info.Mode().Perm(), 0o755)
	}
}

func TestReplaceExecutablePropagatesFilesystemFailures(t *testing.T) {
	t.Parallel()

	missingPath := filepath.Join(t.TempDir(), "missing", "chilly")
	if err := ReplaceExecutable(missingPath, []byte("new"), 0o755); err == nil {
		t.Fatal("ReplaceExecutable() error = nil, want create temp failure")
	}

	targetDir := filepath.Join(t.TempDir(), "chilly-dir")
	if err := os.Mkdir(targetDir, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	if err := ReplaceExecutable(targetDir, []byte("new"), 0o755); err == nil {
		t.Fatal("ReplaceExecutable() error = nil, want replace failure")
	}
}

func TestSameVersion(t *testing.T) {
	t.Parallel()

	if !SameVersion("1.2.3", "v1.2.3") {
		t.Fatal("expected versions to match")
	}
}

func TestNormalizeAndValidateVersion(t *testing.T) {
	t.Parallel()

	if NormalizeVersion(" 1.2.3 ") != "v1.2.3" {
		t.Fatalf("NormalizeVersion() = %q", NormalizeVersion(" 1.2.3 "))
	}
	if NormalizeVersion(" ") != "" {
		t.Fatalf("NormalizeVersion() = %q, want empty", NormalizeVersion(" "))
	}
	if _, err := ValidateVersion(""); err == nil {
		t.Fatal("ValidateVersion() error = nil, want empty version error")
	}
}

func TestValidateVersionRejectsPathTraversal(t *testing.T) {
	t.Parallel()

	if _, err := ValidateVersion("../../evil"); err == nil {
		t.Fatal("ValidateVersion() error = nil, want invalid version")
	}
}

func TestLatestAndByTag(t *testing.T) {
	t.Parallel()

	var requested []string
	client := NewClient(&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requested = append(requested, request.URL.String())
		payload, err := json.Marshal(Release{TagName: "1.2.3"})
		if err != nil {
			return nil, err
		}
		return okResponse(payload), nil
	})})

	release, err := client.Latest(context.Background())
	if err != nil {
		t.Fatalf("Latest() error = %v", err)
	}
	if release.TagName != "v1.2.3" {
		t.Fatalf("TagName = %q, want normalized v1.2.3", release.TagName)
	}
	if _, err := client.ByTag(context.Background(), "1.2.3"); err != nil {
		t.Fatalf("ByTag() error = %v", err)
	}

	want := []string{
		"https://api.github.com/repos/chill-institute/chill-cli/releases/latest",
		"https://api.github.com/repos/chill-institute/chill-cli/releases/tags/v1.2.3",
	}
	if !slices.Equal(requested, want) {
		t.Fatalf("requested = %q, want %q", requested, want)
	}
}

func TestDownload(t *testing.T) {
	t.Parallel()

	client := NewClient(&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() != "https://github.com/chill-institute/chill-cli/releases/download/v1.2.3/chilly_1.2.3_darwin_arm64.tar.gz" {
			t.Fatalf("request.URL = %q", request.URL.String())
		}
		return okResponse([]byte("archive-bytes")), nil
	})})
	payload, err := client.Download(context.Background(), "https://github.com/chill-institute/chill-cli/releases/download/v1.2.3/chilly_1.2.3_darwin_arm64.tar.gz")
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	if string(payload) != "archive-bytes" {
		t.Fatalf("payload = %q", string(payload))
	}
}

func TestDownloadRejectsDisallowedHost(t *testing.T) {
	t.Parallel()

	client := NewClient(&http.Client{})
	if _, err := client.Download(context.Background(), "https://example.invalid/chilly.tar.gz"); err == nil {
		t.Fatal("Download() error = nil, want invalid host")
	}
}

func TestValidateDownloadURLRejectsUnsupportedScheme(t *testing.T) {
	t.Parallel()

	if _, err := validateDownloadURL("http://github.com/chill-institute/chill-cli/releases/download/v1.2.3/chilly.tar.gz"); err == nil {
		t.Fatal("validateDownloadURL() error = nil, want unsupported scheme")
	}
}

func TestValidateDownloadURLAcceptsAllowedHosts(t *testing.T) {
	t.Parallel()

	for _, rawURL := range []string{
		"https://github.com/chill-institute/chill-cli/releases/download/v1.2.3/chilly.tar.gz",
		"https://objects.githubusercontent.com/archive",
		"https://release-assets.githubusercontent.com/archive",
		"https://github-releases.githubusercontent.com/archive",
	} {
		rawURL := rawURL
		t.Run(rawURL, func(t *testing.T) {
			t.Parallel()

			if _, err := validateDownloadURL(rawURL); err != nil {
				t.Fatalf("validateDownloadURL(%q) error = %v", rawURL, err)
			}
		})
	}
}

func TestDownloadRejectsUnexpectedStatus(t *testing.T) {
	t.Parallel()

	client := NewClient(&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadGateway,
			Body:       io.NopCloser(strings.NewReader("upstream failed")),
			Header:     make(http.Header),
		}, nil
	})})

	_, err := client.Download(context.Background(), "https://github.com/chill-institute/chill-cli/releases/download/v1.2.3/chilly_1.2.3_darwin_arm64.tar.gz")
	if err == nil || !strings.Contains(err.Error(), "unexpected status 502") {
		t.Fatalf("Download() error = %v", err)
	}
}

func TestDownloadReadBodyError(t *testing.T) {
	t.Parallel()

	client := NewClient(&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       errReadCloser{},
			Header:     make(http.Header),
		}, nil
	})})

	if _, err := client.Download(context.Background(), "https://github.com/chill-institute/chill-cli/releases/download/v1.2.3/chilly_1.2.3_darwin_arm64.tar.gz"); err == nil || !strings.Contains(err.Error(), "read release asset") {
		t.Fatalf("Download() error = %v", err)
	}
}

func TestFetchReleaseRejectsUnexpectedStatusAndInvalidJSON(t *testing.T) {
	t.Parallel()

	t.Run("unexpected status", func(t *testing.T) {
		t.Parallel()

		client := NewClient(&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("boom")),
				Header:     make(http.Header),
			}, nil
		})})

		if _, err := client.Latest(context.Background()); err == nil || !strings.Contains(err.Error(), "unexpected status 500") {
			t.Fatalf("Latest() error = %v", err)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		t.Parallel()

		client := NewClient(&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("{")),
				Header:     make(http.Header),
			}, nil
		})})

		if _, err := client.Latest(context.Background()); err == nil || !strings.Contains(err.Error(), "decode release metadata") {
			t.Fatalf("Latest() error = %v", err)
		}
	})
}

func TestFetchReleaseReadBodyError(t *testing.T) {
	t.Parallel()

	client := NewClient(&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       errReadCloser{},
			Header:     make(http.Header),
		}, nil
	})})

	if _, err := client.Latest(context.Background()); err == nil || !strings.Contains(err.Error(), "read release metadata") {
		t.Fatalf("Latest() error = %v", err)
	}
}

func TestByTagRejectsInvalidVersion(t *testing.T) {
	t.Parallel()

	client := NewClient(&http.Client{})
	if _, err := client.ByTag(context.Background(), "../v1.2.3"); err == nil {
		t.Fatal("ByTag() error = nil, want invalid version")
	}
}

func TestFetchReleasePropagatesTransportErrors(t *testing.T) {
	t.Parallel()

	client := NewClient(&http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return nil, errors.New("boom")
	})})

	if _, err := client.Latest(context.Background()); err == nil || !strings.Contains(err.Error(), "fetch release metadata") {
		t.Fatalf("Latest() error = %v", err)
	}
}

func TestExtractBinaryRejectsUnsupportedTarget(t *testing.T) {
	t.Parallel()

	if _, err := ExtractBinary([]byte("archive"), "plan9"); err == nil {
		t.Fatal("ExtractBinary() error = nil, want unsupported os")
	}
}

func TestExtractTarGZBinaryRejectsMissingBinary(t *testing.T) {
	t.Parallel()

	var archive bytes.Buffer
	gzipWriter := gzip.NewWriter(&archive)
	tarWriter := tar.NewWriter(gzipWriter)

	payload := []byte("notes")
	header := &tar.Header{
		Name: "README.txt",
		Mode: 0o644,
		Size: int64(len(payload)),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatalf("WriteHeader() error = %v", err)
	}
	if _, err := tarWriter.Write(payload); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("Close() tar error = %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("Close() gzip error = %v", err)
	}

	if _, err := extractTarGZBinary(archive.Bytes()); err == nil {
		t.Fatal("extractTarGZBinary() error = nil, want missing binary")
	}
}

func TestExtractTarGZBinaryRejectsInvalidArchive(t *testing.T) {
	t.Parallel()

	if _, err := extractTarGZBinary([]byte("not-a-gzip")); err == nil {
		t.Fatal("extractTarGZBinary() error = nil, want invalid archive")
	}
}

func TestParseChecksumHandlesErrorsAndPointerFormat(t *testing.T) {
	t.Parallel()

	assetName := "chilly_1.2.3_darwin_arm64.tar.gz"
	want := fmt.Sprintf("%x", sha256.Sum256([]byte("payload")))
	got, err := parseChecksum([]byte(want+"  *"+assetName+"\n"), assetName)
	if err != nil {
		t.Fatalf("parseChecksum() error = %v", err)
	}
	if got != want {
		t.Fatalf("parseChecksum() = %q, want %q", got, want)
	}

	if _, err := parseChecksum([]byte("deadbeef  "+assetName+"\n"), assetName); err == nil {
		t.Fatal("parseChecksum() error = nil, want invalid checksum")
	}
	if _, err := parseChecksum([]byte(want+"  other.tar.gz\n"), assetName); err == nil {
		t.Fatal("parseChecksum() error = nil, want missing checksum")
	}
	if _, err := parseChecksum([]byte(want+"  "+assetName+"\n"), " "); err == nil {
		t.Fatal("parseChecksum() error = nil, want empty asset name")
	}
}

func TestExtractZipBinaryRejectsMissingBinary(t *testing.T) {
	t.Parallel()

	var archive bytes.Buffer
	zipWriter := zip.NewWriter(&archive)
	fileWriter, err := zipWriter.Create("README.txt")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := fileWriter.Write([]byte("notes")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := zipWriter.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if _, err := extractZipBinary(archive.Bytes()); err == nil {
		t.Fatal("extractZipBinary() error = nil, want missing binary")
	}
}

func TestExtractZipBinaryRejectsInvalidArchive(t *testing.T) {
	t.Parallel()

	if _, err := extractZipBinary([]byte("not-a-zip")); err == nil {
		t.Fatal("extractZipBinary() error = nil, want invalid archive")
	}
}

func TestReplaceExecutableValidatesInputs(t *testing.T) {
	t.Parallel()

	if err := ReplaceExecutable("", []byte("binary"), 0o755); err == nil {
		t.Fatal("ReplaceExecutable() error = nil, want empty path error")
	}
	if err := ReplaceExecutable(filepath.Join(t.TempDir(), "chilly"), nil, 0o755); err == nil {
		t.Fatal("ReplaceExecutable() error = nil, want empty payload error")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func okResponse(payload []byte) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(payload)),
		Header:     make(http.Header),
	}
}

type errReadCloser struct{}

func (errReadCloser) Read([]byte) (int, error) {
	return 0, errors.New("boom")
}

func (errReadCloser) Close() error { return nil }
