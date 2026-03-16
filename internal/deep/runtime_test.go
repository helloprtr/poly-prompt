package deep

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	deepevent "github.com/helloprtr/poly-prompt/internal/deep/event"
	deepplan "github.com/helloprtr/poly-prompt/internal/deep/plan"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func runPatch(t *testing.T, repoRoot string) Result {
	t.Helper()
	result, err := ExecutePatchRun(context.Background(), Options{
		Action:     "patch",
		Source:     "fix the login bug in internal/auth/auth.go",
		SourceKind: "clipboard",
		TargetApp:  "codex",
		RepoRoot:   repoRoot,
	})
	if err != nil {
		t.Fatalf("ExecutePatchRun() error = %v", err)
	}
	return result
}

func mustReadJSON(t *testing.T, path string, dst any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	if err := json.Unmarshal(data, dst); err != nil {
		t.Fatalf("Unmarshal(%s) error = %v", path, err)
	}
}

func mustExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file %s to exist: %v", path, err)
	}
}

// ---------------------------------------------------------------------------
// Artifact structure
// ---------------------------------------------------------------------------

// TestArtifactStructureCreated verifies that all canonical artifact files and
// directories are created by a successful run.
func TestArtifactStructureCreated(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	result := runPatch(t, repoRoot)
	root := result.Run.ArtifactRoot

	required := []string{
		"manifest.json",
		"lineage.json",
		"plan.json",
		"events.jsonl",
		"source.md",
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
	for _, rel := range required {
		mustExist(t, filepath.Join(root, rel))
	}
}

// ---------------------------------------------------------------------------
// plan.json todo graph
// ---------------------------------------------------------------------------

// TestPlanJSONTodoGraph verifies that plan.json encodes the fixed
// planner→patcher→(critic|tester)→reconciler graph.
func TestPlanJSONTodoGraph(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	result := runPatch(t, repoRoot)

	var plan deepplan.WorkPlan
	mustReadJSON(t, filepath.Join(result.Run.ArtifactRoot, "plan.json"), &plan)

	wantWorkers := []string{"planner", "patcher", "critic", "tester", "reconciler"}
	gotWorkers := make(map[string]bool)
	for _, todo := range plan.Todos {
		gotWorkers[todo.Worker] = true
	}
	for _, w := range wantWorkers {
		if !gotWorkers[w] {
			t.Errorf("plan.json missing todo for worker %q", w)
		}
	}

	// Dependency chain: patch depends on plan, reconcile depends on critique+tests.
	depMap := make(map[string][]string)
	for _, todo := range plan.Todos {
		depMap[todo.ID] = todo.DependsOn
	}
	if !containsAll(depMap["patch"], []string{"plan"}) {
		t.Errorf("patch.DependsOn = %v, want [plan]", depMap["patch"])
	}
	if !containsAll(depMap["critique"], []string{"patch"}) {
		t.Errorf("critique.DependsOn = %v, want [patch]", depMap["critique"])
	}
	if !containsAll(depMap["tests"], []string{"patch"}) {
		t.Errorf("tests.DependsOn = %v, want [patch]", depMap["tests"])
	}
	if !containsAll(depMap["reconcile"], []string{"critique", "tests"}) {
		t.Errorf("reconcile.DependsOn = %v, want [critique tests]", depMap["reconcile"])
	}
}

func containsAll(haystack, needles []string) bool {
	set := make(map[string]bool, len(haystack))
	for _, v := range haystack {
		set[v] = true
	}
	for _, n := range needles {
		if !set[n] {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// Completed_with_warnings on soft failure
// ---------------------------------------------------------------------------

// TestCompletedWithWarningsOnSoftFailure is tested indirectly: the graph
// propagates soft-blocker warnings, and the runtime merges them into the bundle.
// Here we verify the final run status is completed_with_warnings when the
// reconciler surfaces bundle warnings (no concrete file path).
func TestBundleWarningsProducesCompletedWithWarnings(t *testing.T) {
	t.Parallel()

	// Source with no recognisable file path → reconciler adds a warning.
	result, err := ExecutePatchRun(context.Background(), Options{
		Action:     "patch",
		Source:     "just fix it", // no file path → warning
		SourceKind: "clipboard",
		RepoRoot:   t.TempDir(),
	})
	if err != nil {
		t.Fatalf("ExecutePatchRun() error = %v", err)
	}
	if result.Run.Status != RunStatusCompletedWithWarning {
		t.Errorf("Status = %q, want %q", result.Run.Status, RunStatusCompletedWithWarning)
	}
	if result.Run.WarningCount == 0 {
		t.Error("WarningCount should be > 0 when status is completed_with_warnings")
	}
}

// ---------------------------------------------------------------------------
// Artifact root path logic
// ---------------------------------------------------------------------------

// TestArtifactRootInsideRepo verifies that when a repoRoot is provided the
// artifacts land under <repoRoot>/.prtr/runs/<id>.
func TestArtifactRootInsideRepo(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	result := runPatch(t, repoRoot)

	prefix := filepath.Join(repoRoot, ".prtr", "runs")
	if !strings.HasPrefix(result.Run.ArtifactRoot, prefix) {
		t.Errorf("ArtifactRoot = %q, want prefix %q", result.Run.ArtifactRoot, prefix)
	}
}

// TestArtifactRootFallbackWithoutRepo verifies that when repoRoot is empty
// the artifacts land in the prtr data directory (not the repo).
func TestArtifactRootFallbackWithoutRepo(t *testing.T) {
	t.Parallel()

	result, err := ExecutePatchRun(context.Background(), Options{
		Action:     "patch",
		Source:     "fix the bug",
		SourceKind: "clipboard",
		RepoRoot:   "", // no repo
	})
	if err != nil {
		t.Fatalf("ExecutePatchRun() error = %v", err)
	}
	// The fallback path must not be empty and must not start with ".prtr".
	if result.Run.ArtifactRoot == "" {
		t.Fatal("ArtifactRoot should not be empty even without a repo root")
	}
	if strings.HasPrefix(result.Run.ArtifactRoot, ".prtr") {
		t.Errorf("ArtifactRoot %q looks like a relative repo path; expected absolute fallback", result.Run.ArtifactRoot)
	}
}

// ---------------------------------------------------------------------------
// manifest.json reflects final run state
// ---------------------------------------------------------------------------

// TestManifestReflectsFinalStatus verifies that manifest.json is updated with
// the final run status after completion.
func TestManifestReflectsFinalStatus(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	result := runPatch(t, repoRoot)

	var manifest Run
	mustReadJSON(t, filepath.Join(result.Run.ArtifactRoot, "manifest.json"), &manifest)

	if manifest.ID != result.Run.ID {
		t.Errorf("manifest.ID = %q, want %q", manifest.ID, result.Run.ID)
	}
	if manifest.Engine != "deep" {
		t.Errorf("manifest.Engine = %q, want deep", manifest.Engine)
	}
	if manifest.ResultRef == "" {
		t.Error("manifest.ResultRef should be set")
	}
	if manifest.CompletedAt == nil {
		t.Error("manifest.CompletedAt should be set on completion")
	}
}

// ---------------------------------------------------------------------------
// events.jsonl golden test
// ---------------------------------------------------------------------------

// TestEventsJSONLOrderAndPayload is a golden test that verifies:
//   - events.jsonl is valid JSONL (each line is valid JSON)
//   - the mandatory event sequence is present and in order
//   - each event has a non-zero Timestamp and a Type
func TestEventsJSONLOrderAndPayload(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	result := runPatch(t, repoRoot)

	eventsPath := filepath.Join(result.Run.ArtifactRoot, "events.jsonl")
	f, err := os.Open(eventsPath)
	if err != nil {
		t.Fatalf("open events.jsonl: %v", err)
	}
	defer f.Close()

	var events []deepevent.Event
	scanner := bufio.NewScanner(f)
	for i := 1; scanner.Scan(); i++ {
		line := scanner.Text()
		var e deepevent.Event
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("line %d is not valid JSON: %v\n%s", i, err, line)
		}
		if e.Timestamp.IsZero() {
			t.Errorf("line %d: Timestamp is zero", i)
		}
		if e.Type == "" {
			t.Errorf("line %d: Type is empty", i)
		}
		events = append(events, e)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner error: %v", err)
	}

	// Mandatory sequence (parallel critic/tester events are not checked for order).
	mandatorySequence := []deepevent.Type{
		deepevent.RunStarted,
		deepevent.ContextCompiled,
		// worker events happen here (order partially non-deterministic for parallel stage)
		deepevent.PlanCreated,
		deepevent.ArtifactReady,
		deepevent.RunCompleted,
	}
	seqIdx := 0
	for _, e := range events {
		if seqIdx < len(mandatorySequence) && e.Type == mandatorySequence[seqIdx] {
			seqIdx++
		}
	}
	if seqIdx < len(mandatorySequence) {
		t.Errorf("mandatory event sequence incomplete; stopped at index %d (%s)", seqIdx, mandatorySequence[seqIdx])
	}

	// All expected event types must appear at least once.
	seen := make(map[deepevent.Type]bool)
	for _, e := range events {
		seen[e.Type] = true
	}
	required := []deepevent.Type{
		deepevent.RunStarted,
		deepevent.ContextCompiled,
		deepevent.WorkerStarted,
		deepevent.WorkerCompleted,
		deepevent.PlanCreated,
		deepevent.ArtifactReady,
		deepevent.RunCompleted,
	}
	for _, et := range required {
		if !seen[et] {
			t.Errorf("event type %q not found in events.jsonl", et)
		}
	}

	// run.started must carry run_id.
	for _, e := range events {
		if e.Type == deepevent.RunStarted {
			if _, ok := e.Data["run_id"]; !ok {
				t.Error("run.started event missing run_id in Data")
			}
			break
		}
	}
}

// TestEventsJSONLWorkerSet verifies that all five workers emit started+completed events.
func TestEventsJSONLWorkerSet(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	result := runPatch(t, repoRoot)

	eventsPath := filepath.Join(result.Run.ArtifactRoot, "events.jsonl")
	data, err := os.ReadFile(eventsPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	startedWorkers := make(map[string]bool)
	completedWorkers := make(map[string]bool)
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var e deepevent.Event
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		name, _ := e.Data["worker"].(string)
		if name == "" {
			continue
		}
		switch e.Type {
		case deepevent.WorkerStarted:
			startedWorkers[name] = true
		case deepevent.WorkerCompleted:
			completedWorkers[name] = true
		}
	}

	for _, w := range []string{"planner", "patcher", "critic", "tester", "reconciler"} {
		if !startedWorkers[w] {
			t.Errorf("worker.started not found for %q", w)
		}
		if !completedWorkers[w] {
			t.Errorf("worker.completed not found for %q", w)
		}
	}
}
