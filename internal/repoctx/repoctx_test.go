package repoctx

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectReturnsSummary(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}

	repoDir := t.TempDir()
	runGit(t, repoDir, "init")
	runGit(t, repoDir, "config", "user.email", "test@example.com")
	runGit(t, repoDir, "config", "user.name", "Test User")
	writeFile(t, filepath.Join(repoDir, "README.md"), "# demo\n")
	runGit(t, repoDir, "add", "README.md")
	runGit(t, repoDir, "commit", "-m", "init")
	writeFile(t, filepath.Join(repoDir, "main.go"), "package main\n")
	writeFile(t, filepath.Join(repoDir, "notes.txt"), "todo\n")

	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	defer func() {
		_ = os.Chdir(previous)
	}()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	summary, err := New().Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if summary.RepoName == "" {
		t.Fatal("RepoName is empty")
	}
	if summary.Branch == "" {
		t.Fatal("Branch is empty")
	}
	if len(summary.Changes) != 2 {
		t.Fatalf("len(Changes) = %d, want 2 (%v)", len(summary.Changes), summary.Changes)
	}
	if !containsPrefix(summary.Changes, "?? notes.txt") {
		t.Fatalf("Changes = %v", summary.Changes)
	}
}

func TestCollectReturnsNotGitRepo(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	defer func() {
		_ = os.Chdir(previous)
	}()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	_, err = New().Collect(context.Background())
	if err == nil {
		t.Fatal("Collect() expected an error, got nil")
	}
	if err != ErrNotGitRepo {
		t.Fatalf("Collect() error = %v, want %v", err, ErrNotGitRepo)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v: %s", strings.Join(args, " "), err, string(output))
	}
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func containsPrefix(values []string, want string) bool {
	for _, value := range values {
		if strings.HasPrefix(value, want) {
			return true
		}
	}
	return false
}
