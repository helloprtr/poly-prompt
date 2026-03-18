package capsule

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/helloprtr/poly-prompt/internal/history"
	"github.com/helloprtr/poly-prompt/internal/repoctx"
)

// BuildInput holds everything needed to construct a Capsule.
type BuildInput struct {
	Label        string
	Note         string
	Kind         string
	HistoryEntry *history.Entry // optional — nil if no prior run
	RepoSummary  repoctx.Summary
	RepoRoot     string
	Todos        []TodoItem // optional override; if nil, extracted from history entry
}

// Build constructs a Capsule from the given inputs.
// Missing optional inputs (HistoryEntry, Todos) result in empty/zero session/work fields.
func Build(in BuildInput) Capsule {
	now := time.Now().UTC()

	kind := in.Kind
	if kind == "" {
		kind = KindAuto
	}

	c := Capsule{
		ID:        NewID(),
		Label:     strings.TrimSpace(in.Label),
		Note:      strings.TrimSpace(in.Note),
		Kind:      kind,
		CreatedAt: now,
		UpdatedAt: now,
		Repo: RepoState{
			Root:         in.RepoRoot,
			Name:         in.RepoSummary.RepoName,
			Branch:       in.RepoSummary.Branch,
			HeadSHA:      in.RepoSummary.HeadSHA,
			TouchedFiles: changedFiles(in.RepoSummary.Changes),
		},
	}

	if in.HistoryEntry != nil {
		e := in.HistoryEntry
		c.Session = SessionState{
			TargetApp:       e.Target,
			Engine:          e.Engine,
			Mode:            e.ResultType,
			SourceHistoryID: e.ID,
			SourceRunID:     e.RunID,
			ArtifactRoot:    relativeArtifactRoot(in.RepoRoot, e.ArtifactRoot),
		}
		c.Work = WorkState{
			OriginalRequest: e.Original,
			NormalizedGoal:  NormalizeGoal(e.Original),
		}
		if len(in.Todos) > 0 {
			c.Work.Todos = in.Todos
		}
	}

	return c
}

// NormalizeGoal produces a deterministic dedup key from a request string:
// lowercase, trimmed, max 100 chars.
func NormalizeGoal(request string) string {
	s := strings.ToLower(strings.TrimSpace(request))
	if len(s) > 100 {
		s = s[:100]
	}
	return s
}

// changedFiles extracts bare file paths from git status --short output lines.
// Input lines look like: "M internal/auth/auth.go" or "?? newfile.go"
func changedFiles(changes []string) []string {
	files := make([]string, 0, len(changes))
	for _, line := range changes {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			files = append(files, parts[len(parts)-1])
		}
	}
	return files
}

// relativeArtifactRoot converts an absolute artifact root path to a path
// relative to repoRoot, for portability. Falls back to the original if
// it cannot be made relative.
func relativeArtifactRoot(repoRoot, artifactRoot string) string {
	if repoRoot == "" || artifactRoot == "" {
		return artifactRoot
	}
	rel, err := filepath.Rel(repoRoot, artifactRoot)
	if err != nil {
		return artifactRoot
	}
	return rel
}
