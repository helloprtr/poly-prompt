package lastresponse

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Entry is the JSON structure stored in last-response.json.
type Entry struct {
	CapturedAt time.Time `json:"captured_at"`
	Source     string    `json:"source"` // "terminal" or "clipboard"
	Response   string    `json:"response"`
}

// Store reads and writes last-response.json at a fixed path.
type Store struct {
	path string
}

// New returns a Store backed by the given file path.
func New(path string) *Store {
	return &Store{path: path}
}

// DefaultPath returns ~/.config/prtr/last-response.json,
// respecting XDG_CONFIG_HOME if set.
func DefaultPath() (string, error) {
	base := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "prtr", "last-response.json"), nil
}

// Read returns the stored entry and true, or zero Entry and false if the
// file does not exist. Other I/O or parse errors are returned as non-nil error.
func (s *Store) Read() (Entry, bool, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Entry{}, false, nil
		}
		return Entry{}, false, fmt.Errorf("read last-response: %w", err)
	}
	var entry Entry
	if err := json.Unmarshal(data, &entry); err != nil {
		return Entry{}, false, fmt.Errorf("parse last-response: %w", err)
	}
	return entry, true, nil
}

// Path returns the file path this store reads from and writes to.
func (s *Store) Path() string {
	return s.path
}

// Write atomically stores a new entry with the current UTC time.
func (s *Store) Write(source, response string) error {
	entry := Entry{
		CapturedAt: time.Now().UTC(),
		Source:     source,
		Response:   response,
	}
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("encode last-response: %w", err)
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".last-response-*.json")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("write last-response: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("write last-response: %w", err)
	}
	return nil
}
