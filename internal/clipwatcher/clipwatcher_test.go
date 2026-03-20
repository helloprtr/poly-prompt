package clipwatcher_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/helloprtr/poly-prompt/internal/clipwatcher"
	"github.com/helloprtr/poly-prompt/internal/lastresponse"
)

type stubClip struct {
	values []string
	index  int
}

func (s *stubClip) Read(_ context.Context) (string, error) {
	if s.index >= len(s.values) {
		return s.values[len(s.values)-1], nil
	}
	v := s.values[s.index]
	s.index++
	return v, nil
}

func TestWatcherCapturesNewContent(t *testing.T) {
	dir := t.TempDir()
	lrStore := lastresponse.New(filepath.Join(dir, "last-response.json"))
	pidFile := filepath.Join(dir, "watcher.pid")

	// First read = baseline (the sent prompt), subsequent = AI response
	clip := &stubClip{values: []string{
		"sent prompt content",         // baseline
		"sent prompt content",         // first poll: no change
		"This is a long AI response that has more than one hundred characters in total to pass the length check.",
	}}

	w := clipwatcher.New(lrStore, pidFile, clip)
	w.PollInterval = 10 * time.Millisecond
	w.Timeout = 5 * time.Second

	ctx := context.Background()
	if err := w.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	entry, ok, err := lrStore.Read()
	if err != nil {
		t.Fatalf("lrStore.Read() error = %v", err)
	}
	if !ok {
		t.Fatal("no entry written")
	}
	if entry.Source != "clipboard" {
		t.Errorf("Source = %q, want clipboard", entry.Source)
	}
	if len(entry.Response) < 100 {
		t.Errorf("Response length = %d, want >= 100", len(entry.Response))
	}
}

func TestWatcherExitsOnTimeout(t *testing.T) {
	dir := t.TempDir()
	lrStore := lastresponse.New(filepath.Join(dir, "last-response.json"))
	pidFile := filepath.Join(dir, "watcher.pid")

	// Clipboard never changes
	clip := &stubClip{values: []string{"same content forever"}}

	w := clipwatcher.New(lrStore, pidFile, clip)
	w.PollInterval = 10 * time.Millisecond
	w.Timeout = 50 * time.Millisecond

	start := time.Now()
	if err := w.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if time.Since(start) > 500*time.Millisecond {
		t.Error("Run() took too long — timeout not respected")
	}

	_, ok, _ := lrStore.Read()
	if ok {
		t.Error("expected no entry written on timeout")
	}
}

func TestWatcherSkipsShortContent(t *testing.T) {
	dir := t.TempDir()
	lrStore := lastresponse.New(filepath.Join(dir, "last-response.json"))
	pidFile := filepath.Join(dir, "watcher.pid")

	clip := &stubClip{values: []string{
		"baseline",
		"short", // changed but < 100 chars
		"also short response",
	}}

	w := clipwatcher.New(lrStore, pidFile, clip)
	w.PollInterval = 10 * time.Millisecond
	w.Timeout = 50 * time.Millisecond

	_ = w.Run(context.Background())

	_, ok, _ := lrStore.Read()
	if ok {
		t.Error("expected no entry written for short content")
	}
}

func TestIsRunningStaleFile(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "stale.pid")

	// Write a PID that definitely doesn't exist
	_ = os.WriteFile(pidFile, []byte("99999999"), 0o600)

	if clipwatcher.IsRunning(pidFile) {
		t.Error("IsRunning() = true for stale PID, want false")
	}
	// Stale file should be cleaned up
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Error("stale PID file was not removed")
	}
}
