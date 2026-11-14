package config

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

func NormalizeAPIBaseURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("api-base-url cannot be empty")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("parse api-base-url: %v", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("api-base-url must include scheme and host")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("api-base-url must start with http:// or https://")
	}
	if parsed.Scheme == "http" && !isLoopbackHost(parsed.Hostname()) {
		return "", fmt.Errorf("api-base-url must use https except for loopback development hosts")
	}
	if parsed.User != nil {
		return "", fmt.Errorf("api-base-url must not include user info")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("api-base-url must not include query or fragment data")
	}
	if strings.TrimSpace(parsed.Path) != "" && parsed.Path != "/" {
		return "", fmt.Errorf("api-base-url must not include a path")
	}

	return strings.TrimRight(trimmed, "/"), nil
}

func isLoopbackHost(raw string) bool {
	host := strings.ToLower(strings.TrimSpace(raw))
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
