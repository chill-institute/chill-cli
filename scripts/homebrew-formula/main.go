package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"

	"github.com/chill-institute/chill-cli/v2/internal/releaseassets"
)

type platform struct {
	block  string
	goOS   string
	goArch string
}

var platforms = []platform{
	{block: "on_macos", goOS: "darwin", goArch: "amd64"},
	{block: "on_macos", goOS: "darwin", goArch: "arm64"},
	{block: "on_linux", goOS: "linux", goArch: "amd64"},
	{block: "on_linux", goOS: "linux", goArch: "arm64"},
}

type archive struct {
	URL    string
	SHA256 string
}

type formula struct {
	MacIntel, MacARM, LinuxIntel, LinuxARM archive
}

var formulaTemplate = template.Must(template.New("formula").Parse(`# typed: false
# frozen_string_literal: true

class Chilly < Formula
  desc "Search chill.institute and send transfers from the terminal"
  homepage "https://chill.institute"
  license "MIT"

  on_macos do
    on_intel do
      url "{{ .MacIntel.URL }}"
      sha256 "{{ .MacIntel.SHA256 }}"
    end
    on_arm do
      url "{{ .MacARM.URL }}"
      sha256 "{{ .MacARM.SHA256 }}"
    end
  end

  on_linux do
    on_intel do
      url "{{ .LinuxIntel.URL }}"
      sha256 "{{ .LinuxIntel.SHA256 }}"
    end
    on_arm do
      url "{{ .LinuxARM.URL }}"
      sha256 "{{ .LinuxARM.SHA256 }}"
    end
  end

  def install
    bin.install "chilly"
  end

  test do
    system bin/"chilly", "version", "--output", "json"
  end
end
`))

func main() {
	var version, checksums string
	flag.StringVar(&version, "version", "", "release version (vX.Y.Z)")
	flag.StringVar(&checksums, "checksums", "", "path to the release checksums.txt")
	flag.Parse()

	if err := run(os.Stdout, version, checksums); err != nil {
		fmt.Fprintf(os.Stderr, "generate Homebrew formula: %v\n", err)
		os.Exit(1)
	}
}

func run(w io.Writer, version string, checksums string) error {
	version, err := releaseassets.NormalizeVersion(version)
	if err != nil {
		return err
	}
	if strings.TrimSpace(checksums) == "" {
		return fmt.Errorf("checksums path is required")
	}
	sums, err := releaseassets.ReadChecksums(checksums)
	if err != nil {
		return err
	}
	archives := make([]archive, 0, len(platforms))
	for _, p := range platforms {
		name := releaseassets.ArchiveName(version, p.goOS, p.goArch)
		digest, err := sums.Digest(name)
		if err != nil {
			return err
		}
		archives = append(archives, archive{URL: releaseassets.ArchiveURL(version, name), SHA256: digest})
	}
	return formulaTemplate.Execute(w, formula{
		MacIntel:   archives[0],
		MacARM:     archives[1],
		LinuxIntel: archives[2],
		LinuxARM:   archives[3],
	})
}
