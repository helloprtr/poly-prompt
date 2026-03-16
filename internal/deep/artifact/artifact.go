package artifact

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	deeprun "github.com/helloprtr/poly-prompt/internal/deep/run"
	"github.com/helloprtr/poly-prompt/internal/history"
)

// Writer manages the on-disk artifact directory for a single deep run.
//
// Directory layout:
//
//	<root>/
//	  manifest.json   – canonical run record (updated throughout the run)
//	  lineage.json    – provenance: parent history, source kind, target app
//	  plan.json       – serialised WorkPlan
//	  events.jsonl    – append-only structured event log
//	  evidence/       – read-only inputs (repo context, history, memory, git diff)
//	  workers/        – per-worker request.json + result.json pairs
//	  result/         – final output artifacts
type Writer struct {
	Root string
}

// New resolves the artifact root for runID and returns a Writer.
// If repoRoot is non-empty the root is <repoRoot>/.prtr/runs/<runID>.
// Otherwise it falls back to the prtr history directory.
func New(repoRoot, runID string) (*Writer, error) {
	root, err := resolveRoot(repoRoot, runID)
	if err != nil {
		return nil, err
	}
	return &Writer{Root: root}, nil
}

// Init creates the canonical directory structure for a run.
// workerNames lists the worker subdirectories to pre-create under workers/.
func (w *Writer) Init(workerNames []string) error {
	dirs := []string{
		w.Root,
		filepath.Join(w.Root, "evidence"),
		filepath.Join(w.Root, "result"),
	}
	for _, name := range workerNames {
		dirs = append(dirs, filepath.Join(w.Root, "workers", name))
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("init artifact dir %s: %w", dir, err)
		}
	}
	return nil
}

// WriteManifest serialises r as manifest.json – the canonical record for this run.
// Callers should call this after every status transition.
func (w *Writer) WriteManifest(r deeprun.DeepRun) error {
	return w.WriteJSON("manifest.json", r)
}

// WriteJSON writes value as indented JSON to <root>/<rel>, creating parent dirs.
func (w *Writer) WriteJSON(rel string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", rel, err)
	}
	return w.WriteText(rel, string(data)+"\n")
}

// WriteText writes content to <root>/<rel>, creating parent dirs as needed.
func (w *Writer) WriteText(rel, content string) error {
	path := filepath.Join(w.Root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create directory for %s: %w", rel, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", rel, err)
	}
	return nil
}

// Path returns the absolute path for a relative artifact reference.
func (w *Writer) Path(rel string) string {
	return filepath.Join(w.Root, rel)
}

// EventLogPath returns the canonical path for the run's events.jsonl file.
func (w *Writer) EventLogPath() string {
	return filepath.Join(w.Root, "events.jsonl")
}

func resolveRoot(repoRoot, runID string) (string, error) {
	if strings.TrimSpace(repoRoot) != "" {
		return filepath.Join(repoRoot, ".prtr", "runs", runID), nil
	}
	historyPath, err := history.DefaultPath()
	if err != nil {
		return "", fmt.Errorf("resolve deep run storage: %w", err)
	}
	return filepath.Join(filepath.Dir(historyPath), "runs", runID), nil
}
