package deep

import (
	"testing"
)

func TestPipelineModelStages(t *testing.T) {
	m := NewPipelineModel([]string{"planner", "patcher", "critic", "tester", "reconciler"})
	if len(m.Stages()) != 5 {
		t.Errorf("expected 5 stages, got %d", len(m.Stages()))
	}
	for _, s := range m.Stages() {
		if s.State != StagePending {
			t.Errorf("stage %q: expected pending, got %v", s.Name, s.State)
		}
	}
}

func TestPipelineModelAdvance(t *testing.T) {
	m := NewPipelineModel([]string{"planner", "patcher"})
	m2, _ := m.Advance("planner")
	stages := m2.Stages()
	if stages[0].State != StageRunning {
		t.Errorf("expected planner=running after Advance, got %v", stages[0].State)
	}
	m3, _ := m2.Complete("planner")
	stages3 := m3.Stages()
	if stages3[0].State != StageDone {
		t.Errorf("expected planner=done after Complete, got %v", stages3[0].State)
	}
}
