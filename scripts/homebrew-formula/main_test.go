package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
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

func TestRunMatchesGoldenFormula(t *testing.T) {
	var out bytes.Buffer
	if err := run(&out, "v2.6.7", filepath.Join("testdata", "checksums-2.6.7.txt")); err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "chilly-2.6.7.rb"))
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != string(want) {
		t.Fatalf("formula differs from testdata/chilly-2.6.7.rb:\n%s", out.String())
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
