package termbook

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestFindRepoRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	nested := filepath.Join(root, "internal", "app")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	got, err := FindRepoRoot(nested)
	if err != nil {
		t.Fatalf("FindRepoRoot() error = %v", err)
	}
	if got != root {
		t.Fatalf("FindRepoRoot() = %q, want %q", got, root)
	}
}

func TestFindRepoRootReturnsErrorOutsideRepo(t *testing.T) {
	t.Parallel()

	_, err := FindRepoRoot(t.TempDir())
	if err != ErrNotGitRepo {
		t.Fatalf("FindRepoRoot() error = %v, want %v", err, ErrNotGitRepo)
	}
}

func TestSaveAndLoad(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	book := Book{
		GeneratedAt:    time.Unix(1700000000, 0).UTC(),
		Sources:        []string{"docs/guide.md", "README.md"},
		ProtectedTerms: []string{"MyHTTPClient", "PRTR_TARGET", "snake_case"},
	}

	path, err := Save(root, book)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if path != filepath.Join(root, ".prtr", "termbook.toml") {
		t.Fatalf("path = %q", path)
	}

	loaded, err := Load(root)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Version != 1 {
		t.Fatalf("Version = %d", loaded.Version)
	}
	if !loaded.GeneratedAt.Equal(book.GeneratedAt) {
		t.Fatalf("GeneratedAt = %v, want %v", loaded.GeneratedAt, book.GeneratedAt)
	}
	if !slices.Equal(loaded.ProtectedTerms, []string{"MyHTTPClient", "PRTR_TARGET", "snake_case"}) {
		t.Fatalf("ProtectedTerms = %v", loaded.ProtectedTerms)
	}
}
