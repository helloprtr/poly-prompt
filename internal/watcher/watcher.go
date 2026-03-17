// internal/watcher/watcher.go
package watcher

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/helloprtr/poly-prompt/internal/repoctx"
)

const pollInterval = 500 * time.Millisecond

type shellEvent struct {
	ExitCode   int    `json:"exit_code"`
	Cmd        string `json:"cmd"`
	OutputFile string `json:"output_file"`
}

// Run starts the watcher subprocess. Blocks until ctx is cancelled.
func Run(ctx context.Context) error {
	sockPath, err := socketPath()
	if err != nil {
		return err
	}
	_ = os.Remove(sockPath)

	pidPath, err := PIDPath()
	if err != nil {
		return err
	}
	if err := writePID(pidPath); err != nil {
		return err
	}
	defer os.Remove(pidPath)

	if ln, err := net.Listen("unix", sockPath); err == nil {
		defer ln.Close()
		defer os.Remove(sockPath)
		return runSocketServer(ctx, ln)
	}
	return runPollingServer(ctx)
}

func runSocketServer(ctx context.Context, ln net.Listener) error {
	events := make(chan shellEvent, 8)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go handleConn(conn, events)
		}
	}()
	return processEvents(ctx, events)
}

func handleConn(conn net.Conn, events chan<- shellEvent) {
	defer conn.Close()
	var ev shellEvent
	if err := json.NewDecoder(conn).Decode(&ev); err == nil {
		select {
		case events <- ev:
		default:
		}
	}
}

func runPollingServer(ctx context.Context) error {
	eventFile := filepath.Join(os.TempDir(), "prtr-watch-event")
	events := make(chan shellEvent, 8)
	go func() {
		var lastMod time.Time
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(pollInterval):
				info, err := os.Stat(eventFile)
				if err != nil || !info.ModTime().After(lastMod) {
					continue
				}
				lastMod = info.ModTime()
				data, err := os.ReadFile(eventFile)
				if err != nil {
					continue
				}
				var ev shellEvent
				if json.Unmarshal(data, &ev) == nil {
					select {
					case events <- ev:
					default:
					}
				}
			}
		}
	}()
	return processEvents(ctx, events)
}

func processEvents(ctx context.Context, events <-chan shellEvent) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case ev := <-events:
			if err := handleEvent(ctx, ev); err != nil {
				fmt.Fprintf(os.Stderr, "prtr watch: %v\n", err)
			}
		}
	}
}

func handleEvent(ctx context.Context, ev shellEvent) error {
	// Note: output is only available if the user's shell captures it to ev.OutputFile.
	// In v0.8, the shell hook does not capture output automatically (requires tee-based wrapping).
	// Exit-code detection and git conflict detection remain functional.
	output, _ := repoctx.LastTestOutput(ev.OutputFile)
	action := DetectEvent(ev.ExitCode, output)
	if action == "" {
		return nil
	}

	suggestPath, err := SuggestPath()
	if err != nil {
		return err
	}

	var contextLines []string
	if lines := countLines(output); lines > 0 {
		contextLines = append(contextLines, fmt.Sprintf("%d output lines", lines))
	}

	if diff, err := repoctx.GitDiff(ctx, "."); err == nil && diff != "" {
		summary := summarizeDiff(diff)
		if summary != "" {
			contextLines = append(contextLines, "git diff: "+summary)
		}
	}

	branch := currentBranch(ctx)

	return WriteSuggest(suggestPath, Suggestion{
		Action:       action,
		ContextLines: contextLines,
		Branch:       branch,
	})
}

func summarizeDiff(diff string) string {
	var files []string
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "+++ b/") {
			files = append(files, strings.TrimPrefix(line, "+++ b/"))
		}
	}
	if len(files) == 0 {
		return ""
	}
	if len(files) == 1 {
		return files[0]
	}
	return fmt.Sprintf("%d files changed", len(files))
}

func currentBranch(ctx context.Context) string {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	return len(strings.Split(strings.TrimRight(s, "\n"), "\n"))
}

func socketPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "prtr", "watch.sock"), nil
}

func writePID(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0o644)
}
