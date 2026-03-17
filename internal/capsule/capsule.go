package capsule

import (
	"strings"
	"time"

	"github.com/helloprtr/poly-prompt/internal/repoctx"
)

// Kind distinguishes manual saves from auto-saves.
const (
	KindManual = "manual"
	KindAuto   = "auto"
)

// Capsule is the complete in-memory and on-disk record of a single Work Capsule.
// All session fields are optional — capsule.json uses omitempty throughout.
type Capsule struct {
	ID        string    `json:"id"`
	Label     string    `json:"label,omitempty"`
	Note      string    `json:"note,omitempty"`
	Kind      string    `json:"kind"`
	Pinned    bool      `json:"pinned,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Repo    RepoState    `json:"repo"`
	Session SessionState `json:"session,omitempty"`
	Work    WorkState    `json:"work"`
}

// RepoState captures git state at save time.
type RepoState struct {
	Root         string   `json:"root"`
	Name         string   `json:"name"`
	Branch       string   `json:"branch"`
	HeadSHA      string   `json:"head_sha"`
	TouchedFiles []string `json:"touched_files,omitempty"`
	DiffStat     string   `json:"diff_stat,omitempty"`
}

// SessionState links back to the history entry and deep run that preceded this save.
// All fields are optional — zero values are omitted in JSON.
type SessionState struct {
	TargetApp       string `json:"target_app,omitempty"`
	Engine          string `json:"engine,omitempty"`
	Mode            string `json:"mode,omitempty"`
	SourceHistoryID string `json:"source_history_id,omitempty"`
	SourceRunID     string `json:"source_run_id,omitempty"`
	ArtifactRoot    string `json:"artifact_root,omitempty"`
}

// WorkState holds the semantic content of the work being done.
type WorkState struct {
	OriginalRequest string     `json:"original_request,omitempty"`
	NormalizedGoal  string     `json:"normalized_goal,omitempty"`
	NextAction      string     `json:"next_action,omitempty"`
	Summary         string     `json:"summary,omitempty"`
	ProtectedTerms  []string   `json:"protected_terms,omitempty"`
	Todos           []TodoItem `json:"todos,omitempty"`
	Decisions       []string   `json:"decisions,omitempty"`
	OpenQuestions   []string   `json:"open_questions,omitempty"`
	Risks           []string   `json:"risks,omitempty"`
}

// TodoItem mirrors a single task from deep run plan.json.
type TodoItem struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"` // "pending" | "completed" | "failed"
}

// DriftReport describes how the current repo state differs from a saved capsule.
type DriftReport struct {
	BranchChanged bool
	SavedBranch   string
	CurrentBranch string
	SHAChanged    bool
	SavedSHA      string
	CurrentSHA    string
	FilesChanged  bool
}

// HasDrift returns true if any drift was detected.
func (d DriftReport) HasDrift() bool {
	return d.BranchChanged || d.SHAChanged || d.FilesChanged
}

// DetectDrift compares a saved capsule's repo state to the current repoctx summary.
func DetectDrift(c Capsule, current repoctx.Summary) DriftReport {
	// Build a set from current changed files (strip git status prefix like "M ")
	currentFiles := make(map[string]bool, len(current.Changes))
	for _, line := range current.Changes {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			currentFiles[parts[len(parts)-1]] = true
		}
	}

	// Check if any saved touched files are no longer in the current changed set
	filesChanged := false
	for _, f := range c.Repo.TouchedFiles {
		if !currentFiles[f] {
			filesChanged = true
			break
		}
	}
	// Also check if there are new files not in the saved set
	if !filesChanged && len(current.Changes) != len(c.Repo.TouchedFiles) {
		filesChanged = true
	}

	return DriftReport{
		BranchChanged: c.Repo.Branch != current.Branch,
		SavedBranch:   c.Repo.Branch,
		CurrentBranch: current.Branch,
		SHAChanged:    c.Repo.HeadSHA != current.HeadSHA,
		SavedSHA:      c.Repo.HeadSHA,
		CurrentSHA:    current.HeadSHA,
		FilesChanged:  filesChanged,
	}
}
