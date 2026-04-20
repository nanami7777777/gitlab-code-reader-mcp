package tools

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestAddLineNumbers(t *testing.T) {
	content := "line one\nline two\nline three"
	result := addLineNumbers(content, 1)
	lines := strings.Split(result, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "1\tline one") {
		t.Errorf("first line should contain '1\\tline one', got %q", lines[0])
	}
	if !strings.Contains(lines[2], "3\tline three") {
		t.Errorf("third line should contain '3\\tline three', got %q", lines[2])
	}
}

func TestAddLineNumbersOffset(t *testing.T) {
	content := "hello\nworld"
	result := addLineNumbers(content, 50)
	if !strings.Contains(result, "50\thello") {
		t.Errorf("expected line 50, got %q", result)
	}
	if !strings.Contains(result, "51\tworld") {
		t.Errorf("expected line 51, got %q", result)
	}
}

func TestTruncateLine(t *testing.T) {
	short := "hello world"
	if truncateLine(short, 500) != short {
		t.Error("short line should not be truncated")
	}

	long := strings.Repeat("a", 600)
	result := truncateLine(long, 500)
	if len(result) >= 600 {
		t.Error("long line should be truncated")
	}
	if !strings.Contains(result, "[truncated:") {
		t.Error("truncated line should contain marker")
	}
}

func TestIsBinary(t *testing.T) {
	// Text content
	text := base64.StdEncoding.EncodeToString([]byte("hello world\nfoo bar"))
	if isBinary(text) {
		t.Error("text content should not be detected as binary")
	}

	// Binary content (contains null bytes)
	bin := base64.StdEncoding.EncodeToString([]byte{0x89, 0x50, 0x4E, 0x47, 0x00, 0x00})
	if !isBinary(bin) {
		t.Error("binary content should be detected as binary")
	}
}

func TestDecodeBase64(t *testing.T) {
	original := "hello world"
	encoded := base64.StdEncoding.EncodeToString([]byte(original))
	decoded, err := decodeBase64(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decoded != original {
		t.Errorf("expected %q, got %q", original, decoded)
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
	}
	for _, tt := range tests {
		got := formatSize(tt.bytes)
		if got != tt.want {
			t.Errorf("formatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}

func TestMinMaxInt(t *testing.T) {
	if minInt(3, 5) != 3 {
		t.Error("minInt(3,5) should be 3")
	}
	if maxInt(3, 5) != 5 {
		t.Error("maxInt(3,5) should be 5")
	}
}

func TestGetArgs(t *testing.T) {
	m := map[string]any{"key": "value", "num": 42.0}
	args := getArgs(m)
	if args["key"] != "value" {
		t.Error("getArgs should return the map")
	}

	args2 := getArgs("not a map")
	if len(args2) != 0 {
		t.Error("getArgs on non-map should return empty map")
	}
}

func TestArgStr(t *testing.T) {
	args := map[string]any{"name": "test", "empty": ""}
	if argStr(args, "name") != "test" {
		t.Error("argStr should return string value")
	}
	if argStr(args, "missing") != "" {
		t.Error("argStr on missing key should return empty")
	}
}

func TestArgFloat(t *testing.T) {
	args := map[string]any{"val": 3.14}
	v, ok := argFloat(args, "val")
	if !ok || v != 3.14 {
		t.Error("argFloat should return float value")
	}
	_, ok = argFloat(args, "missing")
	if ok {
		t.Error("argFloat on missing key should return false")
	}
}
