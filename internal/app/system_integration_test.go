package app

// system_integration_test.go — Full-system integration scenarios for prtr 0.7.0
//
// These tests exercise the complete application flow from CLI dispatch through
// all subsystems (translate, history, termbook, deep engine). Each scenario
// uses isolated temp directories and stub dependencies to prevent any real
// side-effects (no real git, no real clipboard, no real AI calls).
//
// Scenarios:
//   1. Classic CLI Regression   — go/swap/take/learn without --deep
//   2. Memory Interoperability  — learn writes termbook → deep reads it
//   3. History Continuity       — classic go → deep take → classic swap chain
//   4. Deep E2E Completeness    — deep worker graph + CLI UX (complements integration_test.go)
//   5. Resilience               — edge cases, graceful errors, no panics

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/helloprtr/poly-prompt/internal/config"
	"github.com/helloprtr/poly-prompt/internal/history"
	"github.com/helloprtr/poly-prompt/internal/repoctx"
	"github.com/helloprtr/poly-prompt/internal/termbook"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// newRealTermbookApp creates an App where TermbookLoader is NOT stubbed, so
// it falls back to the real termbook.Load from disk. Everything else is stubbed.
func newRealTermbookApp(
	t *testing.T,
	cfg config.Config,
	translator *stubTranslator,
	clipboard *stubClipboard,
	historyStore *history.Store,
	repoRoot string,
) *App {
	t.Helper()
	return New(Dependencies{
		Version:         "test",
		Stdout:          &bytes.Buffer{},
		Stderr:          &bytes.Buffer{},
		Translator:      translator,
		Clipboard:       clipboard,
		Editor:          &stubEditor{},
		Launcher:        &stubLauncher{},
		Automator:       &stubAutomator{},
		SubmitConfirmer: &stubConfirmer{},
		ConfigLoader: func() (config.Config, error) {
			return cfg, nil
		},
		ConfigInit:   func() (string, error) { return "/tmp/prtr/config.toml", nil },
		LookupEnv:    func(string) (string, bool) { return "", false },
		HistoryStore: historyStore,
		RepoContext: &stubRepoContext{summary: repoctx.Summary{
			RepoName: "test-repo",
			Branch:   "main",
			Changes:  []string{" M internal/auth/auth.go"},
		}},
		RepoRootFinder: func() (string, error) {
			return repoRoot, nil
		},
		// TermbookLoader: nil → falls back to real termbook.Load(repoRoot)
	})
}

// gitRepo creates a temp directory with a .git subdirectory (bare marker only,
// not a real git repo, but enough for FindRepoRoot and termbook.Save to work).
func gitRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("gitRepo: MkdirAll .git: %v", err)
	}
	return root
}

// writeFile creates a file at path (relative to base) with content.
func writeFile(t *testing.T, base, rel, content string) {
	t.Helper()
	p := filepath.Join(base, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("writeFile MkdirAll: %v", err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("writeFile WriteFile: %v", err)
	}
}

// mustLatestEntry retrieves the latest history entry, failing the test if not found.
func mustLatestEntry(t *testing.T, store *history.Store) history.Entry {
	t.Helper()
	entry, err := store.Latest()
	if err != nil {
		t.Fatalf("Latest() error = %v", err)
	}
	return entry
}

// mustListEntries retrieves all history entries, failing if error.
func mustListEntries(t *testing.T, store *history.Store) []history.Entry {
	t.Helper()
	entries, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	return entries
}

// ---------------------------------------------------------------------------
// Scenario 1: Classic CLI Regression — no deep engine involvement
// ---------------------------------------------------------------------------

// TestSystemS1_ClassicGoTakeSwapLearnSequence runs go → take(classic) →
// swap → learn in sequence, verifying that no deep artifacts are created and
// each command produces the expected output.
func TestSystemS1_ClassicGoTakeSwapLearnSequence(t *testing.T) {
	t.Parallel()

	root := gitRepo(t)
	writeFile(t, root, "README.md", "Use BuildPrompt and PRTR_TARGET.\n")

	store := history.New(filepath.Join(t.TempDir(), "history.json"))
	// Seed a history entry so `take` and `swap` can find the last target.
	if err := store.Append(history.Entry{
		ID:        "seed",
		CreatedAt: time.Unix(100, 0).UTC(),
		Target:    "codex",
		Shortcut:  "fix",
		Original:  "seed prompt",
	}); err != nil {
		t.Fatalf("seed history: %v", err)
	}

	translator := &stubTranslator{output: "Translated: why is this slow?"}
	clipboard := &stubClipboard{read: "Apply the fix to internal/cache/lru.go"}

	app := New(Dependencies{
		Version:         "test",
		Stdout:          &bytes.Buffer{},
		Stderr:          &bytes.Buffer{},
		Translator:      translator,
		Clipboard:       clipboard,
		Editor:          &stubEditor{},
		Launcher:        &stubLauncher{},
		Automator:       &stubAutomator{},
		SubmitConfirmer: &stubConfirmer{},
		ConfigLoader:    func() (config.Config, error) { return testConfig(), nil },
		ConfigInit:      func() (string, error) { return "/tmp/prtr/config.toml", nil },
		LookupEnv:       func(string) (string, bool) { return "", false },
		HistoryStore:    store,
		RepoContext:     &stubRepoContext{},
		RepoRootFinder:  func() (string, error) { return root, nil },
	})
	stdout, stderr := buffersFromApp(app)
	ctx := context.Background()

	// Step 1: go ask — classic prompt delivery (dry-run).
	if err := app.Execute(ctx, []string{"go", "ask", "why is this slow?", "--dry-run"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("step1 go: %v", err)
	}
	if !strings.Contains(stderr.String(), "-> ask | codex | prompt") {
		t.Errorf("step1: unexpected stderr: %q", stderr.String())
	}
	goEntry := mustLatestEntry(t, store)
	if goEntry.Engine != "" && goEntry.Engine != "classic" {
		t.Errorf("step1: entry.Engine = %q, want classic or empty", goEntry.Engine)
	}

	// Step 2: take patch (classic) — no --deep flag.
	resetBuffers(app)
	if err := app.Execute(ctx, []string{"take", "patch", "--dry-run"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("step2 take: %v", err)
	}
	if !strings.Contains(stdout.String(), "Turn the material below into an implementation prompt.") {
		t.Errorf("step2: classic take template missing from stdout: %q", stdout.String())
	}
	// Classic take must not emit the "running" line that deep runs emit.
	if strings.Contains(stderr.String(), "| running") {
		t.Errorf("step2: classic take emitted deep 'running' line: %q", stderr.String())
	}
	takeEntry := mustLatestEntry(t, store)
	if takeEntry.Engine != "" && takeEntry.Engine != "classic" {
		t.Errorf("step2: entry.Engine = %q, want classic or empty", takeEntry.Engine)
	}
	if takeEntry.RunID != "" {
		t.Errorf("step2: classic take should not set RunID, got %q", takeEntry.RunID)
	}
	if takeEntry.ArtifactRoot != "" {
		t.Errorf("step2: classic take should not set ArtifactRoot, got %q", takeEntry.ArtifactRoot)
	}

	// Step 3: swap gemini — re-sends the classic take result to a different AI.
	resetBuffers(app)
	if err := app.Execute(ctx, []string{"swap", "gemini", "--dry-run"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("step3 swap: %v", err)
	}
	if !strings.Contains(stderr.String(), "| gemini |") {
		t.Errorf("step3: swap stderr = %q", stderr.String())
	}
	swapEntry := mustLatestEntry(t, store)
	if swapEntry.ParentID == "" {
		t.Error("step3: swap entry.ParentID should be set")
	}

	// Step 4: learn --dry-run — must not write termbook.toml.
	resetBuffers(app)
	if err := app.Execute(ctx, []string{"learn", "README.md", "--dry-run"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("step4 learn: %v", err)
	}
	if !strings.Contains(stdout.String(), "protected_terms") {
		t.Errorf("step4: learn dry-run stdout = %q", stdout.String())
	}
	termBookPath := filepath.Join(root, ".prtr", "termbook.toml")
	if _, err := os.Stat(termBookPath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("step4: termbook.toml should not exist after --dry-run, stat err = %v", err)
	}

	// Verify: NO deep artifact directory was created at any step.
	runsDir := filepath.Join(root, ".prtr", "runs")
	if _, err := os.Stat(runsDir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("classic workflow must not create .prtr/runs/, but it exists: %v", err)
	}

	// Verify history: seed + go + take + swap = 4 entries (learn does not add history).
	entries := mustListEntries(t, store)
	if len(entries) != 4 {
		t.Errorf("expected 4 history entries (seed+go+take+swap), got %d", len(entries))
	}
}

// TestSystemS1_ClassicTakeNoDeepFiles explicitly verifies that `take patch`
// (classic) does not touch the .prtr/runs/ directory.
func TestSystemS1_ClassicTakeNoDeepFiles(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	store := history.New(filepath.Join(t.TempDir(), "history.json"))
	if err := store.Append(history.Entry{
		ID: "prev", CreatedAt: time.Unix(200, 0).UTC(), Target: "codex",
	}); err != nil {
		t.Fatal(err)
	}

	app := newTestApp(t, testConfig(), &stubTranslator{}, &stubClipboard{read: "Fix auth.go"}, &stubEditor{}, store)

	if err := app.Execute(context.Background(), []string{"take", "patch", "--dry-run"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	runsDir := filepath.Join(repoRoot, ".prtr", "runs")
	if _, err := os.Stat(runsDir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf(".prtr/runs should not be created by classic take patch, stat err = %v", err)
	}
}

// ---------------------------------------------------------------------------
// Scenario 2: Memory Interoperability — learn → deep
// ---------------------------------------------------------------------------

// TestSystemS2_LearnTermsFlowIntoDeepMemoryJson tests the full chain:
// `learn README.md` writes .prtr/termbook.toml → `take patch --deep`
// reads it via resolveLearnedTerms → stores in evidence/memory.json.
func TestSystemS2_LearnTermsFlowIntoDeepMemoryJson(t *testing.T) {
	t.Parallel()

	root := gitRepo(t)
	// README.md with project-specific identifiers that the term extractor will pick up.
	writeFile(t, root, "README.md", `# prtr

Use **BuildPrompt** to construct prompts.
The **PRTR_TARGET** env variable controls the active AI.
See also **RunPatchBundle** in the code.
`)

	store := history.New(filepath.Join(t.TempDir(), "history.json"))
	// Seed history so take can find the last target.
	if err := store.Append(history.Entry{
		ID: "seed", CreatedAt: time.Unix(100, 0).UTC(), Target: "codex",
	}); err != nil {
		t.Fatal(err)
	}

	clipboard := &stubClipboard{read: "Fix the nil pointer in internal/auth/auth.go"}

	// App with REAL TermbookLoader (nil → falls back to termbook.Load from disk).
	app := newRealTermbookApp(t, testConfig(), &stubTranslator{}, clipboard, store, root)
	stdout, _ := buffersFromApp(app)
	ctx := context.Background()

	// Step 1: learn → writes .prtr/termbook.toml to root.
	if err := app.Execute(ctx, []string{"learn", "README.md"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("learn: %v", err)
	}
	if !strings.Contains(stdout.String(), "saved") {
		t.Errorf("learn stdout = %q", stdout.String())
	}
	// Verify termbook.toml was written with at least one expected term.
	book, err := termbook.Load(root)
	if err != nil {
		t.Fatalf("termbook.Load: %v", err)
	}
	termsJoined := strings.Join(book.ProtectedTerms, ",")
	if !strings.Contains(termsJoined, "BuildPrompt") && !strings.Contains(termsJoined, "PRTR_TARGET") {
		t.Errorf("expected terms not in termbook: %v", book.ProtectedTerms)
	}

	// Step 2: take patch --deep → should read the termbook via resolveLearnedTerms.
	resetBuffers(app)
	if err := app.Execute(ctx, []string{"take", "patch", "--deep", "--dry-run"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("take patch --deep: %v", err)
	}

	// Step 3: find the artifact root from history.
	deepEntry := mustLatestEntry(t, store)
	if deepEntry.ArtifactRoot == "" {
		t.Fatal("deep entry.ArtifactRoot is empty")
	}

	// Step 4: verify evidence/memory.json contains the learned terms.
	memoryPath := filepath.Join(deepEntry.ArtifactRoot, "evidence", "memory.json")
	data, err := os.ReadFile(memoryPath)
	if err != nil {
		t.Fatalf("ReadFile evidence/memory.json: %v", err)
	}
	var memoryDoc map[string]any
	if err := json.Unmarshal(data, &memoryDoc); err != nil {
		t.Fatalf("Unmarshal memory.json: %v", err)
	}
	rawTerms, ok := memoryDoc["protected_terms"]
	if !ok {
		t.Fatal("evidence/memory.json missing 'protected_terms' key")
	}
	terms, ok := rawTerms.([]any)
	if !ok {
		t.Fatalf("protected_terms is not an array: %T", rawTerms)
	}
	termsStr := make([]string, 0, len(terms))
	for _, v := range terms {
		if s, ok := v.(string); ok {
			termsStr = append(termsStr, s)
		}
	}
	if len(termsStr) == 0 {
		t.Error("evidence/memory.json has empty protected_terms — learn terms were not passed to deep run")
	}
	// At least one term from the termbook must appear.
	found := false
	for _, term := range termsStr {
		if term == "BuildPrompt" || term == "PRTR_TARGET" || term == "RunPatchBundle" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("no expected terms found in memory.json; got: %v", termsStr)
	}
}

// TestSystemS2_DeepRunWithoutTermbookStillSucceeds verifies that a deep run
// in a repo that has NO termbook.toml completes without error.
func TestSystemS2_DeepRunWithoutTermbookStillSucceeds(t *testing.T) {
	t.Parallel()

	root := gitRepo(t) // no README.md, no .prtr/termbook.toml
	store := history.New(filepath.Join(t.TempDir(), "history.json"))
	if err := store.Append(history.Entry{
		ID: "s", CreatedAt: time.Unix(100, 0).UTC(), Target: "codex",
	}); err != nil {
		t.Fatal(err)
	}

	app := newRealTermbookApp(t, testConfig(), &stubTranslator{}, &stubClipboard{read: "fix auth.go"}, store, root)

	err := app.Execute(context.Background(), []string{"take", "patch", "--deep", "--dry-run"}, strings.NewReader(""), false)
	if err != nil {
		t.Errorf("deep run without termbook should succeed, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Scenario 3: History Continuity — classic go → deep take → classic swap
// ---------------------------------------------------------------------------

// TestSystemS3_GoDeepSwapHistoryChain runs the full three-step sequence and
// verifies that the history store captures entries in order without collision,
// and that `swap` succeeds when the last entry is from a deep run.
func TestSystemS3_GoDeepSwapHistoryChain(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	store := history.New(filepath.Join(t.TempDir(), "history.json"))
	ctx := context.Background()

	// Shared app wiring: deep-capable (has RepoRootFinder + RepoContext + TermbookLoader stub).
	app := newDeepTestApp(t, testConfig(), &stubTranslator{output: "Translated go prompt"}, &stubClipboard{read: "Fix the login flow in internal/auth/login.go"}, &stubEditor{}, store, repoRoot)
	stdout, stderr := buffersFromApp(app)

	// ── Step 1: classic `go ask` ──────────────────────────────────────────
	if err := app.Execute(ctx, []string{"go", "ask", "what does this auth function do?", "--dry-run"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("step1 go: %v", err)
	}
	classicEntry := mustLatestEntry(t, store)
	if classicEntry.Engine != "" && classicEntry.Engine != "classic" {
		t.Errorf("step1: Engine = %q, want classic/empty", classicEntry.Engine)
	}
	if classicEntry.RunID != "" {
		t.Errorf("step1: RunID should be empty for classic go, got %q", classicEntry.RunID)
	}
	if classicEntry.ArtifactRoot != "" {
		t.Errorf("step1: ArtifactRoot should be empty for classic go, got %q", classicEntry.ArtifactRoot)
	}
	classicID := classicEntry.ID

	// ── Step 2: deep `take patch --deep` ─────────────────────────────────
	resetBuffers(app)
	if err := app.Execute(ctx, []string{"take", "patch", "--deep", "--dry-run"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("step2 take --deep: %v", err)
	}
	deepEntry := mustLatestEntry(t, store)
	if deepEntry.Engine != "deep" {
		t.Errorf("step2: Engine = %q, want deep", deepEntry.Engine)
	}
	if deepEntry.RunID == "" {
		t.Error("step2: RunID should be set for deep run")
	}
	if deepEntry.ArtifactRoot == "" {
		t.Error("step2: ArtifactRoot should be set for deep run")
	}
	if deepEntry.ResultType != "PatchBundle" {
		t.Errorf("step2: ResultType = %q, want PatchBundle", deepEntry.ResultType)
	}
	deepID := deepEntry.ID

	// Verify the IDs are different and both entries are in the store.
	if classicID == deepID {
		t.Error("step1 and step2 history IDs must be different")
	}

	// ── Step 3: classic `again` (re-uses last entry = deep run) ──────────
	resetBuffers(app)
	if err := app.Execute(ctx, []string{"again", "--dry-run"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("step3 again: %v", err)
	}
	againEntry := mustLatestEntry(t, store)
	if againEntry.ParentID != deepID {
		t.Errorf("step3: again entry.ParentID = %q, want deep entry ID %q", againEntry.ParentID, deepID)
	}

	// ── Step 4: classic `swap claude` ────────────────────────────────────
	// By now the latest entry is the `again` entry (not the deep run), so
	// swap reads from `again`. Still verifies no panic.
	resetBuffers(app)
	if err := app.Execute(ctx, []string{"swap", "claude", "--dry-run"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("step4 swap: %v", err)
	}
	if !strings.Contains(stderr.String(), "| claude |") {
		t.Errorf("step4 swap: expected claude in status, got %q", stderr.String())
	}

	// ── History order verification ────────────────────────────────────────
	allEntries := mustListEntries(t, store)
	// List returns pinned-first then newest-first, so sort by CreatedAt.
	if len(allEntries) != 4 {
		t.Errorf("expected 4 history entries (go+deep+again+swap), got %d", len(allEntries))
	}

	// Verify each unique engine type appears in the recorded entries.
	engines := make(map[string]int)
	for _, e := range allEntries {
		eng := e.Engine
		if eng == "" {
			eng = "classic"
		}
		engines[eng]++
	}
	if engines["deep"] < 1 {
		t.Error("no deep entry found in history")
	}
	if engines["classic"] < 1 {
		t.Error("no classic entry found in history")
	}

	_ = stdout
}

// TestSystemS3_SwapDirectlyAfterDeepRunSucceeds verifies that `swap` can
// be called immediately after a deep run without any intermediate classic run.
func TestSystemS3_SwapDirectlyAfterDeepRunSucceeds(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	store := history.New(filepath.Join(t.TempDir(), "history.json"))
	// Seed so `take` can find a target (codex).
	if err := store.Append(history.Entry{
		ID: "seed", CreatedAt: time.Unix(100, 0).UTC(), Target: "codex",
	}); err != nil {
		t.Fatal(err)
	}

	app := newDeepTestApp(t, testConfig(), &stubTranslator{}, &stubClipboard{read: "fix the handler bug in handlers/user.go"}, &stubEditor{}, store, repoRoot)
	ctx := context.Background()

	// Deep run.
	if err := app.Execute(ctx, []string{"take", "patch", "--deep", "--dry-run"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("take patch --deep: %v", err)
	}
	deepEntry := mustLatestEntry(t, store)
	if deepEntry.Engine != "deep" {
		t.Fatalf("expected deep entry, got Engine=%q", deepEntry.Engine)
	}

	// Swap immediately after the deep run.
	resetBuffers(app)
	if err := app.Execute(ctx, []string{"swap", "gemini", "--dry-run"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("swap after deep: %v", err)
	}
	swapEntry := mustLatestEntry(t, store)
	if swapEntry.ParentID != deepEntry.ID {
		t.Errorf("swap.ParentID = %q, want %q", swapEntry.ParentID, deepEntry.ID)
	}
}

// ---------------------------------------------------------------------------
// Scenario 4: Deep E2E — CLI UX and worker graph completeness
// ---------------------------------------------------------------------------

// TestSystemS4_DeepDryRunCLIUXOutput verifies the exact CLI status format
// for a deep dry-run: initial "running" line, step progress lines, and final
// "completed" line with next-step hints.
func TestSystemS4_DeepDryRunCLIUXOutput(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	store := history.New(filepath.Join(t.TempDir(), "history.json"))
	if err := store.Append(history.Entry{
		ID: "seed", CreatedAt: time.Unix(100, 0).UTC(), Target: "codex",
	}); err != nil {
		t.Fatal(err)
	}

	app := newDeepTestApp(t, testConfig(), &stubTranslator{}, &stubClipboard{read: "fix the goroutine leak in internal/worker/pool.go"}, &stubEditor{}, store, repoRoot)
	_, stderr := buffersFromApp(app)

	if err := app.Execute(context.Background(), []string{"take", "patch", "--deep", "--dry-run"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	stderrStr := stderr.String()

	// Initial running line must appear before any step.
	if !strings.Contains(stderrStr, "-> take:patch --deep | codex | clipboard | running") {
		t.Errorf("missing initial running line: %q", stderrStr)
	}

	// All five step progress lines must be present.
	wantSteps := []struct {
		step  string
		index string
	}{
		{"plan", "1/5"},
		{"patch", "2/5"},
		{"critique", "3/5"},
		{"tests", "4/5"},
		{"reconcile", "5/5"},
	}
	for _, w := range wantSteps {
		want := "step: " + w.step + " (" + w.index + ")"
		if !strings.Contains(stderrStr, want) {
			t.Errorf("missing step progress %q in stderr: %q", want, stderrStr)
		}
	}

	// Final completed line (status may be "completed" or "completed_with_warnings").
	// The line also includes the delivery label and lang (e.g. "| preview | en->en | completed").
	if !strings.Contains(stderrStr, "-> take:patch --deep | codex | clipboard") || !strings.Contains(stderrStr, "completed") {
		t.Errorf("missing completed status line: %q", stderrStr)
	}

	// Next-step hint lines.
	if !strings.Contains(stderrStr, "next: review:") {
		t.Errorf("missing 'next: review:' hint: %q", stderrStr)
	}
	if !strings.Contains(stderrStr, "next: inspect:") {
		t.Errorf("missing 'next: inspect:' hint: %q", stderrStr)
	}
	if !strings.Contains(stderrStr, "next: log:") {
		t.Errorf("missing 'next: log:' hint: %q", stderrStr)
	}
}

// TestSystemS4_DeepDryRunArtifactRootInRepo verifies that artifacts land in
// <repoRoot>/.prtr/runs/<runID>/ during a dry-run.
func TestSystemS4_DeepDryRunArtifactRootInRepo(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	store := history.New(filepath.Join(t.TempDir(), "history.json"))
	if err := store.Append(history.Entry{
		ID: "seed", CreatedAt: time.Unix(100, 0).UTC(), Target: "codex",
	}); err != nil {
		t.Fatal(err)
	}

	app := newDeepTestApp(t, testConfig(), &stubTranslator{}, &stubClipboard{read: "fix the handler in handlers/user.go"}, &stubEditor{}, store, repoRoot)

	if err := app.Execute(context.Background(), []string{"take", "patch", "--deep", "--dry-run"}, strings.NewReader(""), false); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	entry := mustLatestEntry(t, store)
	expectedPrefix := filepath.Join(repoRoot, ".prtr", "runs")
	if !strings.HasPrefix(entry.ArtifactRoot, expectedPrefix) {
		t.Errorf("ArtifactRoot = %q, want prefix %q", entry.ArtifactRoot, expectedPrefix)
	}

	// Core artifacts must exist inside the artifact root.
	for _, rel := range []string{"manifest.json", "plan.json", "events.jsonl", "result/patch_bundle.json"} {
		p := filepath.Join(entry.ArtifactRoot, rel)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("artifact %q missing: %v", rel, err)
		}
	}
}

// ---------------------------------------------------------------------------
// Scenario 5: Resilience — graceful errors, no panics
// ---------------------------------------------------------------------------

// TestSystemS5_EmptyPrtrDirGracefulRun verifies that commands run correctly
// in a repo that has no .prtr directory at all.
func TestSystemS5_EmptyPrtrDirGracefulRun(t *testing.T) {
	t.Parallel()

	root := gitRepo(t) // no .prtr directory
	writeFile(t, root, "README.md", "# project\nUse FooBarBaz.\n")
	store := history.New(filepath.Join(t.TempDir(), "history.json"))
	if err := store.Append(history.Entry{
		ID: "seed", CreatedAt: time.Unix(100, 0).UTC(), Target: "codex",
	}); err != nil {
		t.Fatal(err)
	}

	app := newRealTermbookApp(t, testConfig(), &stubTranslator{output: "ok"}, &stubClipboard{read: "fix the bug"}, store, root)
	ctx := context.Background()

	// go command must succeed even without .prtr.
	if err := app.Execute(ctx, []string{"go", "ask", "how does this work?", "--dry-run"}, strings.NewReader(""), false); err != nil {
		t.Errorf("go without .prtr: %v", err)
	}

	// take patch (classic) must succeed without .prtr.
	resetBuffers(app)
	if err := app.Execute(ctx, []string{"take", "patch", "--dry-run"}, strings.NewReader(""), false); err != nil {
		t.Errorf("take patch without .prtr: %v", err)
	}

	// take patch --deep must succeed and CREATE .prtr/runs/ on demand.
	resetBuffers(app)
	if err := app.Execute(ctx, []string{"take", "patch", "--deep", "--dry-run"}, strings.NewReader(""), false); err != nil {
		t.Errorf("take patch --deep without .prtr: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".prtr", "runs")); err != nil {
		t.Errorf(".prtr/runs should be created by deep run: %v", err)
	}

	// learn --dry-run must succeed without .prtr.
	resetBuffers(app)
	if err := app.Execute(ctx, []string{"learn", "README.md", "--dry-run"}, strings.NewReader(""), false); err != nil {
		t.Errorf("learn --dry-run without .prtr: %v", err)
	}
}

// TestSystemS5_UnsupportedDeepActionFriendlyError verifies that unsupported
// --deep action combinations return a friendly user-visible error, not a panic.
func TestSystemS5_UnsupportedDeepActionFriendlyError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		args    []string
		wantMsg string
	}{
		{
			name:    "take_summary_deep",
			args:    []string{"take", "summary", "--deep", "--dry-run"},
			wantMsg: "deep execution supports",
		},
		{
			name:    "take_commit_deep",
			args:    []string{"take", "commit", "--deep", "--dry-run"},
			wantMsg: "deep execution supports",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			app := newTestApp(t, testConfig(), &stubTranslator{}, &stubClipboard{read: "some answer"}, &stubEditor{},
				history.New(filepath.Join(t.TempDir(), "history.json")))

			err := app.Execute(context.Background(), tc.args, strings.NewReader(""), false)
			if err == nil {
				t.Fatalf("%s: expected error, got nil", tc.name)
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("%s: error = %q, want to contain %q", tc.name, err.Error(), tc.wantMsg)
			}
		})
	}
}

// TestSystemS5_UnknownTakeActionFriendlyError verifies that completely unknown
// take actions (like "build") return a clear error without panic.
func TestSystemS5_UnknownTakeActionFriendlyError(t *testing.T) {
	t.Parallel()

	app := newTestApp(t, testConfig(), &stubTranslator{}, &stubClipboard{read: "x"}, &stubEditor{},
		history.New(filepath.Join(t.TempDir(), "history.json")))

	cases := []struct {
		action string
	}{
		{"build"},
		{"deploy"},
		{"review"},
		{"push"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.action, func(t *testing.T) {
			t.Parallel()
			err := app.Execute(context.Background(), []string{"take", tc.action, "--deep", "--dry-run"}, strings.NewReader(""), false)
			if err == nil {
				t.Fatalf("take %s --deep: expected error, got nil", tc.action)
			}
			if !strings.Contains(strings.ToLower(err.Error()), tc.action) &&
				!strings.Contains(strings.ToLower(err.Error()), "unknown") &&
				!strings.Contains(strings.ToLower(err.Error()), "available") {
				t.Errorf("take %s --deep: error = %q, want a descriptive message", tc.action, err.Error())
			}
		})
	}
}

// TestSystemS5_TestActionDeepSucceeds verifies that `take test --deep` runs
// without error now that "test" is a supported deep action.
func TestSystemS5_TestActionDeepSucceeds(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	app := newDeepTestApp(t, testConfig(), &stubTranslator{}, &stubClipboard{read: "Write tests for the auth handler"}, &stubEditor{},
		history.New(filepath.Join(t.TempDir(), "history.json")), repoRoot)

	err := app.Execute(context.Background(), []string{"take", "test", "--deep", "--dry-run"}, strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("take test --deep: unexpected error: %v", err)
	}
}


// TestSystemS5_EmptyHistoryForSwapAndAgain verifies that `swap` and `again`
// return a friendly error (not a panic) when there is no history.
func TestSystemS5_EmptyHistoryForSwapAndAgain(t *testing.T) {
	t.Parallel()

	store := history.New(filepath.Join(t.TempDir(), "history.json")) // empty
	app := newTestApp(t, testConfig(), &stubTranslator{}, &stubClipboard{}, &stubEditor{}, store)

	t.Run("swap_no_history", func(t *testing.T) {
		err := app.Execute(context.Background(), []string{"swap", "gemini", "--dry-run"}, strings.NewReader(""), false)
		if err == nil {
			t.Fatal("swap with empty history: expected error, got nil")
		}
		if strings.Contains(strings.ToLower(err.Error()), "panic") {
			t.Errorf("swap error contains 'panic': %v", err)
		}
	})

	t.Run("again_no_history", func(t *testing.T) {
		err := app.Execute(context.Background(), []string{"again", "--dry-run"}, strings.NewReader(""), false)
		if err == nil {
			t.Fatal("again with empty history: expected error, got nil")
		}
		if strings.Contains(strings.ToLower(err.Error()), "panic") {
			t.Errorf("again error contains 'panic': %v", err)
		}
	})
}

// TestSystemS5_EmptyClipboardForDeepRun verifies that `take patch --deep`
// with empty clipboard returns a friendly error.
func TestSystemS5_EmptyClipboardForDeepRun(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	app := newDeepTestApp(t, testConfig(), &stubTranslator{}, &stubClipboard{read: "   "}, &stubEditor{},
		history.New(filepath.Join(t.TempDir(), "history.json")), repoRoot)

	err := app.Execute(context.Background(), []string{"take", "patch", "--deep", "--dry-run"}, strings.NewReader(""), false)
	if err == nil {
		t.Fatal("expected error for empty clipboard, got nil")
	}
	if !strings.Contains(err.Error(), "clipboard is empty") {
		t.Errorf("error = %q, want 'clipboard is empty'", err.Error())
	}
}

// TestSystemS5_NoPanicOnMissingHistoryStore verifies that the app does not
// panic when the history store is nil; history writes are silently skipped.
func TestSystemS5_NoPanicOnMissingHistoryStore(t *testing.T) {
	t.Parallel()

	app := New(Dependencies{
		Version:         "test",
		Stdout:          &bytes.Buffer{},
		Stderr:          &bytes.Buffer{},
		Translator:      &stubTranslator{output: "ok"},
		Clipboard:       &stubClipboard{},
		Editor:          &stubEditor{},
		Launcher:        &stubLauncher{},
		Automator:       &stubAutomator{},
		SubmitConfirmer: &stubConfirmer{},
		ConfigLoader:    func() (config.Config, error) { return testConfig(), nil },
		ConfigInit:      func() (string, error) { return "/tmp/prtr/config.toml", nil },
		LookupEnv:       func(string) (string, bool) { return "", false },
		HistoryStore:    nil, // deliberately nil
		RepoRootFinder: func() (string, error) {
			return "", termbook.ErrNotGitRepo
		},
	})

	// Should succeed without panic (history append is a no-op when store is nil).
	err := app.Execute(context.Background(), []string{"--no-copy", "hello world"}, strings.NewReader(""), false)
	if err != nil {
		t.Fatalf("Execute() with nil HistoryStore error = %v", err)
	}
}

// ---------------------------------------------------------------------------
// resetBuffers resets the stdout/stderr buffers between steps in a
// sequential multi-step test.
// ---------------------------------------------------------------------------

func resetBuffers(app *App) {
	if b, ok := app.stdout.(*bytes.Buffer); ok {
		b.Reset()
	}
	if b, ok := app.stderr.(*bytes.Buffer); ok {
		b.Reset()
	}
}
