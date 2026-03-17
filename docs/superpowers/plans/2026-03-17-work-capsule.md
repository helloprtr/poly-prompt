# Work Capsule Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `prtr save` / `prtr resume` / `prtr status` / `prtr list` / `prtr prune` commands that capture and restore AI work state locally as Work Capsules.

**Architecture:** Five independent chunks: (1) config + repoctx additions, (2) capsule package types + store + builder + prune, (3) render package, (4) app command wiring, (5) auto-save hooks. Each chunk is independently testable. Chunks 2–5 require Chunk 1; Chunk 4 requires Chunks 2–3; Chunk 5 requires Chunk 4.

**Tech Stack:** Go 1.24, cobra, encoding/json, github.com/helloprtr/poly-prompt (existing module). No new external dependencies.

---

## File Map

### New Files

```
internal/capsule/capsule.go   — Capsule + sub-types, constants, zero values
internal/capsule/store.go     — filesystem CRUD under .prtr/capsules/
internal/capsule/builder.go   — history + run + repoctx → Capsule assembly
internal/capsule/render.go    — Capsule → resume prompt string
internal/capsule/prune.go     — retention policy + storage calculation
internal/capsule/store_test.go
internal/capsule/builder_test.go
internal/capsule/render_test.go
internal/capsule/prune_test.go
```

### Modified Files

```
internal/config/config.go           — add MemoryConfig, wire into Config + fileConfig + Load()
internal/config/config_test.go      — add MemoryConfig default tests
internal/repoctx/repoctx.go         — add HeadSHA to Summary, populate in Collect()
internal/repoctx/repoctx_test.go    — add HeadSHA test
internal/app/app.go                 — add runSave, runResume, runCapsuleStatus, runCapsuleList, runPrune; wire auto-save
internal/app/command.go             — register save, resume, status, list, prune cobra commands
```

---

## Chunk 1: Config + repoctx

**Prerequisite for all other chunks.**

---

### Task 1.1: Add MemoryConfig to config

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Write failing test**

Add to `internal/config/config_test.go`:

```go
func TestMemoryConfigDefaults(t *testing.T) {
    cfg, err := Load()
    if err != nil {
        t.Skipf("no config available: %v", err)
    }
    if !cfg.Memory.Enabled {
        t.Error("Memory.Enabled should default to true")
    }
    if !cfg.Memory.AutoSave {
        t.Error("Memory.AutoSave should default to true")
    }
    if cfg.Memory.CapsuleRetentionDays != 30 {
        t.Errorf("CapsuleRetentionDays: got %d, want 30", cfg.Memory.CapsuleRetentionDays)
    }
    if cfg.Memory.AutosaveRetentionDays != 14 {
        t.Errorf("AutosaveRetentionDays: got %d, want 14", cfg.Memory.AutosaveRetentionDays)
    }
    if cfg.Memory.StoreDiff != "stat" {
        t.Errorf("StoreDiff: got %q, want %q", cfg.Memory.StoreDiff, "stat")
    }
}
```

- [ ] **Step 2: Run test to confirm it fails**

```bash
cd /Users/koo/dev/translateCLI-brew
go test ./internal/config/... -run TestMemoryConfigDefaults -v
```

Expected: FAIL — `cfg.Memory` field does not exist

- [ ] **Step 3: Add MemoryConfig struct and wire into config.go**

After the existing `LauncherConfig` struct, add:

```go
type MemoryConfig struct {
    Enabled               bool   `toml:"enabled"`
    AutoSave              bool   `toml:"auto_save"`
    PruneOnWrite          bool   `toml:"prune_on_write"`
    PruneOnResume         bool   `toml:"prune_on_resume"`
    CapsuleRetentionDays  int    `toml:"capsule_retention_days"`
    AutosaveRetentionDays int    `toml:"autosave_retention_days"`
    RunRetentionDays      int    `toml:"run_retention_days"`
    MaxCapsulesPerRepo    int    `toml:"max_capsules_per_repo"`
    MaxStorageMBPerRepo   int    `toml:"max_storage_mb_per_repo"`
    StoreDiff             string `toml:"store_diff"`
}
```

Add `Memory MemoryConfig` to `Config` struct (no toml tag — Config is the runtime struct).

Add `Memory MemoryConfig \`toml:"memory"\`` to `fileConfig` struct.

Add `defaultMemoryConfig()` function:

```go
func defaultMemoryConfig() MemoryConfig {
    return MemoryConfig{
        Enabled:               true,
        AutoSave:              true,
        PruneOnWrite:          true,
        PruneOnResume:         true,
        CapsuleRetentionDays:  30,
        AutosaveRetentionDays: 14,
        RunRetentionDays:      7,
        MaxCapsulesPerRepo:    200,
        MaxStorageMBPerRepo:   256,
        StoreDiff:             "stat",
    }
}
```

In `Load()`, initialize defaults alongside the other defaults in the `cfg := Config{...}` literal:

```go
cfg := Config{
    // ... existing fields ...
    Memory: defaultMemoryConfig(),
}
```

In `applyFileConfig`, merge memory config from the file if fields are non-zero:

```go
if raw.Memory.CapsuleRetentionDays != 0 {
    cfg.Memory = raw.Memory
    // re-apply individual zero-value guards for booleans
    if !raw.Memory.Enabled {
        cfg.Memory.Enabled = false
    }
}
```

Actually, a cleaner approach: always apply the file's memory section by overriding defaults field by field:

```go
// at end of applyFileConfig, after all other sections:
if raw.Memory.CapsuleRetentionDays != 0 || raw.Memory.AutosaveRetentionDays != 0 ||
    raw.Memory.StoreDiff != "" || raw.Memory.MaxCapsulesPerRepo != 0 {
    if raw.Memory.CapsuleRetentionDays != 0 {
        cfg.Memory.CapsuleRetentionDays = raw.Memory.CapsuleRetentionDays
    }
    if raw.Memory.AutosaveRetentionDays != 0 {
        cfg.Memory.AutosaveRetentionDays = raw.Memory.AutosaveRetentionDays
    }
    if raw.Memory.RunRetentionDays != 0 {
        cfg.Memory.RunRetentionDays = raw.Memory.RunRetentionDays
    }
    if raw.Memory.MaxCapsulesPerRepo != 0 {
        cfg.Memory.MaxCapsulesPerRepo = raw.Memory.MaxCapsulesPerRepo
    }
    if raw.Memory.MaxStorageMBPerRepo != 0 {
        cfg.Memory.MaxStorageMBPerRepo = raw.Memory.MaxStorageMBPerRepo
    }
    if strings.TrimSpace(raw.Memory.StoreDiff) != "" {
        cfg.Memory.StoreDiff = strings.TrimSpace(raw.Memory.StoreDiff)
    }
    // booleans: file always wins if [memory] section is present
    cfg.Memory.Enabled = raw.Memory.Enabled
    cfg.Memory.AutoSave = raw.Memory.AutoSave
    cfg.Memory.PruneOnWrite = raw.Memory.PruneOnWrite
    cfg.Memory.PruneOnResume = raw.Memory.PruneOnResume
}
```

- [ ] **Step 4: Run test to confirm it passes**

```bash
go test ./internal/config/... -run TestMemoryConfigDefaults -v
```

Expected: PASS

- [ ] **Step 5: Run full config test suite**

```bash
go test ./internal/config/... -v
```

Expected: all pass

- [ ] **Step 6: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): add MemoryConfig with retention defaults"
```

---

### Task 1.2: Add HeadSHA to repoctx.Summary

**Files:**
- Modify: `internal/repoctx/repoctx.go`
- Modify: `internal/repoctx/repoctx_test.go`

- [ ] **Step 1: Write failing test**

In `internal/repoctx/repoctx_test.go`, add:

```go
func TestCollectIncludesHeadSHA(t *testing.T) {
    ctx := context.Background()
    c := New()
    summary, err := c.Collect(ctx)
    if errors.Is(err, ErrNotGitRepo) {
        t.Skip("not in a git repo")
    }
    if err != nil {
        t.Fatalf("Collect: %v", err)
    }
    if summary.HeadSHA == "" {
        t.Error("HeadSHA should be non-empty in a git repo")
    }
    if len(summary.HeadSHA) != 7 {
        t.Errorf("HeadSHA should be 7 chars (short SHA), got %d: %q", len(summary.HeadSHA), summary.HeadSHA)
    }
}
```

- [ ] **Step 2: Run test to confirm it fails**

```bash
go test ./internal/repoctx/... -run TestCollectIncludesHeadSHA -v
```

Expected: FAIL — `summary.HeadSHA` field does not exist

- [ ] **Step 3: Add HeadSHA to Summary and populate in Collect()**

In `repoctx.go`, add `HeadSHA string` to the `Summary` struct:

```go
type Summary struct {
    RepoName  string
    Branch    string
    HeadSHA   string
    Changes   []string
    Truncated int
}
```

In `GitCollector.Collect()`, after collecting `branch`, add:

```go
headSHA, err := gitOutput(ctx, "rev-parse", "--short", "HEAD")
if err != nil {
    if isNotGitRepo(err) {
        return Summary{}, ErrNotGitRepo
    }
    return Summary{}, err
}
```

Set `summary.HeadSHA = strings.TrimSpace(headSHA)` when building the `Summary`.

- [ ] **Step 4: Run test to confirm it passes**

```bash
go test ./internal/repoctx/... -run TestCollectIncludesHeadSHA -v
```

Expected: PASS

- [ ] **Step 5: Run full repoctx test suite**

```bash
go test ./internal/repoctx/... -v
```

Expected: all pass

- [ ] **Step 6: Commit**

```bash
git add internal/repoctx/repoctx.go internal/repoctx/repoctx_test.go
git commit -m "feat(repoctx): add HeadSHA to Summary for drift detection"
```

---

## Chunk 2: Capsule Package — Types, Store, Builder, Prune

---

### Task 2.1: Capsule types (capsule.go)

**Files:**
- Create: `internal/capsule/capsule.go`

No test needed for pure type definitions.

- [ ] **Step 1: Create `internal/capsule/capsule.go`**

```go
package capsule

import "time"

// Kind distinguishes manual saves from auto-saves.
const (
    KindManual = "manual"
    KindAuto   = "auto"
)

// Capsule is the complete in-memory and on-disk record of a single Work Capsule.
// All session fields are optional — capsule.json uses omitempty throughout.
type Capsule struct {
    ID        string    `json:"id"`
    Label     string    `json:"label,omitempty"`
    Note      string    `json:"note,omitempty"`
    Kind      string    `json:"kind"`
    Pinned    bool      `json:"pinned,omitempty"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`

    Repo    RepoState   `json:"repo"`
    Session SessionState `json:"session,omitempty"`
    Work    WorkState   `json:"work"`
}

// RepoState captures git state at save time.
type RepoState struct {
    Root         string   `json:"root"`
    Name         string   `json:"name"`
    Branch       string   `json:"branch"`
    HeadSHA      string   `json:"head_sha"`
    TouchedFiles []string `json:"touched_files,omitempty"`
    DiffStat     string   `json:"diff_stat,omitempty"`
}

// SessionState links back to the history entry and deep run that preceded this save.
// All fields are optional — zero values are omitted in JSON.
type SessionState struct {
    TargetApp       string `json:"target_app,omitempty"`
    Engine          string `json:"engine,omitempty"`
    Mode            string `json:"mode,omitempty"`
    SourceHistoryID string `json:"source_history_id,omitempty"`
    SourceRunID     string `json:"source_run_id,omitempty"`
    ArtifactRoot    string `json:"artifact_root,omitempty"`
}

// WorkState holds the semantic content of the work being done.
type WorkState struct {
    OriginalRequest string     `json:"original_request,omitempty"`
    NormalizedGoal  string     `json:"normalized_goal,omitempty"`
    NextAction      string     `json:"next_action,omitempty"`
    Summary         string     `json:"summary,omitempty"`
    ProtectedTerms  []string   `json:"protected_terms,omitempty"`
    Todos           []TodoItem `json:"todos,omitempty"`
    Decisions       []string   `json:"decisions,omitempty"`
    OpenQuestions   []string   `json:"open_questions,omitempty"`
    Risks           []string   `json:"risks,omitempty"`
}

// TodoItem mirrors a single task from deep run plan.json.
type TodoItem struct {
    ID     string `json:"id"`
    Title  string `json:"title"`
    Status string `json:"status"` // "pending" | "completed" | "failed"
}

// DriftReport describes how the current repo state differs from a saved capsule.
type DriftReport struct {
    BranchChanged bool
    SavedBranch   string
    CurrentBranch string
    SHAChanged    bool
    SavedSHA      string
    CurrentSHA    string
    FilesChanged  bool
}

// HasDrift returns true if any drift was detected.
func (d DriftReport) HasDrift() bool {
    return d.BranchChanged || d.SHAChanged || d.FilesChanged
}
```

- [ ] **Step 2: Build to confirm it compiles**

```bash
go build ./internal/capsule/...
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/capsule/capsule.go
git commit -m "feat(capsule): add Capsule types and DriftReport"
```

---

### Task 2.2: Capsule store (store.go)

**Files:**
- Create: `internal/capsule/store.go`
- Create: `internal/capsule/store_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/capsule/store_test.go`:

```go
package capsule_test

import (
    "os"
    "path/filepath"
    "testing"
    "time"

    "github.com/helloprtr/poly-prompt/internal/capsule"
)

func newTestCapsule(id, label, kind string) capsule.Capsule {
    now := time.Now().UTC()
    return capsule.Capsule{
        ID:        id,
        Label:     label,
        Kind:      kind,
        CreatedAt: now,
        UpdatedAt: now,
        Repo: capsule.RepoState{
            Root:    "/test/repo",
            Name:    "repo",
            Branch:  "main",
            HeadSHA: "abc1234",
        },
        Work: capsule.WorkState{
            OriginalRequest: "test request",
            NormalizedGoal:  "test goal",
        },
    }
}

func TestStoreSaveAndLoad(t *testing.T) {
    dir := t.TempDir()
    store := capsule.NewStore(dir)

    c := newTestCapsule("cap_001", "test label", capsule.KindManual)
    if err := store.Save(c); err != nil {
        t.Fatalf("Save: %v", err)
    }

    loaded, err := store.Load("cap_001")
    if err != nil {
        t.Fatalf("Load: %v", err)
    }
    if loaded.Label != "test label" {
        t.Errorf("Label: got %q, want %q", loaded.Label, "test label")
    }
    if loaded.Kind != capsule.KindManual {
        t.Errorf("Kind: got %q, want %q", loaded.Kind, capsule.KindManual)
    }
}

func TestStoreList(t *testing.T) {
    dir := t.TempDir()
    store := capsule.NewStore(dir)

    for _, id := range []string{"cap_001", "cap_002", "cap_003"} {
        c := newTestCapsule(id, id, capsule.KindManual)
        if err := store.Save(c); err != nil {
            t.Fatalf("Save %s: %v", id, err)
        }
    }

    list, err := store.List()
    if err != nil {
        t.Fatalf("List: %v", err)
    }
    if len(list) != 3 {
        t.Errorf("List count: got %d, want 3", len(list))
    }
}

func TestStoreLatest(t *testing.T) {
    dir := t.TempDir()
    store := capsule.NewStore(dir)

    t1 := time.Now().UTC().Add(-time.Hour)
    t2 := time.Now().UTC()

    c1 := newTestCapsule("cap_001", "old", capsule.KindManual)
    c1.CreatedAt = t1
    c2 := newTestCapsule("cap_002", "new", capsule.KindManual)
    c2.CreatedAt = t2

    _ = store.Save(c1)
    _ = store.Save(c2)

    latest, err := store.Latest()
    if err != nil {
        t.Fatalf("Latest: %v", err)
    }
    if latest.ID != "cap_002" {
        t.Errorf("Latest ID: got %q, want %q", latest.ID, "cap_002")
    }
}

func TestStoreLatestEmpty(t *testing.T) {
    dir := t.TempDir()
    store := capsule.NewStore(dir)
    _, err := store.Latest()
    if err == nil {
        t.Error("Latest on empty store should return error")
    }
}

func TestStoreDelete(t *testing.T) {
    dir := t.TempDir()
    store := capsule.NewStore(dir)

    c := newTestCapsule("cap_001", "to delete", capsule.KindManual)
    _ = store.Save(c)
    if err := store.Delete("cap_001"); err != nil {
        t.Fatalf("Delete: %v", err)
    }

    _, err := store.Load("cap_001")
    if err == nil {
        t.Error("Load after delete should return error")
    }
}

// Note: TestStoreSummaryFileWritten is in Task 2.3 (after renderSummaryMD is added).
// Do not add it here — store_test.go will not compile until renderSummaryMD exists.
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./internal/capsule/... -run "TestStore" -v
```

Expected: FAIL — package not compilable yet

- [ ] **Step 3: Create `internal/capsule/store.go`**

```go
package capsule

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

var ErrNotFound = errors.New("capsule not found")

// Store manages capsules under <repoRoot>/.prtr/capsules/.
type Store struct {
    root string // absolute path to the capsules directory
}

// NewStore returns a Store rooted at dir (the capsules directory, not repo root).
func NewStore(dir string) *Store {
    return &Store{root: dir}
}

// DefaultDir resolves the capsules directory for the given repo root.
// Creates the directory if it does not exist.
func DefaultDir(repoRoot string) (string, error) {
    dir := filepath.Join(repoRoot, ".prtr", "capsules")
    if err := os.MkdirAll(dir, 0o755); err != nil {
        return "", fmt.Errorf("create capsules dir: %w", err)
    }
    return dir, nil
}

// Save writes capsule.json and summary.md to <root>/<id>/.
func (s *Store) Save(c Capsule) error {
    dir := filepath.Join(s.root, c.ID)
    if err := os.MkdirAll(dir, 0o755); err != nil {
        return fmt.Errorf("create capsule dir: %w", err)
    }

    data, err := json.MarshalIndent(c, "", "  ")
    if err != nil {
        return fmt.Errorf("encode capsule: %w", err)
    }
    if err := writeAtomic(filepath.Join(dir, "capsule.json"), data); err != nil {
        return err
    }

    summary := renderSummaryMD(c)
    if err := writeAtomic(filepath.Join(dir, "summary.md"), []byte(summary)); err != nil {
        return err
    }

    return nil
}

// Load reads and parses capsule.json for the given id.
func (s *Store) Load(id string) (Capsule, error) {
    path := filepath.Join(s.root, id, "capsule.json")
    data, err := os.ReadFile(path)
    if err != nil {
        if errors.Is(err, os.ErrNotExist) {
            return Capsule{}, ErrNotFound
        }
        return Capsule{}, fmt.Errorf("read capsule %s: %w", id, err)
    }
    var c Capsule
    if err := json.Unmarshal(data, &c); err != nil {
        return Capsule{}, fmt.Errorf("parse capsule %s: %w", id, err)
    }
    return c, nil
}

// List returns all capsules sorted by CreatedAt descending (newest first).
func (s *Store) List() ([]Capsule, error) {
    entries, err := os.ReadDir(s.root)
    if err != nil {
        if errors.Is(err, os.ErrNotExist) {
            return nil, nil
        }
        return nil, fmt.Errorf("list capsules: %w", err)
    }

    var caps []Capsule
    for _, e := range entries {
        if !e.IsDir() || !strings.HasPrefix(e.Name(), "cap_") {
            continue
        }
        c, err := s.Load(e.Name())
        if err != nil {
            continue // skip corrupt entries silently
        }
        caps = append(caps, c)
    }

    sort.Slice(caps, func(i, j int) bool {
        return caps[i].CreatedAt.After(caps[j].CreatedAt)
    })

    return caps, nil
}

// Latest returns the most recently created capsule.
func (s *Store) Latest() (Capsule, error) {
    caps, err := s.List()
    if err != nil {
        return Capsule{}, err
    }
    if len(caps) == 0 {
        return Capsule{}, ErrNotFound
    }
    return caps[0], nil
}

// Delete removes the capsule directory for id.
func (s *Store) Delete(id string) error {
    dir := filepath.Join(s.root, id)
    if err := os.RemoveAll(dir); err != nil {
        return fmt.Errorf("delete capsule %s: %w", id, err)
    }
    return nil
}

// Update loads a capsule, applies fn, and saves it back (same id, bumped UpdatedAt).
func (s *Store) Update(id string, fn func(*Capsule)) error {
    c, err := s.Load(id)
    if err != nil {
        return err
    }
    fn(&c)
    c.UpdatedAt = time.Now().UTC()
    return s.Save(c)
}

// NewID generates a unique capsule ID using current UnixNano.
func NewID() string {
    return fmt.Sprintf("cap_%d", time.Now().UTC().UnixNano())
}

func writeAtomic(path string, data []byte) error {
    dir := filepath.Dir(path)
    tmp, err := os.CreateTemp(dir, ".cap-*.tmp")
    if err != nil {
        return fmt.Errorf("create temp file: %w", err)
    }
    tmpPath := tmp.Name()
    if _, err := tmp.Write(data); err != nil {
        _ = tmp.Close()
        _ = os.Remove(tmpPath)
        return fmt.Errorf("write temp file: %w", err)
    }
    if err := tmp.Close(); err != nil {
        _ = os.Remove(tmpPath)
        return fmt.Errorf("close temp file: %w", err)
    }
    if err := os.Rename(tmpPath, path); err != nil {
        _ = os.Remove(tmpPath)
        return fmt.Errorf("commit file: %w", err)
    }
    return nil
}
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
go test ./internal/capsule/... -run "TestStore" -v
```

Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add internal/capsule/store.go internal/capsule/store_test.go
git commit -m "feat(capsule): add Store — filesystem CRUD for capsules"
```

---

### Task 2.3: Summary MD renderer (summary.go helper)

The `renderSummaryMD` function is used by `store.go`. Add it in `store.go` (same file, unexported):

- [ ] **Step 1: Add `renderSummaryMD` to `store.go`**

At the bottom of `store.go`, add:

```go
func renderSummaryMD(c Capsule) string {
    var b strings.Builder

    label := c.Label
    if label == "" {
        label = "[auto]"
    }
    fmt.Fprintf(&b, "# %s\n", label)
    fmt.Fprintf(&b, "**Saved:** %s · branch: %s · sha: %s\n\n",
        c.CreatedAt.Format("2006-01-02 15:04"), c.Repo.Branch, c.Repo.HeadSHA)

    if c.Work.OriginalRequest != "" {
        fmt.Fprintf(&b, "## What was being worked on\n%s\n\n", c.Work.OriginalRequest)
    }

    if len(c.Work.Todos) > 0 {
        fmt.Fprintf(&b, "## Progress\n")
        for _, t := range c.Work.Todos {
            mark := "○"
            if t.Status == "completed" {
                mark = "✓"
            } else if t.Status == "failed" {
                mark = "✕"
            }
            fmt.Fprintf(&b, "- %s %s\n", mark, t.Title)
        }
        fmt.Fprintln(&b)
    }

    if len(c.Work.Decisions) > 0 {
        fmt.Fprintf(&b, "## Decisions made\n")
        for _, d := range c.Work.Decisions {
            fmt.Fprintf(&b, "- %s\n", d)
        }
        fmt.Fprintln(&b)
    }

    if len(c.Work.OpenQuestions) > 0 {
        fmt.Fprintf(&b, "## Open questions\n")
        for _, q := range c.Work.OpenQuestions {
            fmt.Fprintf(&b, "- %s\n", q)
        }
        fmt.Fprintln(&b)
    }

    if len(c.Work.Risks) > 0 {
        fmt.Fprintf(&b, "## Risks\n")
        for _, r := range c.Work.Risks {
            fmt.Fprintf(&b, "- %s\n", r)
        }
        fmt.Fprintln(&b)
    }

    if c.Work.NextAction != "" {
        fmt.Fprintf(&b, "## Next action\n%s\n", c.Work.NextAction)
    }

    return b.String()
}
```

- [ ] **Step 2: Add summary.md test to `store_test.go` now that renderSummaryMD exists**

Add to `internal/capsule/store_test.go`:

```go
func TestStoreSummaryFileWritten(t *testing.T) {
    dir := t.TempDir()
    store := capsule.NewStore(dir)

    c := newTestCapsule("cap_001", "test", capsule.KindManual)
    _ = store.Save(c)

    summaryPath := filepath.Join(dir, "cap_001", "summary.md")
    if _, err := os.Stat(summaryPath); err != nil {
        t.Errorf("summary.md not written: %v", err)
    }
}
```

- [ ] **Step 3: Build and run full capsule test suite**

```bash
go test ./internal/capsule/... -v
```

Expected: all PASS including TestStoreSummaryFileWritten

- [ ] **Step 4: Commit**

```bash
git add internal/capsule/store.go internal/capsule/store_test.go
git commit -m "feat(capsule): add renderSummaryMD for summary.md generation"
```

---

### Task 2.4: Capsule builder (builder.go)

**Files:**
- Create: `internal/capsule/builder.go`
- Create: `internal/capsule/builder_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/capsule/builder_test.go`:

```go
package capsule_test

import (
    "testing"
    "time"

    "github.com/helloprtr/poly-prompt/internal/capsule"
    "github.com/helloprtr/poly-prompt/internal/history"
    "github.com/helloprtr/poly-prompt/internal/repoctx"
)

func TestBuildFromInputs(t *testing.T) {
    entry := history.Entry{
        ID:           "hist_001",
        CreatedAt:    time.Now().UTC(),
        Original:     "implement JWT auth",
        Target:       "claude",
        Engine:       "deep",
        RunID:        "run_001",
        ArtifactRoot: ".prtr/runs/run_001",
    }
    summary := repoctx.Summary{
        RepoName: "myrepo",
        Branch:   "fix/auth",
        HeadSHA:  "abc1234",
        Changes:  []string{"M internal/auth/auth.go"},
    }

    in := capsule.BuildInput{
        Label:       "auth refactor paused",
        Note:        "JWT decided",
        Kind:        capsule.KindManual,
        HistoryEntry: &entry,
        RepoSummary: summary,
        RepoRoot:    "/test/repo",
    }

    c := capsule.Build(in)

    if c.ID == "" {
        t.Error("ID should be set")
    }
    if c.Label != "auth refactor paused" {
        t.Errorf("Label: got %q", c.Label)
    }
    if c.Session.TargetApp != "claude" {
        t.Errorf("TargetApp: got %q", c.Session.TargetApp)
    }
    if c.Repo.HeadSHA != "abc1234" {
        t.Errorf("HeadSHA: got %q", c.Repo.HeadSHA)
    }
    if c.Work.NormalizedGoal == "" {
        t.Error("NormalizedGoal should not be empty")
    }
    if len(c.Session.SourceHistoryID) == 0 {
        t.Error("SourceHistoryID should be set from history entry")
    }
}

func TestBuildWithNoHistoryEntry(t *testing.T) {
    summary := repoctx.Summary{
        RepoName: "myrepo",
        Branch:   "main",
        HeadSHA:  "def5678",
    }

    in := capsule.BuildInput{
        Kind:        capsule.KindAuto,
        RepoSummary: summary,
        RepoRoot:    "/test/repo",
    }

    c := capsule.Build(in)

    if c.ID == "" {
        t.Error("ID should be set even without history entry")
    }
    if c.Session.TargetApp != "" {
        t.Errorf("TargetApp should be empty when no history entry: got %q", c.Session.TargetApp)
    }
    if c.Repo.Branch != "main" {
        t.Errorf("Branch: got %q", c.Repo.Branch)
    }
}

func TestNormalizeGoal(t *testing.T) {
    tests := []struct {
        input string
        want  string
    }{
        {"Implement JWT Auth", "implement jwt auth"},
        {
            "This is a very long request that exceeds one hundred characters and should be truncated by the normalizer function",
            "this is a very long request that exceeds one hundred characters and should be truncated by the norm",
        },
        {"", ""},
    }
    for _, tt := range tests {
        got := capsule.NormalizeGoal(tt.input)
        if got != tt.want {
            t.Errorf("NormalizeGoal(%q) = %q, want %q", tt.input, got, tt.want)
        }
    }
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./internal/capsule/... -run "TestBuild|TestNormalize" -v
```

Expected: FAIL — builder not implemented

- [ ] **Step 3: Create `internal/capsule/builder.go`**

```go
package capsule

import (
    "path/filepath"
    "strings"
    "time"

    "github.com/helloprtr/poly-prompt/internal/history"
    "github.com/helloprtr/poly-prompt/internal/repoctx"
)

// BuildInput holds everything needed to construct a Capsule.
type BuildInput struct {
    Label        string
    Note         string
    Kind         string
    HistoryEntry *history.Entry  // optional — nil if no prior run
    RepoSummary  repoctx.Summary
    RepoRoot     string
    Todos        []TodoItem // optional override; if nil, extracted from history entry
}

// Build constructs a Capsule from the given inputs.
// Missing optional inputs (HistoryEntry, Todos) result in empty/zero session/work fields.
func Build(in BuildInput) Capsule {
    now := time.Now().UTC()

    kind := in.Kind
    if kind == "" {
        kind = KindAuto
    }

    c := Capsule{
        ID:        NewID(),
        Label:     strings.TrimSpace(in.Label),
        Note:      strings.TrimSpace(in.Note),
        Kind:      kind,
        CreatedAt: now,
        UpdatedAt: now,
        Repo: RepoState{
            Root:         in.RepoRoot,
            Name:         in.RepoSummary.RepoName,
            Branch:       in.RepoSummary.Branch,
            HeadSHA:      in.RepoSummary.HeadSHA,
            TouchedFiles: changedFiles(in.RepoSummary.Changes),
        },
    }

    if in.HistoryEntry != nil {
        e := in.HistoryEntry
        c.Session = SessionState{
            TargetApp:       e.Target,
            Engine:          e.Engine,
            Mode:            e.ResultType,
            SourceHistoryID: e.ID,
            SourceRunID:     e.RunID,
            ArtifactRoot:    relativeArtifactRoot(in.RepoRoot, e.ArtifactRoot),
        }
        c.Work = WorkState{
            OriginalRequest: e.Original,
            NormalizedGoal:  NormalizeGoal(e.Original),
            // ProtectedTerms are not stored on history.Entry;
            // they can be added via --note or future deep run integration.
        }
        if len(in.Todos) > 0 {
            c.Work.Todos = in.Todos
        }
    }

    return c
}

// NormalizeGoal produces a deterministic dedup key from a request string:
// lowercase, trimmed, max 100 chars.
func NormalizeGoal(request string) string {
    s := strings.ToLower(strings.TrimSpace(request))
    if len(s) > 100 {
        s = s[:100]
    }
    return s
}

// changedFiles extracts bare file paths from git status --short output lines.
// Input lines look like: "M internal/auth/auth.go" or "?? newfile.go"
func changedFiles(changes []string) []string {
    files := make([]string, 0, len(changes))
    for _, line := range changes {
        parts := strings.Fields(line)
        if len(parts) >= 2 {
            files = append(files, parts[len(parts)-1])
        }
    }
    return files
}

// relativeArtifactRoot converts an absolute artifact root path to a path
// relative to repoRoot, for portability. Falls back to the original if
// it cannot be made relative.
func relativeArtifactRoot(repoRoot, artifactRoot string) string {
    if repoRoot == "" || artifactRoot == "" {
        return artifactRoot
    }
    rel, err := filepath.Rel(repoRoot, artifactRoot)
    if err != nil {
        return artifactRoot
    }
    return rel
}
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
go test ./internal/capsule/... -run "TestBuild|TestNormalize" -v
```

Expected: all PASS

- [ ] **Step 5: Run full capsule test suite**

```bash
go test ./internal/capsule/... -v
```

Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add internal/capsule/builder.go internal/capsule/builder_test.go
git commit -m "feat(capsule): add Builder — assembles Capsule from history + repoctx"
```

---

### Task 2.5: Prune (prune.go)

**Files:**
- Create: `internal/capsule/prune.go`
- Create: `internal/capsule/prune_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/capsule/prune_test.go`:

```go
package capsule_test

import (
    "testing"
    "time"

    "github.com/helloprtr/poly-prompt/internal/capsule"
    "github.com/helloprtr/poly-prompt/internal/config"
)

func makeCapsuleAge(id, kind string, age time.Duration, pinned bool) capsule.Capsule {
    now := time.Now().UTC()
    return capsule.Capsule{
        ID:        id,
        Kind:      kind,
        Pinned:    pinned,
        CreatedAt: now.Add(-age),
        UpdatedAt: now.Add(-age),
        Repo:      capsule.RepoState{Branch: "main", HeadSHA: "abc"},
    }
}

func TestPruneByRetentionDays(t *testing.T) {
    cfg := config.MemoryConfig{
        CapsuleRetentionDays:  30,
        AutosaveRetentionDays: 14,
    }

    caps := []capsule.Capsule{
        makeCapsuleAge("cap_keep_manual",   capsule.KindManual, 10*24*time.Hour, false),  // 10d — keep
        makeCapsuleAge("cap_drop_manual",   capsule.KindManual, 31*24*time.Hour, false),  // 31d — drop
        makeCapsuleAge("cap_keep_auto",     capsule.KindAuto,   5*24*time.Hour,  false),  // 5d  — keep
        makeCapsuleAge("cap_drop_auto",     capsule.KindAuto,   15*24*time.Hour, false),  // 15d — drop
        makeCapsuleAge("cap_pinned_old",    capsule.KindManual, 365*24*time.Hour, true),  // 1yr pinned — keep
    }

    toDelete := capsule.ApplyRetentionPolicy(caps, cfg)

    deleteSet := map[string]bool{}
    for _, id := range toDelete {
        deleteSet[id] = true
    }

    if deleteSet["cap_keep_manual"] {
        t.Error("cap_keep_manual should not be deleted (10d < 30d)")
    }
    if !deleteSet["cap_drop_manual"] {
        t.Error("cap_drop_manual should be deleted (31d > 30d)")
    }
    if deleteSet["cap_keep_auto"] {
        t.Error("cap_keep_auto should not be deleted (5d < 14d)")
    }
    if !deleteSet["cap_drop_auto"] {
        t.Error("cap_drop_auto should be deleted (15d > 14d)")
    }
    if deleteSet["cap_pinned_old"] {
        t.Error("cap_pinned_old should never be deleted (pinned)")
    }
}
```

- [ ] **Step 2: Run test to confirm it fails**

```bash
go test ./internal/capsule/... -run "TestPrune" -v
```

Expected: FAIL

- [ ] **Step 3: Create `internal/capsule/prune.go`**

```go
package capsule

import (
    "time"

    "github.com/helloprtr/poly-prompt/internal/config"
)

// ApplyRetentionPolicy returns a list of capsule IDs that should be deleted
// according to the configured retention days. Pinned capsules are never included.
func ApplyRetentionPolicy(caps []Capsule, cfg config.MemoryConfig) []string {
    now := time.Now().UTC()
    var toDelete []string

    for _, c := range caps {
        if c.Pinned {
            continue
        }
        var maxAge time.Duration
        switch c.Kind {
        case KindManual:
            maxAge = time.Duration(cfg.CapsuleRetentionDays) * 24 * time.Hour
        case KindAuto:
            maxAge = time.Duration(cfg.AutosaveRetentionDays) * 24 * time.Hour
        default:
            maxAge = time.Duration(cfg.CapsuleRetentionDays) * 24 * time.Hour
        }
        if maxAge > 0 && now.Sub(c.CreatedAt) > maxAge {
            toDelete = append(toDelete, c.ID)
        }
    }

    return toDelete
}

// ApplyOlderThan returns IDs of capsules older than the given duration.
// Pinned capsules are never included.
func ApplyOlderThan(caps []Capsule, d time.Duration) []string {
    now := time.Now().UTC()
    var toDelete []string
    for _, c := range caps {
        if c.Pinned {
            continue
        }
        if now.Sub(c.CreatedAt) > d {
            toDelete = append(toDelete, c.ID)
        }
    }
    return toDelete
}
```

- [ ] **Step 4: Run test to confirm it passes**

```bash
go test ./internal/capsule/... -run "TestPrune" -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/capsule/prune.go internal/capsule/prune_test.go
git commit -m "feat(capsule): add prune — retention policy implementation"
```

---

## Chunk 3: Render

---

### Task 3.1: Resume prompt renderer (render.go)

**Files:**
- Create: `internal/capsule/render.go`
- Create: `internal/capsule/render_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/capsule/render_test.go`:

```go
package capsule_test

import (
    "strings"
    "testing"
    "time"

    "github.com/helloprtr/poly-prompt/internal/capsule"
)

func testCapsuleForRender() capsule.Capsule {
    now := time.Now().UTC()
    return capsule.Capsule{
        ID:        "cap_001",
        Label:     "auth refactor",
        Kind:      capsule.KindManual,
        CreatedAt: now,
        UpdatedAt: now,
        Repo: capsule.RepoState{
            Name:    "myrepo",
            Branch:  "fix/auth",
            HeadSHA: "abc1234",
        },
        Session: capsule.SessionState{
            TargetApp: "claude",
        },
        Work: capsule.WorkState{
            OriginalRequest: "implement JWT auth",
            NormalizedGoal:  "implement jwt auth",
            NextAction:      "add token refresh in auth.go",
            Summary:         "JWT base done.",
            Todos: []capsule.TodoItem{
                {ID: "a", Title: "Design auth", Status: "completed"},
                {ID: "b", Title: "Token refresh", Status: "pending"},
            },
            Decisions:     []string{"Use JWT"},
            OpenQuestions: []string{"Refresh interval?"},
            Risks:         []string{"No revocation"},
        },
    }
}

func TestRenderPromptContainsKeyFields(t *testing.T) {
    c := testCapsuleForRender()
    prompt := capsule.RenderResumePrompt(c, "claude", capsule.DriftReport{})

    checks := []string{
        "implement JWT auth",     // original request
        "Token refresh",          // todo item
        "Use JWT",                // decision
        "Refresh interval?",      // open question
        "No revocation",          // risk
        "add token refresh",      // next action
        "fix/auth",               // branch
    }
    for _, s := range checks {
        if !strings.Contains(prompt, s) {
            t.Errorf("prompt missing %q", s)
        }
    }
}

func TestRenderPromptIncludesDriftWarning(t *testing.T) {
    c := testCapsuleForRender()
    drift := capsule.DriftReport{
        BranchChanged: true,
        SavedBranch:   "fix/auth",
        CurrentBranch: "main",
    }
    prompt := capsule.RenderResumePrompt(c, "claude", drift)

    if !strings.Contains(prompt, "drift") && !strings.Contains(prompt, "branch changed") {
        t.Error("prompt with drift should mention branch change")
    }
}

func TestRenderPromptNoDriftSection(t *testing.T) {
    c := testCapsuleForRender()
    prompt := capsule.RenderResumePrompt(c, "claude", capsule.DriftReport{})

    if strings.Contains(strings.ToLower(prompt), "drift") {
        t.Error("prompt without drift should not mention drift")
    }
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./internal/capsule/... -run "TestRender" -v
```

Expected: FAIL

- [ ] **Step 3: Create `internal/capsule/render.go`**

```go
package capsule

import (
    "fmt"
    "strings"
)

// RenderResumePrompt generates a target-agnostic resume prompt from the capsule.
// The prompt is structured plain text suitable for pasting into any AI app.
// If drift has been detected, a warning section is prepended.
// Target is used to tailor tone (reserved for future LLM-enhanced rendering).
func RenderResumePrompt(c Capsule, target string, drift DriftReport) string {
    var b strings.Builder

    // Drift warning (prepended if any drift exists)
    if drift.HasDrift() {
        fmt.Fprintf(&b, "## ⚠ Repo Drift Detected\n")
        if drift.BranchChanged {
            fmt.Fprintf(&b, "- branch changed: %s → %s\n", drift.SavedBranch, drift.CurrentBranch)
        }
        if drift.SHAChanged {
            fmt.Fprintf(&b, "- commits since save: %s → %s\n", drift.SavedSHA, drift.CurrentSHA)
        }
        if drift.FilesChanged {
            fmt.Fprintf(&b, "- changed files differ from save point\n")
        }
        fmt.Fprintf(&b, "\nPlease review the changes above before continuing.\n\n---\n\n")
    }

    // Resume context
    fmt.Fprintf(&b, "## Resume Context\n\n")
    fmt.Fprintf(&b, "**Repo:** %s  **Branch:** %s  **SHA:** %s\n\n",
        c.Repo.Name, c.Repo.Branch, c.Repo.HeadSHA)

    if c.Work.OriginalRequest != "" {
        fmt.Fprintf(&b, "**What was being worked on:**\n%s\n\n", c.Work.OriginalRequest)
    }
    if c.Work.Summary != "" {
        fmt.Fprintf(&b, "**Progress summary:**\n%s\n\n", c.Work.Summary)
    }

    if len(c.Work.Todos) > 0 {
        fmt.Fprintf(&b, "**TODOs:**\n")
        for _, t := range c.Work.Todos {
            mark := "○"
            if t.Status == "completed" {
                mark = "✓"
            }
            fmt.Fprintf(&b, "- [%s] %s\n", mark, t.Title)
        }
        fmt.Fprintln(&b)
    }

    if len(c.Work.Decisions) > 0 {
        fmt.Fprintf(&b, "**Decisions already made:**\n")
        for _, d := range c.Work.Decisions {
            fmt.Fprintf(&b, "- %s\n", d)
        }
        fmt.Fprintln(&b)
    }

    if len(c.Work.OpenQuestions) > 0 {
        fmt.Fprintf(&b, "**Open questions:**\n")
        for _, q := range c.Work.OpenQuestions {
            fmt.Fprintf(&b, "- %s\n", q)
        }
        fmt.Fprintln(&b)
    }

    if len(c.Work.Risks) > 0 {
        fmt.Fprintf(&b, "**Risks:**\n")
        for _, r := range c.Work.Risks {
            fmt.Fprintf(&b, "- %s\n", r)
        }
        fmt.Fprintln(&b)
    }

    if c.Work.NextAction != "" {
        fmt.Fprintf(&b, "**Next recommended action:**\n%s\n\n", c.Work.NextAction)
    }

    if c.Session.ArtifactRoot != "" {
        fmt.Fprintf(&b, "**Artifacts to inspect:** `%s`\n\n", c.Session.ArtifactRoot)
    }

    if len(c.Repo.TouchedFiles) > 0 {
        fmt.Fprintf(&b, "**Touched files:**\n")
        for _, f := range c.Repo.TouchedFiles {
            fmt.Fprintf(&b, "- %s\n", f)
        }
        fmt.Fprintln(&b)
    }

    return b.String()
}
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
go test ./internal/capsule/... -run "TestRender" -v
```

Expected: all PASS

- [ ] **Step 5: Run full capsule suite**

```bash
go test ./internal/capsule/... -v
```

Expected: all PASS

- [ ] **Step 6: Commit**

```bash
git add internal/capsule/render.go internal/capsule/render_test.go
git commit -m "feat(capsule): add RenderResumePrompt with drift warning support"
```

---

## Chunk 4: App Wiring — Commands

---

### Task 4.1: Register cobra commands

**Files:**
- Modify: `internal/app/command.go`

- [ ] **Step 1: Add command stubs to `command.go`**

Add the following to `command.go` after `newInitCommand`. These are stubs that call `app.go` methods we will add next:

```go
func (a *App) newSaveCommand() *cobra.Command {
    var note string
    cmd := &cobra.Command{
        Use:   "save [label]",
        Short: "Save the current work state as a capsule.",
        RunE: func(cmd *cobra.Command, args []string) error {
            label := ""
            if len(args) > 0 {
                label = strings.Join(args, " ")
            }
            return a.runSave(label, note)
        },
    }
    cmd.Flags().StringVar(&note, "note", "", "Short annotation (e.g. decisions, open questions)")
    return cmd
}

func (a *App) newResumeCommand(ctx context.Context) *cobra.Command {
    var to, olderThan string
    var dryRun bool
    cmd := &cobra.Command{
        Use:   "resume [capsule-id|latest]",
        Short: "Restore a saved work capsule and continue.",
        RunE: func(cmd *cobra.Command, args []string) error {
            id := ""
            if len(args) > 0 && args[0] != "latest" {
                id = args[0]
            }
            return a.runResume(ctx, id, to, dryRun)
        },
    }
    cmd.Flags().StringVar(&to, "to", "", "Target app to render the resume prompt for (claude|codex|gemini)")
    cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print the resume prompt without sending it")
    return cmd
}

func (a *App) newCapsuleStatusCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "status",
        Short: "Show the latest capsule state and detect repo drift.",
        RunE: func(cmd *cobra.Command, args []string) error {
            return a.runCapsuleStatus()
        },
    }
}

func (a *App) newCapsuleListCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "list",
        Short: "List saved work capsules for this repo.",
        RunE: func(cmd *cobra.Command, args []string) error {
            return a.runCapsuleList()
        },
    }
}

func (a *App) newPruneCommand() *cobra.Command {
    var olderThan string
    var dryRun bool
    cmd := &cobra.Command{
        Use:   "prune",
        Short: "Delete old capsules according to retention policy.",
        RunE: func(cmd *cobra.Command, args []string) error {
            return a.runPrune(olderThan, dryRun)
        },
    }
    cmd.Flags().StringVar(&olderThan, "older-than", "", "Delete capsules older than this duration (e.g. 30d)")
    cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print what would be deleted without deleting")
    return cmd
}
```

In `Command()`, add the new commands to the root command after `newInitCommand`:

```go
root.AddCommand(a.newSaveCommand())
root.AddCommand(a.newResumeCommand(ctx))
root.AddCommand(a.newCapsuleStatusCommand())
root.AddCommand(a.newCapsuleListCommand())
root.AddCommand(a.newPruneCommand())
```

Also add `"import \"strings\""` if not already present at the top of the file (it should already be there).

- [ ] **Step 2: Build to confirm it compiles**

```bash
go build ./...
```

Expected: FAIL because `a.runSave`, `a.runResume`, etc. do not exist yet — that is expected

- [ ] **Step 3: Add stub implementations to `app.go`**

These stubs make the build pass. Real implementations follow in subsequent tasks.

Add to `app.go`:

```go
func (a *App) runSave(label, note string) error {
    return errors.New("save: not yet implemented")
}

func (a *App) runResume(ctx context.Context, id, to string, dryRun bool) error {
    return errors.New("resume: not yet implemented")
}

func (a *App) runCapsuleStatus() error {
    return errors.New("status: not yet implemented")
}

func (a *App) runCapsuleList() error {
    return errors.New("list: not yet implemented")
}

func (a *App) runPrune(olderThan string, dryRun bool) error {
    return errors.New("prune: not yet implemented")
}
```

- [ ] **Step 4: Build to confirm it compiles**

```bash
go build ./...
```

Expected: SUCCESS

- [ ] **Step 5: Commit**

```bash
git add internal/app/command.go internal/app/app.go
git commit -m "feat(app): register save/resume/status/list/prune commands (stubs)"
```

---

### Task 4.2: `prtr save` implementation

**Files:**
- Modify: `internal/app/app.go`

The `runSave` function needs: repo root, repoctx summary, latest history entry, config (for MemoryConfig), capsule store, capsule builder.

- [ ] **Step 1: Add capsule import to `app.go`**

Add to the import block in `app.go`:
```go
"github.com/helloprtr/poly-prompt/internal/capsule"
```

- [ ] **Step 2: Replace the `runSave` stub with the real implementation**

```go
func (a *App) runSave(label, note string) error {
    cfg, err := a.configLoader()
    if err != nil {
        return fmt.Errorf("load config: %w", err)
    }
    if !cfg.Memory.Enabled {
        _, _ = fmt.Fprintln(a.stdout, "capsules are disabled (memory.enabled = false)")
        return nil
    }

    repoRoot, err := a.repoRootFinder()
    if err != nil {
        return fmt.Errorf("find repo root: %w", err)
    }

    ctx := context.Background()
    repoSummary, err := a.repoContext.Collect(ctx)
    if err != nil {
        return fmt.Errorf("collect repo context: %w", err)
    }

    // Latest history entry is optional — proceed without it on ErrNotFound.
    var histEntry *history.Entry
    if a.historyStore != nil {
        if e, err := a.historyStore.Latest(); err == nil {
            histEntry = &e
        }
    }

    in := capsule.BuildInput{
        Label:        label,
        Note:         note,
        Kind:         capsule.KindManual,
        HistoryEntry: histEntry,
        RepoSummary:  repoSummary,
        RepoRoot:     repoRoot,
    }
    c := capsule.Build(in)

    dir, err := capsule.DefaultDir(repoRoot)
    if err != nil {
        return fmt.Errorf("resolve capsule dir: %w", err)
    }
    store := capsule.NewStore(dir)
    if err := store.Save(c); err != nil {
        return fmt.Errorf("save capsule: %w", err)
    }

    todoCount := len(c.Work.Todos)
    displayLabel := c.Label
    if displayLabel == "" {
        displayLabel = "[auto]"
    }
    _, _ = fmt.Fprintf(a.stdout, "✓ capsule saved  %s  %s\n  branch: %s  sha: %s  %d todos\n",
        c.ID, displayLabel, c.Repo.Branch, c.Repo.HeadSHA, todoCount)

    if cfg.Memory.PruneOnWrite {
        _ = a.runPrune("", false) // best-effort, ignore error
    }

    return nil
}
```

- [ ] **Step 3: Write integration test for runSave**

Add to `internal/app/system_integration_test.go` (or create a new `capsule_integration_test.go` in the app package if the integration test file does not exist):

```go
// In the existing system_integration_test.go or a new file:
func TestRunSaveCreatesCapsule(t *testing.T) {
    // This test uses the real App wired with temp directories.
    // It verifies that runSave writes capsule.json without error.
    repoRoot := t.TempDir()
    // Initialize a git repo so repoctx works
    mustRun(t, repoRoot, "git", "init")
    mustRun(t, repoRoot, "git", "commit", "--allow-empty", "-m", "init")

    histPath := filepath.Join(t.TempDir(), "history.json")
    app := buildTestApp(t, repoRoot, histPath)

    if err := app.runSave("test label", "test note"); err != nil {
        t.Fatalf("runSave: %v", err)
    }

    capDir := filepath.Join(repoRoot, ".prtr", "capsules")
    entries, err := os.ReadDir(capDir)
    if err != nil {
        t.Fatalf("read capsule dir: %v", err)
    }
    if len(entries) != 1 {
        t.Errorf("expected 1 capsule, got %d", len(entries))
    }
}
```

Note: `buildTestApp` and `mustRun` are helper functions — check if they exist in `system_integration_test.go`. If not, add minimal versions:

```go
func mustRun(t *testing.T, dir, cmd string, args ...string) {
    t.Helper()
    c := exec.Command(cmd, args...)
    c.Dir = dir
    if out, err := c.CombinedOutput(); err != nil {
        t.Fatalf("run %s %v: %v\n%s", cmd, args, err, out)
    }
}
```

- [ ] **Step 4: Run the test**

```bash
go test ./internal/app/... -run TestRunSaveCreates -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/app.go internal/app/
git commit -m "feat(app): implement runSave — creates capsule from current repo state"
```

---

### Task 4.3: `prtr status` implementation

**Files:**
- Modify: `internal/app/app.go`

- [ ] **Step 1: Replace the `runCapsuleStatus` stub**

```go
func (a *App) runCapsuleStatus() error {
    repoRoot, err := a.repoRootFinder()
    if err != nil {
        return fmt.Errorf("find repo root: %w", err)
    }

    dir, err := capsule.DefaultDir(repoRoot)
    if err != nil {
        return fmt.Errorf("resolve capsule dir: %w", err)
    }
    store := capsule.NewStore(dir)

    c, err := store.Latest()
    if err != nil {
        if errors.Is(err, capsule.ErrNotFound) {
            _, _ = fmt.Fprintln(a.stdout, "no capsules saved for this repo")
            return nil
        }
        return fmt.Errorf("load latest capsule: %w", err)
    }

    ctx := context.Background()
    current, _ := a.repoContext.Collect(ctx)

    drift := capsule.DetectDrift(c, current)

    displayLabel := c.Label
    if displayLabel == "" {
        displayLabel = "[auto]"
    }
    _, _ = fmt.Fprintf(a.stdout, "last save:  %s  %s\n",
        c.CreatedAt.Local().Format("2006-01-02 15:04"), displayLabel)

    if drift.BranchChanged {
        _, _ = fmt.Fprintf(a.stdout, "branch:     %s → %s  ⚠ branch changed\n",
            drift.SavedBranch, drift.CurrentBranch)
    } else {
        _, _ = fmt.Fprintf(a.stdout, "branch:     %s  (no drift)\n", c.Repo.Branch)
    }

    if drift.SHAChanged {
        _, _ = fmt.Fprintf(a.stdout, "sha:        %s → %s  ⚠ commits since save\n",
            drift.SavedSHA, drift.CurrentSHA)
    } else {
        _, _ = fmt.Fprintf(a.stdout, "sha:        %s  (no drift)\n", c.Repo.HeadSHA)
    }

    open, done := 0, 0
    for _, t := range c.Work.Todos {
        if t.Status == "completed" {
            done++
        } else {
            open++
        }
    }
    _, _ = fmt.Fprintf(a.stdout, "todos:      %d open, %d done\n", open, done)
    _, _ = fmt.Fprintf(a.stdout, "target:     %s\n", c.Session.TargetApp)

    return nil
}
```

Add `DetectDrift` to `internal/capsule/capsule.go`:

```go
// DetectDrift compares a saved capsule's repo state to the current repoctx summary.
func DetectDrift(c Capsule, current repoctx.Summary) DriftReport {
    return DriftReport{
        BranchChanged: c.Repo.Branch != current.Branch,
        SavedBranch:   c.Repo.Branch,
        CurrentBranch: current.Branch,
        SHAChanged:    c.Repo.HeadSHA != current.HeadSHA,
        SavedSHA:      c.Repo.HeadSHA,
        CurrentSHA:    current.HeadSHA,
    }
}
```

Add the repoctx import to `capsule.go`:
```go
import (
    "time"
    "github.com/helloprtr/poly-prompt/internal/repoctx"
)
```

- [ ] **Step 2: Build and test**

```bash
go build ./...
go test ./internal/capsule/... -v
```

Expected: all PASS

- [ ] **Step 3: Commit**

```bash
git add internal/app/app.go internal/capsule/capsule.go
git commit -m "feat(app): implement runCapsuleStatus with drift detection"
```

---

### Task 4.4: `prtr list` implementation

**Files:**
- Modify: `internal/app/app.go`

- [ ] **Step 1: Replace the `runCapsuleList` stub**

```go
func (a *App) runCapsuleList() error {
    repoRoot, err := a.repoRootFinder()
    if err != nil {
        return fmt.Errorf("find repo root: %w", err)
    }

    dir, err := capsule.DefaultDir(repoRoot)
    if err != nil {
        return fmt.Errorf("resolve capsule dir: %w", err)
    }
    store := capsule.NewStore(dir)

    caps, err := store.List()
    if err != nil {
        return fmt.Errorf("list capsules: %w", err)
    }

    if len(caps) == 0 {
        _, _ = fmt.Fprintln(a.stdout, "no capsules saved for this repo")
        return nil
    }

    for _, c := range caps {
        label := c.Label
        if label == "" {
            label = "[auto]"
        }
        pinMark := ""
        if c.Pinned {
            pinMark = "  📌"
        }
        _, _ = fmt.Fprintf(a.stdout, "%s  %s  %-30s  %-8s  %dt%s\n",
            c.ID,
            c.CreatedAt.Local().Format("2006-01-02 15:04"),
            label,
            c.Session.TargetApp,
            len(c.Work.Todos),
            pinMark,
        )
    }

    return nil
}
```

- [ ] **Step 2: Build and run**

```bash
go build ./...
```

Expected: SUCCESS

- [ ] **Step 3: Commit**

```bash
git add internal/app/app.go
git commit -m "feat(app): implement runCapsuleList"
```

---

### Task 4.5: `prtr resume` implementation

**Files:**
- Modify: `internal/app/app.go`

- [ ] **Step 1: Replace the `runResume` stub**

```go
func (a *App) runResume(ctx context.Context, id, to string, dryRun bool) error {
    repoRoot, err := a.repoRootFinder()
    if err != nil {
        return fmt.Errorf("find repo root: %w", err)
    }

    dir, err := capsule.DefaultDir(repoRoot)
    if err != nil {
        return fmt.Errorf("resolve capsule dir: %w", err)
    }
    store := capsule.NewStore(dir)

    var c capsule.Capsule
    if id == "" {
        c, err = store.Latest()
    } else {
        c, err = store.Load(id)
    }
    if err != nil {
        if errors.Is(err, capsule.ErrNotFound) {
            return fmt.Errorf("no capsules found for this repo — run `prtr save` first")
        }
        return fmt.Errorf("load capsule: %w", err)
    }

    // Detect drift
    current, _ := a.repoContext.Collect(ctx)
    drift := capsule.DetectDrift(c, current)

    // Resolve target
    target := to
    if target == "" {
        target = c.Session.TargetApp
    }
    if target == "" {
        cfg, _ := a.configLoader()
        target = cfg.DefaultTarget
    }
    if target == "" {
        target = "claude"
    }

    // Render resume prompt
    prompt := capsule.RenderResumePrompt(c, target, drift)

    if dryRun {
        _, _ = fmt.Fprintln(a.stdout, prompt)
        return nil
    }

    // Deliver: copy to clipboard + launch target app
    if err := a.clipboard.Copy(ctx, prompt); err != nil {
        return fmt.Errorf("copy to clipboard: %w", err)
    }

    cfg, err := a.configLoader()
    if err != nil {
        return fmt.Errorf("load config: %w", err)
    }
    launcherCfg, hasLauncher := cfg.Launchers[target]
    if hasLauncher && strings.TrimSpace(launcherCfg.Command) != "" && a.launcher != nil {
        if err := a.launcher.Launch(ctx, launcher.Request{
            Command: launcherCfg.Command,
            Args:    launcherCfg.Args,
        }); err != nil {
            return fmt.Errorf("launch %s: %w", target, err)
        }
    }

    displayLabel := c.Label
    if displayLabel == "" {
        displayLabel = "[auto]"
    }
    _, _ = fmt.Fprintf(a.stdout, "✓ resume prompt copied  %s  → %s\n", displayLabel, target)
    if drift.HasDrift() {
        _, _ = fmt.Fprintln(a.stdout, "  ⚠ repo has drifted since save — review the warning in the prompt")
    }

    cfg2, _ := a.configLoader()
    if cfg2.Memory.PruneOnResume {
        _ = a.runPrune("", false)
    }

    return nil
}
```

- [ ] **Step 2: Build and run existing tests**

```bash
go build ./...
go test ./internal/app/... -v
```

Expected: all PASS

- [ ] **Step 3: Commit**

```bash
git add internal/app/app.go
git commit -m "feat(app): implement runResume with drift detection and clipboard delivery"
```

---

### Task 4.6: `prtr prune` implementation

**Files:**
- Modify: `internal/app/app.go`

- [ ] **Step 1: Replace the `runPrune` stub**

```go
func (a *App) runPrune(olderThan string, dryRun bool) error {
    repoRoot, err := a.repoRootFinder()
    if err != nil {
        return fmt.Errorf("find repo root: %w", err)
    }

    dir, err := capsule.DefaultDir(repoRoot)
    if err != nil {
        return fmt.Errorf("resolve capsule dir: %w", err)
    }
    store := capsule.NewStore(dir)

    caps, err := store.List()
    if err != nil {
        return fmt.Errorf("list capsules: %w", err)
    }

    var toDelete []string
    if olderThan != "" {
        d, err := parseDuration(olderThan)
        if err != nil {
            return fmt.Errorf("parse --older-than: %w", err)
        }
        toDelete = capsule.ApplyOlderThan(caps, d)
    } else {
        cfg, err := a.configLoader()
        if err != nil {
            return fmt.Errorf("load config: %w", err)
        }
        toDelete = capsule.ApplyRetentionPolicy(caps, cfg.Memory)
    }

    if len(toDelete) == 0 {
        _, _ = fmt.Fprintln(a.stdout, "nothing to prune")
        return nil
    }

    if dryRun {
        _, _ = fmt.Fprintf(a.stdout, "would delete %d capsule(s):\n", len(toDelete))
        for _, id := range toDelete {
            _, _ = fmt.Fprintf(a.stdout, "  %s\n", id)
        }
        return nil
    }

    for _, id := range toDelete {
        if err := store.Delete(id); err != nil {
            _, _ = fmt.Fprintf(a.stderr, "warning: failed to delete %s: %v\n", id, err)
        }
    }
    _, _ = fmt.Fprintf(a.stdout, "pruned %d capsule(s)\n", len(toDelete))
    return nil
}

// parseDuration parses a duration string like "30d", "14d", "7d".
// Only day units are supported (e.g. "30d").
func parseDuration(s string) (time.Duration, error) {
    s = strings.TrimSpace(s)
    if strings.HasSuffix(s, "d") {
        n := 0
        if _, err := fmt.Sscanf(s[:len(s)-1], "%d", &n); err != nil {
            return 0, fmt.Errorf("invalid day count in %q", s)
        }
        return time.Duration(n) * 24 * time.Hour, nil
    }
    return 0, fmt.Errorf("unsupported duration format %q — use Nd (e.g. 30d)", s)
}
```

- [ ] **Step 2: Build and run all tests**

```bash
go build ./...
go test ./... -v 2>&1 | tail -20
```

Expected: all PASS

- [ ] **Step 3: Commit**

```bash
git add internal/app/app.go
git commit -m "feat(app): implement runPrune with policy and --older-than support"
```

---

## Chunk 5: Auto-save Hooks

---

### Task 5.1: Auto-save helper + dedup logic

**Files:**
- Modify: `internal/app/app.go`

- [ ] **Step 1: Add `tryAutoSave` helper to `app.go`**

This function is called from `runGo`, `runSwap`, `runAgain`, `runTake` success paths. It is best-effort: any error is logged and silently ignored.

```go
// tryAutoSave creates or deduplicates an auto-save capsule after a successful run.
// Errors are non-fatal — auto-save must never block the main flow.
func (a *App) tryAutoSave(ctx context.Context, histEntry *history.Entry) {
    cfg, err := a.configLoader()
    if err != nil || !cfg.Memory.Enabled || !cfg.Memory.AutoSave {
        return
    }

    repoRoot, err := a.repoRootFinder()
    if err != nil {
        return
    }

    repoSummary, err := a.repoContext.Collect(ctx)
    if err != nil {
        return
    }

    dir, err := capsule.DefaultDir(repoRoot)
    if err != nil {
        return
    }
    store := capsule.NewStore(dir)

    in := capsule.BuildInput{
        Kind:         capsule.KindAuto,
        HistoryEntry: histEntry,
        RepoSummary:  repoSummary,
        RepoRoot:     repoRoot,
    }
    c := capsule.Build(in)

    // Dedup: check if we have a recent auto-save with the same key fields.
    if existing := findDedupeTarget(store, c); existing != nil {
        _ = store.Update(existing.ID, func(old *capsule.Capsule) {
            old.Repo = c.Repo
            old.Session = c.Session
            old.Work = c.Work
        })
        return
    }

    _ = store.Save(c)

    if cfg.Memory.PruneOnWrite {
        _ = a.runPrune("", false)
    }
}

// findDedupeTarget returns the existing auto-save capsule to update, or nil
// if no dedup target is found. Dedup condition: same repo + branch +
// normalized_goal + target_app, and created within the last 10 minutes.
func findDedupeTarget(store *capsule.Store, incoming capsule.Capsule) *capsule.Capsule {
    caps, err := store.List()
    if err != nil {
        return nil
    }
    cutoff := time.Now().UTC().Add(-10 * time.Minute)
    for i := range caps {
        c := &caps[i]
        if c.Kind != capsule.KindAuto {
            continue
        }
        if c.Pinned {
            continue // never update a pinned auto-save
        }
        if c.CreatedAt.Before(cutoff) {
            continue
        }
        if c.Repo.Name != incoming.Repo.Name {
            continue
        }
        if c.Repo.Branch != incoming.Repo.Branch {
            continue
        }
        if c.Work.NormalizedGoal != incoming.Work.NormalizedGoal {
            continue
        }
        if c.Session.TargetApp != incoming.Session.TargetApp {
            continue
        }
        return c
    }
    return nil
}
```

- [ ] **Step 2: Build to confirm it compiles**

```bash
go build ./...
```

Expected: SUCCESS

- [ ] **Step 3: Commit**

```bash
git add internal/app/app.go
git commit -m "feat(app): add tryAutoSave helper with dedup logic"
```

---

### Task 5.2: Wire auto-save into runGo, runSwap, runAgain, runTake

**Files:**
- Modify: `internal/app/app.go`

For each of the four functions, add a `tryAutoSave` call at the end of the success path (after `appendHistory`).

- [ ] **Step 1: Find where history is appended in each function**

```bash
grep -n "appendHistory\|historyStore.Append\|saveHistory" /Users/koo/dev/translateCLI-brew/internal/app/app.go | head -20
```

- [ ] **Step 2: Add tryAutoSave after each appendHistory call**

In `executePrompt` (which is called by `runGo`, `runAgain`, `runSwap`, `runShortcut`), find the history append call. After it, add:

```go
// Auto-save capsule after successful run.
// Use context.Background() — the command's ctx may be cancelled before
// the goroutine finishes, which would silently drop the auto-save.
if latestEntry, err := a.historyStore.Latest(); err == nil {
    entry := latestEntry // capture loop variable
    go a.tryAutoSave(context.Background(), &entry)
}
```

Note: wrap in a goroutine so auto-save doesn't add latency to the main command path.

In `runTake` (and `runTakeDeep`), add the same pattern after the history append.

- [ ] **Step 3: Build and run full test suite**

```bash
go build ./...
go test ./... 2>&1 | tail -30
```

Expected: all PASS

- [ ] **Step 4: Commit**

```bash
git add internal/app/app.go
git commit -m "feat(app): wire auto-save into go/swap/again/take success paths"
```

---

## Final Verification

- [ ] **Step 1: Run complete test suite**

```bash
go test ./... -count=1
```

Expected: all PASS, no skipped tests that matter

- [ ] **Step 2: Build release binary**

```bash
go build -o /tmp/prtr-test ./cmd/prtr/
```

Expected: SUCCESS

- [ ] **Step 3: Smoke test the commands**

```bash
/tmp/prtr-test save "test capsule" --note "smoke test"
/tmp/prtr-test status
/tmp/prtr-test list
/tmp/prtr-test resume --dry-run
/tmp/prtr-test prune --dry-run
```

Expected: each command runs without error or panic

- [ ] **Step 4: Final commit**

```bash
git add .
git commit -m "feat: Work Capsule MVP — save/resume/status/list/prune commands"
```
