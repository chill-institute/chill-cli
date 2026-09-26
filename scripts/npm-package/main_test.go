package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chill-institute/chill-cli/v2/internal/releaseassets"
)

func TestRunCreatesRootAndPlatformPackages(t *testing.T) {
	dir := t.TempDir()
	distDir := filepath.Join(dir, "dist")
	outDir := filepath.Join(dir, "npm")
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeReleaseFixture(t, distDir, "1.2.3")

	if err := run(options{distDir: distDir, outDir: outDir, version: "v1.2.3"}); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	root := readPackageJSON(t, filepath.Join(outDir, "cli", "package.json"))
	if root.Name != rootPackageName {
		t.Fatalf("root package name = %q, want %q", root.Name, rootPackageName)
	}
	if root.Version != "1.2.3" {
		t.Fatalf("root package version = %q, want 1.2.3", root.Version)
	}
	if root.Bin[binaryName] != "bin/chilly.js" {
		t.Fatalf("root bin = %#v, want chilly launcher", root.Bin)
	}
	if len(root.OptionalDependencies) != len(targets) {
		t.Fatalf("optional dependency count = %d, want %d", len(root.OptionalDependencies), len(targets))
	}

	launcherPath := filepath.Join(outDir, "cli", "bin", "chilly.js")
	launcher, err := os.ReadFile(launcherPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(launcher), "@chill-institute/cli-darwin-arm64") {
		t.Fatalf("launcher does not contain platform package map:\n%s", launcher)
	}
	assertExecutable(t, launcherPath)

	platform := readPackageJSON(t, filepath.Join(outDir, "cli-darwin-arm64", "package.json"))
	if platform.Name != "@chill-institute/cli-darwin-arm64" {
		t.Fatalf("platform package name = %q", platform.Name)
	}
	if len(platform.OS) != 1 || platform.OS[0] != "darwin" {
		t.Fatalf("platform os = %#v, want darwin", platform.OS)
	}
	if len(platform.CPU) != 1 || platform.CPU[0] != "arm64" {
		t.Fatalf("platform cpu = %#v, want arm64", platform.CPU)
	}
	assertExecutable(t, filepath.Join(outDir, "cli-darwin-arm64", "bin", "chilly"))
	if data, err := os.ReadFile(filepath.Join(outDir, "cli-win32-x64", "bin", "chilly.exe")); err != nil || string(data) != "chilly.exe" {
		t.Fatalf("windows binary = %q, %v; want archive entry", data, err)
	}
}

func TestRunUsesMetadataVersionWhenVersionFlagIsEmpty(t *testing.T) {
	dir := t.TempDir()
	distDir := filepath.Join(dir, "dist")
	outDir := filepath.Join(dir, "npm")
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeReleaseFixture(t, distDir, "2.3.4")
	writeJSONFixture(t, filepath.Join(distDir, "metadata.json"), metadata{Version: "v2.3.4"})

	if err := run(options{distDir: distDir, outDir: outDir}); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	root := readPackageJSON(t, filepath.Join(outDir, "cli", "package.json"))
	if root.Version != "2.3.4" {
		t.Fatalf("root package version = %q, want 2.3.4", root.Version)
	}
}

func TestRunRejectsMissingOptions(t *testing.T) {
	if err := run(options{outDir: t.TempDir(), version: "1.0.0"}); err == nil {
		t.Fatal("run() error = nil, want missing dist directory error")
	}
	if err := run(options{distDir: t.TempDir(), version: "1.0.0"}); err == nil {
		t.Fatal("run() error = nil, want missing output directory error")
	}
}

func TestRunRejectsTamperedArchive(t *testing.T) {
	dir := t.TempDir()
	distDir := filepath.Join(dir, "dist")
	writeReleaseFixture(t, distDir, "1.0.0")
	if err := os.WriteFile(filepath.Join(distDir, "chilly_1.0.0_linux_arm64.tar.gz"), []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := run(options{distDir: distDir, outDir: filepath.Join(dir, "npm"), version: "1.0.0"})
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("run() error = %v, want digest mismatch", err)
	}
}

func TestRunRejectsArchiveWithoutBinary(t *testing.T) {
	dir := t.TempDir()
	distDir := filepath.Join(dir, "dist")
	writeReleaseFixture(t, distDir, "1.0.0")
	name := "chilly_1.0.0_windows_arm64.zip"
	writeZip(t, filepath.Join(distDir, name), "README.md")
	rewriteChecksums(t, distDir)

	err := run(options{distDir: distDir, outDir: filepath.Join(dir, "npm"), version: "1.0.0"})
	if err == nil || !strings.Contains(err.Error(), "archive has no chilly.exe") {
		t.Fatalf("run() error = %v, want missing binary error", err)
	}
}

func TestRunRejectsInvalidVersion(t *testing.T) {
	err := run(options{distDir: t.TempDir(), outDir: t.TempDir(), version: "latest"})
	if err == nil || !strings.Contains(err.Error(), "invalid release version") {
		t.Fatalf("run() error = %v, want invalid version error", err)
	}
}

func TestResolveVersionRejectsMissingMetadataVersion(t *testing.T) {
	dir := t.TempDir()
	writeJSONFixture(t, filepath.Join(dir, "metadata.json"), metadata{})

	_, err := resolveVersion(options{distDir: dir})
	if err == nil || !strings.Contains(err.Error(), "metadata version is empty") {
		t.Fatalf("resolveVersion() error = %v, want empty metadata version error", err)
	}
}

func TestResetOutputDirRejectsUnsafePath(t *testing.T) {
	if err := resetOutputDir("."); err == nil {
		t.Fatal("resetOutputDir() error = nil, want unsafe path error")
	}
}

func TestWriteJSONRejectsUnmarshalableValue(t *testing.T) {
	err := writeJSON(filepath.Join(t.TempDir(), "package.json"), make(chan struct{}))
	if err == nil || !strings.Contains(err.Error(), "marshal") {
		t.Fatalf("writeJSON() error = %v, want marshal error", err)
	}
}

func writeReleaseFixture(t *testing.T, distDir string, version string) {
	t.Helper()
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, target := range targets {
		path := filepath.Join(distDir, releaseassets.ArchiveName(version, target.goOS, target.goArch))
		if target.goOS == "windows" {
			writeZip(t, path, target.binaryFile)
		} else {
			writeTarGz(t, path, target.binaryFile)
		}
	}
	rewriteChecksums(t, distDir)
}

func writeTarGz(t *testing.T, path string, binary string) {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, name := range []string{"README.md", binary} {
		body := []byte(name)
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeZip(t *testing.T, path string, entry string) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(entry)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(entry)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func rewriteChecksums(t *testing.T, distDir string) {
	t.Helper()
	entries, err := os.ReadDir(distDir)
	if err != nil {
		t.Fatal(err)
	}
	var manifest strings.Builder
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "chilly_") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(distDir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&manifest, "%x  %s\n", sha256.Sum256(data), entry.Name())
	}
	if err := os.WriteFile(filepath.Join(distDir, releaseassets.ChecksumsFile), []byte(manifest.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeJSONFixture(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func readPackageJSON(t *testing.T, path string) packageJSON {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		t.Fatal(err)
	}
	return pkg
}

func assertExecutable(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("%s is not executable: %s", path, info.Mode())
	}
}
