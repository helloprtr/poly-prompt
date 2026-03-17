package capsule

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var ErrNotFound = errors.New("capsule not found")

// Store manages capsules under <repoRoot>/.prtr/capsules/.
type Store struct {
	root string // absolute path to the capsules directory
}

// NewStore returns a Store rooted at dir (the capsules directory, not repo root).
func NewStore(dir string) *Store {
	return &Store{root: dir}
}

// DefaultDir resolves the capsules directory for the given repo root.
// Creates the directory if it does not exist.
func DefaultDir(repoRoot string) (string, error) {
	dir := filepath.Join(repoRoot, ".prtr", "capsules")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create capsules dir: %w", err)
	}
	return dir, nil
}

// Save writes capsule.json and summary.md to <root>/<id>/.
func (s *Store) Save(c Capsule) error {
	dir := filepath.Join(s.root, c.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create capsule dir: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("encode capsule: %w", err)
	}
	if err := writeAtomic(filepath.Join(dir, "capsule.json"), data); err != nil {
		return err
	}

	summary := renderSummaryMD(c)
	if err := writeAtomic(filepath.Join(dir, "summary.md"), []byte(summary)); err != nil {
		return err
	}

	return nil
}

// Load reads and parses capsule.json for the given id.
func (s *Store) Load(id string) (Capsule, error) {
	path := filepath.Join(s.root, id, "capsule.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Capsule{}, ErrNotFound
		}
		return Capsule{}, fmt.Errorf("read capsule %s: %w", id, err)
	}
	var c Capsule
	if err := json.Unmarshal(data, &c); err != nil {
		return Capsule{}, fmt.Errorf("parse capsule %s: %w", id, err)
	}
	return c, nil
}

// List returns all capsules sorted by CreatedAt descending (newest first).
func (s *Store) List() ([]Capsule, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("list capsules: %w", err)
	}

	var caps []Capsule
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "cap_") {
			continue
		}
		c, err := s.Load(e.Name())
		if err != nil {
			continue // skip corrupt entries silently
		}
		caps = append(caps, c)
	}

	sort.Slice(caps, func(i, j int) bool {
		return caps[i].CreatedAt.After(caps[j].CreatedAt)
	})

	return caps, nil
}

// Latest returns the most recently created capsule.
func (s *Store) Latest() (Capsule, error) {
	caps, err := s.List()
	if err != nil {
		return Capsule{}, err
	}
	if len(caps) == 0 {
		return Capsule{}, ErrNotFound
	}
	return caps[0], nil
}

// Delete removes the capsule directory for id.
func (s *Store) Delete(id string) error {
	dir := filepath.Join(s.root, id)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("delete capsule %s: %w", id, err)
	}
	return nil
}

// Update loads a capsule, applies fn, and saves it back (same id, bumped UpdatedAt).
func (s *Store) Update(id string, fn func(*Capsule)) error {
	c, err := s.Load(id)
	if err != nil {
		return err
	}
	fn(&c)
	c.UpdatedAt = time.Now().UTC()
	return s.Save(c)
}

// NewID generates a unique capsule ID using current UnixNano.
func NewID() string {
	return fmt.Sprintf("cap_%d", time.Now().UTC().UnixNano())
}

func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".cap-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("commit file: %w", err)
	}
	return nil
}

// renderSummaryMD is defined in store.go after Task 2.3 adds the real implementation.
// Temporary stub to allow compilation:
func renderSummaryMD(c Capsule) string { return "" }
