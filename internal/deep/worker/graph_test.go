package worker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/helloprtr/poly-prompt/internal/deep/artifact"
	deepplan "github.com/helloprtr/poly-prompt/internal/deep/plan"
	deeprun "github.com/helloprtr/poly-prompt/internal/deep/run"
	deepschema "github.com/helloprtr/poly-prompt/internal/deep/schema"
)

// ---------------------------------------------------------------------------
// Stub worker helpers
// ---------------------------------------------------------------------------

type stubWorker struct {
	name string
	hard bool
	err  error
	fn   func(ctx context.Context, s *State) error
}

func (w *stubWorker) Name() string      { return w.name }
func (w *stubWorker) HardBlocker() bool { return w.hard }
func (w *stubWorker) Run(ctx context.Context, s *State) error {
	if w.fn != nil {
		return w.fn(ctx, s)
	}
	return w.err
}

// minimalPlannerStub sets s.Plan so downstream workers are not starved.
func minimalPlannerStub() *stubWorker {
	return &stubWorker{
		name: "planner",
		hard: true,
		fn: func(ctx context.Context, s *State) error {
			plan := &deepplan.WorkPlan{
				Version:    1,
				Action:     "patch",
				ResultType: "PatchBundle",
				Summary:    "stub plan",
				Todos: []deepplan.TodoItem{
					{ID: "plan", Worker: "planner", Status: deepplan.TodoPending},
					{ID: "patch", Worker: "patcher", Status: deepplan.TodoPending},
					{ID: "critique", Worker: "critic", Status: deepplan.TodoPending},
					{ID: "tests", Worker: "tester", Status: deepplan.TodoPending},
					{ID: "reconcile", Worker: "reconciler", Status: deepplan.TodoPending},
				},
			}
			s.Plan = plan
			return s.AW.WriteJSON("plan.json", plan)
		},
	}
}

// minimalPatcherStub sets s.Patch so the reconciler is not starved.
func minimalPatcherStub() *stubWorker {
	return &stubWorker{
		name: "patcher",
		hard: true,
		fn: func(ctx context.Context, s *State) error {
			s.Patch = &deepschema.PatchDraft{Summary: "stub patch", Diff: "stub diff"}
			return s.AW.WriteJSON("workers/patcher/result.json", s.Patch)
		},
	}
}

// testArtifactWriter creates a temp-dir-backed artifact writer with the
// canonical subdirectory layout pre-created.
func testArtifactWriter(t *testing.T) *artifact.Writer {
	t.Helper()
	dir := t.TempDir()
	aw := &artifact.Writer{Root: dir}
	for _, sub := range []string{
		"evidence", "result",
		"workers/planner", "workers/patcher",
		"workers/critic", "workers/tester", "workers/reconciler",
	} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", sub, err)
		}
	}
	return aw
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestTodoGraphAfterSuccess verifies that every todo is marked completed
// when all workers succeed.
func TestTodoGraphAfterSuccess(t *testing.T) {
	t.Parallel()

	aw := testArtifactWriter(t)
	g := NewPatchGraph()
	opts := deeprun.Options{Action: "patch", Source: "fix bug in main.go"}

	gr, err := g.Run(context.Background(), opts, aw, []string{"main.go"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if gr.Plan == nil {
		t.Fatal("Plan is nil")
	}
	for _, todo := range gr.Plan.Todos {
		if todo.Status != deepplan.TodoCompleted {
			t.Errorf("todo %q status = %q, want completed", todo.ID, todo.Status)
		}
	}
}

// TestHardBlockerFailureReturnsError verifies that a hard-blocker failure
// propagates as an error and does not produce a bundle.
func TestHardBlockerFailureReturnsError(t *testing.T) {
	t.Parallel()

	aw := testArtifactWriter(t)
	hardErr := fmt.Errorf("patcher exploded")
	g := NewPatchGraphWith(
		minimalPlannerStub(),
		&stubWorker{name: "patcher", hard: true, err: hardErr},
		&criticWorker{},
		&testerWorker{},
		&reconcilerWorker{},
	)

	_, err := g.Run(context.Background(), deeprun.Options{Action: "patch", Source: "x"}, aw, nil)
	if err == nil {
		t.Fatal("expected error from hard-blocker patcher, got nil")
	}
}

// TestSoftFailureCriticProducesWarnings verifies that a failing critic (soft
// blocker) does not abort the run; the result contains a warning and a bundle.
func TestSoftFailureCriticProducesWarnings(t *testing.T) {
	t.Parallel()

	aw := testArtifactWriter(t)
	g := NewPatchGraphWith(
		minimalPlannerStub(),
		minimalPatcherStub(),
		&stubWorker{name: "critic", hard: false, err: fmt.Errorf("critic unavailable")},
		&testerWorker{},
		&reconcilerWorker{},
	)

	gr, err := g.Run(context.Background(), deeprun.Options{Action: "patch", Source: "fix main.go"}, aw, []string{"main.go"})
	if err != nil {
		t.Fatalf("Run() error = %v (expected success with warnings)", err)
	}
	if len(gr.Warnings) == 0 {
		t.Fatal("expected at least one warning from soft-blocker critic, got none")
	}
	if gr.Bundle == nil {
		t.Fatal("expected bundle despite soft failure, got nil")
	}
	// Critic failure → Risks should be nil; reconciler handles that gracefully.
	if gr.Bundle.Risks != nil && len(gr.Bundle.Risks) > 0 {
		t.Logf("risks present despite critic failure (may be from reconciler default): %v", gr.Bundle.Risks)
	}
}

// TestSoftFailureTesterProducesWarnings verifies that a failing tester (soft
// blocker) does not abort the run.
func TestSoftFailureTesterProducesWarnings(t *testing.T) {
	t.Parallel()

	aw := testArtifactWriter(t)
	g := NewPatchGraphWith(
		minimalPlannerStub(),
		minimalPatcherStub(),
		&criticWorker{},
		&stubWorker{name: "tester", hard: false, err: fmt.Errorf("tester unavailable")},
		&reconcilerWorker{},
	)

	gr, err := g.Run(context.Background(), deeprun.Options{Action: "patch", Source: "fix main.go"}, aw, []string{"main.go"})
	if err != nil {
		t.Fatalf("Run() error = %v (expected success with warnings)", err)
	}
	if len(gr.Warnings) == 0 {
		t.Fatal("expected at least one warning from soft-blocker tester, got none")
	}
	if gr.Bundle == nil {
		t.Fatal("expected bundle despite soft failure, got nil")
	}
}

// TestBothSoftBlockersFailProducesMultipleWarnings verifies that both critic
// and tester failing accumulates two warnings and still produces a bundle.
func TestBothSoftBlockersFailProducesMultipleWarnings(t *testing.T) {
	t.Parallel()

	aw := testArtifactWriter(t)
	g := NewPatchGraphWith(
		minimalPlannerStub(),
		minimalPatcherStub(),
		&stubWorker{name: "critic", hard: false, err: fmt.Errorf("critic down")},
		&stubWorker{name: "tester", hard: false, err: fmt.Errorf("tester down")},
		&reconcilerWorker{},
	)

	gr, err := g.Run(context.Background(), deeprun.Options{Action: "patch", Source: "fix it"}, aw, nil)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(gr.Warnings) < 2 {
		t.Fatalf("want ≥2 warnings, got %d: %v", len(gr.Warnings), gr.Warnings)
	}
	if gr.Bundle == nil {
		t.Fatal("bundle should not be nil")
	}
}

// TestTodoStatusOnSoftFailure verifies that a failing soft blocker's todo is
// marked failed while hard-blocker todos remain completed.
func TestTodoStatusOnSoftFailure(t *testing.T) {
	t.Parallel()

	aw := testArtifactWriter(t)
	g := NewPatchGraphWith(
		minimalPlannerStub(),
		minimalPatcherStub(),
		&stubWorker{name: "critic", hard: false, err: fmt.Errorf("critic down")},
		&testerWorker{},
		&reconcilerWorker{},
	)

	gr, err := g.Run(context.Background(), deeprun.Options{Action: "patch", Source: "x"}, aw, []string{"x.go"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, todo := range gr.Plan.Todos {
		switch todo.Worker {
		case "critic":
			if todo.Status != deepplan.TodoFailed {
				t.Errorf("critic todo status = %q, want failed", todo.Status)
			}
		case "planner", "patcher", "tester", "reconciler":
			if todo.Status != deepplan.TodoCompleted {
				t.Errorf("%s todo status = %q, want completed", todo.Worker, todo.Status)
			}
		}
	}
}
