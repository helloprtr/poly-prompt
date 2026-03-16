package worker

import (
	"context"
	"fmt"
	"strings"

	deepplan "github.com/helloprtr/poly-prompt/internal/deep/plan"
	deepschema "github.com/helloprtr/poly-prompt/internal/deep/schema"
)

// ---------------------------------------------------------------------------
// plannerWorker — hard blocker
// Inputs:  source.md, evidence/repo_context.json, evidence/history.json,
//          evidence/memory.json
// Outputs: WorkPlan  (written to s.Plan and plan.json)
// ---------------------------------------------------------------------------

type plannerWorker struct{}

func (plannerWorker) Name() string        { return "planner" }
func (plannerWorker) HardBlocker() bool   { return true }

func (plannerWorker) Run(ctx context.Context, s *State) error {
	todos := []deepplan.TodoItem{
		{
			ID: "plan", Title: "Capture the implementation plan", Worker: "planner",
			Status:     deepplan.TodoInProgress,
			InputRefs:  []string{"source.md", "evidence/repo_context.json", "evidence/history.json", "evidence/memory.json"},
			OutputRefs: []string{"plan.json"},
		},
		{
			ID: "patch", Title: "Draft the patch bundle", Worker: "patcher",
			Status:    deepplan.TodoPending,
			DependsOn: []string{"plan"},
			InputRefs: []string{"source.md", "plan.json", "evidence/git.diff"},
			OutputRefs: []string{"workers/patcher/result.json", "result/patch.diff"},
		},
		{
			ID: "critique", Title: "Review the draft for risks", Worker: "critic",
			Status:    deepplan.TodoPending,
			DependsOn: []string{"patch"},
			InputRefs: []string{"workers/patcher/result.json"},
			OutputRefs: []string{"workers/critic/result.json"},
		},
		{
			ID: "tests", Title: "Draft the verification plan", Worker: "tester",
			Status:    deepplan.TodoPending,
			DependsOn: []string{"patch"},
			InputRefs: []string{"workers/patcher/result.json", "evidence/memory.json"},
			OutputRefs: []string{"workers/tester/result.json", "result/tests.md"},
		},
		{
			ID: "reconcile", Title: "Package the final PatchBundle", Worker: "reconciler",
			Status:    deepplan.TodoPending,
			DependsOn: []string{"critique", "tests"},
			InputRefs: []string{"workers/patcher/result.json", "workers/critic/result.json", "workers/tester/result.json"},
			OutputRefs: []string{"workers/reconciler/result.json", "result/patch_bundle.json"},
		},
	}
	graph := []deepplan.WorkerSpec{
		{Name: "planner", Objective: "Capture a fixed todo plan.", Inputs: []string{"source.md", "evidence/repo_context.json", "evidence/history.json", "evidence/memory.json"}, OutputType: "WorkPlan", Required: true},
		{Name: "patcher", Objective: "Turn the source material into an implementation-focused draft.", Inputs: []string{"source.md", "plan.json", "evidence/git.diff"}, OutputType: "PatchDraft", DependsOn: []string{"planner"}, Required: true},
		{Name: "critic", Objective: "Identify the top risks and missing checks.", Inputs: []string{"workers/patcher/result.json"}, OutputType: "RiskReport", DependsOn: []string{"patcher"}, Required: false},
		{Name: "tester", Objective: "Draft targeted verification steps for the patch.", Inputs: []string{"workers/patcher/result.json", "evidence/memory.json"}, OutputType: "TestPlan", DependsOn: []string{"patcher"}, Required: false},
		{Name: "reconciler", Objective: "Merge the draft, risks, and tests into a typed PatchBundle.", Inputs: []string{"workers/patcher/result.json", "workers/critic/result.json", "workers/tester/result.json"}, OutputType: "PatchBundle", DependsOn: []string{"critic", "tester"}, Required: true},
	}
	plan := &deepplan.WorkPlan{
		Version:      1,
		Action:       "patch",
		ResultType:   "PatchBundle",
		Summary:      "Plan the follow-up, draft the patch, critique the risk, draft tests, and package a single PatchBundle.",
		EvidenceRefs: []string{"source.md", "evidence/repo_context.json", "evidence/history.json", "evidence/memory.json", "evidence/git.diff"},
		Todos:        todos,
		WorkerGraph:  graph,
	}
	s.Plan = plan
	if err := s.AW.WriteJSON("plan.json", plan); err != nil {
		return fmt.Errorf("persist plan: %w", err)
	}
	if err := writeWorkerRequest(s, "planner", map[string]any{
		"source":       "source.md",
		"evidence_refs": []string{"evidence/repo_context.json", "evidence/history.json", "evidence/memory.json"},
	}); err != nil {
		return err
	}
	return writeWorkerResult(s, "planner", plan)
}

// ---------------------------------------------------------------------------
// patcherWorker — hard blocker
// Inputs:  source (s.Opts.Source), WorkPlan (s.Plan), evidence
// Outputs: PatchDraft  (written to s.Patch and workers/patcher/result.json)
// ---------------------------------------------------------------------------

type patcherWorker struct{}

func (patcherWorker) Name() string      { return "patcher" }
func (patcherWorker) HardBlocker() bool { return true }

func (patcherWorker) Run(ctx context.Context, s *State) error {
	if s.Plan == nil {
		return fmt.Errorf("plan not available")
	}

	notes := []string{
		"Use the clipboard answer as source intent, not as final code.",
		"Prefer concrete file-by-file changes over abstract rewrites.",
		"Validate behavior with focused regression tests after the patch.",
	}
	if len(s.Opts.ProtectedTerms) > 0 {
		notes = append(notes,
			"Preserve these repo-specific identifiers exactly (do not rename or translate): "+
				strings.Join(s.Opts.ProtectedTerms, ", ")+".")
	}

	constraints := []string{}
	if s.Opts.RepoSummary.Branch != "" {
		constraints = append(constraints, "Respect current branch context: "+s.Opts.RepoSummary.Branch+".")
	}
	if len(s.Opts.RepoSummary.Changes) > 0 {
		// Include actual file list, not just a generic reminder.
		constraints = append(constraints,
			"Local changes in this checkout:\n  "+
				strings.Join(s.Opts.RepoSummary.Changes, "\n  "))
	}

	draft := &deepschema.PatchDraft{
		Summary:             summarize(s.Opts.Source),
		TouchedFiles:        s.Files,
		ImplementationNotes: notes,
		Constraints:         constraints,
		Diff:                buildDraftDiff(s.Files),
	}
	s.Patch = draft

	if err := s.AW.WriteText("result/patch.diff", draft.Diff+"\n"); err != nil {
		return fmt.Errorf("write patch.diff: %w", err)
	}
	if err := writeWorkerRequest(s, "patcher", map[string]any{
		"source":       "source.md",
		"plan":         "plan.json",
		"evidence_refs": []string{"evidence/repo_context.json", "evidence/history.json", "evidence/memory.json", "evidence/git.diff"},
	}); err != nil {
		return err
	}
	return writeWorkerResult(s, "patcher", draft)
}

// ---------------------------------------------------------------------------
// criticWorker — soft blocker
// Inputs:  PatchDraft (s.Patch)
// Outputs: RiskReport  (written to s.Risks and workers/critic/result.json)
// ---------------------------------------------------------------------------

type criticWorker struct{}

func (criticWorker) Name() string      { return "critic" }
func (criticWorker) HardBlocker() bool { return false }

func (criticWorker) Run(ctx context.Context, s *State) error {
	if s.Patch == nil {
		return fmt.Errorf("patch draft not available")
	}

	risks := []deepschema.RiskItem{
		{
			Title:      "Behavior drift",
			Severity:   "high",
			Detail:     "The follow-up may change behavior beyond the user's requested fix if the source answer mixed diagnosis and implementation.",
			Confidence: "medium",
		},
		{
			Title:      "Local context mismatch",
			Severity:   "medium",
			Detail:     "The suggested change may not align with the current repository state, especially if the clipboard answer was produced against an older checkout.",
			Confidence: "medium",
		},
		{
			Title:      "Test gap",
			Severity:   "medium",
			Detail:     "The patch may look plausible without adding a regression check for the user-visible failure path.",
			Confidence: "high",
		},
	}
	if len(s.Files) > 0 && s.Files[0] != "<inspect-changed-files>" {
		risks[1].Detail = "Review the existing logic in " + s.Files[0] + " before applying broad changes."
	}

	missing := []string{
		"Confirm the exact file and symbol names before editing.",
		"Check whether the clipboard answer assumed code that no longer exists.",
		"Run the smallest regression test that reproduces the original problem.",
	}
	src := strings.ToLower(s.Opts.Source)
	if strings.Contains(src, "migration") || strings.Contains(src, "schema") {
		missing = append(missing, "Validate migration and compatibility behavior separately from code edits.")
	}

	report := &deepschema.RiskReport{
		TopRisks:      risks,
		MissingChecks: missing,
		OverallRisk:   "medium",
	}
	s.Risks = report

	if err := writeWorkerRequest(s, "critic", map[string]any{
		"patch_draft": "workers/patcher/result.json",
	}); err != nil {
		return err
	}
	return writeWorkerResult(s, "critic", report)
}

// ---------------------------------------------------------------------------
// testerWorker — soft blocker
// Inputs:  PatchDraft (s.Patch), testing norms from evidence/memory.json
// Outputs: TestPlan  (written to s.Tests and workers/tester/result.json)
// ---------------------------------------------------------------------------

type testerWorker struct{}

func (testerWorker) Name() string      { return "tester" }
func (testerWorker) HardBlocker() bool { return false }

func (testerWorker) Run(ctx context.Context, s *State) error {
	if s.Patch == nil {
		return fmt.Errorf("patch draft not available")
	}

	cases := []string{
		"Add or update one regression test that covers the primary failure path described in the source material.",
		"Verify the happy path still passes after the change.",
	}
	if len(s.Files) > 0 && s.Files[0] != "<inspect-changed-files>" {
		cases = append(cases, "Cover the behavior around "+s.Files[0]+" with the narrowest useful scope.")
	}

	tp := &deepschema.TestPlan{
		TestCases: cases,
		EdgeCases: []string{
			"Empty or nil inputs that were mentioned or implied by the original issue.",
			"Boundary cases around existing conditionals, parsing, or branching logic.",
		},
		VerificationSteps: []string{
			"Run the smallest targeted test command first.",
			"Re-run the original failing command if one was mentioned.",
			"Review logs or output formatting for regressions after the patch.",
		},
	}
	s.Tests = tp

	if err := s.AW.WriteText("result/tests.md", formatTestPlanMD(tp)); err != nil {
		return fmt.Errorf("write tests.md: %w", err)
	}
	if err := writeWorkerRequest(s, "tester", map[string]any{
		"patch_draft": "workers/patcher/result.json",
		"memory":      "evidence/memory.json",
	}); err != nil {
		return err
	}
	return writeWorkerResult(s, "tester", tp)
}

// ---------------------------------------------------------------------------
// reconcilerWorker — hard blocker
// Inputs:  PatchDraft (s.Patch), RiskReport (s.Risks, may be nil), TestPlan (s.Tests, may be nil)
// Outputs: PatchBundle  (written to s.Bundle and result/patch_bundle.json)
// ---------------------------------------------------------------------------

type reconcilerWorker struct{}

func (reconcilerWorker) Name() string      { return "reconciler" }
func (reconcilerWorker) HardBlocker() bool { return true }

func (reconcilerWorker) Run(ctx context.Context, s *State) error {
	if s.Patch == nil {
		return fmt.Errorf("patch draft not available")
	}

	// Soft blockers may have left Risks or Tests nil — use safe defaults.
	risks := safeRisks(s.Risks)
	tests := safeTests(s.Tests)

	warnings := []string{}
	if len(s.Patch.TouchedFiles) == 0 || (len(s.Patch.TouchedFiles) == 1 && s.Patch.TouchedFiles[0] == "<inspect-changed-files>") {
		warnings = append(warnings, "No concrete file path was confidently extracted from the source material.")
	}
	if s.Risks == nil {
		warnings = append(warnings, "Risk review skipped (critic failed); manual risk assessment recommended.")
	}
	if s.Tests == nil {
		warnings = append(warnings, "Test plan skipped (tester failed); manual test planning recommended.")
	}

	openQuestions := []string{
		"Which exact file and symbol should receive the first edit?",
		"Does the repository already contain a closer existing helper or test for this behavior?",
	}
	if strings.Contains(strings.ToLower(s.Opts.Source), "api") {
		openQuestions = append(openQuestions, "Does this change affect external API behavior or compatibility?")
	}

	bundle := &deepschema.PatchBundle{
		Summary:       s.Patch.Summary,
		Diff:          s.Patch.Diff,
		TouchedFiles:  s.Patch.TouchedFiles,
		Risks:         risks,
		TestPlan:      tests,
		OpenQuestions: openQuestions,
		Warnings:      warnings,
	}
	s.Bundle = bundle

	if err := s.AW.WriteJSON("result/patch_bundle.json", bundle); err != nil {
		return fmt.Errorf("write patch_bundle.json: %w", err)
	}
	if err := writeWorkerRequest(s, "reconciler", map[string]any{
		"patch_draft": "workers/patcher/result.json",
		"risk_report": "workers/critic/result.json",
		"test_plan":   "workers/tester/result.json",
	}); err != nil {
		return err
	}
	return writeWorkerResult(s, "reconciler", bundle)
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

func writeWorkerRequest(s *State, name string, req any) error {
	return s.AW.WriteJSON("workers/"+name+"/request.json", req)
}

func writeWorkerResult(s *State, name string, result any) error {
	return s.AW.WriteJSON("workers/"+name+"/result.json", result)
}

func safeRisks(r *deepschema.RiskReport) []deepschema.RiskItem {
	if r == nil {
		return nil
	}
	return r.TopRisks
}

func safeTests(t *deepschema.TestPlan) deepschema.TestPlan {
	if t == nil {
		return deepschema.TestPlan{}
	}
	return *t
}

func summarize(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return "Convert the source answer into a concrete implementation follow-up."
	}
	text = strings.ReplaceAll(text, "\n", " ")
	runes := []rune(text)
	if len(runes) <= 180 {
		return text
	}
	return string(runes[:177]) + "..."
}

func buildDraftDiff(files []string) string {
	if len(files) == 0 {
		return "diff --git a/<inspect-changed-files> b/<inspect-changed-files>\n" +
			"--- a/<inspect-changed-files>\n+++ b/<inspect-changed-files>\n@@\n" +
			"- review the existing implementation\n+ apply the planned fix and add regression coverage"
	}
	parts := make([]string, 0, len(files))
	for _, file := range files {
		parts = append(parts, fmt.Sprintf(
			"diff --git a/%s b/%s\n--- a/%s\n+++ b/%s\n@@\n- inspect the current implementation\n+ apply the change described by the patch bundle",
			file, file, file, file,
		))
	}
	return strings.Join(parts, "\n\n")
}

func formatTestPlanMD(tp *deepschema.TestPlan) string {
	var sb strings.Builder
	sb.WriteString("# Test Plan\n\n## Test Cases\n")
	for _, c := range tp.TestCases {
		sb.WriteString("- " + c + "\n")
	}
	sb.WriteString("\n## Edge Cases\n")
	for _, c := range tp.EdgeCases {
		sb.WriteString("- " + c + "\n")
	}
	sb.WriteString("\n## Verification Steps\n")
	for _, v := range tp.VerificationSteps {
		sb.WriteString("- " + v + "\n")
	}
	return sb.String()
}
