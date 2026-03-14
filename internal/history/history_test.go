package history

import (
	"path/filepath"
	"testing"
	"time"
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
		DeliveryMode:   "open-copy",
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

func TestStoreSearchAndToggleFlags(t *testing.T) {
	t.Parallel()

	store := New(filepath.Join(t.TempDir(), "history.json"))
	if err := store.Append(Entry{
		ID:                  "1",
		Original:            "한국어 리뷰 요청",
		Translated:          "Review this change",
		FinalPrompt:         "Review this change carefully",
		Target:              "claude",
		Role:                "review",
		TemplatePreset:      "claude-review",
		TranslationDecision: "translated",
		DeliveryMode:        "open-copy-paste",
		Pasted:              true,
	}); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	results, err := store.Search("review")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}

	entry, err := store.TogglePinned("1")
	if err != nil {
		t.Fatalf("TogglePinned() error = %v", err)
	}
	if !entry.Pinned {
		t.Fatalf("Pinned = %v, want true", entry.Pinned)
	}

	entry, err = store.ToggleFavorite("1")
	if err != nil {
		t.Fatalf("ToggleFavorite() error = %v", err)
	}
	if !entry.Favorite {
		t.Fatalf("Favorite = %v, want true", entry.Favorite)
	}
}

func TestStoreLatestReturnsMostRecentEntry(t *testing.T) {
	t.Parallel()

	store := New(filepath.Join(t.TempDir(), "history.json"))
	if err := store.Append(Entry{
		ID:        "older",
		CreatedAt: time.Unix(100, 0).UTC(),
		Original:  "first",
		Target:    "claude",
	}); err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if err := store.Append(Entry{
		ID:        "newer",
		CreatedAt: time.Unix(200, 0).UTC(),
		Original:  "second",
		Target:    "codex",
		Pinned:    true,
	}); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	entry, err := store.Latest()
	if err != nil {
		t.Fatalf("Latest() error = %v", err)
	}
	if entry.ID != "newer" {
		t.Fatalf("Latest() ID = %q, want %q", entry.ID, "newer")
	}
}
