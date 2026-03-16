package worker

// factories.go provides exported constructor functions for the five concrete
// workers and two test-helper worker types (error and call-tracking).
//
// These exist primarily to support integration tests in the parent `deep`
// package that need to inject stub workers via NewPatchGraphWith without
// exposing the unexported concrete types.

import (
	"context"
	"fmt"
	"sync"
)

// NewDefaultPlannerWorker returns the production planner worker.
func NewDefaultPlannerWorker() Worker { return &plannerWorker{} }

// NewDefaultPatcherWorker returns the production patcher worker.
func NewDefaultPatcherWorker() Worker { return &patcherWorker{} }

// NewDefaultCriticWorker returns the production critic worker.
func NewDefaultCriticWorker() Worker { return &criticWorker{} }

// NewDefaultTesterWorker returns the production tester worker.
func NewDefaultTesterWorker() Worker { return &testerWorker{} }

// NewDefaultReconcilerWorker returns the production reconciler worker.
func NewDefaultReconcilerWorker() Worker { return &reconcilerWorker{} }

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// NewErrorWorker returns a Worker that always returns err when Run is called.
// hard controls whether it is a hard blocker or soft blocker.
func NewErrorWorker(name string, hard bool, err error) Worker {
	return &errorWorker{name: name, hard: hard, err: err}
}

type errorWorker struct {
	name string
	hard bool
	err  error
}

func (w *errorWorker) Name() string      { return w.name }
func (w *errorWorker) HardBlocker() bool { return w.hard }
func (w *errorWorker) Run(_ context.Context, _ *State) error {
	return fmt.Errorf("%s: %w", w.name, w.err)
}

// NewCallTrackingWorker returns a Worker that appends its name to called on
// each Run invocation, then returns nil (success). This is used in tests to
// verify that downstream workers are NOT called after an upstream hard failure.
func NewCallTrackingWorker(name string, hard bool, called *[]string) Worker {
	return &callTrackingWorker{name: name, hard: hard, called: called}
}

type callTrackingWorker struct {
	name   string
	hard   bool
	called *[]string
	mu     sync.Mutex
}

func (w *callTrackingWorker) Name() string      { return w.name }
func (w *callTrackingWorker) HardBlocker() bool { return w.hard }
func (w *callTrackingWorker) Run(_ context.Context, _ *State) error {
	w.mu.Lock()
	*w.called = append(*w.called, w.name)
	w.mu.Unlock()
	return nil
}
