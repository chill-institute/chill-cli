package cli

import (
	"strconv"
	"strings"
	"unicode"
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
	if strings.IndexFunc(value, unicode.IsControl) < 0 {
		return value
	}

	var sanitized strings.Builder
	sanitized.Grow(len(value))
	for _, char := range value {
		if !unicode.IsControl(char) {
			sanitized.WriteRune(char)
			continue
		}
		escaped := strconv.QuoteRuneToASCII(char)
		sanitized.WriteString(escaped[1 : len(escaped)-1])
	}
	return sanitized.String()
}
