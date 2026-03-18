package repoctx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var ErrNotGitRepo = errors.New("not a git repository")

type Summary struct {
	RepoName  string
	Branch    string
	HeadSHA   string
	Changes   []string
	Truncated int
}

type Collector interface {
	Collect(ctx context.Context) (Summary, error)
}

type GitCollector struct{}

func New() *GitCollector {
	return &GitCollector{}
}

func (c *GitCollector) Collect(ctx context.Context) (Summary, error) {
	root, err := gitOutput(ctx, "rev-parse", "--show-toplevel")
	if err != nil {
		if isNotGitRepo(err) {
			return Summary{}, ErrNotGitRepo
		}
		return Summary{}, err
	}

	branch, err := gitOutput(ctx, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		if isNotGitRepo(err) {
			return Summary{}, ErrNotGitRepo
		}
		return Summary{}, err
	}

	headSHA, err := gitOutput(ctx, "rev-parse", "--short", "HEAD")
	if err != nil {
		if isNotGitRepo(err) {
			return Summary{}, ErrNotGitRepo
		}
		return Summary{}, err
	}

	status, err := gitOutput(ctx, "status", "--short", "--untracked-files=normal")
	if err != nil {
		if isNotGitRepo(err) {
			return Summary{}, ErrNotGitRepo
		}
		return Summary{}, err
	}

	lines := splitNonEmptyLines(status)
	summary := Summary{
		RepoName: filepath.Base(strings.TrimSpace(root)),
		Branch:   strings.TrimSpace(branch),
		HeadSHA:  strings.TrimSpace(headSHA),
	}

	const maxChanges = 20
	if len(lines) > maxChanges {
		summary.Changes = lines[:maxChanges]
		summary.Truncated = len(lines) - maxChanges
	} else {
		summary.Changes = lines
	}

	return summary, nil
}

func gitOutput(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = append(os.Environ(), "LANG=C", "LC_ALL=C")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = strings.TrimSpace(stdout.String())
		}
		if message != "" {
			return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, message)
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

func isNotGitRepo(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "not a git repository")
}

func splitNonEmptyLines(text string) []string {
	lines := strings.Split(text, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		result = append(result, line)
	}
	return result
}

// GitDiff returns staged+unstaged diff relative to HEAD, truncated to 200 lines.
// Returns empty string (no error) if not in a git repo or diff is empty.
func GitDiff(ctx context.Context, root string) (string, error) {
	raw, err := gitOutputInDir(ctx, root, "diff", "HEAD")
	if err != nil {
		// Not a git repo or no commits — not an error for callers
		return "", nil
	}
	return TruncateDiff(raw, 200), nil
}

// TruncateDiff truncates diff output to maxLines, appending a note if cut.
func TruncateDiff(diff string, maxLines int) string {
	lines := strings.Split(diff, "\n")
	if len(lines) <= maxLines {
		return diff
	}
	return strings.Join(lines[:maxLines], "\n") + fmt.Sprintf("\n[diff truncated at %d lines]", maxLines)
}

// LastTestOutput reads the last test run output from path.
// Returns empty string (no error) if file does not exist.
// Returns last 50 lines of the file.
func LastTestOutput(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("read test output: %w", err)
	}
	return tailLines(string(data), 50), nil
}

// LoadIgnorePatterns reads .prtrignore from repoRoot.
// Returns default patterns if file does not exist.
func LoadIgnorePatterns(repoRoot string) []string {
	data, err := os.ReadFile(filepath.Join(repoRoot, ".prtrignore"))
	if err != nil {
		return ParseIgnorePatterns("") // returns defaults only
	}
	return ParseIgnorePatterns(string(data))
}

func tailLines(text string, n int) string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if len(lines) <= n {
		return text
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}

// gitOutputInDir runs git in a specific directory (dir can be empty string for CWD).
// This is alongside the existing gitOutput() helper which uses the CWD by default.
func gitOutputInDir(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg != "" {
			return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, msg)
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}
