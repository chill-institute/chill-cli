// Package releaseassets reads the archives and checksum manifest of a chilly
// GitHub release.
package releaseassets

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

const ChecksumsFile = "checksums.txt"

var (
	versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+([+-][0-9A-Za-z.-]+)?$`)
	digestPattern  = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// NormalizeVersion strips a leading "v" and rejects anything that is not a
// release version.
func NormalizeVersion(version string) (string, error) {
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	if !versionPattern.MatchString(version) {
		return "", fmt.Errorf("invalid release version %q", version)
	}
	return version, nil
}

// ArchiveName returns the release archive name GoReleaser produces for a
// platform.
func ArchiveName(version, goos, goarch string) string {
	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("chilly_%s_%s_%s.%s", version, goos, goarch, ext)
}

// ArchiveURL returns the download URL of a release archive.
func ArchiveURL(version, name string) string {
	return fmt.Sprintf("https://github.com/chill-institute/chill-cli/releases/download/v%s/%s", version, name)
}

// Checksums maps asset names to lowercase hex SHA-256 digests.
type Checksums map[string]string

// ParseChecksums reads a sha256sum-style manifest.
func ParseChecksums(r io.Reader) (Checksums, error) {
	sums := Checksums{}
	scanner := bufio.NewScanner(r)
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		fields := strings.Fields(text)
		if len(fields) != 2 || !digestPattern.MatchString(fields[0]) {
			return nil, fmt.Errorf("checksums line %d is malformed", line)
		}
		name := strings.TrimPrefix(fields[1], "*")
		if _, ok := sums[name]; ok {
			return nil, fmt.Errorf("checksums list %s twice", name)
		}
		sums[name] = fields[0]
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read checksums: %w", err)
	}
	if len(sums) == 0 {
		return nil, fmt.Errorf("checksums are empty")
	}
	return sums, nil
}

// ReadChecksums parses the manifest at path.
func ReadChecksums(path string) (Checksums, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open checksums: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()
	return ParseChecksums(f)
}

// Digest returns the manifest digest for name.
func (c Checksums) Digest(name string) (string, error) {
	digest, ok := c[name]
	if !ok {
		return "", fmt.Errorf("checksums do not list %s", name)
	}
	return digest, nil
}

// VerifyFile checks that the file at path matches the manifest entry for name.
func (c Checksums) VerifyFile(name, path string) error {
	want, err := c.Digest(name)
	if err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", name, err)
	}
	defer func() {
		_ = f.Close()
	}()
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return fmt.Errorf("hash %s: %w", name, err)
	}
	if got := hex.EncodeToString(hash.Sum(nil)); got != want {
		return fmt.Errorf("%s digest %s does not match checksums %s", name, got, want)
	}
	return nil
}
