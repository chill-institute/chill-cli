package main

import (
	"os"
	"testing"
)

func TestMainInvokesExitWithRunResult(t *testing.T) {
	originalExit := exit
	originalArgs := os.Args
	os.Args = []string{"chilly", "no-such-command"}

	called := -1
	exit = func(code int) {
		called = code
	}
	t.Cleanup(func() {
		exit = originalExit
		os.Args = originalArgs
	})

	main()

	// An unknown command is a usage error, so main must forward os.Args and its exit code.
	if called != 2 {
		t.Fatalf("exit code = %d, want 2", called)
	}
}
