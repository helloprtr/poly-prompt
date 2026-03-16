package history

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

const maxEntries = 200

var ErrNotFound = errors.New("history entry not found")

type Entry struct {
	ID                  string    `json:"id"`
	CreatedAt           time.Time `json:"created_at"`
	Original            string    `json:"original"`
	Translated          string    `json:"translated"`
	FinalPrompt         string    `json:"final_prompt"`
	Target              string    `json:"target"`
	Role                string    `json:"role,omitempty"`
	TemplatePreset      string    `json:"template_preset,omitempty"`
	Shortcut            string    `json:"shortcut,omitempty"`
	SourceLang          string    `json:"source_lang,omitempty"`
	TargetLang          string    `json:"target_lang,omitempty"`
	TranslationMode     string    `json:"translation_mode,omitempty"`
	TranslationDecision string    `json:"translation_decision,omitempty"`
	LaunchedTarget      string    `json:"launched_target,omitempty"`
	DeliveryMode        string    `json:"delivery_mode,omitempty"`
	Pasted              bool      `json:"pasted,omitempty"`
	SubmitMode          string    `json:"submit_mode,omitempty"`
	Submitted           bool      `json:"submitted,omitempty"`
	ParentID            string    `json:"parent_id,omitempty"`
	RunID               string    `json:"run_id,omitempty"`
	Engine              string    `json:"engine,omitempty"`
	ResultType          string    `json:"result_type,omitempty"`
	ArtifactRoot        string    `json:"artifact_root,omitempty"`
	RunStatus           string    `json:"run_status,omitempty"`
	Pinned              bool      `json:"pinned,omitempty"`
	Favorite            bool      `json:"favorite,omitempty"`
}

type Store struct {
	path string
}

func New(path string) *Store {
	return &Store{path: path}
}

func DefaultPath() (string, error) {
	base := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		base = filepath.Join(home, ".config")
	}

	return filepath.Join(base, "prtr", "history.json"), nil
}

func (s *Store) Append(entry Entry) error {
	entries, err := s.load()
	if err != nil {
		return err
	}

	if strings.TrimSpace(entry.ID) == "" {
		entry.ID = fmt.Sprintf("%d", time.Now().UTC().UnixNano())
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}

	entries = append(entries, entry)
	if len(entries) > maxEntries {
		// Separate pinned and unpinned; preserve all pinned entries.
		pinned := make([]Entry, 0)
		unpinned := make([]Entry, 0)
		for _, e := range entries {
			if e.Pinned {
				pinned = append(pinned, e)
			} else {
				unpinned = append(unpinned, e)
			}
		}
		// Keep only as many unpinned as there's room for
		allowedUnpinned := maxEntries - len(pinned)
		if allowedUnpinned < 0 {
			allowedUnpinned = 0
		}
		if len(unpinned) > allowedUnpinned {
			unpinned = unpinned[len(unpinned)-allowedUnpinned:]
		}
		// Merge back in timestamp order
		entries = make([]Entry, 0, len(pinned)+len(unpinned))
		pi, ui := 0, 0
		for pi < len(pinned) || ui < len(unpinned) {
			switch {
			case pi >= len(pinned):
				entries = append(entries, unpinned[ui])
				ui++
			case ui >= len(unpinned):
				entries = append(entries, pinned[pi])
				pi++
			case pinned[pi].CreatedAt.Before(unpinned[ui].CreatedAt):
				entries = append(entries, pinned[pi])
				pi++
			default:
				entries = append(entries, unpinned[ui])
				ui++
			}
		}
	}

	return s.save(entries)
}

func (s *Store) List() ([]Entry, error) {
	entries, err := s.load()
	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Pinned != entries[j].Pinned {
			return entries[i].Pinned
		}
		return entries[i].CreatedAt.After(entries[j].CreatedAt)
	})

	return entries, nil
}

func (s *Store) Get(id string) (Entry, error) {
	entries, err := s.load()
	if err != nil {
		return Entry{}, err
	}

	for _, entry := range entries {
		if entry.ID == id {
			return entry, nil
		}
	}

	return Entry{}, ErrNotFound
}

func (s *Store) Latest() (Entry, error) {
	entries, err := s.load()
	if err != nil {
		return Entry{}, err
	}
	if len(entries) == 0 {
		return Entry{}, ErrNotFound
	}
	// Entries are appended in chronological order; last is always newest.
	return entries[len(entries)-1], nil
}

func (s *Store) Search(query string) ([]Entry, error) {
	entries, err := s.List()
	if err != nil {
		return nil, err
	}

	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return entries, nil
	}

	filtered := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		haystack := strings.ToLower(strings.Join([]string{
			entry.Original,
			entry.Translated,
			entry.FinalPrompt,
			entry.Role,
			entry.Target,
			entry.Shortcut,
			entry.TemplatePreset,
			entry.Engine,
			entry.ResultType,
			entry.ArtifactRoot,
			entry.RunStatus,
		}, "\n"))
		if strings.Contains(haystack, query) {
			filtered = append(filtered, entry)
		}
	}
	return filtered, nil
}

func (s *Store) TogglePinned(id string) (Entry, error) {
	return s.update(id, func(entry *Entry) {
		entry.Pinned = !entry.Pinned
	})
}

func (s *Store) ToggleFavorite(id string) (Entry, error) {
	return s.update(id, func(entry *Entry) {
		entry.Favorite = !entry.Favorite
	})
}

func (s *Store) load() ([]Entry, error) {
	if strings.TrimSpace(s.path) == "" {
		return nil, errors.New("history path is empty")
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Entry{}, nil
		}
		return nil, fmt.Errorf("read history: %w", err)
	}
	if len(data) == 0 {
		return []Entry{}, nil
	}

	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse history: %w", err)
	}

	return entries, nil
}

func (s *Store) save(entries []Entry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("encode history: %w", err)
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create history directory: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".history-*.json.tmp")
	if err != nil {
		return fmt.Errorf("create temp history file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write history: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close temp history file: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("commit history: %w", err)
	}
	return nil
}

func (s *Store) update(id string, update func(*Entry)) (Entry, error) {
	entries, err := s.load()
	if err != nil {
		return Entry{}, err
	}

	for i := range entries {
		if entries[i].ID == id {
			update(&entries[i])
			if err := s.save(entries); err != nil {
				return Entry{}, err
			}
			return entries[i], nil
		}
	}

	return Entry{}, ErrNotFound
}
