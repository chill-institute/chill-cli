package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	cfg := options{}
	flag.StringVar(&cfg.distDir, "dist", "dist", "directory holding checksums.txt and the release archives")
	flag.StringVar(&cfg.outDir, "out", "dist/npm", "output directory for npm packages")
	flag.StringVar(&cfg.version, "version", "", "npm package version; defaults to dist/metadata.json")
	flag.Parse()

	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "prepare npm packages: %v\n", err)
		os.Exit(1)
	}
}
