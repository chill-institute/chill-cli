package cli

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

const (
	ansiReset = "\033[0m"
	ansiBold  = "\033[1m"
	ansiDim   = "\033[2m"
)

func bold(s string) string {
	return ansiBold + s + ansiReset
}

func dim(s string) string {
	return ansiDim + s + ansiReset
}

func sanitizeTerminalText(value string) string {
	if strings.IndexFunc(value, isUnsafeTerminalRune) < 0 {
		return value
	}

	var sanitized strings.Builder
	sanitized.Grow(len(value))
	for _, char := range value {
		if !isUnsafeTerminalRune(char) {
			sanitized.WriteRune(char)
			continue
		}
		escaped := strconv.QuoteRuneToASCII(char)
		sanitized.WriteString(escaped[1 : len(escaped)-1])
	}
	return sanitized.String()
}

func isUnsafeTerminalRune(char rune) bool {
	return unicode.IsControl(char) || unicode.Is(unicode.Cf, char)
}

// escapeUnsafeJSONRunes rewrites C1 controls and format characters in encoded
// JSON as \uXXXX escapes. encoding/json escapes ASCII controls but passes
// U+0080 and above through, so hosted strings could otherwise reach a
// terminal as raw control or bidi characters. Non-ASCII runes only occur
// inside JSON strings, so the output decodes to the same values.
func escapeUnsafeJSONRunes(encoded []byte) []byte {
	if bytes.IndexFunc(encoded, isUnsafeNonASCIIRune) < 0 {
		return encoded
	}

	escaped := make([]byte, 0, len(encoded)+16)
	for len(encoded) > 0 {
		char, size := utf8.DecodeRune(encoded)
		switch {
		case !isUnsafeNonASCIIRune(char):
			escaped = append(escaped, encoded[:size]...)
		case char > 0xffff:
			high, low := utf16.EncodeRune(char)
			escaped = fmt.Appendf(escaped, `\u%04x\u%04x`, high, low)
		default:
			escaped = fmt.Appendf(escaped, `\u%04x`, char)
		}
		encoded = encoded[size:]
	}
	return escaped
}

func isUnsafeNonASCIIRune(char rune) bool {
	return char >= utf8.RuneSelf && isUnsafeTerminalRune(char)
}
