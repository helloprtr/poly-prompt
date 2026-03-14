package termbook

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestExtractDefaultSources(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTermbookFile(t, filepath.Join(root, "README.md"), "Use `go test ./...` with MyHTTPClient and --dry-run.\n")
	writeTermbookFile(t, filepath.Join(root, "docs", "guide.md"), "Set PRTR_TARGET and keep snake_case plus kebab-case.\n")
	writeTermbookFile(t, filepath.Join(root, "internal", "app", "main.go"), "var _ = BuildPrompt\n")
	writeTermbookFile(t, filepath.Join(root, "images", "banner.png"), "binary")

	extraction, err := Extract(root, nil)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	if !slices.Contains(extraction.Sources, "README.md") {
		t.Fatalf("Sources = %v", extraction.Sources)
	}
	if !slices.Contains(extraction.Sources, "docs/guide.md") {
		t.Fatalf("Sources = %v", extraction.Sources)
	}
	for _, want := range []string{"--dry-run", "BuildPrompt", "MyHTTPClient", "PRTR_TARGET", "kebab-case", "snake_case"} {
		if !slices.Contains(extraction.Terms, want) {
			t.Fatalf("Terms missing %q in %v", want, extraction.Terms)
		}
	}
	for _, unwanted := range []string{"Set", "Use"} {
		if slices.Contains(extraction.Terms, unwanted) {
			t.Fatalf("Terms should not contain %q in %v", unwanted, extraction.Terms)
		}
	}
}

func TestExtractHonorsExplicitPaths(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTermbookFile(t, filepath.Join(root, "README.md"), "Ignore RootThing\n")
	writeTermbookFile(t, filepath.Join(root, "docs", "guide.md"), "KeepOnlyThisToken and PRTR_TARGET\n")

	extraction, err := Extract(root, []string{"docs"})
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	if slices.Contains(extraction.Terms, "RootThing") {
		t.Fatalf("Terms = %v", extraction.Terms)
	}
	if !slices.Contains(extraction.Terms, "KeepOnlyThisToken") {
		t.Fatalf("Terms = %v", extraction.Terms)
	}
}

func writeTermbookFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
