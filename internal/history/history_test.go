package history

import (
	"path/filepath"
	"testing"
)

func TestStoreAppendListAndGet(t *testing.T) {
	t.Parallel()

	store := New(filepath.Join(t.TempDir(), "history.json"))

	if err := store.Append(Entry{
		ID:             "1",
		Original:       "원문",
		Translated:     "translated",
		FinalPrompt:    "final",
		Target:         "claude",
		Role:           "be",
		TemplatePreset: "claude-structured",
	}); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	entries, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}

	entry, err := store.Get("1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if entry.FinalPrompt != "final" {
		t.Fatalf("FinalPrompt = %q, want %q", entry.FinalPrompt, "final")
	}
}
