package session_test

import (
	"os"
	"os/exec"
	"testing"

	"github.com/helloprtr/poly-prompt/internal/session"
)

func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")
	run("commit", "--allow-empty", "-m", "init")
	return dir
}

func TestRepoRoot_InGitRepo(t *testing.T) {
	dir := initGitRepo(t)
	root, err := session.RepoRoot(dir)
	if err != nil {
		t.Fatalf("RepoRoot: %v", err)
	}
	if root == "" {
		t.Error("expected non-empty root")
	}
}

func TestRepoRoot_NotGitRepo(t *testing.T) {
	dir := t.TempDir()
	_, err := session.RepoRoot(dir)
	if err == nil {
		t.Error("expected error for non-git directory")
	}
}

func TestRepoHash_IsDeterministic(t *testing.T) {
	dir := initGitRepo(t)
	// Use the canonical repo root (resolved by git) for hashing
	root, err := session.RepoRoot(dir)
	if err != nil {
		t.Fatalf("RepoRoot: %v", err)
	}
	h1, err := session.RepoHash(root)
	if err != nil {
		t.Fatalf("RepoHash: %v", err)
	}
	h2, err := session.RepoHash(root)
	if err != nil {
		t.Fatalf("RepoHash second call: %v", err)
	}
	if h1 != h2 {
		t.Errorf("RepoHash not deterministic: %q vs %q", h1, h2)
	}
	if len(h1) != 8 {
		t.Errorf("expected 8-char hash, got %d chars: %q", len(h1), h1)
	}
}

func TestCurrentSHA(t *testing.T) {
	dir := initGitRepo(t)
	sha, err := session.CurrentSHA(dir)
	if err != nil {
		t.Fatalf("CurrentSHA: %v", err)
	}
	if len(sha) < 7 {
		t.Errorf("expected full SHA, got %q", sha)
	}
}

func TestDiff_EmptyWhenNothingChanged(t *testing.T) {
	dir := initGitRepo(t)
	sha, _ := session.CurrentSHA(dir)
	diff, err := session.Diff(dir, sha)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if diff != "" {
		t.Errorf("expected empty diff, got:\n%s", diff)
	}
}

func TestDiff_ShowsCommittedChanges(t *testing.T) {
	dir := initGitRepo(t)
	sha, _ := session.CurrentSHA(dir)

	// Commit a new file so git diff <sha> HEAD shows it
	if err := os.WriteFile(dir+"/hello.txt", []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "add", "hello.txt")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	cmd2 := exec.Command("git", "commit", "-m", "add hello")
	cmd2.Dir = dir
	if out, err := cmd2.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}

	diff, err := session.Diff(dir, sha)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if diff == "" {
		t.Error("expected non-empty diff after committing a file")
	}
}
