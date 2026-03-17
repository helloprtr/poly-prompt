package capsule_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/helloprtr/poly-prompt/internal/capsule"
)

func newTestCapsule(id, label, kind string) capsule.Capsule {
	now := time.Now().UTC()
	return capsule.Capsule{
		ID:        id,
		Label:     label,
		Kind:      kind,
		CreatedAt: now,
		UpdatedAt: now,
		Repo: capsule.RepoState{
			Root:    "/test/repo",
			Name:    "repo",
			Branch:  "main",
			HeadSHA: "abc1234",
		},
		Work: capsule.WorkState{
			OriginalRequest: "test request",
			NormalizedGoal:  "test goal",
		},
	}
}

func TestStoreSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	store := capsule.NewStore(dir)

	c := newTestCapsule("cap_001", "test label", capsule.KindManual)
	if err := store.Save(c); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load("cap_001")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Label != "test label" {
		t.Errorf("Label: got %q, want %q", loaded.Label, "test label")
	}
	if loaded.Kind != capsule.KindManual {
		t.Errorf("Kind: got %q, want %q", loaded.Kind, capsule.KindManual)
	}
}

func TestStoreList(t *testing.T) {
	dir := t.TempDir()
	store := capsule.NewStore(dir)

	for _, id := range []string{"cap_001", "cap_002", "cap_003"} {
		c := newTestCapsule(id, id, capsule.KindManual)
		if err := store.Save(c); err != nil {
			t.Fatalf("Save %s: %v", id, err)
		}
	}

	list, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("List count: got %d, want 3", len(list))
	}
}

func TestStoreLatest(t *testing.T) {
	dir := t.TempDir()
	store := capsule.NewStore(dir)

	t1 := time.Now().UTC().Add(-time.Hour)
	t2 := time.Now().UTC()

	c1 := newTestCapsule("cap_001", "old", capsule.KindManual)
	c1.CreatedAt = t1
	c2 := newTestCapsule("cap_002", "new", capsule.KindManual)
	c2.CreatedAt = t2

	_ = store.Save(c1)
	_ = store.Save(c2)

	latest, err := store.Latest()
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if latest.ID != "cap_002" {
		t.Errorf("Latest ID: got %q, want %q", latest.ID, "cap_002")
	}
}

func TestStoreLatestEmpty(t *testing.T) {
	dir := t.TempDir()
	store := capsule.NewStore(dir)
	_, err := store.Latest()
	if err == nil {
		t.Error("Latest on empty store should return error")
	}
}

func TestStoreDelete(t *testing.T) {
	dir := t.TempDir()
	store := capsule.NewStore(dir)

	c := newTestCapsule("cap_001", "to delete", capsule.KindManual)
	_ = store.Save(c)
	if err := store.Delete("cap_001"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := store.Load("cap_001")
	if err == nil {
		t.Error("Load after delete should return error")
	}
}

// Note: TestStoreSummaryFileWritten is added in Task 2.3 after renderSummaryMD exists.

func TestStoreSummaryFileWritten(t *testing.T) {
	dir := t.TempDir()
	store := capsule.NewStore(dir)

	c := newTestCapsule("cap_001", "test", capsule.KindManual)
	_ = store.Save(c)

	summaryPath := filepath.Join(dir, "cap_001", "summary.md")
	if _, err := os.Stat(summaryPath); err != nil {
		t.Errorf("summary.md not written: %v", err)
	}
}
