# prtr v0.9 — Command System + Auto-Capture Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `resume` command, make `again` a hidden alias, update `take` to read from `last-response.json`, add `inspect --response`, and add a clipboard watcher subprocess that auto-captures AI responses.

**Architecture:** New `lastresponse` package owns the JSON store. `App` gets a `lastResponseStore` field following the same dependency-injection pattern as `historyStore`. The clipboard watcher runs as a detached subprocess (`prtr _watcher`) spawned by `runGo` after delivery. The terminal AI capture path (shell hook) requires v0.8 watch and is out of scope here.

**Tech Stack:** Go 1.24, cobra, existing `clipboard`, `history`, `config`, `app` packages. No new external dependencies.

---

## File Map

| Action | File | Responsibility |
|--------|------|----------------|
| Create | `internal/lastresponse/lastresponse.go` | JSON store: Read, Write, DefaultPath |
| Create | `internal/lastresponse/lastresponse_test.go` | Unit tests for store |
| Create | `internal/clipwatcher/clipwatcher.go` | Background watcher: poll clipboard, PID file, write lastresponse |
| Create | `internal/clipwatcher/clipwatcher_test.go` | Unit tests for watcher logic |
| Modify | `internal/app/app.go` | Add lastResponseStore dep; readLastResponse helper; runResume; runWatcher; spawnClipboardWatcher; update runTake, runInspect, runGo |
| Modify | `internal/app/command.go` | newResumeCommand; update newAgainCommand to hidden alias; update newInspectCommand; register hidden _watcher command |
| Modify | `internal/app/app_test.go` | Tests for new behaviors |
| Modify | `cmd/prtr/main.go` | Wire lastresponse.New into Dependencies |

---

## Chunk 1: `lastresponse` Package

**Files:**
- Create: `internal/lastresponse/lastresponse.go`
- Create: `internal/lastresponse/lastresponse_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/lastresponse/lastresponse_test.go
package lastresponse_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/helloprtr/poly-prompt/internal/lastresponse"
)

func TestStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := lastresponse.New(filepath.Join(dir, "last-response.json"))

	if err := store.Write("clipboard", "AI said hello"); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	entry, ok, err := store.Read()
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if !ok {
		t.Fatal("Read() ok = false, want true")
	}
	if entry.Source != "clipboard" {
		t.Errorf("Source = %q, want %q", entry.Source, "clipboard")
	}
	if entry.Response != "AI said hello" {
		t.Errorf("Response = %q, want %q", entry.Response, "AI said hello")
	}
	if entry.CapturedAt.IsZero() {
		t.Error("CapturedAt is zero")
	}
}

func TestStoreReadMissingFile(t *testing.T) {
	dir := t.TempDir()
	store := lastresponse.New(filepath.Join(dir, "nonexistent.json"))

	_, ok, err := store.Read()
	if err != nil {
		t.Fatalf("Read() error = %v, want nil", err)
	}
	if ok {
		t.Error("Read() ok = true, want false for missing file")
	}
}

func TestStoreAge(t *testing.T) {
	dir := t.TempDir()
	store := lastresponse.New(filepath.Join(dir, "last-response.json"))
	_ = store.Write("terminal", "response text")

	entry, _, _ := store.Read()
	age := time.Since(entry.CapturedAt)
	if age > 5*time.Second {
		t.Errorf("age = %v, expected < 5s", age)
	}
}

func TestDefaultPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	path, err := lastresponse.DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath() error = %v", err)
	}
	want := filepath.Join(dir, "prtr", "last-response.json")
	if path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

```bash
cd /Users/koo/dev/translateCLI-brew
go test ./internal/lastresponse/... -v
```
Expected: compile error — package does not exist

- [ ] **Step 3: Implement the package**

```go
// internal/lastresponse/lastresponse.go
package lastresponse

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Entry is the JSON structure stored in last-response.json.
type Entry struct {
	CapturedAt time.Time `json:"captured_at"`
	Source     string    `json:"source"` // "terminal" or "clipboard"
	Response   string    `json:"response"`
}

// Store reads and writes last-response.json at a fixed path.
type Store struct {
	path string
}

// New returns a Store backed by the given file path.
func New(path string) *Store {
	return &Store{path: path}
}

// DefaultPath returns ~/.config/prtr/last-response.json,
// respecting XDG_CONFIG_HOME if set.
func DefaultPath() (string, error) {
	base := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "prtr", "last-response.json"), nil
}

// Read returns the stored entry and true, or zero Entry and false if the
// file does not exist. Other I/O or parse errors are returned as non-nil error.
func (s *Store) Read() (Entry, bool, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Entry{}, false, nil
		}
		return Entry{}, false, fmt.Errorf("read last-response: %w", err)
	}
	var entry Entry
	if err := json.Unmarshal(data, &entry); err != nil {
		return Entry{}, false, fmt.Errorf("parse last-response: %w", err)
	}
	return entry, true, nil
}

// Path returns the file path this store reads from and writes to.
func (s *Store) Path() string {
	return s.path
}

// Write atomically stores a new entry with the current UTC time.
func (s *Store) Write(source, response string) error {
	entry := Entry{
		CapturedAt: time.Now().UTC(),
		Source:     source,
		Response:   response,
	}
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("encode last-response: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	return os.WriteFile(s.path, data, 0o600)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/lastresponse/... -v
```
Expected: all 4 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/lastresponse/
git commit -m "feat: add lastresponse package for AI response JSON store"
```

---

## Chunk 2: Wire `lastResponseStore` into App

**Files:**
- Modify: `internal/app/app.go:37-75` (Dependencies and App structs, New function)
- Modify: `internal/app/app_test.go`
- Modify: `cmd/prtr/main.go`

- [ ] **Step 1: Write failing compile-check test**

Add the following test to `internal/app/app_test.go`. It must fail before the field exists:

```go
func TestDependenciesAcceptsLastResponseStore(t *testing.T) {
	dir := t.TempDir()
	store := lastresponse.New(filepath.Join(dir, "last-response.json"))
	deps := Dependencies{LastResponseStore: store}
	_ = deps // compile-time field existence check
}
```

Add import `"github.com/helloprtr/poly-prompt/internal/lastresponse"` to `app_test.go`.

- [ ] **Step 2: Run to verify it fails**

```bash
go test ./internal/app/... -run TestDependenciesAcceptsLastResponseStore -v
```
Expected: compile error — `unknown field LastResponseStore`

- [ ] **Step 3: Add `lastResponseStore` to Dependencies and App**

In `internal/app/app.go`, add to the `Dependencies` struct (after `HistoryStore`):

```go
LastResponseStore *lastresponse.Store
```

Add to the `App` struct (after `historyStore`):

```go
lastResponseStore *lastresponse.Store
```

In the `New` function (wherever `historyStore` is assigned), add:

```go
lastResponseStore: deps.LastResponseStore,
```

Add the import:

```go
"github.com/helloprtr/poly-prompt/internal/lastresponse"
```

- [ ] **Step 4: Wire in main.go**

In `cmd/prtr/main.go`, add the import and wire the store:

```go
import "github.com/helloprtr/poly-prompt/internal/lastresponse"

// inside main(), after historyPath resolution:
lastResponsePath, err := lastresponse.DefaultPath()
if err != nil {
    fmt.Fprintf(os.Stderr, "failed to resolve last-response path: %v\n", err)
    os.Exit(1)
}

// inside app.New(Dependencies{...}):
LastResponseStore: lastresponse.New(lastResponsePath),
```

- [ ] **Step 5: Add `readLastResponse` helper to app.go**

This helper is used by both `runTake` and `runResume`. Add it to `internal/app/app.go`:

```go
// readLastResponse returns (responseText, source, error).
// It tries lastResponseStore first (if configured), then falls back to clipboard.
// Emits a warning to stderr when the stored response is >= 5 minutes old.
// Returns ("", "", nil) when lastResponseStore is absent and clipboard is empty.
func (a *App) readLastResponse(ctx context.Context) (text, source string, err error) {
	if a.lastResponseStore != nil {
		entry, ok, readErr := a.lastResponseStore.Read()
		if readErr != nil {
			return "", "", fmt.Errorf("read last-response: %w", readErr)
		}
		if ok && strings.TrimSpace(entry.Response) != "" {
			age := time.Since(entry.CapturedAt)
			if age >= 5*time.Minute {
				_, _ = fmt.Fprintf(a.stderr, "Using response from %s ago\n",
					age.Round(time.Minute).String())
			}
			return strings.TrimSpace(entry.Response), entry.Source, nil
		}
	}
	// Fallback: clipboard
	clipText, clipErr := a.clipboard.Read(ctx)
	if clipErr != nil {
		return "", "", clipErr
	}
	return strings.TrimSpace(clipText), "clipboard", nil
}
```

- [ ] **Step 6: Run compile-check test to verify it passes**

```bash
go test ./internal/app/... -run TestDependenciesAcceptsLastResponseStore -v
```
Expected: PASS

- [ ] **Step 7: Build to verify it compiles**

```bash
go build ./...
```
Expected: no errors

- [ ] **Step 8: Commit**

```bash
git add internal/app/app.go internal/app/app_test.go cmd/prtr/main.go
git commit -m "feat: wire lastResponseStore into App and main"
```

---

## Chunk 3: Update `runTake`

**Files:**
- Modify: `internal/app/app.go` (runTake, around line 1097)
- Modify: `internal/app/app_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/app/app_test.go — add these two tests

func TestRunTakeUsesLastResponseWhenAvailable(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	lrPath := filepath.Join(dir, "last-response.json")
	lrStore := lastresponse.New(lrPath)
	_ = lrStore.Write("clipboard", "This is the AI response text from lastresponse store.")

	var stdout, stderr bytes.Buffer
	app := New(Dependencies{
		Version:           "test",
		Stdout:            &stdout,
		Stderr:            &stderr,
		Clipboard:         &stubClipboard{read: "old clipboard content"},
		Launcher:          &stubLauncher{desc: "Terminal.app"},
		Automator:         &stubAutomator{desc: "Terminal.app"},
		SubmitConfirmer:   &stubConfirmer{},
		ConfigLoader:      config.Load,
		ConfigInit:        config.Init,
		LookupEnv:         func(string) (string, bool) { return "", false },
		HistoryStore:      history.New(filepath.Join(dir, "history.json")),
		LastResponseStore: lrStore,
		RepoContext:       &stubRepoContext{},
		RepoRootFinder:    func() (string, error) { return "", termbook.ErrNotGitRepo },
	})

	err := app.Execute(context.Background(), []string{"take", "--dry-run", "patch"}, nil, false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "AI response text from lastresponse") {
		t.Errorf("stdout = %q, want it to contain lastresponse content", stdout.String())
	}
}

func TestRunTakeFallsBackToClipboard(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	var stdout, stderr bytes.Buffer
	app := New(Dependencies{
		Version:         "test",
		Stdout:          &stdout,
		Stderr:          &stderr,
		Clipboard:       &stubClipboard{read: "clipboard fallback response content here."},
		Launcher:        &stubLauncher{desc: "Terminal.app"},
		Automator:       &stubAutomator{desc: "Terminal.app"},
		SubmitConfirmer: &stubConfirmer{},
		ConfigLoader:    config.Load,
		ConfigInit:      config.Init,
		LookupEnv:       func(string) (string, bool) { return "", false },
		HistoryStore:    history.New(filepath.Join(dir, "history.json")),
		RepoContext:     &stubRepoContext{},
		// LastResponseStore is nil — no JSON file wired, clipboard fallback expected
		RepoRootFinder: func() (string, error) { return "", termbook.ErrNotGitRepo },
	})

	err := app.Execute(context.Background(), []string{"take", "--dry-run", "patch"}, nil, false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "clipboard fallback response content") {
		t.Errorf("stdout = %q, want clipboard content", stdout.String())
	}
}

func TestRunTakeErrorWhenNothingCaptured(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	app := New(Dependencies{
		Version:         "test",
		Stdout:          &bytes.Buffer{},
		Stderr:          &bytes.Buffer{},
		Clipboard:       &stubClipboard{read: ""},
		Launcher:        &stubLauncher{desc: "Terminal.app"},
		Automator:       &stubAutomator{desc: "Terminal.app"},
		SubmitConfirmer: &stubConfirmer{},
		ConfigLoader:    config.Load,
		ConfigInit:      config.Init,
		LookupEnv:       func(string) (string, bool) { return "", false },
		HistoryStore:    history.New(filepath.Join(dir, "history.json")),
		RepoContext:     &stubRepoContext{},
		RepoRootFinder:  func() (string, error) { return "", termbook.ErrNotGitRepo },
	})

	err := app.Execute(context.Background(), []string{"take", "patch"}, nil, false)
	if err == nil {
		t.Fatal("Execute() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "No response captured") {
		t.Errorf("error = %q, want 'No response captured'", err.Error())
	}
}
```

Also add import at top of `app_test.go`:
```go
"github.com/helloprtr/poly-prompt/internal/lastresponse"
```

Also add a staleness warning test:

```go
func TestRunTakeWarnsStaleness(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	// Write a last-response entry with a backdated timestamp
	lrPath := filepath.Join(dir, "last-response.json")
	oldEntry := `{"captured_at":"2020-01-01T00:00:00Z","source":"clipboard","response":"old AI response content here for staleness test."}`
	_ = os.WriteFile(lrPath, []byte(oldEntry), 0o600)
	lrStore := lastresponse.New(lrPath)

	var stdout, stderr bytes.Buffer
	app := New(Dependencies{
		Version:           "test",
		Stdout:            &stdout,
		Stderr:            &stderr,
		Clipboard:         &stubClipboard{},
		Launcher:          &stubLauncher{desc: "Terminal.app"},
		Automator:         &stubAutomator{desc: "Terminal.app"},
		SubmitConfirmer:   &stubConfirmer{},
		ConfigLoader:      config.Load,
		ConfigInit:        config.Init,
		LookupEnv:         func(string) (string, bool) { return "", false },
		HistoryStore:      history.New(filepath.Join(dir, "history.json")),
		LastResponseStore: lrStore,
		RepoContext:       &stubRepoContext{},
		RepoRootFinder:    func() (string, error) { return "", termbook.ErrNotGitRepo },
	})

	err := app.Execute(context.Background(), []string{"take", "--dry-run", "patch"}, nil, false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(stderr.String(), "ago") {
		t.Errorf("stderr = %q, want staleness warning containing 'ago'", stderr.String())
	}
}
```

Add `"os"` to the imports in `app_test.go` if not already present.

- [ ] **Step 2: Run to verify they fail**

```bash
go test ./internal/app/... -run "TestRunTake" -v
```
Expected: compile error (lastresponse not imported yet) or FAIL

- [ ] **Step 3: Update `runTake` in app.go**

Replace lines 1103–1110 (the clipboard read block):

```go
// OLD:
clipboardText, err := a.clipboard.Read(ctx)
if err != nil {
    return err
}
clipboardText = strings.TrimSpace(clipboardText)
if clipboardText == "" {
    return errors.New("clipboard is empty; copy an answer and try again")
}
```

With:

```go
responseText, responseSource, err := a.readLastResponse(ctx)
if err != nil {
    return err
}
if responseText == "" {
    return errors.New("No response captured yet. Run prtr go first, then copy the AI's response.")
}
```

Then update every place in `runTake` where `clipboardText` or the hardcoded source label appears. There are **five** locations across the `--deep` and non-deep branches:

1. `Source: clipboardText` (deep branch, `deep.ExecutePatchRun` options) → `Source: responseText`
2. `SourceKind: "clipboard"` (deep branch, `deep.ExecutePatchRun` options) → `SourceKind: responseSource`
3. The stderr status format string in the deep branch:
   `"-> take:%s --deep | %s | clipboard | running\n"` → `"-> take:%s --deep | %s | %s | running\n"` with `responseSource` as the third `%s` argument
4. `surfaceInput: "clipboard"` in the deep opts block → `surfaceInput: responseSource`
5. `takePrompt(command.action, clipboardText)` (non-deep branch) → `takePrompt(command.action, responseText)`
6. `surfaceInput: "clipboard"` in the non-deep opts block → `surfaceInput: responseSource`

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/app/... -run "TestRunTake" -v
```
Expected: all 4 tests PASS (`TestRunTakeUsesLastResponseWhenAvailable`, `TestRunTakeFallsBackToClipboard`, `TestRunTakeErrorWhenNothingCaptured`, `TestRunTakeWarnsStaleness`)

- [ ] **Step 5: Commit**

```bash
git add internal/app/app.go internal/app/app_test.go
git commit -m "feat: update take to read from lastresponse store with clipboard fallback"
```

---

## Chunk 4: Add `prtr resume` + `again` hidden alias

**Files:**
- Modify: `internal/app/app.go` (add runResume, parseResumeCommand, resumePrompt)
- Modify: `internal/app/command.go` (newResumeCommand, update newAgainCommand)
- Modify: `internal/app/app_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/app/app_test.go

func TestRunResumeNoArgRerunsLastPrompt(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	histStore := history.New(filepath.Join(dir, "history.json"))
	_ = histStore.Append(history.Entry{
		ID:       "abc",
		Original: "explain this function",
		Target:   "claude",
	})

	var stdout bytes.Buffer
	app := New(Dependencies{
		Version:         "test",
		Stdout:          &stdout,
		Stderr:          &bytes.Buffer{},
		Clipboard:       &stubClipboard{},
		Launcher:        &stubLauncher{desc: "Terminal.app"},
		Automator:       &stubAutomator{desc: "Terminal.app"},
		SubmitConfirmer: &stubConfirmer{},
		ConfigLoader:    config.Load,
		ConfigInit:      config.Init,
		LookupEnv:       func(string) (string, bool) { return "", false },
		HistoryStore:    histStore,
		RepoContext:     &stubRepoContext{},
		RepoRootFinder:  func() (string, error) { return "", termbook.ErrNotGitRepo },
	})

	err := app.Execute(context.Background(), []string{"resume", "--dry-run"}, nil, false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "explain this function") {
		t.Errorf("stdout = %q, want original prompt", stdout.String())
	}
}

func TestRunResumeWithArgPrependsResponse(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	lrPath := filepath.Join(dir, "last-response.json")
	lrStore := lastresponse.New(lrPath)
	_ = lrStore.Write("clipboard", "The function uses O(n^2) complexity.")

	var stdout bytes.Buffer
	app := New(Dependencies{
		Version:           "test",
		Stdout:            &stdout,
		Stderr:            &bytes.Buffer{},
		Clipboard:         &stubClipboard{},
		Launcher:          &stubLauncher{desc: "Terminal.app"},
		Automator:         &stubAutomator{desc: "Terminal.app"},
		SubmitConfirmer:   &stubConfirmer{},
		ConfigLoader:      config.Load,
		ConfigInit:        config.Init,
		LookupEnv:         func(string) (string, bool) { return "", false },
		HistoryStore:      history.New(filepath.Join(dir, "history.json")),
		LastResponseStore: lrStore,
		RepoContext:       &stubRepoContext{},
		RepoRootFinder:    func() (string, error) { return "", termbook.ErrNotGitRepo },
	})

	err := app.Execute(context.Background(), []string{"resume", "--dry-run", "더 자세히 설명해줘"}, nil, false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "O(n^2) complexity") {
		t.Errorf("stdout = %q, want response content", out)
	}
	if !strings.Contains(out, "더 자세히 설명해줘") {
		t.Errorf("stdout = %q, want user message", out)
	}
}

func TestAgainIsHiddenAliasOfResume(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	histStore := history.New(filepath.Join(dir, "history.json"))
	_ = histStore.Append(history.Entry{
		ID:       "xyz",
		Original: "fix the bug",
		Target:   "claude",
	})

	var stdout bytes.Buffer
	app := New(Dependencies{
		Version:         "test",
		Stdout:          &stdout,
		Stderr:          &bytes.Buffer{},
		Clipboard:       &stubClipboard{},
		Launcher:        &stubLauncher{desc: "Terminal.app"},
		Automator:       &stubAutomator{desc: "Terminal.app"},
		SubmitConfirmer: &stubConfirmer{},
		ConfigLoader:    config.Load,
		ConfigInit:      config.Init,
		LookupEnv:       func(string) (string, bool) { return "", false },
		HistoryStore:    histStore,
		RepoContext:     &stubRepoContext{},
		RepoRootFinder:  func() (string, error) { return "", termbook.ErrNotGitRepo },
	})

	// `again` with no args should behave exactly like `resume` with no args
	err := app.Execute(context.Background(), []string{"again", "--dry-run"}, nil, false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "fix the bug") {
		t.Errorf("stdout = %q, want original prompt", stdout.String())
	}
}
```

- [ ] **Step 2: Run to verify they fail**

```bash
go test ./internal/app/... -run "TestRunResume|TestAgainIsHidden" -v
```
Expected: FAIL — `resume` command not found

- [ ] **Step 3: Add `parseResumeCommand` and `resumePrompt` to app.go**

Add near `parseReplayCommand` (around line 1996):

```go
type resumeCommandOptions struct {
	prompt  []string
	dryRun  bool
	noCopy  bool
	edit    bool
}

func parseResumeCommand(args []string) (resumeCommandOptions, error) {
	command := resumeCommandOptions{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--dry-run":
			command.dryRun = true
		case arg == "--no-copy":
			command.noCopy = true
		case arg == "--edit" || arg == "--interactive" || arg == "-i":
			command.edit = true
		case strings.HasPrefix(arg, "-"):
			return resumeCommandOptions{}, usageError{message: fmt.Sprintf("unknown resume flag %q", arg)}
		default:
			command.prompt = append(command.prompt, arg)
		}
	}
	return command, nil
}

func resumePrompt(response, userMessage string) string {
	return strings.Join([]string{
		response,
		"",
		"---",
		userMessage,
	}, "\n")
}
```

- [ ] **Step 4: Add `runResume` to app.go**

Add after `runAgain` (around line 1050):

```go
func (a *App) runResume(ctx context.Context, args []string, stdin io.Reader, stdinPiped bool) error {
	command, err := parseResumeCommand(args)
	if err != nil {
		return err
	}

	// No-arg: re-run last prompt verbatim (same behavior as legacy `again`)
	if len(command.prompt) == 0 && !stdinPiped {
		entry, err := a.latestHistoryEntry()
		if err != nil {
			// Wrap to match spec §1.1 error message exactly.
			return errors.New("No previous prompt found. Run prtr go first.")
		}
		opts := runOptions{
			target:          entry.Target,
			role:            entry.Role,
			templatePreset:  entry.TemplatePreset,
			sourceLang:      entry.SourceLang,
			targetLang:      entry.TargetLang,
			translationMode: entry.TranslationMode,
			interactive:     command.edit,
			noCopy:          command.dryRun || command.noCopy,
			launch:          !command.dryRun,
			paste:           !command.dryRun,
			compactStatus:   true,
			surfaceMode:     blankDefault(entry.Shortcut, "ask"),
			surfaceInput:    "history",
			surfaceDelivery: surfaceDeliveryLabel(command.dryRun),
			engine:          blankDefault(entry.Engine, "classic"),
			parentID:        entry.ID,
		}
		return a.executePrompt(ctx, opts, entry.Original, entry.Shortcut)
	}

	// With message: prepend last response as context
	responseText, _, err := a.readLastResponse(ctx)
	if err != nil {
		return err
	}
	if responseText == "" {
		return errors.New("No response captured yet. Run prtr go first, then copy the AI's response.")
	}

	userMsg := strings.Join(command.prompt, " ")
	if stdinPiped {
		stdinBytes, readErr := io.ReadAll(stdin)
		if readErr != nil {
			return fmt.Errorf("read stdin: %w", readErr)
		}
		if userMsg == "" {
			userMsg = strings.TrimSpace(string(stdinBytes))
		}
	}

	// latestHistoryEntry error is intentionally ignored here: if history is
	// unavailable, we proceed with zero-value Entry (empty target, no parent).
	// executePrompt handles the empty target via its own config fallback.
	entry, _ := a.latestHistoryEntry()
	opts := runOptions{
		target:          entry.Target,
		sourceLang:      entry.SourceLang,
		targetLang:      entry.TargetLang,
		translationMode: entry.TranslationMode,
		interactive:     command.edit,
		noCopy:          command.dryRun || command.noCopy,
		launch:          !command.dryRun,
		paste:           !command.dryRun,
		compactStatus:   true,
		surfaceMode:     "resume",
		surfaceInput:    "last-response",
		surfaceDelivery: surfaceDeliveryLabel(command.dryRun),
		engine:          blankDefault(entry.Engine, "classic"),
		parentID:        entry.ID,
	}
	return a.executePrompt(ctx, opts, resumePrompt(responseText, userMsg), "")
}
```

- [ ] **Step 5: Register `resume` command and update `again` in command.go**

Add `newResumeCommand` to `internal/app/command.go`:

```go
func (a *App) newResumeCommand(ctx context.Context, stdin io.Reader, stdinPiped bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "resume [message...]",
		Short:              "Continue the last AI conversation, or re-run it if no message is given.",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return cmd.Help()
			}
			return a.runResume(ctx, args, stdin, stdinPiped)
		},
	}
	return cmd
}
```

Replace `newAgainCommand` registration in `Command()` with:

```go
root.AddCommand(a.newResumeCommand(ctx, stdin, stdinPiped))
```

Update `newAgainCommand` to be a hidden alias:

```go
func (a *App) newAgainCommand(ctx context.Context, stdin io.Reader, stdinPiped bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "again [message...]",
		Short:              "Alias for resume.",
		Hidden:             true,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if wantsHelp(args) {
				return a.newResumeCommand(ctx, stdin, stdinPiped).Help()
			}
			return a.runResume(ctx, args, stdin, stdinPiped)
		},
	}
	return cmd
}
```

In `Command()`, keep `root.AddCommand(a.newAgainCommand(ctx, stdin, stdinPiped))` — it just doesn't appear in help now.

- [ ] **Step 6: Run tests to verify they pass**

```bash
go test ./internal/app/... -run "TestRunResume|TestAgainIsHidden" -v
```
Expected: all 3 tests PASS

- [ ] **Step 7: Commit**

```bash
git add internal/app/app.go internal/app/command.go internal/app/app_test.go
git commit -m "feat: add resume command and make again a hidden alias"
```

---

## Chunk 5: `inspect --response`

**Files:**
- Modify: `internal/app/app.go` (runInspect)
- Modify: `internal/app/app_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/app/app_test.go

func TestInspectResponseShowsCapturedEntry(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	lrPath := filepath.Join(dir, "last-response.json")
	lrStore := lastresponse.New(lrPath)
	_ = lrStore.Write("terminal", "Here is the implementation: func foo() { return 42 }")

	var stdout bytes.Buffer
	app := New(Dependencies{
		Version:           "test",
		Stdout:            &stdout,
		Stderr:            &bytes.Buffer{},
		Clipboard:         &stubClipboard{},
		Launcher:          &stubLauncher{desc: "Terminal.app"},
		Automator:         &stubAutomator{desc: "Terminal.app"},
		SubmitConfirmer:   &stubConfirmer{},
		ConfigLoader:      config.Load,
		ConfigInit:        config.Init,
		LookupEnv:         func(string) (string, bool) { return "", false },
		HistoryStore:      history.New(filepath.Join(dir, "history.json")),
		LastResponseStore: lrStore,
		RepoContext:       &stubRepoContext{},
		RepoRootFinder:    func() (string, error) { return "", termbook.ErrNotGitRepo },
	})

	err := app.Execute(context.Background(), []string{"inspect", "--response"}, nil, false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "terminal") {
		t.Errorf("stdout = %q, want source label 'terminal'", out)
	}
	if !strings.Contains(out, "Here is the implementation") {
		t.Errorf("stdout = %q, want response content", out)
	}
}

func TestInspectResponseWhenNothingCaptured(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	var stdout bytes.Buffer
	app := New(Dependencies{
		Version:         "test",
		Stdout:          &stdout,
		Stderr:          &bytes.Buffer{},
		Clipboard:       &stubClipboard{},
		Launcher:        &stubLauncher{desc: "Terminal.app"},
		Automator:       &stubAutomator{desc: "Terminal.app"},
		SubmitConfirmer: &stubConfirmer{},
		ConfigLoader:    config.Load,
		ConfigInit:      config.Init,
		LookupEnv:       func(string) (string, bool) { return "", false },
		HistoryStore:    history.New(filepath.Join(dir, "history.json")),
		RepoContext:     &stubRepoContext{},
		RepoRootFinder:  func() (string, error) { return "", termbook.ErrNotGitRepo },
	})

	err := app.Execute(context.Background(), []string{"inspect", "--response"}, nil, false)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "No response captured") {
		t.Errorf("stdout = %q, want 'No response captured'", stdout.String())
	}
}
```

- [ ] **Step 2: Run to verify they fail**

```bash
go test ./internal/app/... -run "TestInspectResponse" -v
```
Expected: FAIL — `inspect --response` not handled

- [ ] **Step 3: Add `runInspectResponse` and update `runInspect`**

Add this helper to `internal/app/app.go`:

```go
func (a *App) runInspectResponse() error {
	const maxPreview = 500

	if a.lastResponseStore == nil {
		_, _ = fmt.Fprintln(a.stdout, "No response captured yet. Run prtr go, then copy or let prtr capture the AI's response.")
		return nil
	}

	entry, ok, err := a.lastResponseStore.Read()
	if err != nil {
		return fmt.Errorf("read last-response: %w", err)
	}
	if !ok || strings.TrimSpace(entry.Response) == "" {
		_, _ = fmt.Fprintln(a.stdout, "No response captured yet. Run prtr go, then copy or let prtr capture the AI's response.")
		return nil
	}

	age := time.Since(entry.CapturedAt).Round(time.Minute)
	ageStr := age.String()
	if age < time.Minute {
		ageStr = "just now"
	}

	preview := entry.Response
	truncated := false
	if len(preview) > maxPreview {
		preview = preview[:maxPreview]
		truncated = true
	}

	_, _ = fmt.Fprintf(a.stdout, "Source:   %s\nCaptured: %s\n%s\n%s\n",
		entry.Source, ageStr,
		strings.Repeat("─", 40),
		preview,
	)
	if truncated {
		_, _ = fmt.Fprintf(a.stdout, "\n(truncated at %d chars — full content in %s)\n",
			maxPreview, a.lastResponseStore.Path())
	}
	return nil
}
```

At the top of `runInspect`, add an early branch before calling `parseRunOptions`:

```go
func (a *App) runInspect(ctx context.Context, args []string, stdin io.Reader, stdinPiped bool) error {
	// Handle --response flag before normal option parsing
	for _, arg := range args {
		if arg == "--response" {
			return a.runInspectResponse()
		}
	}

	// ... existing code continues unchanged
	opts, positional, err := parseRunOptions(args)
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/app/... -run "TestInspectResponse" -v
```
Expected: both tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/app.go internal/app/app_test.go
git commit -m "feat: add inspect --response to preview captured AI response"
```

---

## Chunk 6: Clipboard Watcher Subprocess

**Files:**
- Create: `internal/clipwatcher/clipwatcher.go`
- Create: `internal/clipwatcher/clipwatcher_test.go`
- Modify: `internal/app/app.go` (runWatcher, spawnClipboardWatcher, update runGo)
- Modify: `internal/app/command.go` (register hidden _watcher command)

### Part A: `clipwatcher` package

- [ ] **Step 1: Write failing tests**

```go
// internal/clipwatcher/clipwatcher_test.go
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
```

- [ ] **Step 2: Run to verify they fail**

```bash
go test ./internal/clipwatcher/... -v
```
Expected: compile error — package does not exist

- [ ] **Step 3: Implement the package**

```go
// internal/clipwatcher/clipwatcher.go
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
	if err := os.MkdirAll(filepath.Dir(w.pidFile), 0o700); err != nil {
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
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/clipwatcher/... -v
```
Expected: all 4 tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/clipwatcher/
git commit -m "feat: add clipboard watcher package for background AI response capture"
```

### Part B: Wire watcher into App

- [ ] **Step 6: Add `runWatcher` and `spawnClipboardWatcher` to app.go**

Add these methods to `internal/app/app.go`:

```go
// runWatcher is the entry point for the hidden `prtr _watcher` subprocess.
// It runs the clipboard watcher until capture, timeout, or context cancellation.
func (a *App) runWatcher(ctx context.Context) error {
	if a.lastResponseStore == nil {
		return errors.New("last-response store not configured")
	}

	pidPath, err := clipwatcher.DefaultPIDFile()
	if err != nil {
		return err
	}

	if clipwatcher.IsRunning(pidPath) {
		return nil // another instance is already running
	}

	w := clipwatcher.New(a.lastResponseStore, pidPath, a.clipboard)
	return w.Run(ctx)
}

// spawnClipboardWatcher detaches a background `prtr _watcher` subprocess.
// Failures are silently ignored — take and resume fall back to clipboard.
//
// NOTE: The parent intentionally does NOT check clipwatcher.IsRunning before
// spawning. Deduplication is the subprocess's responsibility: runWatcher calls
// IsRunning on startup and exits early if another instance is alive. This
// keeps PID file cleanup in one place (the subprocess) and avoids a TOCTOU race
// in the parent.
func (a *App) spawnClipboardWatcher() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	cmd := exec.Command(exe, "_watcher")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		return
	}
	_ = cmd.Process.Release()
}
```

Add imports as needed:
```go
"os/exec"
"github.com/helloprtr/poly-prompt/internal/clipwatcher"
```

- [ ] **Step 7: Register hidden `_watcher` command in command.go**

```go
func (a *App) newWatcherCommand(ctx context.Context) *cobra.Command {
	return &cobra.Command{
		Use:    "_watcher",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runWatcher(ctx)
		},
	}
}
```

In `Command()`, add:
```go
root.AddCommand(a.newWatcherCommand(ctx))
```

- [ ] **Step 8: Spawn watcher in `runGo`**

In `runGo` (around line 943), change the return to:

```go
if err := a.executePrompt(ctx, opts, text, command.mode); err != nil {
    return err
}
// Spawn clipboard watcher to capture AI response in the background.
// Fails silently; take and resume fall back to clipboard if nothing is captured.
if !command.dryRun && !command.noCopy {
    a.spawnClipboardWatcher()
}
return nil
```

- [ ] **Step 9: Build to verify it compiles**

```bash
go build ./...
```
Expected: no errors

- [ ] **Step 10: Smoke test (manual)**

```bash
go install ./cmd/prtr
prtr go --dry-run "test prompt"
# No subprocess spawned on --dry-run; no error expected
```

- [ ] **Step 11: Run full test suite**

```bash
go test ./... -v 2>&1 | tail -30
```
Expected: all tests PASS

- [ ] **Step 12: Commit**

```bash
git add internal/app/app.go internal/app/command.go
git commit -m "feat: spawn clipboard watcher subprocess after prtr go"
```

---

## Out of Scope for v0.9

The following is tracked for a later milestone (after v0.8 watch is implemented):

- **Terminal AI shell hook capture** — extends the v0.8 `watch` shell hook to write `last-response.json` when a terminal AI command exits. Requires v0.8 shell hook infrastructure.
- **Post-capture hint via shell** — printing `✓ Response captured` in the user's terminal requires the v0.8 `watch-suggest` IPC mechanism (shell's `precmd` hook reads and prints it).
- **Signal watcher on `take`/`resume`** — spec §2.2 states that running `prtr take` or `prtr resume` should signal the clipboard watcher to exit. This requires reading the PID file and sending a signal from the `runTake`/`runResume` path. Deferred to a follow-up: the watcher will time out naturally after 5 minutes, and the PID file deduplication prevents multiple instances from accumulating.

---

## Verification

After all chunks are complete:

```bash
go test ./... -count=1
go vet ./...
go build ./cmd/prtr
```

All must pass with zero errors before the branch is ready for review.
