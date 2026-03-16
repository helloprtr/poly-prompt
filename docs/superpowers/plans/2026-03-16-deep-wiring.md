# Deep Worker Wiring Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire already-collected evidence (git diff, repo changes, protected terms, test files) into the deep workers so `prtr take patch --deep` produces a meaningfully richer prompt than `prtr take patch`.

**Architecture:** All data is already written to the artifact directory by `runtime.go` before workers run. Workers receive a `*State` that holds `Opts` (with `RepoSummary`, `ProtectedTerms`) and `AW` (artifact writer whose `Path()` method resolves any evidence file). Changes are confined to `workers.go`; no new packages or interfaces needed.

**Tech Stack:** Go 1.24, stdlib only (`os`, `strings`, `path/filepath`). No external deps added.

---

## Files

| Action | Path | What changes |
|--------|------|--------------|
| Modify | `internal/deep/worker/workers.go` | patcher, critic, tester, summarize, buildDraftDiff |
| Modify | `internal/deep/worker/graph_test.go` | add wiring assertions to existing stubs |
| Add    | `internal/deep/worker/workers_wiring_test.go` | focused unit tests for the 4 wiring points |

---

## Chunk 1: artifact read helper + patcher git diff wiring

### Task 1: Read helper on artifact.Writer

**Files:**
- Modify: `internal/deep/artifact/artifact.go`

- [ ] **Step 1: Write failing test**

Add to `internal/deep/worker/workers_wiring_test.go` (new file):
```go
package worker

import (
    "os"
    "path/filepath"
    "testing"

    "github.com/helloprtr/poly-prompt/internal/deep/artifact"
)

func TestReadEvidenceFile(t *testing.T) {
    dir := t.TempDir()
    aw := &artifact.Writer{Root: dir}
    _ = aw.WriteText("evidence/git.diff", "diff --git a/foo.go b/foo.go\n+func New() {}")

    content, err := aw.ReadText("evidence/git.diff")
    if err != nil {
        t.Fatalf("ReadText error: %v", err)
    }
    if content == "" {
        t.Fatal("expected non-empty content")
    }
}
```

- [ ] **Step 2: Run test — expect FAIL**
```
go test ./internal/deep/... -run TestReadEvidenceFile
```
Expected: `aw.ReadText undefined`

- [ ] **Step 3: Add ReadText to artifact.Writer**

In `internal/deep/artifact/artifact.go`, add after `WriteText`:
```go
// ReadText reads the content of <root>/<rel>. Returns "" if the file does
// not exist or is empty.
func (w *Writer) ReadText(rel string) (string, error) {
    data, err := os.ReadFile(filepath.Join(w.Root, rel))
    if os.IsNotExist(err) {
        return "", nil
    }
    if err != nil {
        return "", fmt.Errorf("read %s: %w", rel, err)
    }
    return string(data), nil
}
```

- [ ] **Step 4: Run test — expect PASS**
```
go test ./internal/deep/... -run TestReadEvidenceFile
```

- [ ] **Step 5: Commit**
```
git add internal/deep/artifact/artifact.go internal/deep/worker/workers_wiring_test.go
git commit -m "feat(deep/artifact): add ReadText for evidence file access"
```

---

### Task 2: Wire real git diff into patcherWorker

**Files:**
- Modify: `internal/deep/worker/workers.go` — `patcherWorker.Run()`, `buildDraftDiff()`

- [ ] **Step 1: Write failing test**

In `workers_wiring_test.go`, add:
```go
func TestPatcherIncludesRealGitDiff(t *testing.T) {
    dir := t.TempDir()
    for _, sub := range []string{"evidence", "result", "workers/planner", "workers/patcher",
        "workers/critic", "workers/tester", "workers/reconciler"} {
        _ = os.MkdirAll(filepath.Join(dir, sub), 0o755)
    }
    aw := &artifact.Writer{Root: dir}

    realDiff := "diff --git a/main.go b/main.go\n--- a/main.go\n+++ b/main.go\n@@ -1 +1 @@\n-old\n+new"
    _ = aw.WriteText("evidence/git.diff", realDiff)

    plan := makePlan(t, aw)
    s := &State{
        AW:    aw,
        Opts:  deeprun.Options{Source: "fix nil panic in main.go", Action: "patch"},
        Files: []string{"main.go"},
        Plan:  plan,
    }
    w := &patcherWorker{}
    if err := w.Run(context.Background(), s); err != nil {
        t.Fatalf("patcher.Run: %v", err)
    }
    if s.Patch == nil {
        t.Fatal("Patch is nil")
    }
    if !strings.Contains(s.Patch.Diff, "-old") || !strings.Contains(s.Patch.Diff, "+new") {
        t.Errorf("Diff does not contain real git diff content:\n%s", s.Patch.Diff)
    }
}

// helper: write a minimal plan.json and return a WorkPlan
func makePlan(t *testing.T, aw *artifact.Writer) *deepplan.WorkPlan {
    t.Helper()
    plan := &deepplan.WorkPlan{Version: 1, Action: "patch", ResultType: "PatchBundle",
        Summary: "test plan"}
    _ = aw.WriteJSON("plan.json", plan)
    return plan
}
```

Add imports at top of `workers_wiring_test.go`:
```go
import (
    "context"
    "os"
    "path/filepath"
    "strings"
    "testing"

    "github.com/helloprtr/poly-prompt/internal/deep/artifact"
    deepplan "github.com/helloprtr/poly-prompt/internal/deep/plan"
    deeprun "github.com/helloprtr/poly-prompt/internal/deep/run"
)
```

- [ ] **Step 2: Run test — expect FAIL**
```
go test ./internal/deep/worker/... -run TestPatcherIncludesRealGitDiff
```
Expected: FAIL — diff is placeholder, not real content.

- [ ] **Step 3: Update patcherWorker.Run() to read git.diff**

In `workers.go`, replace `buildDraftDiff(s.Files)` call in `patcherWorker.Run()`:
```go
// Read real git diff from evidence; fall back to placeholder if absent/empty.
realDiff, _ := s.AW.ReadText("evidence/git.diff")
diff := buildDraftDiff(s.Files, strings.TrimSpace(realDiff))
```

Update `buildDraftDiff` signature and body:
```go
// buildDraftDiff returns the git diff to include in the patch draft.
// If realDiff is non-empty it is used (truncated to 120 lines).
// Otherwise a per-file placeholder is generated.
func buildDraftDiff(files []string, realDiff string) string {
    if strings.TrimSpace(realDiff) != "" {
        lines := strings.Split(realDiff, "\n")
        const maxLines = 120
        if len(lines) > maxLines {
            lines = append(lines[:maxLines], "... (truncated)")
        }
        return strings.Join(lines, "\n")
    }
    // Fallback: placeholder per file
    if len(files) == 0 {
        return "diff --git a/<inspect-changed-files> b/<inspect-changed-files>\n" +
            "--- a/<inspect-changed-files>\n+++ b/<inspect-changed-files>\n@@\n" +
            "- review the existing implementation\n+ apply the planned fix and add regression coverage"
    }
    parts := make([]string, 0, len(files))
    for _, file := range files {
        parts = append(parts, fmt.Sprintf(
            "diff --git a/%s b/%s\n--- a/%s\n+++ b/%s\n@@\n- inspect the current implementation\n+ apply the change described by the patch bundle",
            file, file, file, file,
        ))
    }
    return strings.Join(parts, "\n\n")
}
```

- [ ] **Step 4: Run test — expect PASS**
```
go test ./internal/deep/worker/... -run TestPatcherIncludesRealGitDiff
```

- [ ] **Step 5: Run all deep tests to confirm no regression**
```
go test ./internal/deep/...
```

- [ ] **Step 6: Commit**
```
git add internal/deep/worker/workers.go internal/deep/worker/workers_wiring_test.go
git commit -m "feat(deep/patcher): wire real git diff into patch draft"
```

---

## Chunk 2: repo changes list + protected terms wiring

### Task 3: Wire actual Changes list into patcherWorker

**Files:**
- Modify: `internal/deep/worker/workers.go` — `patcherWorker.Run()`

- [ ] **Step 1: Write failing test**

In `workers_wiring_test.go`, add:
```go
func TestPatcherIncludesChangedFilesList(t *testing.T) {
    dir := t.TempDir()
    for _, sub := range []string{"evidence", "result", "workers/planner", "workers/patcher",
        "workers/critic", "workers/tester", "workers/reconciler"} {
        _ = os.MkdirAll(filepath.Join(dir, sub), 0o755)
    }
    aw := &artifact.Writer{Root: dir}
    _ = aw.WriteText("evidence/git.diff", "")

    plan := makePlan(t, aw)
    s := &State{
        AW: aw,
        Opts: deeprun.Options{
            Source: "fix bug",
            Action: "patch",
            RepoSummary: repoctx.Summary{
                Branch:  "fix/nil-panic",
                Changes: []string{"M internal/foo/foo.go", "M internal/foo/foo_test.go"},
            },
        },
        Files: []string{"internal/foo/foo.go"},
        Plan:  plan,
    }
    w := &patcherWorker{}
    if err := w.Run(context.Background(), s); err != nil {
        t.Fatalf("patcher.Run: %v", err)
    }
    found := false
    for _, c := range s.Patch.Constraints {
        if strings.Contains(c, "internal/foo/foo.go") && strings.Contains(c, "internal/foo/foo_test.go") {
            found = true
        }
    }
    if !found {
        t.Errorf("Constraints do not include actual changed files list:\n%v", s.Patch.Constraints)
    }
}
```

Add import `repoctx "github.com/helloprtr/poly-prompt/internal/repoctx"` to the test file imports.

- [ ] **Step 2: Run test — expect FAIL**
```
go test ./internal/deep/worker/... -run TestPatcherIncludesChangedFilesList
```

- [ ] **Step 3: Update patcherWorker.Run() to include Changes list**

In `workers.go`, replace the `RepoSummary.Changes` block in `patcherWorker.Run()`:
```go
if len(s.Opts.RepoSummary.Changes) > 0 {
    // Include actual file list, not just a generic reminder.
    constraints = append(constraints,
        "Local changes in this checkout:\n  "+
            strings.Join(s.Opts.RepoSummary.Changes, "\n  "))
}
```

- [ ] **Step 4: Run test — expect PASS**
```
go test ./internal/deep/worker/... -run TestPatcherIncludesChangedFilesList
```

---

### Task 4: Wire protected terms list into patcherWorker

**Files:**
- Modify: `internal/deep/worker/workers.go` — `patcherWorker.Run()`

- [ ] **Step 1: Write failing test**

In `workers_wiring_test.go`, add:
```go
func TestPatcherIncludesProtectedTerms(t *testing.T) {
    dir := t.TempDir()
    for _, sub := range []string{"evidence", "result", "workers/planner", "workers/patcher",
        "workers/critic", "workers/tester", "workers/reconciler"} {
        _ = os.MkdirAll(filepath.Join(dir, sub), 0o755)
    }
    aw := &artifact.Writer{Root: dir}
    _ = aw.WriteText("evidence/git.diff", "")

    plan := makePlan(t, aw)
    s := &State{
        AW: aw,
        Opts: deeprun.Options{
            Source:         "fix bug",
            Action:         "patch",
            ProtectedTerms: []string{"ApplyPolicy", "DecisionSkipped", "PRTR_TARGET"},
        },
        Files: []string{"foo.go"},
        Plan:  plan,
    }
    w := &patcherWorker{}
    if err := w.Run(context.Background(), s); err != nil {
        t.Fatalf("patcher.Run: %v", err)
    }
    found := false
    for _, n := range s.Patch.ImplementationNotes {
        if strings.Contains(n, "ApplyPolicy") && strings.Contains(n, "PRTR_TARGET") {
            found = true
        }
    }
    if !found {
        t.Errorf("ImplementationNotes do not list protected terms:\n%v", s.Patch.ImplementationNotes)
    }
}
```

- [ ] **Step 2: Run test — expect FAIL**
```
go test ./internal/deep/worker/... -run TestPatcherIncludesProtectedTerms
```

- [ ] **Step 3: Update patcherWorker.Run() to include actual term list**

Replace the `ProtectedTerms` block in `patcherWorker.Run()`:
```go
if len(s.Opts.ProtectedTerms) > 0 {
    notes = append(notes,
        "Preserve these repo-specific identifiers exactly (do not rename or translate): "+
            strings.Join(s.Opts.ProtectedTerms, ", ")+".")
}
```

- [ ] **Step 4: Run test — expect PASS**
```
go test ./internal/deep/worker/... -run TestPatcherIncludesProtectedTerms
```

- [ ] **Step 5: Run all deep tests**
```
go test ./internal/deep/...
```

- [ ] **Step 6: Commit**
```
git add internal/deep/worker/workers.go internal/deep/worker/workers_wiring_test.go
git commit -m "feat(deep/patcher): wire actual changes list and protected terms into prompt"
```

---

## Chunk 3: test file detection in testerWorker

### Task 5: Detect existing *_test.go files for touched files

**Files:**
- Modify: `internal/deep/worker/workers.go` — `testerWorker.Run()`

- [ ] **Step 1: Write failing test**

In `workers_wiring_test.go`, add:
```go
func TestTesterReferencesExistingTestFile(t *testing.T) {
    dir := t.TempDir()
    for _, sub := range []string{"evidence", "result", "workers/planner", "workers/patcher",
        "workers/critic", "workers/tester", "workers/reconciler"} {
        _ = os.MkdirAll(filepath.Join(dir, sub), 0o755)
    }
    aw := &artifact.Writer{Root: dir}

    // Simulate repo root with a test file present
    repoRoot := t.TempDir()
    srcDir := filepath.Join(repoRoot, "internal", "translate")
    _ = os.MkdirAll(srcDir, 0o755)
    _ = os.WriteFile(filepath.Join(srcDir, "policy_test.go"), []byte("package translate"), 0o644)

    patch := &deepschema.PatchDraft{
        Summary:      "fix nil",
        TouchedFiles: []string{"internal/translate/policy.go"},
        Diff:         "",
    }
    s := &State{
        AW:    aw,
        Opts:  deeprun.Options{Source: "fix nil panic", Action: "patch", RepoRoot: repoRoot},
        Files: []string{"internal/translate/policy.go"},
        Patch: patch,
    }
    w := &testerWorker{}
    if err := w.Run(context.Background(), s); err != nil {
        t.Fatalf("tester.Run: %v", err)
    }
    found := false
    for _, c := range s.Tests.TestCases {
        if strings.Contains(c, "policy_test.go") {
            found = true
        }
    }
    if !found {
        t.Errorf("TestCases do not reference existing test file:\n%v", s.Tests.TestCases)
    }
}
```

Add import `deepschema "github.com/helloprtr/poly-prompt/internal/deep/schema"` to test imports.

- [ ] **Step 2: Run test — expect FAIL**
```
go test ./internal/deep/worker/... -run TestTesterReferencesExistingTestFile
```

- [ ] **Step 3: Add findTestFile helper and update testerWorker.Run()**

In `workers.go`, add helper before `formatTestPlanMD`:
```go
// findTestFile returns the conventional *_test.go path for srcFile if it
// exists under repoRoot. Returns "" if not found or repoRoot is empty.
func findTestFile(repoRoot, srcFile string) string {
    if strings.TrimSpace(repoRoot) == "" {
        return ""
    }
    // Strip extension and append _test.go
    ext := filepath.Ext(srcFile)
    base := strings.TrimSuffix(srcFile, ext)
    candidate := filepath.Join(repoRoot, base+"_test"+ext)
    if _, err := os.Stat(candidate); err == nil {
        return base + "_test" + ext
    }
    return ""
}
```

In `testerWorker.Run()`, replace the first-file block:
```go
if len(s.Files) > 0 && s.Files[0] != "<inspect-changed-files>" {
    primary := s.Files[0]
    testFile := findTestFile(s.Opts.RepoRoot, primary)
    if testFile != "" {
        cases = append(cases,
            fmt.Sprintf("Add a regression test in %s that covers the primary failure path.", testFile))
    } else {
        cases = append(cases,
            fmt.Sprintf("Cover the behavior around %s with the narrowest useful scope.", primary))
    }
}
```

Add `"os"` and `"path/filepath"` to the import block in `workers.go` if not already present.

- [ ] **Step 4: Run test — expect PASS**
```
go test ./internal/deep/worker/... -run TestTesterReferencesExistingTestFile
```

- [ ] **Step 5: Run all deep tests**
```
go test ./internal/deep/...
```

- [ ] **Step 6: Commit**
```
git add internal/deep/worker/workers.go internal/deep/worker/workers_wiring_test.go
git commit -m "feat(deep/tester): detect existing test files and reference them in test plan"
```

---

## Chunk 4: summary improvement + final verification

### Task 6: Better summarize() — extract intent, not just truncate

**Files:**
- Modify: `internal/deep/worker/workers.go` — `summarize()`

- [ ] **Step 1: Write failing test**

In `workers_wiring_test.go`, add:
```go
func TestSummarizeExtractsFirstMeaningfulSentence(t *testing.T) {
    cases := []struct {
        input string
        want  string // must be contained in output
    }{
        {
            input: "ApplyPolicy에서 nil translator일 때 panic 발생.\nprotectSegments 호출 전에 nil 체크가 필요합니다.",
            want:  "ApplyPolicy",
        },
        {
            input: "\n\n  \n실제 내용은 여기서 시작합니다. 파일을 수정해야 합니다.",
            want:  "실제 내용은 여기서 시작합니다",
        },
        {
            input: "",
            want:  "Convert the source answer",
        },
    }
    for _, tc := range cases {
        got := summarize(tc.input)
        if !strings.Contains(got, tc.want) {
            t.Errorf("summarize(%q) = %q, want it to contain %q", tc.input, got, tc.want)
        }
    }
}
```

- [ ] **Step 2: Run test — expect PASS** (current truncation already passes these; baseline check)
```
go test ./internal/deep/worker/... -run TestSummarizeExtractsFirstMeaningfulSentence
```

- [ ] **Step 3: Improve summarize() to extract first non-empty meaningful line**

Replace `summarize()` in `workers.go`:
```go
// summarize extracts the first meaningful sentence or line from text.
// It strips blank lines, picks the first non-empty line as the lead,
// then appends a second line if present and the combined length is ≤220 runes.
func summarize(text string) string {
    text = strings.TrimSpace(text)
    if text == "" {
        return "Convert the source answer into a concrete implementation follow-up."
    }
    lines := strings.Split(text, "\n")
    var nonEmpty []string
    for _, l := range lines {
        if t := strings.TrimSpace(l); t != "" {
            nonEmpty = append(nonEmpty, t)
        }
    }
    if len(nonEmpty) == 0 {
        return "Convert the source answer into a concrete implementation follow-up."
    }
    result := nonEmpty[0]
    if len(nonEmpty) > 1 {
        combined := result + " " + nonEmpty[1]
        if len([]rune(combined)) <= 220 {
            result = combined
        }
    }
    runes := []rune(result)
    if len(runes) > 220 {
        return string(runes[:217]) + "..."
    }
    return result
}
```

- [ ] **Step 4: Run test — expect PASS**
```
go test ./internal/deep/worker/... -run TestSummarizeExtractsFirstMeaningfulSentence
```

- [ ] **Step 5: Full test suite**
```
go test ./internal/deep/... ./internal/app/...
```
All must pass.

- [ ] **Step 6: Commit**
```
git add internal/deep/worker/workers.go internal/deep/worker/workers_wiring_test.go
git commit -m "feat(deep/patcher): improve summarize to extract first meaningful sentence"
```

---

## Final: sync to codex/take-deep-runtime and verify

- [ ] Confirm all 4 wiring tests pass:
```
go test ./internal/deep/worker/... -v -run "TestReadEvidenceFile|TestPatcherIncludesRealGitDiff|TestPatcherIncludesChangedFilesList|TestPatcherIncludesProtectedTerms|TestTesterReferencesExistingTestFile|TestSummarizeExtractsFirstMeaningfulSentence"
```

- [ ] Run integration tests:
```
go test ./internal/deep/... -v 2>&1 | tail -20
```

- [ ] Build check:
```
go build ./...
```
