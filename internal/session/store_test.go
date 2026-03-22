package session_test

import (
	"testing"
	"time"

	"github.com/helloprtr/poly-prompt/internal/session"
)

func TestStore_SaveAndResolve(t *testing.T) {
	dir := t.TempDir()
	store := session.NewStore(dir)

	repoHash := "aabbccdd"
	s := session.Session{
		ID:           "s_001",
		Repo:         "/tmp/myapp",
		RepoHash:     repoHash,
		TaskGoal:     "refactor auth",
		Mode:         session.ModeEdit,
		Status:       session.StatusActive,
		StartedAt:    time.Now().UTC(),
		LastActivity: time.Now().UTC(),
	}

	if err := store.Save(s); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.ActiveFor(repoHash)
	if err != nil {
		t.Fatalf("ActiveFor: %v", err)
	}
	if got.ID != s.ID {
		t.Errorf("expected ID %q, got %q", s.ID, got.ID)
	}
}

func TestStore_ActiveFor_None(t *testing.T) {
	dir := t.TempDir()
	store := session.NewStore(dir)

	_, err := store.ActiveFor("nonexistent")
	if err != session.ErrNoActiveSession {
		t.Errorf("expected ErrNoActiveSession, got %v", err)
	}
}

func TestStore_ActiveFor_MultiplePicksLatest(t *testing.T) {
	dir := t.TempDir()
	store := session.NewStore(dir)
	hash := "aabbccdd"

	older := session.Session{
		ID: "s_old", RepoHash: hash,
		Status:       session.StatusActive,
		LastActivity: time.Now().Add(-2 * time.Hour),
		StartedAt:    time.Now().Add(-3 * time.Hour),
	}
	newer := session.Session{
		ID: "s_new", RepoHash: hash,
		Status:       session.StatusActive,
		LastActivity: time.Now().Add(-10 * time.Minute),
		StartedAt:    time.Now().Add(-1 * time.Hour),
	}

	_ = store.Save(older)
	_ = store.Save(newer)

	got, err := store.ActiveFor(hash)
	if err != nil {
		t.Fatalf("ActiveFor: %v", err)
	}
	if got.ID != "s_new" {
		t.Errorf("expected newest session s_new, got %q", got.ID)
	}
}

func TestStore_List(t *testing.T) {
	dir := t.TempDir()
	store := session.NewStore(dir)

	for i, id := range []string{"s_1", "s_2", "s_3"} {
		_ = store.Save(session.Session{
			ID: id, RepoHash: "abc",
			Status:       session.StatusActive,
			LastActivity: time.Now().Add(time.Duration(i) * time.Minute),
			StartedAt:    time.Now(),
		})
	}

	sessions, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(sessions) != 3 {
		t.Errorf("expected 3 sessions, got %d", len(sessions))
	}
}

func TestStore_Complete(t *testing.T) {
	dir := t.TempDir()
	store := session.NewStore(dir)
	hash := "deadbeef"

	s := session.Session{
		ID: "s_done", RepoHash: hash,
		Status:       session.StatusActive,
		LastActivity: time.Now(),
		StartedAt:    time.Now(),
	}
	_ = store.Save(s)

	if err := store.Complete(s); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	// Should no longer be active
	_, err := store.ActiveFor(hash)
	if err != session.ErrNoActiveSession {
		t.Errorf("expected ErrNoActiveSession after Complete, got %v", err)
	}

	// Should still appear in List
	all, _ := store.List()
	var found bool
	for _, sess := range all {
		if sess.ID == "s_done" && sess.Status == session.StatusCompleted {
			found = true
		}
	}
	if !found {
		t.Error("completed session not found in List")
	}
}

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
