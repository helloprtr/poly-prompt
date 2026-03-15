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
	TargetSource        string    `json:"target_source,omitempty"`
	TargetReason        string    `json:"target_reason,omitempty"`
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
	Action              string    `json:"action,omitempty"`
	SourceKind          string    `json:"source_kind,omitempty"`
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
		entries = entries[len(entries)-maxEntries:]
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

	latest := entries[0]
	for _, entry := range entries[1:] {
		if entry.CreatedAt.After(latest.CreatedAt) {
			latest = entry
		}
	}

	return latest, nil
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
			entry.TargetSource,
			entry.TargetReason,
			entry.Shortcut,
			entry.Action,
			entry.SourceKind,
			entry.TemplatePreset,
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

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create history directory: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return fmt.Errorf("write history: %w", err)
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
