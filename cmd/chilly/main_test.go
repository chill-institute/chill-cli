package main

import (
	"os"
	"testing"
)

func TestMainInvokesExitWithRunResult(t *testing.T) {
	originalExit := exit
	originalArgs := os.Args
	os.Args = []string{"chilly", "--help"}

	called := -1
	exit = func(code int) {
		called = code
	}
	t.Cleanup(func() {
		exit = originalExit
		os.Args = originalArgs
	})

	main()

	if called != 0 {
		t.Fatalf("exit code = %d", called)
	}
}
