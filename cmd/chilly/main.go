package main

import (
	"io"
	"os"

	"github.com/chill-institute/chill-cli/v2/internal/cli"
)

var exit = os.Exit

func run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	return cli.Run(args, stdin, stdout, stderr)
}

func main() {
	exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
