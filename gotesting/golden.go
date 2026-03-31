// Package gotesting provides test helpers for golden file testing.
package gotesting

import (
	"os"
	"path/filepath"
	"testing"
)

// CheckGoldenOutput compares actual against the golden file at path.
//
// When the environment variable WRITE_GOLDEN_OUTPUT=1 is set, the golden file
// is created or updated with actual content. The test fails if the file was
// created or its content changed, so that CI never silently accepts updates.
//
// In normal mode (no env var), the test fails if the golden file is missing or
// its content differs from actual.
func CheckGoldenOutput(t *testing.T, path string, actual string) {
	t.Helper()

	if os.Getenv("WRITE_GOLDEN_OUTPUT") == "1" {
		writeGolden(t, path, actual)
		return
	}

	expected, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("golden file %s does not exist; run with WRITE_GOLDEN_OUTPUT=1 to create it", path)
		}
		t.Fatalf("failed to read golden file %s: %v", path, err)
	}

	if string(expected) != actual {
		t.Errorf("golden file mismatch: %s\nRun with WRITE_GOLDEN_OUTPUT=1 to update.\n\n%s", path, diff(string(expected), actual))
	}
}

// writeGolden writes actual to path, failing the test if the file is new or changed.
func writeGolden(t *testing.T, path string, actual string) {
	t.Helper()

	existing, err := os.ReadFile(path)
	isNew := os.IsNotExist(err)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create directory for golden file %s: %v", path, err)
	}

	if err := os.WriteFile(path, []byte(actual), 0o644); err != nil {
		t.Fatalf("failed to write golden file %s: %v", path, err)
	}

	if isNew {
		t.Errorf("golden file %s created (new); re-run without WRITE_GOLDEN_OUTPUT to verify", path)
	} else if string(existing) != actual {
		t.Errorf("golden file %s updated (content changed); re-run without WRITE_GOLDEN_OUTPUT to verify", path)
	}
}

// diff produces a simple line-by-line comparison showing the first difference.
func diff(expected, actual string) string {
	eLines := splitLines(expected)
	aLines := splitLines(actual)

	maxLines := len(eLines)
	if len(aLines) > maxLines {
		maxLines = len(aLines)
	}

	for i := 0; i < maxLines; i++ {
		var eLine, aLine string
		if i < len(eLines) {
			eLine = eLines[i]
		}
		if i < len(aLines) {
			aLine = aLines[i]
		}
		if eLine != aLine {
			context := "first difference at line " + itoa(i+1) + ":\n"
			context += "  expected: " + truncate(eLine, 200) + "\n"
			context += "  actual:   " + truncate(aLine, 200) + "\n"
			context += "expected " + itoa(len(eLines)) + " lines, got " + itoa(len(aLines)) + " lines"
			return context
		}
	}

	return "files differ but no line difference found (possible trailing newline difference)"
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	for i > 0 {
		buf = append(buf, byte('0'+i%10))
		i /= 10
	}
	// reverse
	for l, r := 0, len(buf)-1; l < r; l, r = l+1, r-1 {
		buf[l], buf[r] = buf[r], buf[l]
	}
	return string(buf)
}
