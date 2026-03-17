package capsule_test

import (
	"testing"
	"time"

	"github.com/helloprtr/poly-prompt/internal/capsule"
	"github.com/helloprtr/poly-prompt/internal/history"
	"github.com/helloprtr/poly-prompt/internal/repoctx"
)

func TestBuildFromInputs(t *testing.T) {
	entry := history.Entry{
		ID:           "hist_001",
		CreatedAt:    time.Now().UTC(),
		Original:     "implement JWT auth",
		Target:       "claude",
		Engine:       "deep",
		RunID:        "run_001",
		ArtifactRoot: ".prtr/runs/run_001",
	}
	summary := repoctx.Summary{
		RepoName: "myrepo",
		Branch:   "fix/auth",
		HeadSHA:  "abc1234",
		Changes:  []string{"M internal/auth/auth.go"},
	}

	in := capsule.BuildInput{
		Label:        "auth refactor paused",
		Note:         "JWT decided",
		Kind:         capsule.KindManual,
		HistoryEntry: &entry,
		RepoSummary:  summary,
		RepoRoot:     "/test/repo",
	}

	c := capsule.Build(in)

	if c.ID == "" {
		t.Error("ID should be set")
	}
	if c.Label != "auth refactor paused" {
		t.Errorf("Label: got %q", c.Label)
	}
	if c.Session.TargetApp != "claude" {
		t.Errorf("TargetApp: got %q", c.Session.TargetApp)
	}
	if c.Repo.HeadSHA != "abc1234" {
		t.Errorf("HeadSHA: got %q", c.Repo.HeadSHA)
	}
	if c.Work.NormalizedGoal == "" {
		t.Error("NormalizedGoal should not be empty")
	}
	if len(c.Session.SourceHistoryID) == 0 {
		t.Error("SourceHistoryID should be set from history entry")
	}
}

func TestBuildWithNoHistoryEntry(t *testing.T) {
	summary := repoctx.Summary{
		RepoName: "myrepo",
		Branch:   "main",
		HeadSHA:  "def5678",
	}

	in := capsule.BuildInput{
		Kind:        capsule.KindAuto,
		RepoSummary: summary,
		RepoRoot:    "/test/repo",
	}

	c := capsule.Build(in)

	if c.ID == "" {
		t.Error("ID should be set even without history entry")
	}
	if c.Session.TargetApp != "" {
		t.Errorf("TargetApp should be empty when no history entry: got %q", c.Session.TargetApp)
	}
	if c.Repo.Branch != "main" {
		t.Errorf("Branch: got %q", c.Repo.Branch)
	}
}

func TestNormalizeGoal(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Implement JWT Auth", "implement jwt auth"},
		{
			"This is a very long request that exceeds one hundred characters and should be truncated by the normalizer function",
			"this is a very long request that exceeds one hundred characters and should be truncated by the norma",
		},
		{"", ""},
	}
	for _, tt := range tests {
		got := capsule.NormalizeGoal(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeGoal(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
