package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/chill-institute/chill-cli/v2/internal/releaseassets"
)

type metadata struct {
	Version string `json:"version"`
}

func resolveVersion(cfg options) (string, error) {
	version := strings.TrimSpace(cfg.version)
	if version == "" {
		data, err := os.ReadFile(filepath.Join(cfg.distDir, "metadata.json"))
		if err != nil {
			return "", fmt.Errorf("read metadata version: %w", err)
		}
		var meta metadata
		if err := json.Unmarshal(data, &meta); err != nil {
			return "", fmt.Errorf("parse metadata version: %w", err)
		}
		version = strings.TrimSpace(meta.Version)
		if version == "" {
			return "", errors.New("metadata version is empty")
		}
	}
	return releaseassets.NormalizeVersion(version)
}

func archivePath(distDir string, sums releaseassets.Checksums, version string, t target) (string, error) {
	name := releaseassets.ArchiveName(version, t.goOS, t.goArch)
	path := filepath.Join(distDir, name)
	if err := sums.VerifyFile(name, path); err != nil {
		return "", fmt.Errorf("release archive for %s/%s: %w", t.goOS, t.goArch, err)
	}
	return path, nil
}

func extractBinary(archive string, entry string, destination string) error {
	var (
		src io.Reader
		err error
	)
	if strings.HasSuffix(archive, ".zip") {
		zr, openErr := zip.OpenReader(archive)
		if openErr != nil {
			return fmt.Errorf("open %s: %w", filepath.Base(archive), openErr)
		}
		defer func() {
			_ = zr.Close()
		}()
		src, err = zipEntry(&zr.Reader, entry)
	} else {
		f, openErr := os.Open(archive)
		if openErr != nil {
			return fmt.Errorf("open %s: %w", filepath.Base(archive), openErr)
		}
		defer func() {
			_ = f.Close()
		}()
		src, err = tarGzEntry(f, entry)
	}
	if err != nil {
		return fmt.Errorf("%s: %w", filepath.Base(archive), err)
	}
	return writeExecutable(src, destination)
}

func zipEntry(zr *zip.Reader, entry string) (io.Reader, error) {
	for _, file := range zr.File {
		if file.Name == entry && !file.FileInfo().IsDir() {
			return file.Open()
		}
	}
	return nil, fmt.Errorf("archive has no %s", entry)
}

func tarGzEntry(r io.Reader, entry string) (io.Reader, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("archive has no %s", entry)
		}
		if err != nil {
			return nil, err
		}
		if header.Name == entry && header.Typeflag == tar.TypeReg {
			return tr, nil
		}
	}
}

func writeExecutable(src io.Reader, destination string) error {
	out, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, src)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
