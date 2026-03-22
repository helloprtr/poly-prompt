// internal/session/store.go
package session

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

var ErrNoActiveSession = errors.New("no active session")

type Store struct {
	dir string
}

func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

func DefaultDir() (string, error) {
	base := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "prtr", "sessions"), nil
}

func (s *Store) Save(sess Session) error {
	if sess.ID == "" {
		sess.ID = fmt.Sprintf("s_%d", time.Now().UTC().UnixNano())
	}
	if sess.LastActivity.IsZero() {
		sess.LastActivity = time.Now().UTC()
	}

	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("create sessions directory: %w", err)
	}

	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return fmt.Errorf("encode session: %w", err)
	}

	path := s.pathFor(sess.RepoHash, sess.ID)
	tmp, err := os.CreateTemp(s.dir, ".session-*.json.tmp")
	if err != nil {
		return fmt.Errorf("create temp session file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write session: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close temp session file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("commit session: %w", err)
	}
	return nil
}

func (s *Store) ActiveFor(repoHash string) (Session, error) {
	all, err := s.List()
	if err != nil {
		return Session{}, err
	}

	var candidates []Session
	for _, sess := range all {
		if sess.RepoHash == repoHash && sess.Status == StatusActive {
			candidates = append(candidates, sess)
		}
	}
	if len(candidates) == 0 {
		return Session{}, ErrNoActiveSession
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].LastActivity.After(candidates[j].LastActivity)
	})
	return candidates[0], nil
}

func (s *Store) List() ([]Session, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read sessions directory: %w", err)
	}

	var sessions []Session
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, e.Name()))
		if err != nil {
			continue
		}
		var sess Session
		if err := json.Unmarshal(data, &sess); err != nil {
			continue
		}
		sessions = append(sessions, sess)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].LastActivity.After(sessions[j].LastActivity)
	})
	return sessions, nil
}

func (s *Store) Update(sess Session) error {
	sess.LastActivity = time.Now().UTC()
	return s.Save(sess)
}

func (s *Store) Complete(sess Session) error {
	sess.Status = StatusCompleted
	sess.LastActivity = time.Now().UTC()
	return s.Save(sess)
}

func (s *Store) pathFor(repoHash, id string) string {
	return filepath.Join(s.dir, fmt.Sprintf("%s-%s.json", repoHash, id))
}
