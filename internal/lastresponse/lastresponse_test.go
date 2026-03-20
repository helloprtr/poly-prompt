package lastresponse_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/helloprtr/poly-prompt/internal/lastresponse"
)

func TestStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := lastresponse.New(filepath.Join(dir, "last-response.json"))

	if err := store.Write("clipboard", "AI said hello"); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	entry, ok, err := store.Read()
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if !ok {
		t.Fatal("Read() ok = false, want true")
	}
	if entry.Source != "clipboard" {
		t.Errorf("Source = %q, want %q", entry.Source, "clipboard")
	}
	if entry.Response != "AI said hello" {
		t.Errorf("Response = %q, want %q", entry.Response, "AI said hello")
	}
	if entry.CapturedAt.IsZero() {
		t.Error("CapturedAt is zero")
	}
}

func TestStoreReadMissingFile(t *testing.T) {
	dir := t.TempDir()
	store := lastresponse.New(filepath.Join(dir, "nonexistent.json"))

	_, ok, err := store.Read()
	if err != nil {
		t.Fatalf("Read() error = %v, want nil", err)
	}
	if ok {
		t.Error("Read() ok = true, want false for missing file")
	}
}

func TestStoreAge(t *testing.T) {
	dir := t.TempDir()
	store := lastresponse.New(filepath.Join(dir, "last-response.json"))
	if err := store.Write("terminal", "response text"); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	entry, _, err := store.Read()
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	age := time.Since(entry.CapturedAt)
	if age > 5*time.Second {
		t.Errorf("age = %v, expected < 5s", age)
	}
}

func TestDefaultPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	path, err := lastresponse.DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath() error = %v", err)
	}
	want := filepath.Join(dir, "prtr", "last-response.json")
	if path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
}
