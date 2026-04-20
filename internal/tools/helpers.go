package tools

import (
	"encoding/base64"
	"fmt"
	"strings"
	"unicode/utf8"
)

// addLineNumbers formats content with line numbers like Claude Code's Read tool.
func addLineNumbers(content string, startLine int) string {
	lines := strings.Split(content, "\n")
	maxDigits := len(fmt.Sprintf("%d", startLine+len(lines)-1))
	var b strings.Builder
	for i, line := range lines {
		fmt.Fprintf(&b, "%*d\t%s\n", maxDigits, startLine+i, line)
	}
	return strings.TrimRight(b.String(), "\n")
}

// truncateLine cuts a line at maxLen characters (minified file protection).
func truncateLine(line string, maxLen int) string {
	if utf8.RuneCountInString(line) <= maxLen {
		return line
	}
	runes := []rune(line)
	return string(runes[:maxLen]) + fmt.Sprintf(" [truncated: %d chars]", len(runes))
}

// decodeBase64 decodes base64 file content to a UTF-8 string.
func decodeBase64(encoded string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// isBinary checks if base64-decoded content looks like a binary file.
func isBinary(encoded string) bool {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return true
	}
	sample := data
	if len(sample) > 8192 {
		sample = sample[:8192]
	}
	for _, b := range sample {
		if b == 0 {
			return true
		}
	}
	return false
}

// formatSize returns a human-readable file size.
func formatSize(bytes int64) string {
	switch {
	case bytes < 1024:
		return fmt.Sprintf("%d B", bytes)
	case bytes < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	}
}

// getArgs extracts the arguments map from a CallToolRequest.
func getArgs(arguments any) map[string]any {
	if m, ok := arguments.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func argStr(args map[string]any, key string) string {
	v, _ := args[key].(string)
	return v
}

func argFloat(args map[string]any, key string) (float64, bool) {
	v, ok := args[key].(float64)
	return v, ok
}

func argInt(args map[string]any, key, fallback string, def int) int {
	if v, ok := argFloat(args, key); ok && v > 0 {
		return int(v)
	}
	return def
}

// min returns the smaller of two ints.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// maxInt returns the larger of two ints.
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
