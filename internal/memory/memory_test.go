package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEncodeNormalizesBook(t *testing.T) {
	data, err := Encode(Book{
		ProtectedTerms: []string{"prtr", "prtr", "codex"},
		Guidance:       []string{"Prefer concrete next actions.", "Prefer concrete next actions."},
	})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "version = 1") {
		t.Fatalf("encoded book = %q", text)
	}
	if strings.Count(text, "prtr") != 1 {
		t.Fatalf("encoded book = %q", text)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	repoRoot := t.TempDir()

	path, err := Save(repoRoot, Book{
		RepoSummary:    "Project summary",
		ProtectedTerms: []string{"prtr", "codex"},
		Guidance:       []string{"Prefer concrete next actions."},
	})
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if _, statErr := os.Stat(path); statErr != nil {
		t.Fatalf("Stat() error = %v", statErr)
	}

	book, err := Load(repoRoot)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if book.RepoSummary != "Project summary" {
		t.Fatalf("RepoSummary = %q", book.RepoSummary)
	}
	if len(book.ProtectedTerms) != 2 {
		t.Fatalf("ProtectedTerms = %#v", book.ProtectedTerms)
	}
}

func TestExtractBuildsRepoSummary(t *testing.T) {
	repoRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	readme := filepath.Join(repoRoot, "README.md")
	content := strings.Join([]string{
		"# prtr",
		"",
		"prtr turns intent into the next action.",
		"",
		"- Prefer concrete next actions.",
		"- Validate changes with tests.",
	}, "\n")
	if err := os.WriteFile(readme, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	book, err := Extract(repoRoot, []string{"README.md"})
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}
	if !strings.Contains(book.RepoSummary, "prtr") {
		t.Fatalf("RepoSummary = %q", book.RepoSummary)
	}
	if len(book.Guidance) == 0 {
		t.Fatalf("Guidance = %#v", book.Guidance)
	}
}
