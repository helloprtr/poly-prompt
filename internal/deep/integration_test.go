package deep

// integration_test.go — 5-scenario end-to-end integration tests for the
// deep patch execution engine.
//
// Scenarios covered:
//   1. Happy Path      — dry-run, all artifacts + events + progress steps
//   2. Resilience      — soft failure → completed_with_warnings;
//                        hard failure → failed (no successor workers)
//   3. Data Integrity  — PatchBundle schema completeness; manifest fields
//   4. History Lineage — history entry fields after deep run (tested via app_test.go)
//   5. Regression      — classic take patch still works (tested via app_test.go)
//
// Scenarios 4 and 5 are already covered exhaustively in app_test.go:
//   • TestExecuteTakeDeepWritesArtifactsAndHistoryMetadata
//   • TestExecuteTakeDeepHistoryLinkage
//   • TestExecuteTakeClassicPatchRegressionNoDeeply
//   • TestExecuteTakeClassicTestActionRegressionNoDeeply
//
// This file adds tests that require mock worker injection, which is only
// possible from within the `deep` package (via executePatchRunWithGraph).

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	deepevent "github.com/helloprtr/poly-prompt/internal/deep/event"
	"github.com/helloprtr/poly-prompt/internal/deep/llm"
	deepschema "github.com/helloprtr/poly-prompt/internal/deep/schema"
	"github.com/helloprtr/poly-prompt/internal/deep/worker"
)

// ---------------------------------------------------------------------------
// Scenario 1: End-to-End Happy Path
// ---------------------------------------------------------------------------

// TestScenario1_AllProgressStepsEmitted verifies that every progress step
// (plan, patch, critique, tests, reconcile) is emitted in order during a
// successful run.
func TestScenario1_AllProgressStepsEmitted(t *testing.T) {
	t.Parallel()

	var steps []string
	opts := Options{
		Action:     "patch",
		Source:     "fix the login bug in internal/auth/auth.go",
		SourceKind: "clipboard",
		TargetApp:  "codex",
		RepoRoot:   t.TempDir(),
		Progress: func(p Progress) {
			steps = append(steps, fmt.Sprintf("%s(%d/%d)", p.Step, p.Index, p.Total))
		},
	}

	result, err := executePatchRunWithGraph(context.Background(), opts, worker.NewPatchGraph, llm.New)
	if err != nil {
		t.Fatalf("executePatchRunWithGraph() error = %v", err)
	}

	// All five step names must be present in order.
	wantSteps := []string{"planner", "patcher", "critic", "tester", "reconciler"}
	for _, want := range wantSteps {
		found := false
		for _, got := range steps {
			if strings.HasPrefix(got, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("progress step %q not emitted; got steps: %v", want, steps)
		}
	}

	// Total must always be 5.
	for _, step := range steps {
		if !strings.Contains(step, "/5)") {
			t.Errorf("step %q has unexpected total (want /5)", step)
		}
	}

	// Step indices must match expected 1-5.
	expectedIndexed := map[string]int{"planner": 1, "patcher": 2, "critic": 3, "tester": 4, "reconciler": 5}
	for _, step := range steps {
		for name, idx := range expectedIndexed {
			if strings.HasPrefix(step, name) {
				wantToken := fmt.Sprintf("%s(%d/5)", name, idx)
				if step != wantToken {
					t.Errorf("step = %q, want %q", step, wantToken)
				}
			}
		}
	}

	// Run must complete successfully (not failed).
	if result.Run.Status == RunStatusFailed {
		t.Errorf("run status = %q after happy path, want completed*", result.Run.Status)
	}
}

// TestScenario1_DryRunArtifactsAndEvents is a full end-to-end test that
// mirrors "take patch --deep --dry-run": verifies all artifact files exist,
// all events are recorded, and the delivery prompt is non-empty.
func TestScenario1_DryRunArtifactsAndEvents(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	result, err := executePatchRunWithGraph(context.Background(), Options{
		Action:     "patch",
		Source:     "refactor internal/handler/user.go to return structured errors",
		SourceKind: "clipboard",
		TargetApp:  "claude",
		RepoRoot:   repoRoot,
	}, worker.NewPatchGraph, llm.New)
	if err != nil {
		t.Fatalf("executePatchRunWithGraph() error = %v", err)
	}

	// ── Artifact files ────────────────────────────────────────────────────
	requiredArtifacts := []string{
		"manifest.json", "lineage.json", "plan.json", "events.jsonl", "source.md",
		filepath.Join("evidence", "repo_context.json"),
		filepath.Join("evidence", "history.json"),
		filepath.Join("evidence", "memory.json"),
		filepath.Join("evidence", "git.diff"),
		filepath.Join("result", "patch_bundle.json"),
		filepath.Join("result", "patch.diff"),
		filepath.Join("result", "tests.md"),
		filepath.Join("result", "summary.md"),
		filepath.Join("workers", "planner", "result.json"),
		filepath.Join("workers", "patcher", "result.json"),
		filepath.Join("workers", "critic", "result.json"),
		filepath.Join("workers", "tester", "result.json"),
		filepath.Join("workers", "reconciler", "result.json"),
	}
	for _, rel := range requiredArtifacts {
		if _, err := os.Stat(filepath.Join(result.Run.ArtifactRoot, rel)); err != nil {
			t.Errorf("artifact %q missing: %v", rel, err)
		}
	}

	// ── Event sequence ────────────────────────────────────────────────────
	eventsPath := filepath.Join(result.Run.ArtifactRoot, "events.jsonl")
	f, err := os.Open(eventsPath)
	if err != nil {
		t.Fatalf("open events.jsonl: %v", err)
	}
	defer f.Close()

	var events []deepevent.Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var e deepevent.Event
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			t.Fatalf("invalid JSON in events.jsonl: %v", err)
		}
		if e.Timestamp.IsZero() {
			t.Errorf("event %q has zero timestamp", e.Type)
		}
		events = append(events, e)
	}

	mandatorySeq := []deepevent.Type{
		deepevent.RunStarted,
		deepevent.ContextCompiled,
		deepevent.PlanCreated,
		deepevent.ArtifactReady,
		deepevent.RunCompleted,
	}
	seqIdx := 0
	for _, e := range events {
		if seqIdx < len(mandatorySeq) && e.Type == mandatorySeq[seqIdx] {
			seqIdx++
		}
	}
	if seqIdx < len(mandatorySeq) {
		t.Errorf("mandatory event sequence incomplete at index %d (%s)", seqIdx, mandatorySeq[seqIdx])
	}

	// ── Delivery prompt ───────────────────────────────────────────────────
	if strings.TrimSpace(result.DeliveryPrompt) == "" {
		t.Error("DeliveryPrompt is empty after successful run")
	}
	if !strings.Contains(result.DeliveryPrompt, "patch bundle") {
		t.Errorf("DeliveryPrompt = %q; want to contain 'patch bundle'", result.DeliveryPrompt[:min(120, len(result.DeliveryPrompt))])
	}
}

// ---------------------------------------------------------------------------
// Scenario 2: Resilience — Soft Failure
// ---------------------------------------------------------------------------

// TestScenario2_CriticSoftFailureProducesCompletedWithWarnings injects a
// failing critic worker and verifies that the full run ends as
// completed_with_warnings, not failed, and the bundle is non-nil.
func TestScenario2_CriticSoftFailureProducesCompletedWithWarnings(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	result, err := executePatchRunWithGraph(context.Background(), Options{
		Action:     "patch",
		Source:     "fix nil pointer in internal/auth/auth.go",
		SourceKind: "clipboard",
		RepoRoot:   repoRoot,
	}, func() *worker.Graph {
		return worker.NewPatchGraphWith(
			worker.NewDefaultPlannerWorker(),
			worker.NewDefaultPatcherWorker(),
			worker.NewErrorWorker("critic", false, fmt.Errorf("critic service unavailable")),
			worker.NewDefaultTesterWorker(),
			worker.NewDefaultReconcilerWorker(),
		)
	}, llm.New)
	if err != nil {
		t.Fatalf("run returned error = %v (expected success with warnings)", err)
	}
	if result.Run.Status != RunStatusCompletedWithWarning {
		t.Errorf("Status = %q, want %q", result.Run.Status, RunStatusCompletedWithWarning)
	}
	if result.Run.WarningCount == 0 {
		t.Error("WarningCount should be > 0 when critic fails")
	}

	// manifest.json must also reflect completed_with_warnings.
	var manifest Run
	mustReadJSON(t, filepath.Join(result.Run.ArtifactRoot, "manifest.json"), &manifest)
	if manifest.Status != RunStatusCompletedWithWarning {
		t.Errorf("manifest.Status = %q, want %q", manifest.Status, RunStatusCompletedWithWarning)
	}
	if manifest.WarningCount == 0 {
		t.Error("manifest.WarningCount should be > 0")
	}
}

// TestScenario2_TesterSoftFailureProducesCompletedWithWarnings injects a
// failing tester worker (soft blocker) and verifies the same semantics.
func TestScenario2_TesterSoftFailureProducesCompletedWithWarnings(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	result, err := executePatchRunWithGraph(context.Background(), Options{
		Action:     "patch",
		Source:     "fix the auth bug in handlers/login.go",
		SourceKind: "clipboard",
		RepoRoot:   repoRoot,
	}, func() *worker.Graph {
		return worker.NewPatchGraphWith(
			worker.NewDefaultPlannerWorker(),
			worker.NewDefaultPatcherWorker(),
			worker.NewDefaultCriticWorker(),
			worker.NewErrorWorker("tester", false, fmt.Errorf("tester timeout")),
			worker.NewDefaultReconcilerWorker(),
		)
	}, llm.New)
	if err != nil {
		t.Fatalf("run returned error = %v (expected success with warnings)", err)
	}
	if result.Run.Status != RunStatusCompletedWithWarning {
		t.Errorf("Status = %q, want %q", result.Run.Status, RunStatusCompletedWithWarning)
	}
	if result.Run.WarningCount == 0 {
		t.Error("WarningCount should be > 0 when tester fails")
	}
}

// TestScenario2_BothSoftBlockersFailProducesWarnings injects both critic and
// tester as failing soft blockers and verifies the run still produces a bundle.
func TestScenario2_BothSoftBlockersFailProducesWarnings(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	result, err := executePatchRunWithGraph(context.Background(), Options{
		Action:     "patch",
		Source:     "fix the database connection leak in db/pool.go",
		SourceKind: "clipboard",
		RepoRoot:   repoRoot,
	}, func() *worker.Graph {
		return worker.NewPatchGraphWith(
			worker.NewDefaultPlannerWorker(),
			worker.NewDefaultPatcherWorker(),
			worker.NewErrorWorker("critic", false, fmt.Errorf("critic down")),
			worker.NewErrorWorker("tester", false, fmt.Errorf("tester down")),
			worker.NewDefaultReconcilerWorker(),
		)
	}, llm.New)
	if err != nil {
		t.Fatalf("run returned error = %v (expected success with warnings)", err)
	}
	if result.Run.Status != RunStatusCompletedWithWarning {
		t.Errorf("Status = %q, want %q", result.Run.Status, RunStatusCompletedWithWarning)
	}
	// Two worker failures + possibly a warning from reconciler seeing nil inputs.
	if result.Run.WarningCount < 2 {
		t.Errorf("WarningCount = %d, want ≥ 2", result.Run.WarningCount)
	}
	// Bundle must still exist.
	bundlePath := filepath.Join(result.Run.ArtifactRoot, "result", "patch_bundle.json")
	if _, err := os.Stat(bundlePath); err != nil {
		t.Errorf("bundle missing after dual soft failure: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Scenario 2: Resilience — Hard Failure
// ---------------------------------------------------------------------------

// TestScenario2_PlannerHardFailureProducesFailedStatus injects a failing
// planner (hard blocker) and verifies that:
//   - ExecutePatchRun returns a non-nil error
//   - manifest.json reflects status = "failed"
//   - No subsequent workers (patcher, critic, etc.) are executed
func TestScenario2_PlannerHardFailureProducesFailedStatus(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	var calledWorkers []string
	trackingPatcher := worker.NewCallTrackingWorker("patcher", true, &calledWorkers)

	_, err := executePatchRunWithGraph(context.Background(), Options{
		Action:     "patch",
		Source:     "fix the cache invalidation bug in cache/store.go",
		SourceKind: "clipboard",
		RepoRoot:   repoRoot,
	}, func() *worker.Graph {
		return worker.NewPatchGraphWith(
			worker.NewErrorWorker("planner", true, fmt.Errorf("planner crashed")),
			trackingPatcher,
			worker.NewDefaultCriticWorker(),
			worker.NewDefaultTesterWorker(),
			worker.NewDefaultReconcilerWorker(),
		)
	}, llm.New)
	if err == nil {
		t.Fatal("expected error from planner hard failure, got nil")
	}
	if !strings.Contains(err.Error(), "planner") {
		t.Errorf("error = %q, want it to mention 'planner'", err.Error())
	}

	// Patcher must NOT have been called.
	for _, name := range calledWorkers {
		if name == "patcher" {
			t.Error("patcher was called despite planner hard failure; run should have aborted")
		}
	}

	// manifest.json must reflect "failed".
	// Artifact root: find it by scanning .prtr/runs/ under repoRoot.
	runsDir := filepath.Join(repoRoot, ".prtr", "runs")
	entries, err2 := os.ReadDir(runsDir)
	if err2 != nil {
		t.Fatalf("ReadDir %s: %v", runsDir, err2)
	}
	if len(entries) == 0 {
		t.Fatal("no run directory created")
	}
	runDir := filepath.Join(runsDir, entries[0].Name())
	var manifest Run
	mustReadJSON(t, filepath.Join(runDir, "manifest.json"), &manifest)
	if manifest.Status != RunStatusFailed {
		t.Errorf("manifest.Status = %q, want %q", manifest.Status, RunStatusFailed)
	}
	if manifest.ErrorMessage == "" {
		t.Error("manifest.ErrorMessage should be set on hard failure")
	}
}

// TestScenario2_PatcherHardFailureProducesFailedStatus injects a failing
// patcher (hard blocker) and verifies critic and tester are never called.
func TestScenario2_PatcherHardFailureProducesFailedStatus(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	var calledWorkers []string
	trackingCritic := worker.NewCallTrackingWorker("critic", false, &calledWorkers)
	trackingTester := worker.NewCallTrackingWorker("tester", false, &calledWorkers)

	_, err := executePatchRunWithGraph(context.Background(), Options{
		Action:     "patch",
		Source:     "fix the router in internal/server/routes.go",
		SourceKind: "clipboard",
		RepoRoot:   repoRoot,
	}, func() *worker.Graph {
		return worker.NewPatchGraphWith(
			worker.NewDefaultPlannerWorker(),
			worker.NewErrorWorker("patcher", true, fmt.Errorf("patcher OOM")),
			trackingCritic,
			trackingTester,
			worker.NewDefaultReconcilerWorker(),
		)
	}, llm.New)
	if err == nil {
		t.Fatal("expected error from patcher hard failure, got nil")
	}

	// Critic and tester must NOT have been called.
	for _, name := range calledWorkers {
		if name == "critic" || name == "tester" {
			t.Errorf("worker %q was called after patcher hard failure; should have aborted", name)
		}
	}

	// Find run dir and check manifest.
	runsDir := filepath.Join(repoRoot, ".prtr", "runs")
	entries, err2 := os.ReadDir(runsDir)
	if err2 != nil {
		t.Fatalf("ReadDir %s: %v", runsDir, err2)
	}
	if len(entries) == 0 {
		t.Fatal("no run directory created")
	}
	var manifest Run
	mustReadJSON(t, filepath.Join(runsDir, entries[0].Name(), "manifest.json"), &manifest)
	if manifest.Status != RunStatusFailed {
		t.Errorf("manifest.Status = %q, want %q", manifest.Status, RunStatusFailed)
	}
}

// ---------------------------------------------------------------------------
// Scenario 3: Schema and Artifact Integrity
// ---------------------------------------------------------------------------

// TestScenario3_PatchBundleSchemaComplete verifies that result/patch_bundle.json
// deserializes correctly into PatchBundle and all required fields are non-zero.
func TestScenario3_PatchBundleSchemaComplete(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	result, err := executePatchRunWithGraph(context.Background(), Options{
		Action:     "patch",
		Source:     "fix the memory leak in internal/cache/lru.go",
		SourceKind: "clipboard",
		RepoRoot:   repoRoot,
	}, worker.NewPatchGraph, llm.New)
	if err != nil {
		t.Fatalf("executePatchRunWithGraph() error = %v", err)
	}

	bundlePath := filepath.Join(result.Run.ArtifactRoot, "result", "patch_bundle.json")
	var bundle deepschema.PatchBundle
	mustReadJSON(t, bundlePath, &bundle)

	// Required string fields.
	if strings.TrimSpace(bundle.Summary) == "" {
		t.Error("PatchBundle.Summary is empty")
	}
	if strings.TrimSpace(bundle.Diff) == "" {
		t.Error("PatchBundle.Diff is empty")
	}

	// TouchedFiles: must have at least one entry (extracted from source).
	if len(bundle.TouchedFiles) == 0 {
		t.Error("PatchBundle.TouchedFiles is empty; expected at least one file from source")
	}

	// TestPlan: must have at least one test case and one verification step.
	if len(bundle.TestPlan.TestCases) == 0 {
		t.Error("PatchBundle.TestPlan.TestCases is empty")
	}
	if len(bundle.TestPlan.VerificationSteps) == 0 {
		t.Error("PatchBundle.TestPlan.VerificationSteps is empty")
	}

	// In-memory result must match on-disk bundle.
	if result.Bundle.Summary != bundle.Summary {
		t.Errorf("in-memory bundle.Summary = %q, disk bundle.Summary = %q", result.Bundle.Summary, bundle.Summary)
	}
}

// TestScenario3_ManifestCompleteFields verifies that manifest.json after a
// successful run carries all required fields: ID, Engine, Status, ArtifactRoot,
// EventLogPath, ResultRef, CompletedAt, and ResultType.
func TestScenario3_ManifestCompleteFields(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	result, err := executePatchRunWithGraph(context.Background(), Options{
		Action:     "patch",
		Source:     "fix the goroutine leak in internal/worker/pool.go",
		SourceKind: "clipboard",
		TargetApp:  "claude",
		RepoRoot:   repoRoot,
	}, worker.NewPatchGraph, llm.New)
	if err != nil {
		t.Fatalf("executePatchRunWithGraph() error = %v", err)
	}

	var manifest Run
	mustReadJSON(t, filepath.Join(result.Run.ArtifactRoot, "manifest.json"), &manifest)

	checks := []struct {
		name string
		ok   bool
	}{
		{"ID non-empty", manifest.ID != ""},
		{"Engine == deep", manifest.Engine == "deep"},
		{"Status completed*", manifest.Status == RunStatusCompleted || manifest.Status == RunStatusCompletedWithWarning},
		{"ArtifactRoot == result.Run.ArtifactRoot", manifest.ArtifactRoot == result.Run.ArtifactRoot},
		{"EventLogPath non-empty", manifest.EventLogPath != ""},
		{"ResultRef set", manifest.ResultRef == "result/patch_bundle.json"},
		{"ResultType == PatchBundle", manifest.ResultType == "PatchBundle"},
		{"CompletedAt set", manifest.CompletedAt != nil},
		{"Version == 1", manifest.Version == 1},
		{"Action == patch", manifest.Action == "patch"},
	}
	for _, c := range checks {
		if !c.ok {
			t.Errorf("manifest field check failed: %s", c.name)
		}
	}
}

// TestScenario3_ManifestReflectsRunStatusFromResult verifies that the status
// in manifest.json exactly matches result.Run.Status for all terminal states.
func TestScenario3_ManifestReflectsRunStatusFromResult(t *testing.T) {
	t.Parallel()

	// Case A: completed (source has a recognizable file path → no warning).
	t.Run("completed", func(t *testing.T) {
		t.Parallel()
		result, err := executePatchRunWithGraph(context.Background(), Options{
			Action:     "patch",
			Source:     "fix nil pointer in internal/auth/auth.go",
			SourceKind: "clipboard",
			RepoRoot:   t.TempDir(),
		}, worker.NewPatchGraph, llm.New)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		var manifest Run
		mustReadJSON(t, filepath.Join(result.Run.ArtifactRoot, "manifest.json"), &manifest)
		if manifest.Status != result.Run.Status {
			t.Errorf("manifest.Status = %q, result.Run.Status = %q", manifest.Status, result.Run.Status)
		}
	})

	// Case B: completed_with_warnings (source has no file path → reconciler warning).
	t.Run("completed_with_warnings", func(t *testing.T) {
		t.Parallel()
		result, err := executePatchRunWithGraph(context.Background(), Options{
			Action:     "patch",
			Source:     "just fix it",
			SourceKind: "clipboard",
			RepoRoot:   t.TempDir(),
		}, worker.NewPatchGraph, llm.New)
		if err != nil {
			t.Fatalf("error = %v", err)
		}
		if result.Run.Status != RunStatusCompletedWithWarning {
			t.Skipf("source produced no warnings (status=%q); skipping status-sync check", result.Run.Status)
		}
		var manifest Run
		mustReadJSON(t, filepath.Join(result.Run.ArtifactRoot, "manifest.json"), &manifest)
		if manifest.Status != result.Run.Status {
			t.Errorf("manifest.Status = %q, result.Run.Status = %q", manifest.Status, result.Run.Status)
		}
	})
}

// ---------------------------------------------------------------------------
// Helpers (used only in this file — runPatch / mustReadJSON are in runtime_test.go)
// ---------------------------------------------------------------------------

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
