package termbook

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	toml "github.com/pelletier/go-toml/v2"
)

var ErrNotGitRepo = errors.New("learn requires a git repository")

type Book struct {
	Version        int       `toml:"version"`
	GeneratedAt    time.Time `toml:"generated_at"`
	Sources        []string  `toml:"sources"`
	ProtectedTerms []string  `toml:"protected_terms"`
}

func FindRepoRoot(start string) (string, error) {
	current := strings.TrimSpace(start)
	if current == "" {
		var err error
		current, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("resolve working directory: %w", err)
		}
	}

	current, err := filepath.Abs(current)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path: %w", err)
	}

	for {
		gitPath := filepath.Join(current, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return current, nil
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("inspect %s: %w", gitPath, err)
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", ErrNotGitRepo
		}
		current = parent
	}
}

func Path(repoRoot string) string {
	return filepath.Join(repoRoot, ".prtr", "termbook.toml")
}

func Load(repoRoot string) (Book, error) {
	path := Path(repoRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Book{}, os.ErrNotExist
		}
		return Book{}, fmt.Errorf("read termbook: %w", err)
	}

	var book Book
	if err := toml.Unmarshal(data, &book); err != nil {
		return Book{}, fmt.Errorf("parse termbook: %w", err)
	}

	book.Sources = normalizeList(book.Sources)
	book.ProtectedTerms = normalizeList(book.ProtectedTerms)
	return book, nil
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
		return "", fmt.Errorf("write termbook: %w", err)
	}

	return path, nil
}

func Encode(book Book) ([]byte, error) {
	book.Version = 1
	if book.GeneratedAt.IsZero() {
		book.GeneratedAt = time.Now().UTC()
	}
	book.Sources = normalizeList(book.Sources)
	book.ProtectedTerms = normalizeList(book.ProtectedTerms)

	data, err := toml.Marshal(book)
	if err != nil {
		return nil, fmt.Errorf("encode termbook: %w", err)
	}
	return data, nil
}

func Merge(existing, next Book) Book {
	merged := Book{
		Version:        1,
		GeneratedAt:    next.GeneratedAt,
		Sources:        append(append([]string{}, existing.Sources...), next.Sources...),
		ProtectedTerms: append(append([]string{}, existing.ProtectedTerms...), next.ProtectedTerms...),
	}
	if merged.GeneratedAt.IsZero() {
		merged.GeneratedAt = time.Now().UTC()
	}
	merged.Sources = normalizeList(merged.Sources)
	merged.ProtectedTerms = normalizeList(merged.ProtectedTerms)
	return merged
}

func normalizeList(values []string) []string {
	seen := make(map[string]bool, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		normalized = append(normalized, value)
	}
	sort.Strings(normalized)
	return normalized
}
