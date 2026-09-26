package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func writeChecksums(t *testing.T, version string, skip string) string {
	t.Helper()
	var manifest strings.Builder
	for i, name := range []string{"darwin_amd64.tar.gz", "darwin_arm64.tar.gz", "linux_amd64.tar.gz", "linux_arm64.tar.gz", "windows_amd64.zip"} {
		if name == skip {
			continue
		}
		fmt.Fprintf(&manifest, "%064d  chilly_%s_%s\n", i+1, version, name)
	}
	path := filepath.Join(t.TempDir(), "checksums.txt")
	if err := os.WriteFile(path, []byte(manifest.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunPairsEachPlatformWithItsDigest(t *testing.T) {
	var out bytes.Buffer
	if err := run(&out, "v2.6.7", writeChecksums(t, "2.6.7", "")); err != nil {
		t.Fatal(err)
	}

	pairs := regexp.MustCompile(`url "([^"]+)"\n\s+sha256 "([0-9a-f]{64})"`).FindAllStringSubmatch(out.String(), -1)
	want := map[string]string{
		"darwin_amd64": fmt.Sprintf("%064d", 1),
		"darwin_arm64": fmt.Sprintf("%064d", 2),
		"linux_amd64":  fmt.Sprintf("%064d", 3),
		"linux_arm64":  fmt.Sprintf("%064d", 4),
	}
	if len(pairs) != len(want) {
		t.Fatalf("formula has %d url/sha256 pairs, want %d:\n%s", len(pairs), len(want), out.String())
	}
	for _, pair := range pairs {
		url, digest := pair[1], pair[2]
		prefix := "https://github.com/chill-institute/chill-cli/releases/download/v2.6.7/chilly_2.6.7_"
		platform := strings.TrimSuffix(strings.TrimPrefix(url, prefix), ".tar.gz")
		if want[platform] != digest {
			t.Fatalf("%s has sha256 %s, want %s", url, digest, want[platform])
		}
	}
	for _, order := range [][2]string{{"on_macos do", "darwin_amd64"}, {"on_intel do", "darwin_amd64"}, {"on_linux do", "linux_amd64"}} {
		if !strings.Contains(out.String(), order[0]) || strings.Index(out.String(), order[0]) > strings.Index(out.String(), order[1]) {
			t.Fatalf("%s must precede %s:\n%s", order[0], order[1], out.String())
		}
	}
	if strings.Contains(out.String(), "\n  version ") {
		t.Fatal("formula repeats the version the archive URL already carries")
	}
}

func TestRunRejectsIncompleteInput(t *testing.T) {
	var out bytes.Buffer
	if err := run(&out, "v2.6.7", writeChecksums(t, "2.6.7", "linux_arm64.tar.gz")); err == nil || !strings.Contains(err.Error(), "linux_arm64") {
		t.Fatalf("run() error = %v, want missing linux_arm64 digest", err)
	}
	if err := run(&out, "v2.6.8", writeChecksums(t, "2.6.7", "")); err == nil {
		t.Fatal("run() accepted checksums for another version")
	}
	if err := run(&out, "latest", writeChecksums(t, "2.6.7", "")); err == nil {
		t.Fatal("run() accepted an invalid version")
	}
	if err := run(&out, "v2.6.7", ""); err == nil {
		t.Fatal("run() accepted an empty checksums path")
	}
}
