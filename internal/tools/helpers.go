package tools

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
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

// guidedError wraps an API error with actionable suggestions for the AI.
// This is a core Claude Code design principle: errors should guide the next action.
func guidedError(err error, toolContext string, args map[string]any) string {
	msg := err.Error()

	// File not found
	if strings.Contains(msg, "404") {
		switch toolContext {
		case "read_file", "read_symbols", "blame":
			filePath := argStr(args, "file_path")
			// Extract just the filename for search suggestion
			parts := strings.Split(filePath, "/")
			name := parts[len(parts)-1]
			nameStem := strings.TrimSuffix(name, filepath.Ext(name))
			return fmt.Sprintf(
				"❌ File not found: %s\n\n"+
					"Suggestions:\n"+
					"  → gl_find_files(pattern: \"**/%s*\") to search for the file by name\n"+
					"  → gl_search_code(query: \"%s\") to find references to it\n"+
					"  → gl_list_directory() to browse the project structure",
				filePath, name, nameStem,
			)
		case "list_directory":
			dirPath := argStr(args, "path")
			return fmt.Sprintf(
				"❌ Directory not found: %s\n\n"+
					"Suggestions:\n"+
					"  → gl_list_directory() with no path to see the root structure\n"+
					"  → gl_find_files(pattern: \"**/*\") to search the entire repository",
				dirPath,
			)
		default:
			projectID := argStr(args, "project_id")
			return fmt.Sprintf(
				"❌ Not found (404) for project %s\n\n"+
					"Suggestions:\n"+
					"  → Check the project_id — use numeric ID (e.g. 609) or full path (e.g. mygroup/myproject)\n"+
					"  → Verify the ref (branch/tag) exists",
				projectID,
			)
		}
	}

	// Auth errors
	if strings.Contains(msg, "401") || strings.Contains(msg, "403") {
		return fmt.Sprintf(
			"❌ Access denied\n\n" +
				"The GitLab token does not have permission for this operation.\n" +
				"Ensure the token has `read_api` scope and access to the project.",
		)
	}

	// Rate limit
	if strings.Contains(msg, "429") {
		return "❌ Rate limited by GitLab API. Wait a moment and retry."
	}

	// Fallback
	return fmt.Sprintf("❌ Error: %v", err)
}
