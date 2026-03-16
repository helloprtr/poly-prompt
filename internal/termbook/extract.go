package termbook

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

var (
	codeSpanPattern = regexp.MustCompile("`([^`\\n]{3,})`")
	tokenPattern    = regexp.MustCompile(`--[a-z0-9][a-z0-9-]*|[A-Z][A-Za-z0-9]*[a-z0-9][A-Za-z0-9]*|[a-z0-9]+_[a-z0-9_]+|[a-z0-9]+(?:-[a-z0-9]+){1,}|[A-Z][A-Z0-9_]{2,}|[A-Za-z0-9_.-]+/[A-Za-z0-9_./-]+`)
)

var ignoredDirs = map[string]bool{
	".git":         true,
	".prtr":        true,
	"build":        true,
	"dist":         true,
	"node_modules": true,
	"vendor":       true,
}

var ignoredExtensions = map[string]bool{
	".gif":   true,
	".gz":    true,
	".ico":   true,
	".jpeg":  true,
	".jpg":   true,
	".pdf":   true,
	".png":   true,
	".svg":   true,
	".tar":   true,
	".tgz":   true,
	".webp":  true,
	".woff":  true,
	".woff2": true,
	".zip":   true,
}

var stopWords = map[string]bool{
	"about": true, "after": true, "before": true, "branch": true, "build": true,
	"changes": true, "command": true, "commit": true, "context": true, "design": true,
	"default": true, "docker": true, "error": true, "example": true, "feature": true,
	"fix": true, "guide": true, "history": true, "input": true, "issue": true,
	"learn": true, "output": true, "patch": true, "prompt": true, "review": true,
	"setup": true, "source": true, "status": true, "summary": true, "target": true,
	"template": true, "test": true, "translation": true, "usage": true,
}

type Extraction struct {
	Sources []string
	Terms   []string
}

func Extract(repoRoot string, paths []string) (Extraction, error) {
	candidates, err := resolveSources(repoRoot, paths)
	if err != nil {
		return Extraction{}, err
	}

	terms := make(map[string]bool)
	sources := make([]string, 0, len(candidates))
	for _, source := range candidates {
		relative, err := filepath.Rel(repoRoot, source)
		if err != nil {
			return Extraction{}, fmt.Errorf("make relative path: %w", err)
		}
		if collectErr := collectFileTerms(source, terms); collectErr != nil {
			return Extraction{}, collectErr
		}
		sources = append(sources, filepath.ToSlash(relative))
	}

	return Extraction{
		Sources: normalizeList(sources),
		Terms:   mapKeysSorted(terms),
	}, nil
}

func resolveSources(repoRoot string, paths []string) ([]string, error) {
	if len(paths) == 0 {
		paths = []string{"README.md", "README", "README.txt", "docs", "cmd", "internal"}
	}

	seen := map[string]bool{}
	files := make([]string, 0)
	for _, input := range paths {
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		path := input
		if !filepath.IsAbs(path) {
			path = filepath.Join(repoRoot, input)
		}

		info, err := os.Stat(path)
		if err != nil {
			if errorsIsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("inspect %s: %w", path, err)
		}

		if info.IsDir() {
			err = filepath.WalkDir(path, func(current string, entry fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if entry.IsDir() {
					if ignoredDirs[entry.Name()] {
						return filepath.SkipDir
					}
					return nil
				}
				if shouldSkipFile(current) {
					return nil
				}
				if !seen[current] {
					seen[current] = true
					files = append(files, current)
				}
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("walk %s: %w", path, err)
			}
			continue
		}

		if shouldSkipFile(path) {
			continue
		}
		if !seen[path] {
			seen[path] = true
			files = append(files, path)
		}
	}

	sort.Strings(files)
	return files, nil
}

func collectFileTerms(path string, terms map[string]bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if len(data) == 0 || len(data) > 1<<20 || !utf8.Valid(data) {
		return nil
	}

	text := string(data)
	for _, match := range codeSpanPattern.FindAllStringSubmatch(text, -1) {
		for _, token := range extractFromSnippet(match[1]) {
			terms[token] = true
		}
	}
	for _, token := range tokenPattern.FindAllString(text, -1) {
		if keepToken(token) {
			terms[token] = true
		}
	}
	return nil
}

func extractFromSnippet(snippet string) []string {
	parts := strings.FieldsFunc(snippet, func(r rune) bool {
		switch r {
		case ' ', '\t', '\n', '\r', ',', ':', ';', '(', ')', '[', ']', '{', '}', '"', '\'':
			return true
		default:
			return false
		}
	})

	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if keepToken(part) {
			result = append(result, part)
		}
	}
	return result
}

func keepToken(token string) bool {
	token = strings.TrimSpace(token)
	if len(token) < 3 {
		return false
	}

	trimmed := strings.Trim(token, ".,!?<>")
	if len(trimmed) < 3 {
		return false
	}
	token = trimmed

	lower := strings.ToLower(token)
	if stopWords[lower] {
		return false
	}
	if strings.Count(token, "/") > 4 {
		return false
	}
	if strings.IndexFunc(token, func(r rune) bool { return r > utf8.RuneSelf }) >= 0 {
		return false
	}

	hasSignal := strings.Contains(token, "_") ||
		strings.Contains(token, "-") ||
		strings.Contains(token, "/") ||
		strings.HasPrefix(token, "--") ||
		hasIdentifierCase(token) ||
		isAllCapsToken(token)
	if !hasSignal {
		return false
	}

	return true
}

func shouldSkipFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ignoredExtensions[ext]
}

func hasIdentifierCase(token string) bool {
	upperCount := 0
	lowerCount := 0
	for i, r := range token {
		switch {
		case r >= 'A' && r <= 'Z':
			upperCount++
			if i > 0 {
				return true
			}
		case r >= 'a' && r <= 'z':
			lowerCount++
		}
	}
	return upperCount > 1 && lowerCount > 0
}

func isAllCapsToken(token string) bool {
	hasUpper := false
	for _, r := range token {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			return false
		}
	}
	return hasUpper
}

func mapKeysSorted(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func errorsIsNotExist(err error) bool {
	return err != nil && os.IsNotExist(err)
}

func isOrderedBullet(line string) bool {
	if len(line) < 3 {
		return false
	}
	if line[0] < '0' || line[0] > '9' {
		return false
	}
	// Scan past all leading digits
	i := 1
	for i < len(line) && line[i] >= '0' && line[i] <= '9' {
		i++
	}
	// Must be followed by ". "
	return i+1 < len(line) && line[i] == '.' && line[i+1] == ' '
}
