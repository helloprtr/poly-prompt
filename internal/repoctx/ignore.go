package repoctx

import (
	"fmt"
	"path"
	"strings"
)

// defaultIgnorePatterns are always excluded, no .prtrignore needed.
var defaultIgnorePatterns = []string{
	".env", ".env.*", "*.key", "*.pem", "*secret*", "*password*",
}

// ParseIgnorePatterns parses a .prtrignore file body (newline-separated globs).
// Lines starting with # and empty lines are ignored.
// Supported: exact match, *.ext, *contains*, prefix.* — uses path.Match per filename.
// Unsupported gitignore features: negation (!), **, directory-only (/).
func ParseIgnorePatterns(content string) []string {
	var patterns []string
	patterns = append(patterns, defaultIgnorePatterns...)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns
}

// MatchesIgnore returns true if the basename of filePath matches any pattern.
func MatchesIgnore(patterns []string, filePath string) bool {
	base := path.Base(filePath)
	for _, pattern := range patterns {
		if ok, _ := path.Match(pattern, base); ok {
			return true
		}
	}
	return false
}

// FilterDiffHunks removes diff hunks for files matching patterns.
// Each suppressed hunk is replaced with a one-line note.
func FilterDiffHunks(diff string, patterns []string) string {
	if len(patterns) == 0 {
		return diff
	}
	var out strings.Builder
	// Split on "diff --git" boundaries
	sections := strings.Split(diff, "diff --git ")
	for i, section := range sections {
		if i == 0 {
			out.WriteString(section)
			continue
		}
		// Extract file path from "+++ b/<path>" line
		filePath := extractDiffFilePath(section)
		if filePath != "" && MatchesIgnore(patterns, filePath) {
			fmt.Fprintf(&out, "[excluded by .prtrignore: %s]\n", filePath)
		} else {
			out.WriteString("diff --git ")
			out.WriteString(section)
		}
	}
	return out.String()
}

func extractDiffFilePath(section string) string {
	for _, line := range strings.Split(section, "\n") {
		if strings.HasPrefix(line, "+++ b/") {
			return strings.TrimPrefix(line, "+++ b/")
		}
	}
	return ""
}
