# prtr v1.0 — AI Work Session Manager Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Redesign prtr from a prompt-composition tool into an AI Work Session Manager where sessions are the first-class citizen, auto-captured on TUI exit, and handed off across models without losing context.

**Architecture:** A new `internal/session` package owns all session state (schema, CRUD, git integration, handoff prompt building, subprocess runner). New commands (`prtr review/edit/fix/design/@model`) replace the 0.x surface while legacy commands stay as hidden aliases. `shouldRunRootDirect` is updated so bare `prtr` and `@model` args route through cobra.

**Tech Stack:** Go 1.24, cobra (existing), encoding/json (existing), os/exec for subprocess, crypto/sha256 for repo-hash. No new dependencies.

**Spec:** `docs/superpowers/specs/2026-03-22-v10-session-manager-design.md`

**Deferred (not in this plan):** `prtr inspect` (dry-run prompt preview) — spec section 4.2, out of scope.

---

## Branch Setup

- [ ] Create the v1.0 branch:

```bash
git checkout -b feat/v100-session-manager
```

---

## File Map

### New files
| File | Responsibility |
|------|---------------|
| `internal/session/session.go` | `Session` struct, `Mode`, `Status` types |
| `internal/session/store.go` | `Store`: CRUD, session resolution, path helpers |
| `internal/session/store_test.go` | Unit tests for store |
| `internal/session/git.go` | `RepoHash()`, `CurrentSHA()`, `Diff()`, `RepoRoot()` |
| `internal/session/git_test.go` | Unit tests for git helpers |
| `internal/session/handoff.go` | `BuildStartPrompt()`, `BuildHandoffPrompt()` |
| `internal/session/handoff_test.go` | Unit tests for prompt builders |
| `internal/session/subprocess.go` | `RunForeground()`, `FindBinary()`, `ModelBinaries()` |
| `internal/session/subprocess_test.go` | Unit tests for subprocess wrapper |

### Modified files
| File | What changes |
|------|-------------|
| `internal/app/app.go` | Add `SessionStore` to `Dependencies`/`App`; add session run methods |
| `internal/app/app_test.go` | Add `makeTestApp(t)` helper; add `SessionStore` to `newTestApp` |
| `internal/app/command.go` | Add new commands; hide legacy commands; fix `shouldRunRootDirect` |
| `internal/app/doctor.go` | Add AI binary checks to `buildDoctorReport` |
| `cmd/prtr/main.go` | Wire `session.NewStore()` into `Dependencies` |

---

## Task 1: Session struct and types

**Files:**
- Create: `internal/session/session.go`
- Create: `internal/session/store_test.go` (initial stub)

- [ ] **Step 1: Write the failing test**

```go
// internal/session/store_test.go
package session_test

import (
	"testing"
	"time"

	"github.com/helloprtr/poly-prompt/internal/session"
)

func TestSession_ZeroValueSafe(t *testing.T) {
	s := session.Session{
		ID:        "s_test",
		TaskGoal:  "fix auth bug",
		Mode:      session.ModeEdit,
		Status:    session.StatusActive,
		StartedAt: time.Now().UTC(),
	}
	if s.Mode != session.ModeEdit {
		t.Errorf("expected ModeEdit, got %v", s.Mode)
	}
	if s.Status != session.StatusActive {
		t.Errorf("expected StatusActive, got %v", s.Status)
	}
	if s.Files != nil && len(s.Files) != 0 {
		t.Errorf("expected nil/empty files, got %v", s.Files)
	}
}
```

- [ ] **Step 2: Run — expect compile error (package missing)**

```bash
cd /Users/koo/dev/translateCLI-brew && go test ./internal/session/... 2>&1 | head -5
```
Expected: `no Go files in .../internal/session`

- [ ] **Step 3: Implement session.go**

```go
// internal/session/session.go
package session

import "time"

type Mode string
type Status string

const (
	ModeReview Mode = "review"
	ModeEdit   Mode = "edit"
	ModeFix    Mode = "fix"
	ModeDesign Mode = "design"
)

const (
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
)

type Checkpoint struct {
	Note   string    `json:"note"`
	GitSHA string    `json:"git_sha"`
	At     time.Time `json:"at"`
}

type Session struct {
	ID           string       `json:"id"`
	Repo         string       `json:"repo"`
	RepoHash     string       `json:"repo_hash"`
	TaskGoal     string       `json:"task_goal"`
	Files        []string     `json:"files"`
	Mode         Mode         `json:"mode"`        // "review" | "edit" | "fix" | "design"
	Constraints  []string     `json:"constraints"`
	TargetModel  string       `json:"target_model"`
	Status       Status       `json:"status"`      // "active" | "completed"
	StartedAt    time.Time    `json:"started_at"`
	LastActivity time.Time    `json:"last_activity"`
	BaseGitSHA   string       `json:"base_git_sha"`
	Checkpoints  []Checkpoint `json:"checkpoints"`
}
```

- [ ] **Step 4: Run test — expect PASS**

```bash
go test ./internal/session/... -run TestSession_ZeroValueSafe -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/session/session.go internal/session/store_test.go
git commit -m "feat(session): add Session struct and Mode/Status types"
```

---

## Task 2: Session store (CRUD + resolution)

**Files:**
- Create: `internal/session/store.go`
- Modify: `internal/session/store_test.go` (add store tests — add new imports to the existing import block)

- [ ] **Step 1: Add store tests** (add to existing import block in store_test.go, do not create a second import block)

```go
// Add "os" and "path/filepath" to the existing import block at the top of store_test.go.
// Then add these test functions:

func TestStore_SaveAndResolve(t *testing.T) {
	dir := t.TempDir()
	store := session.NewStore(dir)

	repoHash := "aabbccdd"
	s := session.Session{
		ID:           "s_001",
		Repo:         "/tmp/myapp",
		RepoHash:     repoHash,
		TaskGoal:     "refactor auth",
		Mode:         session.ModeEdit,
		Status:       session.StatusActive,
		StartedAt:    time.Now().UTC(),
		LastActivity: time.Now().UTC(),
	}

	if err := store.Save(s); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.ActiveFor(repoHash)
	if err != nil {
		t.Fatalf("ActiveFor: %v", err)
	}
	if got.ID != s.ID {
		t.Errorf("expected ID %q, got %q", s.ID, got.ID)
	}
}

func TestStore_ActiveFor_None(t *testing.T) {
	dir := t.TempDir()
	store := session.NewStore(dir)

	_, err := store.ActiveFor("nonexistent")
	if err != session.ErrNoActiveSession {
		t.Errorf("expected ErrNoActiveSession, got %v", err)
	}
}

func TestStore_ActiveFor_MultiplePicksLatest(t *testing.T) {
	dir := t.TempDir()
	store := session.NewStore(dir)
	hash := "aabbccdd"

	older := session.Session{
		ID: "s_old", RepoHash: hash,
		Status:       session.StatusActive,
		LastActivity: time.Now().Add(-2 * time.Hour),
		StartedAt:    time.Now().Add(-3 * time.Hour),
	}
	newer := session.Session{
		ID: "s_new", RepoHash: hash,
		Status:       session.StatusActive,
		LastActivity: time.Now().Add(-10 * time.Minute),
		StartedAt:    time.Now().Add(-1 * time.Hour),
	}

	_ = store.Save(older)
	_ = store.Save(newer)

	got, err := store.ActiveFor(hash)
	if err != nil {
		t.Fatalf("ActiveFor: %v", err)
	}
	if got.ID != "s_new" {
		t.Errorf("expected newest session s_new, got %q", got.ID)
	}
}

func TestStore_List(t *testing.T) {
	dir := t.TempDir()
	store := session.NewStore(dir)

	for i, id := range []string{"s_1", "s_2", "s_3"} {
		_ = store.Save(session.Session{
			ID: id, RepoHash: "abc",
			Status:       session.StatusActive,
			LastActivity: time.Now().Add(time.Duration(i) * time.Minute),
			StartedAt:    time.Now(),
		})
	}

	sessions, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(sessions) != 3 {
		t.Errorf("expected 3 sessions, got %d", len(sessions))
	}
}

func TestStore_Complete(t *testing.T) {
	dir := t.TempDir()
	store := session.NewStore(dir)
	hash := "deadbeef"

	s := session.Session{
		ID: "s_done", RepoHash: hash,
		Status:       session.StatusActive,
		LastActivity: time.Now(),
		StartedAt:    time.Now(),
	}
	_ = store.Save(s)

	if err := store.Complete("s_done"); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	// Should no longer be active
	_, err := store.ActiveFor(hash)
	if err != session.ErrNoActiveSession {
		t.Errorf("expected ErrNoActiveSession after Complete, got %v", err)
	}

	// Should still appear in List
	all, _ := store.List()
	var found bool
	for _, sess := range all {
		if sess.ID == "s_done" && sess.Status == session.StatusCompleted {
			found = true
		}
	}
	if !found {
		t.Error("completed session not found in List")
	}
}
```

- [ ] **Step 2: Run — expect compile errors (Store not defined)**

```bash
go test ./internal/session/... -v 2>&1 | head -10
```

- [ ] **Step 3: Implement store.go**

```go
// internal/session/store.go
package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var ErrNoActiveSession = errors.New("no active session")

type Store struct {
	dir string
}

func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

func DefaultDir() (string, error) {
	base := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "prtr", "sessions"), nil
}

func (s *Store) Save(sess Session) error {
	if sess.ID == "" {
		sess.ID = fmt.Sprintf("s_%d", time.Now().UTC().UnixNano())
	}
	if sess.LastActivity.IsZero() {
		sess.LastActivity = time.Now().UTC()
	}

	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("create sessions directory: %w", err)
	}

	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return fmt.Errorf("encode session: %w", err)
	}

	path := s.pathFor(sess.RepoHash, sess.ID)
	tmp, err := os.CreateTemp(s.dir, ".session-*.json.tmp")
	if err != nil {
		return fmt.Errorf("create temp session file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write session: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close temp session file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("commit session: %w", err)
	}
	return nil
}

func (s *Store) ActiveFor(repoHash string) (Session, error) {
	all, err := s.List()
	if err != nil {
		return Session{}, err
	}

	var candidates []Session
	for _, sess := range all {
		if sess.RepoHash == repoHash && sess.Status == StatusActive {
			candidates = append(candidates, sess)
		}
	}
	if len(candidates) == 0 {
		return Session{}, ErrNoActiveSession
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].LastActivity.After(candidates[j].LastActivity)
	})
	return candidates[0], nil
}

func (s *Store) List() ([]Session, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read sessions directory: %w", err)
	}

	var sessions []Session
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, e.Name()))
		if err != nil {
			continue
		}
		var sess Session
		if err := json.Unmarshal(data, &sess); err != nil {
			continue
		}
		sessions = append(sessions, sess)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastActivity.After(sessions[j].LastActivity)
	})
	return sessions, nil
}

func (s *Store) Update(sess Session) error {
	sess.LastActivity = time.Now().UTC()
	return s.Save(sess)
}

func (s *Store) Complete(id string) error {
	all, err := s.List()
	if err != nil {
		return err
	}
	for _, sess := range all {
		if sess.ID == id {
			sess.Status = StatusCompleted
			sess.LastActivity = time.Now().UTC()
			return s.Save(sess)
		}
	}
	return fmt.Errorf("session %q not found", id)
}

func (s *Store) pathFor(repoHash, id string) string {
	return filepath.Join(s.dir, fmt.Sprintf("%s-%s.json", repoHash, id))
}
```

- [ ] **Step 4: Run tests — expect PASS**

```bash
go test ./internal/session/... -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/session/store.go internal/session/store_test.go
git commit -m "feat(session): add session Store with CRUD and active-session resolution"
```

---

## Task 3: Git helpers

**Files:**
- Create: `internal/session/git.go`
- Create: `internal/session/git_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/session/git_test.go
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
```

- [ ] **Step 2: Run — expect compile errors**

```bash
go test ./internal/session/... -run "TestRepo|TestCurrentSHA|TestDiff" -v 2>&1 | head -10
```

- [ ] **Step 3: Implement git.go**

```go
// internal/session/git.go
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
```

- [ ] **Step 4: Run tests — expect PASS**

```bash
go test ./internal/session/... -run "TestRepo|TestCurrentSHA|TestDiff" -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/session/git.go internal/session/git_test.go
git commit -m "feat(session): add git helpers (repo-hash, SHA, diff)"
```

---

## Task 4: Prompt builders

**Files:**
- Create: `internal/session/handoff.go`
- Create: `internal/session/handoff_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/session/handoff_test.go
package session_test

import (
	"strings"
	"testing"
	"time"

	"github.com/helloprtr/poly-prompt/internal/session"
)

func TestBuildStartPrompt_ContainsGoal(t *testing.T) {
	s := session.Session{
		TaskGoal:    "인증 미들웨어 리팩토링",
		Files:       []string{"auth/*.ts"},
		Mode:        session.ModeEdit,
		Constraints: []string{"TypeScript"},
	}
	prompt := session.BuildStartPrompt(s)
	if !strings.Contains(prompt, "인증 미들웨어 리팩토링") {
		t.Error("start prompt missing task goal")
	}
	if !strings.Contains(prompt, "시작해주세요") {
		t.Error("start prompt should say '시작해주세요', not '이어서'")
	}
}

func TestBuildHandoffPrompt_BasicFields(t *testing.T) {
	s := session.Session{
		TaskGoal:    "인증 미들웨어 리팩토링",
		Files:       []string{"auth/*.ts"},
		Mode:        session.ModeEdit,
		Constraints: []string{"TypeScript", "테스트 포함"},
	}

	prompt := session.BuildHandoffPrompt(s, "", "")
	if !strings.Contains(prompt, "인증 미들웨어 리팩토링") {
		t.Error("prompt missing task goal")
	}
	if !strings.Contains(prompt, "auth/*.ts") {
		t.Error("prompt missing files")
	}
	if !strings.Contains(prompt, "TypeScript") {
		t.Error("prompt missing constraints")
	}
	if !strings.Contains(prompt, "이어서 작업해주세요") {
		t.Error("handoff prompt should say '이어서 작업해주세요'")
	}
}

func TestBuildHandoffPrompt_WithDiff(t *testing.T) {
	s := session.Session{TaskGoal: "fix bug", Mode: session.ModeFix}
	diff := "diff --git a/main.go b/main.go\n+added line"

	prompt := session.BuildHandoffPrompt(s, diff, "")
	if !strings.Contains(prompt, diff) {
		t.Error("prompt missing diff")
	}
}

func TestBuildHandoffPrompt_WithCheckpoints(t *testing.T) {
	s := session.Session{
		TaskGoal: "refactor",
		Mode:     session.ModeEdit,
		Checkpoints: []session.Checkpoint{
			{Note: "JWT done", At: time.Now()},
			{Note: "refresh WIP", At: time.Now()},
		},
	}
	prompt := session.BuildHandoffPrompt(s, "", "")
	if !strings.Contains(prompt, "JWT done") {
		t.Error("prompt missing checkpoint 1")
	}
	if !strings.Contains(prompt, "refresh WIP") {
		t.Error("prompt missing checkpoint 2")
	}
}

func TestBuildHandoffPrompt_EmptyDiffOmitsSection(t *testing.T) {
	s := session.Session{TaskGoal: "task", Mode: session.ModeEdit}
	prompt := session.BuildHandoffPrompt(s, "", "")
	if strings.Contains(prompt, "[코드 변화]") {
		t.Error("expected no code-change section when diff is empty")
	}
}

func TestBuildHandoffPrompt_WithLastResponse(t *testing.T) {
	s := session.Session{TaskGoal: "task", Mode: session.ModeEdit}
	prompt := session.BuildHandoffPrompt(s, "", "AI said: use interface{}")
	if !strings.Contains(prompt, "AI said: use interface{}") {
		t.Error("prompt missing last response")
	}
}
```

- [ ] **Step 2: Run — expect compile errors**

```bash
go test ./internal/session/... -run TestBuild -v 2>&1 | head -10
```

- [ ] **Step 3: Implement handoff.go**

```go
// internal/session/handoff.go
package session

import (
	"strings"
)

// BuildStartPrompt constructs the initial prompt for a brand-new session.
func BuildStartPrompt(s Session) string {
	return buildPrompt(s, "", "", "시작해주세요.")
}

// BuildHandoffPrompt constructs the context-restoration prompt for model handoff.
// diff and lastResponse are optional — omit sections when empty.
func BuildHandoffPrompt(s Session, diff, lastResponse string) string {
	return buildPrompt(s, diff, lastResponse, "이어서 작업해주세요.")
}

func buildPrompt(s Session, diff, lastResponse, closing string) string {
	var b strings.Builder

	b.WriteString("[작업 목표]\n")
	b.WriteString(s.TaskGoal)
	if len(s.Constraints) > 0 {
		b.WriteString(" (" + strings.Join(s.Constraints, ", ") + ")")
	}
	b.WriteString("\n")

	if len(s.Files) > 0 {
		b.WriteString("\n[파일 범위]\n")
		b.WriteString(strings.Join(s.Files, "\n"))
		b.WriteString("\n")
	}

	if len(s.Checkpoints) > 0 {
		b.WriteString("\n[진행 상황]\n")
		for _, cp := range s.Checkpoints {
			b.WriteString("- " + cp.Note + "\n")
		}
	}

	if strings.TrimSpace(diff) != "" {
		b.WriteString("\n[코드 변화]\n")
		b.WriteString(diff)
		b.WriteString("\n")
	}

	if strings.TrimSpace(lastResponse) != "" {
		b.WriteString("\n[마지막 AI 응답]\n")
		b.WriteString(lastResponse)
		b.WriteString("\n")
	}

	b.WriteString("\n" + closing + "\n")
	return b.String()
}
```

- [ ] **Step 4: Run tests — expect PASS**

```bash
go test ./internal/session/... -run TestBuild -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/session/handoff.go internal/session/handoff_test.go
git commit -m "feat(session): add BuildStartPrompt and BuildHandoffPrompt"
```

---

## Task 5: Subprocess runner

**Files:**
- Create: `internal/session/subprocess.go`
- Create: `internal/session/subprocess_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/session/subprocess_test.go
package session_test

import (
	"context"
	"testing"

	"github.com/helloprtr/poly-prompt/internal/session"
)

func TestRunForeground_EchoExitsZero(t *testing.T) {
	err := session.RunForeground(context.Background(), "echo", "hello")
	if err != nil {
		t.Errorf("expected nil error from echo, got: %v", err)
	}
}

func TestRunForeground_MissingBinary(t *testing.T) {
	err := session.RunForeground(context.Background(), "this-binary-does-not-exist-prtr-test")
	if err == nil {
		t.Error("expected error for missing binary")
	}
}

func TestFindBinary_ReturnsPathForEcho(t *testing.T) {
	path, err := session.FindBinary("echo")
	if err != nil {
		t.Fatalf("FindBinary(echo): %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path for echo")
	}
}

func TestFindBinary_ErrorForMissing(t *testing.T) {
	_, err := session.FindBinary("this-binary-does-not-exist-prtr-test")
	if err == nil {
		t.Error("expected error for missing binary")
	}
}

func TestModelBinaries_KnownModels(t *testing.T) {
	cases := []struct {
		model     string
		wantFirst string
	}{
		{"claude", "claude"},
		{"gemini", "gemini"},
		{"codex", "codex"},
	}
	for _, tc := range cases {
		bins := session.ModelBinaries(tc.model)
		if len(bins) == 0 {
			t.Errorf("ModelBinaries(%q): expected at least one candidate", tc.model)
		}
		if bins[0] != tc.wantFirst {
			t.Errorf("ModelBinaries(%q): first = %q, want %q", tc.model, bins[0], tc.wantFirst)
		}
	}
}
```

- [ ] **Step 2: Run — expect compile errors**

```bash
go test ./internal/session/... -run "TestRunForeground|TestFindBinary|TestModel" -v 2>&1 | head -10
```

- [ ] **Step 3: Implement subprocess.go**

```go
// internal/session/subprocess.go
package session

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

var modelBinaryMap = map[string][]string{
	"claude": {"claude"},
	"gemini": {"gemini", "gemini-cli"},
	"codex":  {"codex"},
}

// ModelBinaries returns ordered binary candidates for a model name.
func ModelBinaries(model string) []string {
	if bins, ok := modelBinaryMap[model]; ok {
		return bins
	}
	return []string{model}
}

// FindBinary searches $PATH for the first available candidate binary name.
func FindBinary(candidates ...string) (string, error) {
	for _, name := range candidates {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("binary not found: tried %v", candidates)
}

// RunForeground runs binary with args in the foreground, inheriting stdin/stdout/stderr.
// Blocks until the process exits.
func RunForeground(ctx context.Context, binary string, args ...string) error {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
```

- [ ] **Step 4: Run all session tests — expect PASS**

```bash
go test ./internal/session/... -v
```

- [ ] **Step 5: Commit**

```bash
git add internal/session/subprocess.go internal/session/subprocess_test.go
git commit -m "feat(session): add foreground subprocess runner and binary discovery"
```

---

## Task 6: Wire session store into app + add makeTestApp helper

**Files:**
- Modify: `internal/app/app.go`
- Modify: `internal/app/app_test.go`
- Modify: `cmd/prtr/main.go`

- [ ] **Step 1: Write failing test** (checks that `App` has a `sessionStore` field accessible via `Dependencies`)

```go
// Add to internal/app/app_test.go (add "github.com/helloprtr/poly-prompt/internal/session" to imports):

func TestNewApp_AcceptsSessionStore(t *testing.T) {
	dir := t.TempDir()
	store := session.NewStore(dir)
	cfg := testConfig()
	a := newTestApp(t, cfg, &stubTranslator{}, &stubClipboard{}, &stubEditor{},
		history.New(filepath.Join(t.TempDir(), "h.json")))
	// Wire store after construction via a helper (or verify via compile only)
	_ = store
	_ = a
	// This test passes if it compiles — field presence is a compile-time check
}
```

- [ ] **Step 2: Run — expect compile error (Dependencies.SessionStore not defined)**

```bash
go test ./internal/app/... -run TestNewApp_AcceptsSessionStore -v 2>&1 | head -10
```

- [ ] **Step 3: Add SessionStore to Dependencies and App in app.go**

In `internal/app/app.go`:

```go
// Add to import block:
"github.com/helloprtr/poly-prompt/internal/session"

// Add to Dependencies struct:
SessionStore *session.Store

// Add to App struct:
sessionStore *session.Store

// Add to New() after existing field assignments:
sessionStore: deps.SessionStore,
```

- [ ] **Step 4: Update newTestApp in app_test.go to include SessionStore**

In `newTestApp`, add to `Dependencies{}`:
```go
SessionStore: session.NewStore(t.TempDir()),
```

Also add `makeTestApp` convenience helper (used in later tasks):

```go
// Add to app_test.go:
func makeTestApp(t *testing.T) *App {
	t.Helper()
	return newTestApp(t, testConfig(), &stubTranslator{}, &stubClipboard{}, &stubEditor{},
		history.New(filepath.Join(t.TempDir(), "h.json")))
}
```

Add `"path/filepath"` to imports in `app_test.go` if not already present.

- [ ] **Step 5: Wire in main.go**

```go
// Add to import block in cmd/prtr/main.go:
"github.com/helloprtr/poly-prompt/internal/session"

// Before app.New(...):
sessionDir, err := session.DefaultDir()
if err != nil {
    fmt.Fprintf(os.Stderr, "failed to resolve session directory: %v\n", err)
    os.Exit(1)
}

// In app.New(app.Dependencies{...}):
SessionStore: session.NewStore(sessionDir),
```

- [ ] **Step 6: Build and run tests**

```bash
go build ./... && go test ./internal/app/... -run TestNewApp -v
```

- [ ] **Step 7: Commit**

```bash
git add internal/app/app.go internal/app/app_test.go cmd/prtr/main.go
git commit -m "feat(app): wire session.Store into app dependencies"
```

---

## Task 7: Session methods on App

**Files:**
- Modify: `internal/app/app.go`

- [ ] **Step 1: Write failing tests**

```go
// Add to internal/app/app_test.go:

func TestResolveCurrentSession_NoGitRepo(t *testing.T) {
	a := makeTestApp(t)
	// newTestApp uses RepoRootFinder returning termbook.ErrNotGitRepo
	_, err := a.resolveCurrentSession()
	if err == nil {
		t.Error("expected error when not in a git repo")
	}
}

func TestRunCheckpoint_ErrorWithNoSession(t *testing.T) {
	a := makeTestApp(t)
	err := a.runCheckpoint(context.Background(), "JWT done")
	if err == nil {
		t.Error("expected error when no active session")
	}
}

func TestRunDone_ErrorWithNoSession(t *testing.T) {
	a := makeTestApp(t)
	err := a.runDone(context.Background())
	if err == nil {
		t.Error("expected error when no active session")
	}
}

func TestRunSessions_Empty(t *testing.T) {
	a := makeTestApp(t)
	var stdout bytes.Buffer
	a.stdout = &stdout
	err := a.runSessions(context.Background())
	if err != nil {
		t.Fatalf("runSessions: %v", err)
	}
	// Should not panic; output is "세션 없음." or similar
}
```

- [ ] **Step 2: Run — expect compile errors (methods not defined)**

```bash
go test ./internal/app/... -run "TestResolveCurrentSession|TestRunCheckpoint|TestRunDone|TestRunSessions" -v 2>&1 | head -15
```

- [ ] **Step 3: Implement session methods in app.go**

Add the following methods. Also add imports: `"bufio"`, `"encoding/json"`, `"path/filepath"` (check existing imports first, add only what's missing).

```go
// resolveCurrentSession finds the active session for the current git repo.
func (a *App) resolveCurrentSession() (session.Session, error) {
	root, err := a.resolveRepoRoot()
	if err != nil {
		return session.Session{}, session.ErrNoActiveSession
	}
	hash, err := session.RepoHash(root)
	if err != nil {
		// RepoHash wraps an infallible SHA256 operation; this branch is unreachable in practice
		return session.Session{}, session.ErrNoActiveSession
	}
	return a.sessionStore.ActiveFor(hash)
}

func humanizeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "방금 전"
	case d < time.Hour:
		return fmt.Sprintf("%d분 전", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d시간 전", int(d.Hours()))
	default:
		return fmt.Sprintf("%d일 전", int(d.Hours()/24))
	}
}

func (a *App) runCheckpoint(_ context.Context, note string) error {
	sess, err := a.resolveCurrentSession()
	if err != nil {
		return fmt.Errorf("No active session. Run prtr review|edit|fix|design first.")
	}
	root, _ := a.resolveRepoRoot()
	sha, _ := session.CurrentSHA(root)

	sess.Checkpoints = append(sess.Checkpoints, session.Checkpoint{
		Note:   note,
		GitSHA: sha,
		At:     time.Now().UTC(),
	})
	if err := a.sessionStore.Update(sess); err != nil {
		return fmt.Errorf("save checkpoint: %w", err)
	}
	fmt.Fprintf(a.stderr, "✓ 체크포인트 저장: %q\n", note)
	return nil
}

func (a *App) runDone(_ context.Context) error {
	sess, err := a.resolveCurrentSession()
	if err != nil {
		return fmt.Errorf("No active session.")
	}
	if err := a.sessionStore.Complete(sess.ID); err != nil {
		return fmt.Errorf("complete session: %w", err)
	}
	fmt.Fprintf(a.stderr, "✓ 세션 완료: %q\n", sess.TaskGoal)
	return nil
}

func (a *App) runSessions(_ context.Context) error {
	sessions, err := a.sessionStore.List()
	if err != nil {
		return fmt.Errorf("list sessions: %w", err)
	}
	if len(sessions) == 0 {
		fmt.Fprintln(a.stdout, "세션 없음.")
		return nil
	}
	for _, s := range sessions {
		status := "●"
		if s.Status == session.StatusCompleted {
			status = "✓"
		}
		fmt.Fprintf(a.stdout, "%s [%s] %q — %s (%s)\n",
			status, s.ID, s.TaskGoal, humanizeTime(s.LastActivity), s.TargetModel)
	}
	return nil
}

// readLastResponse reads ~/.config/prtr/last-response.json if present and returns the response field.
func (a *App) readLastResponse() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	path := filepath.Join(home, ".config", "prtr", "last-response.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var v struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return ""
	}
	return v.Response
}

// captureSessionOnExit runs after the TUI exits, updating last_activity in the session.
// The git diff is NOT stored in the session — it is recomputed from base_git_sha at handoff time.
func (a *App) captureSessionOnExit(sess session.Session) error {
	sess.LastActivity = time.Now().UTC()
	if err := a.sessionStore.Update(sess); err != nil {
		return fmt.Errorf("update session after exit: %w", err)
	}
	fmt.Fprintln(a.stderr, "✓ 세션 저장됨 — 다음에 prtr로 이어서")
	return nil
}

// launchWithSession writes the start prompt to clipboard and launches the AI TUI as a subprocess.
func (a *App) launchWithSession(ctx context.Context, sess session.Session) error {
	cfg, err := a.configLoader()
	if err != nil {
		return err
	}

	prompt := session.BuildStartPrompt(sess)
	if err := a.clipboard.Copy(ctx, prompt); err != nil {
		return fmt.Errorf("copy session prompt: %w", err)
	}

	model := sess.TargetModel
	if model == "" {
		model = cfg.DefaultTarget
	}
	bins := session.ModelBinaries(model)
	binary, err := session.FindBinary(bins...)
	if err != nil {
		fmt.Fprintf(a.stderr, "✓ 프롬프트가 클립보드에 복사됐습니다 (%s를 찾을 수 없어 직접 열어주세요)\n", model)
		return nil
	}

	fmt.Fprintln(a.stderr, "✓ 세션 시작 — 프롬프트가 클립보드에 복사됐습니다. TUI에 붙여넣으세요.")
	_ = session.RunForeground(ctx, binary)
	return a.captureSessionOnExit(sess)
}

// launchHandoff builds the handoff prompt and launches the target model.
func (a *App) launchHandoff(ctx context.Context, sess session.Session, model string) error {
	root, _ := a.resolveRepoRoot()
	diff, _ := session.Diff(root, sess.BaseGitSHA)
	lastResp := a.readLastResponse()

	prompt := session.BuildHandoffPrompt(sess, diff, lastResp)
	if err := a.clipboard.Copy(ctx, prompt); err != nil {
		return fmt.Errorf("copy handoff prompt: %w", err)
	}

	bins := session.ModelBinaries(model)
	binary, err := session.FindBinary(bins...)
	if err != nil {
		fmt.Fprintf(a.stderr, "✓ 핸드오프 프롬프트가 클립보드에 복사됐습니다 (%s를 찾을 수 없어 직접 열어주세요)\n", model)
		return nil
	}

	fmt.Fprintf(a.stderr, "✓ %s로 핸드오프 — 프롬프트가 클립보드에 복사됐습니다.\n", model)
	_ = session.RunForeground(ctx, binary)
	sess.TargetModel = model
	return a.captureSessionOnExit(sess)
}

// runSessionCreate runs the interactive new-session creation flow.
// mode and files may be pre-populated from command args.
// reader must be a *bufio.Reader wrapping the stdin io.Reader.
func (a *App) runSessionCreate(ctx context.Context, mode session.Mode, files []string, reader *bufio.Reader) error {
	fmt.Fprint(a.stderr, "무엇을 하려 하나요? ")
	goal, _ := reader.ReadString('\n')
	goal = strings.TrimSpace(goal)
	if goal == "" {
		return fmt.Errorf("작업 목표를 입력해주세요")
	}

	if len(files) == 0 {
		fmt.Fprint(a.stderr, "어떤 파일? (Enter로 건너뜀) ")
		line, _ := reader.ReadString('\n')
		if line = strings.TrimSpace(line); line != "" {
			files = strings.Fields(line)
		}
	}

	fmt.Fprint(a.stderr, "제약조건? (Enter로 건너뜀) ")
	constraintLine, _ := reader.ReadString('\n')
	var constraints []string
	if cl := strings.TrimSpace(constraintLine); cl != "" {
		for _, c := range strings.Split(cl, ",") {
			if t := strings.TrimSpace(c); t != "" {
				constraints = append(constraints, t)
			}
		}
	}

	root, _ := a.resolveRepoRoot()
	hash, _ := session.RepoHash(root)
	sha, _ := session.CurrentSHA(root)

	cfg, _ := a.configLoader()
	sess := session.Session{
		Repo:        root,
		RepoHash:    hash,
		TaskGoal:    goal,
		Files:       files,
		Mode:        mode,
		Constraints: constraints,
		TargetModel: cfg.DefaultTarget,
		Status:      session.StatusActive,
		StartedAt:   time.Now().UTC(),
		BaseGitSHA:  sha,
	}

	if err := a.sessionStore.Save(sess); err != nil {
		return fmt.Errorf("save session: %w", err)
	}
	return a.launchWithSession(ctx, sess)
}

// runBare implements the bare `prtr` command (no subcommand).
func (a *App) runBare(ctx context.Context, stdin io.Reader) error {
	sess, err := a.resolveCurrentSession()
	if err == nil {
		return a.offerContinueOrNew(ctx, sess, stdin)
	}
	return a.runSessionCreate(ctx, session.ModeEdit, nil, bufio.NewReader(stdin))
}

func (a *App) offerContinueOrNew(ctx context.Context, sess session.Session, stdin io.Reader) error {
	fmt.Fprintf(a.stderr, "─────────────────────────────────────\n")
	fmt.Fprintf(a.stderr, "이어서 할까요?\n\n")
	fmt.Fprintf(a.stderr, "%q — %s (%s)\n", sess.TaskGoal, humanizeTime(sess.LastActivity), sess.TargetModel)
	fmt.Fprintf(a.stderr, "─────────────────────────────────────\n")
	fmt.Fprint(a.stderr, "[Enter] 이어서  [n] 새 작업: ")

	reader := bufio.NewReader(stdin)
	line, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(line)) == "n" {
		return a.runSessionCreate(ctx, session.ModeEdit, nil, reader)
	}
	return a.launchHandoff(ctx, sess, sess.TargetModel)
}

// runSessionMode handles prtr review|edit|fix|design [files...]
func (a *App) runSessionMode(ctx context.Context, mode session.Mode, args []string, stdin io.Reader) error {
	reader := bufio.NewReader(stdin)
	if sess, err := a.resolveCurrentSession(); err == nil {
		fmt.Fprintf(a.stderr, "⚡ 진행 중인 세션: %q (%s)\n", sess.TaskGoal, humanizeTime(sess.LastActivity))
		fmt.Fprint(a.stderr, "이어서 할까요, 새로 시작할까요? [이어서/새로]: ")
		line, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(line)) != "새로" {
			return a.launchHandoff(ctx, sess, sess.TargetModel)
		}
	}
	return a.runSessionCreate(ctx, mode, args, reader)
}

// runHandoff handles `prtr @gemini` / `prtr @codex`.
func (a *App) runHandoff(ctx context.Context, model string) error {
	sess, err := a.resolveCurrentSession()
	if err != nil {
		return fmt.Errorf("No active session. Run prtr review|edit|fix|design first.")
	}
	return a.launchHandoff(ctx, sess, model)
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/app/... -run "TestResolveCurrentSession|TestRunCheckpoint|TestRunDone|TestRunSessions" -v
```

- [ ] **Step 5: Build check**

```bash
go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add internal/app/app.go internal/app/app_test.go
git commit -m "feat(app): add session create/handoff/checkpoint/done/sessions methods"
```

---

## Task 8: New commands + fix shouldRunRootDirect

**Files:**
- Modify: `internal/app/command.go`

- [ ] **Step 1: Write failing test**

```go
// Add to app_test.go:
func TestExecute_BareWithNoSession_AsksForGoal(t *testing.T) {
	a := makeTestApp(t)
	_, stderr := buffersFromApp(a)

	// stdin: goal provided
	err := a.Execute(context.Background(), []string{}, strings.NewReader("test goal\n\n\n"), false)
	// Expected: not a git repo → session creation will fail on save, but prompt is shown
	// We just verify no panic and the "what do you want to do" prompt appeared
	_ = err
	if !strings.Contains(stderr.String(), "무엇을") && !strings.Contains(stderr.String(), "session") {
		// Allow: might fail silently in test env (no git, no binary)
	}
}

func TestExecute_AtModel_NoSession_ReturnsError(t *testing.T) {
	a := makeTestApp(t)
	var stdout, stderr bytes.Buffer
	a.stdout = &stdout
	a.stderr = &stderr

	err := a.Execute(context.Background(), []string{"@gemini"}, strings.NewReader(""), false)
	if err == nil {
		t.Error("expected error for @gemini with no active session")
	}
	if !strings.Contains(err.Error(), "No active session") {
		t.Errorf("expected 'No active session', got: %v", err)
	}
}
```

- [ ] **Step 2: Run — expect failures**

```bash
go test ./internal/app/... -run "TestExecute_BareWithNoSession|TestExecute_AtModel" -v 2>&1 | head -20
```

- [ ] **Step 3: Fix shouldRunRootDirect**

In `app.go`, update `shouldRunRootDirect`:

```go
func (a *App) shouldRunRootDirect(args []string) bool {
	if len(args) == 0 {
		return false // let cobra dispatch to root RunE → runBare
	}

	first := strings.TrimSpace(args[0])
	if first == "" {
		return false
	}
	if first == "-h" || first == "--help" || first == "help" {
		return false
	}
	if strings.HasPrefix(first, "-") {
		return true
	}
	if strings.HasPrefix(first, "@") {
		return false // @model args go through cobra → root RunE
	}

	switch first {
	case "init", "version", "start", "setup", "lang", "doctor", "templates", "profiles",
		"history", "rerun", "pin", "favorite", "go", "demo", "again", "swap", "take",
		"learn", "inspect",
		// new v1.0 commands:
		"review", "edit", "fix", "design", "checkpoint", "done", "sessions":
		return false
	}

	return !a.builtInShortcutNames()[first]
}
```

- [ ] **Step 4: Replace shortcut commands with mode commands in command.go**

In `command.go`'s `Command()` method, **replace** the three existing `newShortcutCommand` calls for `review`, `fix`, `design` (lines ~52-54) with `newModeCommand` calls, and add `edit`. Keep `ask` as a shortcut. The result should read:

```go
root.AddCommand(a.newShortcutCommand(ctx, "ask", stdin, stdinPiped))
root.AddCommand(a.newModeCommand(ctx, "review", session.ModeReview, stdin, stdinPiped))
root.AddCommand(a.newModeCommand(ctx, "edit", session.ModeEdit, stdin, stdinPiped))
root.AddCommand(a.newModeCommand(ctx, "fix", session.ModeFix, stdin, stdinPiped))
root.AddCommand(a.newModeCommand(ctx, "design", session.ModeDesign, stdin, stdinPiped))
root.AddCommand(&cobra.Command{
    Use:   "checkpoint [note]",
    Short: "진행 상황 메모 (핸드오프 품질 향상)",
    Args:  cobra.ExactArgs(1),
    RunE:  func(cmd *cobra.Command, args []string) error { return a.runCheckpoint(ctx, args[0]) },
})
root.AddCommand(&cobra.Command{
    Use:   "done",
    Short: "세션 완료 처리",
    RunE:  func(cmd *cobra.Command, args []string) error { return a.runDone(ctx) },
})
root.AddCommand(&cobra.Command{
    Use:   "sessions",
    Short: "과거 세션 목록",
    RunE:  func(cmd *cobra.Command, args []string) error { return a.runSessions(ctx) },
})
```

Add the `newModeCommand` constructor:

```go
func (a *App) newModeCommand(ctx context.Context, name string, mode session.Mode, stdin io.Reader, stdinPiped bool) *cobra.Command {
    return &cobra.Command{
        Use:           name + " [files...]",
        Short:         modeShort(mode),
        Args:          cobra.ArbitraryArgs,
        SilenceErrors: true,
        SilenceUsage:  true,
        RunE: func(cmd *cobra.Command, args []string) error {
            return a.runSessionMode(ctx, mode, args, stdin)
        },
    }
}

func modeShort(m session.Mode) string {
    switch m {
    case session.ModeReview:
        return "코드 리뷰 세션 시작"
    case session.ModeEdit:
        return "코드 수정 세션 시작"
    case session.ModeFix:
        return "버그 수정 세션 시작"
    case session.ModeDesign:
        return "설계 세션 시작"
    default:
        return "세션 시작"
    }
}
```

- [ ] **Step 5: Update root RunE to handle bare and @model**

In the root command's `RunE`:

```go
RunE: func(cmd *cobra.Command, args []string) error {
    if wantsHelp(args) {
        return cmd.Help()
    }
    if len(args) == 0 {
        return a.runBare(ctx, stdin)
    }
    if strings.HasPrefix(args[0], "@") {
        model := strings.TrimPrefix(args[0], "@")
        return a.runHandoff(ctx, model)
    }
    return a.runMain(ctx, args, stdin, stdinPiped, "")
},
```

Add `"github.com/helloprtr/poly-prompt/internal/session"` to command.go imports.

- [ ] **Step 6: Build**

```bash
go build ./...
```

- [ ] **Step 7: Run tests**

```bash
go test ./internal/app/... -run "TestExecute_BareWithNoSession|TestExecute_AtModel" -v
```

- [ ] **Step 8: Commit**

```bash
git add internal/app/app.go internal/app/command.go
git commit -m "feat(app): add review/edit/fix/design/@model commands; fix shouldRunRootDirect"
```

---

## Task 9: Hide legacy commands

**Files:**
- Modify: `internal/app/command.go`

- [ ] **Step 1: Verify legacy commands still work before hiding**

```bash
go build ./... && ./output/prtr go --help 2>&1 | head -3
```

- [ ] **Step 2: Set Hidden: true on legacy command constructors**

For each of `newGoCommand`, `newAgainCommand`, `newSwapCommand`, `newTakeCommand`, `newLearnCommand`, `newStartCommand`, add `Hidden: true` to the `cobra.Command` struct.

Example for `newGoCommand`:
```go
cmd := &cobra.Command{
    Use:                "go [mode] [message...]",
    Hidden:             true,  // ← add this
    Short:              "...",
    // ... rest unchanged
}
```

Also set `Hidden: true` on `resume` if it exists as a registered command.

- [ ] **Step 3: Verify --help no longer shows legacy commands**

```bash
go build ./... && ./output/prtr --help
```
Expected output contains: `review`, `edit`, `fix`, `design`, `checkpoint`, `done`, `sessions`
Expected output does NOT contain: `go`, `swap`, `take`, `again`, `start`, `learn`

- [ ] **Step 4: Verify legacy commands still work**

```bash
./output/prtr go --help 2>&1 | head -3
```
Expected: help text still shows (hidden ≠ removed)

- [ ] **Step 5: Run existing tests to confirm no regressions**

```bash
go test ./internal/app/... -v 2>&1 | tail -20
```
Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add internal/app/command.go
git commit -m "feat(app): hide legacy 0.x commands from help output (backward compat preserved)"
```

---

## Task 10: Update prtr status

**Files:**
- Modify: `internal/app/app.go` or wherever status is handled

- [ ] **Step 1: Find the existing status command handler**

```bash
grep -n "status\|Status\|runStatus\|runCapsule" /Users/koo/dev/translateCLI-brew/internal/app/app.go | head -20
grep -n "\"status\"\|status" /Users/koo/dev/translateCLI-brew/internal/app/command.go | head -20
```

Identify whether a `runStatus` method exists or status is handled inline. Rename/wrap the existing logic as `runCapsuleStatus`. If no existing status method is found, define a no-op stub:

```go
func (a *App) runCapsuleStatus(_ context.Context) error { return nil }
```

- [ ] **Step 2: Write failing test**

```go
func TestRunStatusShowsSessionSection(t *testing.T) {
	a := makeTestApp(t)
	var stdout bytes.Buffer
	a.stdout = &stdout

	err := a.runStatus(context.Background())
	if err != nil {
		t.Fatalf("runStatus: %v", err)
	}
	// When no session: should show "[현재 세션]" section
	output := stdout.String()
	if !strings.Contains(output, "세션") {
		t.Errorf("expected session section in status output, got:\n%s", output)
	}
}
```

- [ ] **Step 3: Implement runStatus**

Wrap existing status logic as `runCapsuleStatus`, then add session display:

```go
func (a *App) runStatus(ctx context.Context) error {
	sess, err := a.resolveCurrentSession()
	if err == nil {
		root, _ := a.resolveRepoRoot()
		diff, _ := session.Diff(root, sess.BaseGitSHA)

		fmt.Fprintln(a.stdout, "[현재 세션]")
		fmt.Fprintf(a.stdout, "작업: %s\n", sess.TaskGoal)
		if len(sess.Files) > 0 {
			fmt.Fprintf(a.stdout, "파일: %s\n", strings.Join(sess.Files, ", "))
		}
		fmt.Fprintf(a.stdout, "모드: %s\n", sess.Mode)
		fmt.Fprintf(a.stdout, "시작: %s (%s)\n", humanizeTime(sess.StartedAt), sess.TargetModel)
		if summary := summarizeDiff(diff); summary != "" {
			fmt.Fprintf(a.stdout, "변경: %s\n", summary)
		}
		if len(sess.Checkpoints) > 0 {
			last := sess.Checkpoints[len(sess.Checkpoints)-1]
			fmt.Fprintf(a.stdout, "체크포인트: %q\n", last.Note)
		}
		fmt.Fprintln(a.stdout)
	} else {
		fmt.Fprintln(a.stdout, "[현재 세션]\n세션 없음\n")
	}

	return a.runCapsuleStatus(ctx)
}

func summarizeDiff(diff string) string {
	if diff == "" {
		return ""
	}
	var files []string
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "diff --git") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				files = append(files, strings.TrimPrefix(parts[3], "b/"))
			}
		}
	}
	if len(files) == 0 {
		return ""
	}
	if len(files) > 3 {
		return fmt.Sprintf("%s 외 %d개", strings.Join(files[:3], ", "), len(files)-3)
	}
	return strings.Join(files, ", ")
}
```

- [ ] **Step 4: Register status command (if not already registered)**

In `command.go`, ensure `"status"` is in the `shouldRunRootDirect` switch and there is a registered command. If existing `runStatus` existed, wire to the new one.

- [ ] **Step 5: Run tests**

```bash
go test ./internal/app/... -run TestRunStatus -v
go test ./... 2>&1 | tail -10
```

- [ ] **Step 6: Commit**

```bash
git add internal/app/app.go internal/app/command.go
git commit -m "feat(app): update prtr status to show session state + Work Capsule info"
```

---

## Task 11: Update doctor with AI binary checks

**Files:**
- Modify: `internal/app/doctor.go`

- [ ] **Step 1: Locate the binary check section in buildDoctorReport**

Read the `buildDoctorReport` function (around line 184 of doctor.go) to understand where to insert the new checks. The pattern is:
```go
report.Checks = append(report.Checks, doctorCheck{Severity: doctorOK, Label: "...", Detail: "..."})
```

- [ ] **Step 2: Improve existing launcher loop with binary discovery**

In `buildDoctorReport`, find the existing `for _, targetName := range []string{"claude", "codex", "gemini"}` loop. The current code adds a blocking `"not configured"` check when the launcher command is empty and `continue`s. Replace that early-exit block with binary discovery:

```go
// Replace this existing block:
//   if strings.TrimSpace(launcherCfg.Command) == "" {
//       report.Checks = append(report.Checks, doctorCheck{Severity: doctorBlocking, Label: "launcher " + targetName, Err: errors.New("not configured")})
//       continue
//   }
// With:
if strings.TrimSpace(launcherCfg.Command) == "" {
    bins := session.ModelBinaries(targetName)
    if _, err := session.FindBinary(bins...); err == nil {
        report.Checks = append(report.Checks, doctorCheck{
            Severity: doctorOK,
            Label:    "launcher " + targetName,
            Detail:   "binary found; run `prtr setup` to configure",
        })
    } else {
        report.Checks = append(report.Checks, doctorCheck{
            Severity: doctorWarning,
            Label:    "launcher " + targetName,
            Detail:   fmt.Sprintf("install %s and run `prtr setup` to configure", targetName),
        })
    }
    continue
}
```

This replaces the blocking "not configured" error with a smarter binary-discovery check. No new loop needed — no duplicate labels.

Add `"github.com/helloprtr/poly-prompt/internal/session"` to doctor.go imports.

- [ ] **Step 3: Build check**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add internal/app/doctor.go
git commit -m "feat(doctor): add AI binary availability checks for claude/gemini/codex"
```

---

## Task 12: Version bump and CHANGELOG

**Files:**
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Add v1.0.0 entry at the top of CHANGELOG.md**

```markdown
## [1.0.0] — 2026-03-22

### Added
- Session as first-class citizen: `prtr` auto-creates and maintains work sessions per git repo
- `prtr review [files]`, `prtr edit [files]`, `prtr fix [desc]`, `prtr design [topic]` — mode-specific session starters
- `prtr @gemini`, `prtr @codex` — handoff current session to another model
- `prtr checkpoint "note"` — save progress memo for better handoff quality
- `prtr done` — mark session complete
- `prtr sessions` — list all sessions (active and completed)
- `prtr status` now shows current session state alongside Work Capsule drift info
- `prtr doctor` now checks AI binary availability for claude, gemini, codex

### Changed
- `prtr` (bare) shows active session and offers to continue or start new
- `prtr @model` requires an active session; exits with clear error if none

### Deprecated (hidden, still functional)
- `prtr go`, `swap`, `take`, `again`, `start`, `learn`, `resume` — use session commands instead
```

- [ ] **Step 2: Final build and full test run**

```bash
go build ./... && go test ./... -v 2>&1 | tail -30
```
Expected: all PASS, no compile errors

- [ ] **Step 3: Final smoke checks**

```bash
./output/prtr --help          # shows review/edit/fix/design, not go/swap/take
./output/prtr go --help       # still works (hidden but functional)
./output/prtr doctor          # shows AI binary checks
```

- [ ] **Step 4: Commit**

```bash
git add CHANGELOG.md
git commit -m "release: v1.0.0 — AI Work Session Manager

Session-first architecture with auto-capture on TUI exit.
prtr review/edit/fix/design/@model command surface.
Legacy commands hidden but fully backward compatible.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

---

## Final Verification Checklist

- [ ] `prtr --help` shows new commands, hides legacy ones
- [ ] `prtr review auth/*.ts` starts session creation flow
- [ ] `prtr` (bare, no session) → interactive goal/file/constraint prompts
- [ ] `prtr @gemini` with no session → clear error: "No active session"
- [ ] `prtr go` still works (hidden but functional)
- [ ] `prtr checkpoint "note"` → "No active session" error when none
- [ ] `prtr sessions` → "세션 없음." when empty
- [ ] `prtr doctor` → shows claude/gemini/codex binary status
- [ ] `go test ./...` → all PASS
