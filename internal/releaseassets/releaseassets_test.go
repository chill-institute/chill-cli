package releaseassets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const helloDigest = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"

func TestNormalizeVersion(t *testing.T) {
	for input, want := range map[string]string{"v2.6.7": "2.6.7", " 1.0.0-rc.1 ": "1.0.0-rc.1"} {
		got, err := NormalizeVersion(input)
		if err != nil || got != want {
			t.Fatalf("NormalizeVersion(%q) = %q, %v; want %q", input, got, err, want)
		}
	}
	for _, input := range []string{"", "latest", "2.6", "2.6.7/../x"} {
		if _, err := NormalizeVersion(input); err == nil {
			t.Fatalf("NormalizeVersion(%q) accepted an invalid version", input)
		}
	}
}

func TestArchiveNameAndURL(t *testing.T) {
	if got := ArchiveName("2.6.7", "darwin", "arm64"); got != "chilly_2.6.7_darwin_arm64.tar.gz" {
		t.Fatalf("darwin archive = %q", got)
	}
	if got := ArchiveName("2.6.7", "windows", "amd64"); got != "chilly_2.6.7_windows_amd64.zip" {
		t.Fatalf("windows archive = %q", got)
	}
	want := "https://github.com/chill-institute/chill-cli/releases/download/v2.6.7/chilly_2.6.7_linux_amd64.tar.gz"
	if got := ArchiveURL("2.6.7", "chilly_2.6.7_linux_amd64.tar.gz"); got != want {
		t.Fatalf("archive URL = %q", got)
	}
}

func TestParseChecksums(t *testing.T) {
	sums, err := ParseChecksums(strings.NewReader(helloDigest + "  a.tar.gz\n\n" + strings.Repeat("0", 64) + " *b.zip\n"))
	if err != nil {
		t.Fatal(err)
	}
	if sums["a.tar.gz"] != helloDigest || sums["b.zip"] != strings.Repeat("0", 64) {
		t.Fatalf("ParseChecksums() = %#v", sums)
	}
	if _, err := sums.Digest("missing"); err == nil {
		t.Fatal("Digest() accepted a missing asset")
	}
}

func TestParseChecksumsRejectsBadManifests(t *testing.T) {
	for name, input := range map[string]string{
		"empty":     "\n",
		"short":     "abc  a.tar.gz\n",
		"fields":    helloDigest + "\n",
		"uppercase": strings.ToUpper(helloDigest) + "  a.tar.gz\n",
		"duplicate": helloDigest + "  a.tar.gz\n" + helloDigest + "  a.tar.gz\n",
	} {
		if _, err := ParseChecksums(strings.NewReader(input)); err == nil {
			t.Fatalf("%s manifest accepted", name)
		}
	}
}

func TestVerifyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.tar.gz")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(dir, ChecksumsFile)
	if err := os.WriteFile(manifest, []byte(helloDigest+"  a.tar.gz\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sums, err := ReadChecksums(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := sums.VerifyFile("a.tar.gz", path); err != nil {
		t.Fatalf("VerifyFile() = %v", err)
	}
	if err := os.WriteFile(path, []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := sums.VerifyFile("a.tar.gz", path); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("VerifyFile() = %v, want digest mismatch", err)
	}
	if err := sums.VerifyFile("a.tar.gz", filepath.Join(dir, "missing")); err == nil {
		t.Fatal("VerifyFile() accepted a missing file")
	}
	if _, err := ReadChecksums(filepath.Join(dir, "missing.txt")); err == nil {
		t.Fatal("ReadChecksums() accepted a missing manifest")
	}
}
