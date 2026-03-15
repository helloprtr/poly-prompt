package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/helloprtr/poly-prompt/internal/termbook"
)

type Book struct {
	Version        int       `toml:"version"`
	GeneratedAt    time.Time `toml:"generated_at"`
	Sources        []string  `toml:"sources"`
	RepoSummary    string    `toml:"repo_summary"`
	ProtectedTerms []string  `toml:"protected_terms"`
	PreferredNames []string  `toml:"preferred_names"`
	Guidance       []string  `toml:"guidance"`
	CodingNorms    []string  `toml:"coding_norms"`
	TestingNorms   []string  `toml:"testing_norms"`
}

func Path(repoRoot string) string {
	return filepath.Join(repoRoot, ".prtr", "memory.toml")
}

func Load(repoRoot string) (Book, error) {
	data, err := os.ReadFile(Path(repoRoot))
	if err != nil {
		return Book{}, err
	}
	var book Book
	if err := toml.Unmarshal(data, &book); err != nil {
		return Book{}, fmt.Errorf("parse memory: %w", err)
	}
	return normalizeBook(book), nil
}

func Save(repoRoot string, book Book) (string, error) {
	data, err := Encode(book)
	if err != nil {
		return "", err
	}
	path := Path(repoRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create .prtr directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("write memory: %w", err)
	}
	return path, nil
}

func Encode(book Book) ([]byte, error) {
	book = normalizeBook(book)
	book.Version = 1
	if book.GeneratedAt.IsZero() {
		book.GeneratedAt = time.Now().UTC()
	}
	data, err := toml.Marshal(book)
	if err != nil {
		return nil, fmt.Errorf("encode memory: %w", err)
	}
	return data, nil
}

func Merge(existing, next Book) Book {
	merged := Book{
		Version:        1,
		GeneratedAt:    next.GeneratedAt,
		Sources:        append(append([]string{}, existing.Sources...), next.Sources...),
		RepoSummary:    firstNonEmpty(next.RepoSummary, existing.RepoSummary),
		ProtectedTerms: append(append([]string{}, existing.ProtectedTerms...), next.ProtectedTerms...),
		PreferredNames: append(append([]string{}, existing.PreferredNames...), next.PreferredNames...),
		Guidance:       append(append([]string{}, existing.Guidance...), next.Guidance...),
		CodingNorms:    append(append([]string{}, existing.CodingNorms...), next.CodingNorms...),
		TestingNorms:   append(append([]string{}, existing.TestingNorms...), next.TestingNorms...),
	}
	if merged.GeneratedAt.IsZero() {
		merged.GeneratedAt = time.Now().UTC()
	}
	return normalizeBook(merged)
}

func Extract(repoRoot string, paths []string) (Book, error) {
	terms, err := termbook.Extract(repoRoot, paths)
	if err != nil {
		return Book{}, err
	}
	texts := make(map[string]string, len(terms.Sources))
	for _, source := range terms.Sources {
		data, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(source)))
		if err != nil {
			return Book{}, fmt.Errorf("read %s: %w", source, err)
		}
		texts[source] = string(data)
	}

	book := Book{
		GeneratedAt:    time.Now().UTC(),
		Sources:        append([]string{}, terms.Sources...),
		RepoSummary:    extractRepoSummary(texts),
		ProtectedTerms: append([]string{}, terms.Terms...),
		PreferredNames: selectPreferredNames(terms.Terms),
		Guidance:       extractBullets(texts, "guidance"),
		CodingNorms:    extractBullets(texts, "coding"),
		TestingNorms:   extractBullets(texts, "testing"),
	}
	return normalizeBook(book), nil
}

func normalizeBook(book Book) Book {
	book.Sources = normalizeList(book.Sources)
	book.ProtectedTerms = normalizeList(book.ProtectedTerms)
	book.PreferredNames = normalizeList(book.PreferredNames)
	book.Guidance = normalizeList(book.Guidance)
	book.CodingNorms = normalizeList(book.CodingNorms)
	book.TestingNorms = normalizeList(book.TestingNorms)
	book.RepoSummary = strings.TrimSpace(book.RepoSummary)
	if len(book.Guidance) == 0 {
		book.Guidance = []string{
			"Preserve project and CLI identifiers exactly.",
			"Prefer the repo's existing names over generic rewrites.",
		}
	}
	if len(book.CodingNorms) == 0 {
		book.CodingNorms = []string{
			"Favor concrete file-level implementation guidance.",
			"Call out tests and validation steps.",
		}
	}
	if len(book.TestingNorms) == 0 {
		book.TestingNorms = []string{
			"Prefer regression-focused tests for changed behavior.",
		}
	}
	return book
}

func extractRepoSummary(texts map[string]string) string {
	readmePaths := prioritizedPaths(texts, func(path string) bool {
		base := strings.ToLower(filepath.Base(path))
		return strings.HasPrefix(base, "readme")
	})
	for _, path := range readmePaths {
		if summary := summarizeText(texts[path]); summary != "" {
			return summary
		}
	}
	docPaths := prioritizedPaths(texts, func(path string) bool {
		return strings.HasPrefix(strings.ToLower(path), "docs/")
	})
	for _, path := range docPaths {
		if summary := summarizeText(texts[path]); summary != "" {
			return summary
		}
	}
	return "Project-local prompt guidance generated from repo documents."
}

func summarizeText(text string) string {
	lines := strings.Split(text, "\n")
	title := ""
	paragraph := make([]string, 0, 4)
	inCode := false
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "```") {
			inCode = !inCode
			continue
		}
		if inCode || line == "" {
			if title != "" && len(paragraph) > 0 {
				break
			}
			continue
		}
		if title == "" && strings.HasPrefix(line, "#") {
			title = strings.TrimSpace(strings.TrimLeft(line, "#"))
			continue
		}
		if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") {
			continue
		}
		paragraph = append(paragraph, line)
		if len(strings.Join(paragraph, " ")) > 220 {
			break
		}
	}
	summary := strings.TrimSpace(strings.Join(paragraph, " "))
	switch {
	case title != "" && summary != "":
		return truncate(title+". "+summary, 260)
	case summary != "":
		return truncate(summary, 260)
	case title != "":
		return truncate(title, 260)
	default:
		return ""
	}
}

func selectPreferredNames(terms []string) []string {
	selected := make([]string, 0, len(terms))
	for _, term := range terms {
		switch {
		case strings.Contains(term, "_"),
			strings.Contains(term, "-"),
			strings.Contains(term, "/"),
			hasInnerUpper(term),
			isAllCaps(term):
			selected = append(selected, term)
		}
	}
	if len(selected) > 20 {
		selected = selected[:20]
	}
	return normalizeList(selected)
}

func extractBullets(texts map[string]string, kind string) []string {
	lines := make([]string, 0, 8)
	for _, path := range prioritizedPaths(texts, func(string) bool { return true }) {
		for _, raw := range strings.Split(texts[path], "\n") {
			line := strings.TrimSpace(raw)
			if !strings.HasPrefix(line, "- ") && !strings.HasPrefix(line, "* ") && !isOrderedBullet(line) {
				continue
			}
			line = strings.TrimSpace(trimBullet(line))
			if line == "" || len(line) > 160 {
				continue
			}
			lower := strings.ToLower(line)
			switch kind {
			case "testing":
				if strings.Contains(lower, "test") || strings.Contains(lower, "verify") || strings.Contains(lower, "regression") || strings.Contains(lower, "validation") {
					lines = append(lines, line)
				}
			case "coding":
				if strings.Contains(lower, "code") || strings.Contains(lower, "implementation") || strings.Contains(lower, "file") || strings.Contains(lower, "prompt") || strings.Contains(lower, "translate") {
					lines = append(lines, line)
				}
			default:
				if strings.Contains(lower, "prefer") || strings.Contains(lower, "preserve") || strings.Contains(lower, "avoid") || strings.Contains(lower, "keep") || strings.Contains(lower, "use") {
					lines = append(lines, line)
				}
			}
		}
	}
	if len(lines) > 10 {
		lines = lines[:10]
	}
	return normalizeList(lines)
}

func prioritizedPaths(texts map[string]string, keep func(string) bool) []string {
	paths := make([]string, 0, len(texts))
	for path := range texts {
		if keep(path) {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	return paths
}

func normalizeList(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func truncate(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	return text[:limit-3] + "..."
}

func isOrderedBullet(line string) bool {
	return len(line) > 3 && line[0] >= '0' && line[0] <= '9' && strings.Contains(line[:4], ".")
}

func trimBullet(line string) string {
	if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
		return line[2:]
	}
	if isOrderedBullet(line) {
		if idx := strings.Index(line, "."); idx >= 0 && idx+1 < len(line) {
			return line[idx+1:]
		}
	}
	return line
}

func hasInnerUpper(text string) bool {
	for i, r := range text {
		if i > 0 && r >= 'A' && r <= 'Z' {
			return true
		}
	}
	return false
}

func isAllCaps(text string) bool {
	hasUpper := false
	for _, r := range text {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			return false
		}
	}
	return hasUpper
}
