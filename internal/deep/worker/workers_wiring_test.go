package worker

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/helloprtr/poly-prompt/internal/deep/artifact"
	deepplan "github.com/helloprtr/poly-prompt/internal/deep/plan"
	deeprun "github.com/helloprtr/poly-prompt/internal/deep/run"
	deepschema "github.com/helloprtr/poly-prompt/internal/deep/schema"
	repoctx "github.com/helloprtr/poly-prompt/internal/repoctx"
)

// ---------------------------------------------------------------------------
// Task 1 — artifact.Writer.ReadText
// ---------------------------------------------------------------------------

func TestReadEvidenceFile(t *testing.T) {
	dir := t.TempDir()
	aw := &artifact.Writer{Root: dir}
	_ = aw.WriteText("evidence/git.diff", "diff --git a/foo.go b/foo.go\n+func New() {}")

	content, err := aw.ReadText("evidence/git.diff")
	if err != nil {
		t.Fatalf("ReadText error: %v", err)
	}
	if content == "" {
		t.Fatal("expected non-empty content")
	}
}

// ---------------------------------------------------------------------------
// Task 2 — patcherWorker uses real git diff
// ---------------------------------------------------------------------------

func TestPatcherIncludesRealGitDiff(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"evidence", "result", "workers/planner", "workers/patcher",
		"workers/critic", "workers/tester", "workers/reconciler"} {
		_ = os.MkdirAll(filepath.Join(dir, sub), 0o755)
	}
	aw := &artifact.Writer{Root: dir}

	realDiff := "diff --git a/main.go b/main.go\n--- a/main.go\n+++ b/main.go\n@@ -1 +1 @@\n-old\n+new"
	_ = aw.WriteText("evidence/git.diff", realDiff)

	plan := makePlan(t, aw)
	s := &State{
		AW:    aw,
		Opts:  deeprun.Options{Source: "fix nil panic in main.go", Action: "patch"},
		Files: []string{"main.go"},
		Plan:  plan,
	}
	w := &patcherWorker{}
	if err := w.Run(context.Background(), s); err != nil {
		t.Fatalf("patcher.Run: %v", err)
	}
	if s.Patch == nil {
		t.Fatal("Patch is nil")
	}
	if !strings.Contains(s.Patch.Diff, "-old") || !strings.Contains(s.Patch.Diff, "+new") {
		t.Errorf("Diff does not contain real git diff content:\n%s", s.Patch.Diff)
	}
}

// helper: write a minimal plan.json and return a WorkPlan
func makePlan(t *testing.T, aw *artifact.Writer) *deepplan.WorkPlan {
	t.Helper()
	plan := &deepplan.WorkPlan{Version: 1, Action: "patch", ResultType: "PatchBundle",
		Summary: "test plan"}
	_ = aw.WriteJSON("plan.json", plan)
	return plan
}

// ---------------------------------------------------------------------------
// Task 3 — patcherWorker includes actual Changes list
// ---------------------------------------------------------------------------

func TestPatcherIncludesChangedFilesList(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"evidence", "result", "workers/planner", "workers/patcher",
		"workers/critic", "workers/tester", "workers/reconciler"} {
		_ = os.MkdirAll(filepath.Join(dir, sub), 0o755)
	}
	aw := &artifact.Writer{Root: dir}
	_ = aw.WriteText("evidence/git.diff", "")

	plan := makePlan(t, aw)
	s := &State{
		AW: aw,
		Opts: deeprun.Options{
			Source: "fix bug",
			Action: "patch",
			RepoSummary: repoctx.Summary{
				Branch:  "fix/nil-panic",
				Changes: []string{"M internal/foo/foo.go", "M internal/foo/foo_test.go"},
			},
		},
		Files: []string{"internal/foo/foo.go"},
		Plan:  plan,
	}
	w := &patcherWorker{}
	if err := w.Run(context.Background(), s); err != nil {
		t.Fatalf("patcher.Run: %v", err)
	}
	found := false
	for _, c := range s.Patch.Constraints {
		if strings.Contains(c, "internal/foo/foo.go") && strings.Contains(c, "internal/foo/foo_test.go") {
			found = true
		}
	}
	if !found {
		t.Errorf("Constraints do not include actual changed files list:\n%v", s.Patch.Constraints)
	}
}

// ---------------------------------------------------------------------------
// Task 4 — patcherWorker includes actual protected terms
// ---------------------------------------------------------------------------

func TestPatcherIncludesProtectedTerms(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"evidence", "result", "workers/planner", "workers/patcher",
		"workers/critic", "workers/tester", "workers/reconciler"} {
		_ = os.MkdirAll(filepath.Join(dir, sub), 0o755)
	}
	aw := &artifact.Writer{Root: dir}
	_ = aw.WriteText("evidence/git.diff", "")

	plan := makePlan(t, aw)
	s := &State{
		AW: aw,
		Opts: deeprun.Options{
			Source:         "fix bug",
			Action:         "patch",
			ProtectedTerms: []string{"ApplyPolicy", "DecisionSkipped", "PRTR_TARGET"},
		},
		Files: []string{"foo.go"},
		Plan:  plan,
	}
	w := &patcherWorker{}
	if err := w.Run(context.Background(), s); err != nil {
		t.Fatalf("patcher.Run: %v", err)
	}
	found := false
	for _, n := range s.Patch.ImplementationNotes {
		if strings.Contains(n, "ApplyPolicy") && strings.Contains(n, "PRTR_TARGET") {
			found = true
		}
	}
	if !found {
		t.Errorf("ImplementationNotes do not list protected terms:\n%v", s.Patch.ImplementationNotes)
	}
}

// ---------------------------------------------------------------------------
// Task 5 — testerWorker references existing test file
// ---------------------------------------------------------------------------

func TestTesterReferencesExistingTestFile(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"evidence", "result", "workers/planner", "workers/patcher",
		"workers/critic", "workers/tester", "workers/reconciler"} {
		_ = os.MkdirAll(filepath.Join(dir, sub), 0o755)
	}
	aw := &artifact.Writer{Root: dir}

	// Simulate repo root with a test file present
	repoRoot := t.TempDir()
	srcDir := filepath.Join(repoRoot, "internal", "translate")
	_ = os.MkdirAll(srcDir, 0o755)
	_ = os.WriteFile(filepath.Join(srcDir, "policy_test.go"), []byte("package translate"), 0o644)

	patch := &deepschema.PatchDraft{
		Summary:      "fix nil",
		TouchedFiles: []string{"internal/translate/policy.go"},
		Diff:         "",
	}
	s := &State{
		AW:    aw,
		Opts:  deeprun.Options{Source: "fix nil panic", Action: "patch", RepoRoot: repoRoot},
		Files: []string{"internal/translate/policy.go"},
		Patch: patch,
	}
	w := &testerWorker{}
	if err := w.Run(context.Background(), s); err != nil {
		t.Fatalf("tester.Run: %v", err)
	}
	found := false
	for _, c := range s.Tests.TestCases {
		if strings.Contains(c, "policy_test.go") {
			found = true
		}
	}
	if !found {
		t.Errorf("TestCases do not reference existing test file:\n%v", s.Tests.TestCases)
	}
}

// ---------------------------------------------------------------------------
// Task 6 — summarize extracts first meaningful sentence
// ---------------------------------------------------------------------------

func TestSummarizeExtractsFirstMeaningfulSentence(t *testing.T) {
	cases := []struct {
		input string
		want  string // must be contained in output
	}{
		{
			input: "ApplyPolicy에서 nil translator일 때 panic 발생.\nprotectSegments 호출 전에 nil 체크가 필요합니다.",
			want:  "ApplyPolicy",
		},
		{
			input: "\n\n  \n실제 내용은 여기서 시작합니다. 파일을 수정해야 합니다.",
			want:  "실제 내용은 여기서 시작합니다",
		},
		{
			input: "",
			want:  "Convert the source answer",
		},
	}
	for _, tc := range cases {
		got := summarize(tc.input)
		if !strings.Contains(got, tc.want) {
			t.Errorf("summarize(%q) = %q, want it to contain %q", tc.input, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Task 7 — criticWorker detects source-specific risks
// ---------------------------------------------------------------------------

func TestCriticDetectsAuthRisk(t *testing.T) {
	risks := detectSourceRisks("Fix the JWT token validation to reject expired tokens", []string{"internal/auth/auth.go"})
	found := false
	for _, r := range risks {
		if strings.Contains(r.Title, "Auth") {
			found = true
		}
	}
	if !found {
		t.Errorf("detectSourceRisks did not detect auth risk; got: %v", risks)
	}
}

func TestCriticDetectsNilRisk(t *testing.T) {
	cases := detectSourceRisks("Fix nil pointer panic in handler", nil)
	// Without specific keyword → should produce Behavior Drift fallback
	if len(cases) == 0 {
		t.Error("detectSourceRisks returned empty slice")
	}
}

func TestCriticFallbackBehaviorDrift(t *testing.T) {
	risks := detectSourceRisks("improve the documentation", nil)
	found := false
	for _, r := range risks {
		if strings.Contains(r.Title, "Behavior Drift") {
			found = true
		}
	}
	if !found {
		t.Errorf("detectSourceRisks should fall back to Behavior Drift; got: %v", risks)
	}
}
