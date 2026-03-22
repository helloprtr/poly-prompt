package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/helloprtr/poly-prompt/internal/deep/artifact"
	deepevent "github.com/helloprtr/poly-prompt/internal/deep/event"
	deepplan "github.com/helloprtr/poly-prompt/internal/deep/plan"
	deeprun "github.com/helloprtr/poly-prompt/internal/deep/run"
	deepschema "github.com/helloprtr/poly-prompt/internal/deep/schema"
)

// GraphResult is the successful outcome of a Graph.Run call.
type GraphResult struct {
	Plan     *deepplan.WorkPlan
	Bundle   *deepschema.PatchBundle
	Warnings []string
}

// Graph is the fixed worker DAG for a deep patch run.
//
// Execution order:
//
//	planner  (hard) → patcher (hard) → critic (soft) ∥ tester (soft) → reconciler (hard)
type Graph struct {
	planner    Worker
	patcher    Worker
	critic     Worker
	tester     Worker
	reconciler Worker
}

// NewPatchGraphWith constructs a Graph with injected workers for testing.
func NewPatchGraphWith(planner, patcher, critic, tester, reconciler Worker) *Graph {
	return &Graph{
		planner:    planner,
		patcher:    patcher,
		critic:     critic,
		tester:     tester,
		reconciler: reconciler,
	}
}

// NewPatchGraph constructs a Graph wired with the default patch workers.
func NewPatchGraph() *Graph {
	return &Graph{
		planner:    &plannerWorker{},
		patcher:    &patcherWorker{},
		critic:     &criticWorker{},
		tester:     &testerWorker{},
		reconciler: &reconcilerWorker{},
	}
}

// Run executes the DAG and returns the final bundle.
//
// Failure semantics:
//   - Hard blocker failure → returns error immediately; caller sets status = failed.
//   - Soft blocker failure → warning added to GraphResult.Warnings; execution continues;
//     caller sets status = completed_with_warnings if len(Warnings) > 0.
func (g *Graph) Run(ctx context.Context, opts deeprun.Options, aw *artifact.Writer, files []string) (GraphResult, error) {
	s := &State{
		Ctx:   ctx,
		Opts:  opts,
		AW:    aw,
		Files: files,
	}

	// Fixed stage ordering — each inner slice runs concurrently.
	stages := [][]Worker{
		{g.planner},
		{g.patcher},
		{g.critic, g.tester}, // parallel soft blockers
		{g.reconciler},
	}

	for _, stage := range stages {
		var err error
		if len(stage) == 1 {
			err = g.runOne(stage[0], s)
		} else {
			err = g.runParallel(stage, s)
		}
		if err != nil {
			return GraphResult{}, err
		}
	}

	return GraphResult{
		Plan:     s.Plan,
		Bundle:   s.Bundle,
		Warnings: s.Warnings,
	}, nil
}

// workerIndex returns the 1-based index of a worker in the pipeline.
func workerIndex(name string) int {
	switch name {
	case "planner":
		return 1
	case "patcher":
		return 2
	case "critic":
		return 3
	case "tester":
		return 4
	case "reconciler":
		return 5
	default:
		return 0
	}
}

// runOne runs a single worker and applies hard/soft blocker semantics.
func (g *Graph) runOne(w Worker, s *State) error {
	_ = deepevent.Append(s.AW.EventLogPath(), deepevent.Event{
		Type:      deepevent.WorkerStarted,
		Timestamp: time.Now().UTC(),
		Data:      map[string]any{"worker": w.Name()},
	})

	// Fire progress when worker actually starts.
	if s.Opts.Progress != nil {
		s.Opts.Progress(deeprun.Progress{
			Step:    w.Name(),
			Index:   workerIndex(w.Name()),
			Total:   5,
			Message: "starting",
		})
	}

	runErr := w.Run(s.Ctx, s)

	if runErr != nil {
		s.MarkTodo(w.Name(), deepplan.TodoFailed)
		if w.HardBlocker() {
			_ = deepevent.Append(s.AW.EventLogPath(), deepevent.Event{
				Type:      deepevent.RunFailed,
				Timestamp: time.Now().UTC(),
				Data:      map[string]any{"worker": w.Name(), "error": runErr.Error()},
			})
			return fmt.Errorf("worker %s: %w", w.Name(), runErr)
		}
		// Soft blocker: record warning and continue.
		s.AddWarning(fmt.Sprintf("%s failed (continuing): %s", w.Name(), runErr.Error()))
	} else {
		s.MarkTodo(w.Name(), deepplan.TodoCompleted)
	}

	_ = deepevent.Append(s.AW.EventLogPath(), deepevent.Event{
		Type:      deepevent.WorkerCompleted,
		Timestamp: time.Now().UTC(),
		Data:      map[string]any{"worker": w.Name(), "ok": runErr == nil},
	})

	return nil
}

// runParallel executes a stage of workers concurrently.
// All workers complete before the function returns.
// A hard-blocker error from any goroutine is returned after all finish.
func (g *Graph) runParallel(workers []Worker, s *State) error {
	type outcome struct{ err error }
	results := make(chan outcome, len(workers))

	var wg sync.WaitGroup
	for _, w := range workers {
		w := w
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- outcome{err: g.runOne(w, s)}
		}()
	}
	wg.Wait()
	close(results)

	for r := range results {
		if r.err != nil {
			// runOne only returns an error for hard blockers; propagate the first one.
			return r.err
		}
	}
	return nil
}
