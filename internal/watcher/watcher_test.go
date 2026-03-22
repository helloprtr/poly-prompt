package watcher_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/helloprtr/poly-prompt/internal/watcher"
)

func TestDetectEvent(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
		output   string
		wantKind string
	}{
		{"test failure", 1, "FAIL src/auth.test.js\n✕ login returns 401", "fix"},
		{"build error", 1, "error[E0001]: cannot find value", "fix"},
		{"panic", 1, "panic: runtime error: index out of range", "debug"},
		{"git conflict", 0, "CONFLICT (content): Merge conflict in main.go", "fix"},
		{"success", 0, "ok  github.com/foo/bar (1.2s)", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := watcher.DetectEvent(tt.exitCode, tt.output)
			if got != tt.wantKind {
				t.Errorf("DetectEvent(%d, ...) = %q, want %q", tt.exitCode, got, tt.wantKind)
			}
		})
	}
}

func TestWriteSuggestAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "watch-suggest")

	suggestion := watcher.Suggestion{
		Action:       "fix",
		ContextLines: []string{"2 failure lines", "git diff: auth.js +3/-1"},
		Branch:       "fix/login",
	}
	if err := watcher.WriteSuggest(path, suggestion); err != nil {
		t.Fatalf("WriteSuggest error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "fix") {
		t.Error("expected action 'fix' in suggest file")
	}
	if !strings.Contains(content, "2 failure lines") {
		t.Error("expected context line in suggest file")
	}
}

func TestReadAndClearSuggest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "watch-suggest")

	suggestion := watcher.Suggestion{Action: "debug"}
	_ = watcher.WriteSuggest(path, suggestion)

	got, err := watcher.ReadAndClearSuggest(path)
	if err != nil {
		t.Fatalf("ReadAndClearSuggest error: %v", err)
	}
	if got == nil || got.Action != "debug" {
		t.Errorf("expected action=debug, got %+v", got)
	}

	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Error("expected suggest file to be removed after read")
	}
}
