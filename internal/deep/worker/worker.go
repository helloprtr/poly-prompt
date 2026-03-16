// Package worker defines the Worker interface, shared execution State, and the
// five concrete workers that form the deep patch pipeline.
package worker

import (
	"context"
	"sync"

	"github.com/helloprtr/poly-prompt/internal/deep/artifact"
	deepplan "github.com/helloprtr/poly-prompt/internal/deep/plan"
	deeprun "github.com/helloprtr/poly-prompt/internal/deep/run"
	deepschema "github.com/helloprtr/poly-prompt/internal/deep/schema"
)

// Worker is the contract for a single step in the deep execution pipeline.
type Worker interface {
	// Name returns the canonical worker identifier (matches WorkerSpec.Name and
	// TodoItem.Worker in the WorkPlan).
	Name() string

	// HardBlocker returns true if a failure in this worker must abort the entire
	// run (status → failed). Soft blockers (false) allow the run to continue
	// with status completed_with_warnings.
	HardBlocker() bool

	// Run executes the worker. It reads its required inputs from s, writes its
	// outputs back into s, and persists any artifacts via s.AW.
	// Returning a non-nil error triggers the hard/soft blocker logic in the Graph.
	Run(ctx context.Context, s *State) error
}

// State threads all mutable execution data through the worker graph.
//
// Fields are populated progressively: each worker reads from fields set by its
// upstream dependencies and writes to its own output field. The graph guarantees
// that stages run in topological order, so downstream workers never race with
// their predecessors.
//
// The only concurrent writes occur during the parallel critic/tester stage;
// those workers write to separate fields (Risks vs Tests), but shared state
// (Warnings, Plan.Todos) is guarded by mu.
type State struct {
	mu sync.Mutex

	// Set once before the graph starts; never written by workers.
	Ctx   context.Context
	Opts  deeprun.Options
	AW    *artifact.Writer
	Files []string // file refs extracted from the source material

	// Populated progressively (one writer per field, set before graph moves on).
	Plan   *deepplan.WorkPlan   // written by planner
	Patch  *deepschema.PatchDraft // written by patcher
	Risks  *deepschema.RiskReport // written by critic  (may be nil on soft failure)
	Tests  *deepschema.TestPlan   // written by tester  (may be nil on soft failure)
	Bundle *deepschema.PatchBundle // written by reconciler

	// Accumulates messages from soft-blocker failures.
	Warnings []string
}

// AddWarning appends msg to Warnings. Safe to call from concurrent workers.
func (s *State) AddWarning(msg string) {
	s.mu.Lock()
	s.Warnings = append(s.Warnings, msg)
	s.mu.Unlock()
}

// MarkTodo updates the status of every TodoItem whose Worker field equals
// workerName, then re-persists plan.json. Safe to call from concurrent workers.
func (s *State) MarkTodo(workerName string, status deepplan.TodoStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Plan == nil {
		return
	}
	for i := range s.Plan.Todos {
		if s.Plan.Todos[i].Worker == workerName {
			s.Plan.Todos[i].Status = status
		}
	}
	_ = s.AW.WriteJSON("plan.json", s.Plan)
}
