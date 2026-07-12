package cli

import (
	"regexp"
	"testing"
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

func TestBoldWrapsWithANSI(t *testing.T) {
	t.Parallel()
	got := bold("hello")
	if stripANSI(got) != "hello" {
		t.Fatalf("bold() stripped = %q, want hello", stripANSI(got))
	}
	if got == "hello" {
		t.Fatal("bold() should contain ANSI codes")
	}
}

func TestDimWrapsWithANSI(t *testing.T) {
	t.Parallel()
	got := dim("label")
	if stripANSI(got) != "label" {
		t.Fatalf("dim() stripped = %q, want label", stripANSI(got))
	}
	if got == "label" {
		t.Fatal("dim() should contain ANSI codes")
	}
}

func TestSanitizeTerminalTextEscapesControls(t *testing.T) {
	t.Parallel()

	input := "Dune\nPart\tTwo\r\x1b[2J\x7f\u0085"
	want := `Dune\nPart\tTwo\r\x1b[2J\x7f\u0085`
	if got := sanitizeTerminalText(input); got != want {
		t.Fatalf("sanitizeTerminalText() = %q, want %q", got, want)
	}
}

func TestSanitizeTerminalTextEscapesUnicodeFormatting(t *testing.T) {
	t.Parallel()

	input := "safe\u202espoof\u200bhidden"
	want := `safe\u202espoof\u200bhidden`
	if got := sanitizeTerminalText(input); got != want {
		t.Fatalf("sanitizeTerminalText() = %q, want %q", got, want)
	}
}

func TestSanitizeTerminalTextPreservesPrintableUnicode(t *testing.T) {
	t.Parallel()

	const input = "Dune: Çöl Gezegeni — 日本語"
	if got := sanitizeTerminalText(input); got != input {
		t.Fatalf("sanitizeTerminalText() = %q, want %q", got, input)
	}
}
