package session

import (
	"crypto/sha256"
	"fmt"
	"os/exec"
	"strings"
)

// RepoRoot returns the absolute git repo root path for the given directory.
func RepoRoot(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// RepoHash returns an 8-character deterministic identifier derived from the repo root path.
// Pass the output of RepoRoot to ensure symlink-resolved canonical path.
// SHA256 of a string cannot fail — the error return exists for interface consistency.
func RepoHash(repoRoot string) (string, error) {
	h := sha256.Sum256([]byte(repoRoot))
	return fmt.Sprintf("%x", h[:4]), nil
}

// CurrentSHA returns the HEAD commit SHA of the git repo at dir.
func CurrentSHA(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Diff returns all committed changes between baseSHA and HEAD.
// Untracked and uncommitted files are not included.
func Diff(dir, baseSHA string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "diff", baseSHA, "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("git diff: %w", err)
	}
	return string(out), nil
}
