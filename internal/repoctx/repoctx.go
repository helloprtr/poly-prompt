package repoctx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

var ErrNotGitRepo = errors.New("not a git repository")

type Summary struct {
	RepoName  string
	Branch    string
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
