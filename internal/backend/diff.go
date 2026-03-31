package backend

import (
	"strconv"
	"strings"
)

// DiffLine represents a single line in a unified diff.
type DiffLine struct {
	Type    string // "add", "remove", "context", "header"
	Content string
	OldLine int // 0 means not applicable
	NewLine int // 0 means not applicable
}

// DiffHunk represents a hunk in a unified diff, starting with an @@ header.
type DiffHunk struct {
	Header string
	Lines  []DiffLine
}

// parsePatch parses a GitHub patch string (unified diff) into structured hunks.
func parsePatch(patch string) []DiffHunk {
	if patch == "" {
		return nil
	}

	lines := strings.Split(patch, "\n")
	var hunks []DiffHunk
	var currentHunk *DiffHunk
	var oldLine, newLine int

	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			// Parse hunk header like "@@ -1,3 +1,4 @@"
			old, new_ := parseHunkHeader(line)
			hunk := DiffHunk{
				Header: line,
			}
			hunks = append(hunks, hunk)
			currentHunk = &hunks[len(hunks)-1]
			oldLine = old
			newLine = new_
			continue
		}

		if currentHunk == nil {
			continue
		}

		if len(line) == 0 {
			// Empty line is a context line
			currentHunk.Lines = append(currentHunk.Lines, DiffLine{
				Type:    "context",
				Content: "",
				OldLine: oldLine,
				NewLine: newLine,
			})
			oldLine++
			newLine++
		} else {
			switch line[0] {
			case '+':
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					Type:    "add",
					Content: line[1:],
					NewLine: newLine,
				})
				newLine++
			case '-':
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					Type:    "remove",
					Content: line[1:],
					OldLine: oldLine,
				})
				oldLine++
			case '\\':
				// "\ No newline at end of file" - skip
			default:
				// Context line (starts with space)
				content := line
				if len(content) > 0 && content[0] == ' ' {
					content = content[1:]
				}
				currentHunk.Lines = append(currentHunk.Lines, DiffLine{
					Type:    "context",
					Content: content,
					OldLine: oldLine,
					NewLine: newLine,
				})
				oldLine++
				newLine++
			}
		}
	}

	return hunks
}

// parseHunkHeader extracts old and new starting line numbers from a hunk header.
// Format: "@@ -old,count +new,count @@"
func parseHunkHeader(header string) (oldStart, newStart int) {
	// Find the range between @@ markers
	parts := strings.SplitN(header, "@@", 3)
	if len(parts) < 2 {
		return 1, 1
	}

	ranges := strings.TrimSpace(parts[1])
	rangeParts := strings.Fields(ranges)

	for _, part := range rangeParts {
		if strings.HasPrefix(part, "-") {
			nums := strings.SplitN(part[1:], ",", 2)
			oldStart, _ = strconv.Atoi(nums[0])
		} else if strings.HasPrefix(part, "+") {
			nums := strings.SplitN(part[1:], ",", 2)
			newStart, _ = strconv.Atoi(nums[0])
		}
	}

	if oldStart == 0 {
		oldStart = 1
	}
	if newStart == 0 {
		newStart = 1
	}

	return oldStart, newStart
}
