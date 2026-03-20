package clipwatcher

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/helloprtr/poly-prompt/internal/lastresponse"
)

const minResponseLength = 100

// ClipboardReader is the subset of clipboard.Accessor used by the watcher.
type ClipboardReader interface {
	Read(ctx context.Context) (string, error)
}

// Watcher polls the clipboard and writes last-response.json on detection.
type Watcher struct {
	store        *lastresponse.Store
	pidFile      string
	clipboard    ClipboardReader
	PollInterval time.Duration
	Timeout      time.Duration
}

// New returns a Watcher with production defaults.
func New(store *lastresponse.Store, pidFile string, clipboard ClipboardReader) *Watcher {
	return &Watcher{
		store:        store,
		pidFile:      pidFile,
		clipboard:    clipboard,
		PollInterval: 500 * time.Millisecond,
		Timeout:      5 * time.Minute,
	}
}

// Run blocks until a response is captured, context is done, or timeout elapses.
// It writes and removes the PID file for deduplication.
func (w *Watcher) Run(ctx context.Context) error {
	if err := w.writePID(); err != nil {
		return fmt.Errorf("write PID file: %w", err)
	}
	defer os.Remove(w.pidFile)

	// Capture baseline after a short delay (clipboard may still have old content)
	time.Sleep(w.PollInterval)
	baseline, _ := w.clipboard.Read(ctx)

	timeout := time.After(w.Timeout)
	ticker := time.NewTicker(w.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timeout:
			return nil
		case <-ticker.C:
			content, err := w.clipboard.Read(ctx)
			if err != nil {
				continue
			}
			if content != baseline && len(content) >= minResponseLength {
				_ = w.store.Write("clipboard", content)
				return nil
			}
		}
	}
}

func (w *Watcher) writePID() error {
	if err := os.MkdirAll(filepath.Dir(w.pidFile), 0o755); err != nil {
		return err
	}
	return os.WriteFile(w.pidFile, []byte(strconv.Itoa(os.Getpid())), 0o600)
}

// IsRunning returns true if a watcher process is alive according to the PID file.
// If the file exists but the process is dead, it removes the stale file and returns false.
func IsRunning(pidFile string) bool {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		_ = os.Remove(pidFile)
		return false
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		_ = os.Remove(pidFile)
		return false
	}
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		if errors.Is(err, os.ErrProcessDone) || errors.Is(err, syscall.ESRCH) {
			_ = os.Remove(pidFile)
			return false
		}
	}
	return err == nil
}

// DefaultPIDFile returns ~/.config/prtr/clipboard-watcher.pid,
// respecting XDG_CONFIG_HOME if set.
func DefaultPIDFile() (string, error) {
	base := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "prtr", "clipboard-watcher.pid"), nil
}
