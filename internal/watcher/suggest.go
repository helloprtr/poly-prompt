// internal/watcher/suggest.go
package watcher

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Suggestion is the payload written to the suggest file.
type Suggestion struct {
	Action       string   `json:"action"`
	ContextLines []string `json:"context_lines"`
	Branch       string   `json:"branch"`
}

// WriteSuggest atomically writes a suggestion to path.
func WriteSuggest(path string, s Suggestion) error {
	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshal suggestion: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create suggest dir: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write suggest tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename suggest: %w", err)
	}
	return nil
}

// ReadAndClearSuggest reads and removes the suggestion file.
// Returns nil, nil if file does not exist.
func ReadAndClearSuggest(path string) (*Suggestion, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read suggest: %w", err)
	}
	_ = os.Remove(path)

	var s Suggestion
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse suggestion: %w", err)
	}
	return &s, nil
}

// SuggestPath returns the canonical suggest file path.
func SuggestPath() (string, error) {
	return configPath("watch-suggest")
}

// PIDPath returns the canonical watcher PID file path.
func PIDPath() (string, error) {
	return configPath("watch.pid")
}

func configPath(name string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "prtr", name), nil
}
