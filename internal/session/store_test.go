package session_test

import (
	"testing"
	"time"

	"github.com/helloprtr/poly-prompt/internal/session"
)

func TestSession_ZeroValueSafe(t *testing.T) {
	s := session.Session{
		ID:        "s_test",
		TaskGoal:  "fix auth bug",
		Mode:      session.ModeEdit,
		Status:    session.StatusActive,
		StartedAt: time.Now().UTC(),
	}
	if s.Mode != session.ModeEdit {
		t.Errorf("expected ModeEdit, got %v", s.Mode)
	}
	if s.Status != session.StatusActive {
		t.Errorf("expected StatusActive, got %v", s.Status)
	}
	if s.Files != nil && len(s.Files) != 0 {
		t.Errorf("expected nil/empty files, got %v", s.Files)
	}
}
