package worker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	deepplan "github.com/helloprtr/poly-prompt/internal/deep/plan"
	deepschema "github.com/helloprtr/poly-prompt/internal/deep/schema"
)

// inspectChangedFiles is the sentinel placeholder used when no concrete file
// path can be extracted from the source text.
const inspectChangedFiles = "<inspect-changed-files>"

// resultTypeFor returns the bundle type name for the given action.
func resultTypeFor(action string) string {
	switch strings.TrimSpace(action) {
	case "test":
		return "TestBundle"
	case "debug":
		return "DebugBundle"
	case "refactor":
		return "RefactorBundle"
	default:
		return "PatchBundle"
	}
}

// ---------------------------------------------------------------------------
// plannerWorker — hard blocker
// Inputs:  source.md, evidence/repo_context.json, evidence/history.json,
//          evidence/memory.json
// Outputs: WorkPlan  (written to s.Plan and plan.json)
// ---------------------------------------------------------------------------

type plannerWorker struct{}

func (plannerWorker) Name() string      { return "planner" }
func (plannerWorker) HardBlocker() bool { return true }

func planSummaryFor(action string) string {
	switch action {
	case "test":
		return "Plan, draft test cases, review coverage risks, verify, package TestBundle."
	case "debug":
		return "Plan, identify root cause, draft fix, verify, package DebugBundle."
	case "refactor":
		return "Plan, draft refactor scope, review safety, define rollback, package RefactorBundle."
	default:
		return "Plan the follow-up, draft the patch, critique the risk, draft tests, and package a single PatchBundle."
	}
}

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
			Status:     deepplan.TodoPending,
			DependsOn:  []string{"plan"},
			InputRefs:  []string{"source.md", "plan.json", "evidence/git.diff"},
			OutputRefs: []string{"workers/patcher/result.json", "result/patch.diff"},
		},
		{
			ID: "critique", Title: "Review the draft for risks", Worker: "critic",
			Status:     deepplan.TodoPending,
			DependsOn:  []string{"patch"},
			InputRefs:  []string{"workers/patcher/result.json"},
			OutputRefs: []string{"workers/critic/result.json"},
		},
		{
			ID: "tests", Title: "Draft the verification plan", Worker: "tester",
			Status:     deepplan.TodoPending,
			DependsOn:  []string{"patch"},
			InputRefs:  []string{"workers/patcher/result.json", "evidence/memory.json"},
			OutputRefs: []string{"workers/tester/result.json", "result/tests.md"},
		},
		{
			ID: "reconcile", Title: "Package the final PatchBundle", Worker: "reconciler",
			Status:     deepplan.TodoPending,
			DependsOn:  []string{"critique", "tests"},
			InputRefs:  []string{"workers/patcher/result.json", "workers/critic/result.json", "workers/tester/result.json"},
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
		Action:       s.Opts.Action,
		ResultType:   resultTypeFor(s.Opts.Action),
		Summary:      planSummaryFor(s.Opts.Action),
		EvidenceRefs: []string{"source.md", "evidence/repo_context.json", "evidence/history.json", "evidence/memory.json", "evidence/git.diff"},
		Todos:        todos,
		WorkerGraph:  graph,
	}
	s.Plan = plan
	if err := s.AW.WriteJSON("plan.json", plan); err != nil {
		return fmt.Errorf("persist plan: %w", err)
	}
	if err := writeWorkerRequest(s, "planner", map[string]any{
		"source":        "source.md",
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

	var constraints []string
	if s.Opts.RepoSummary.Branch != "" {
		constraints = append(constraints, "Respect current branch context: "+s.Opts.RepoSummary.Branch+".")
	}
	if len(s.Opts.RepoSummary.Changes) > 0 {
		// Include actual file list, not just a generic reminder.
		constraints = append(constraints,
			"Local changes in this checkout:\n  "+
				strings.Join(s.Opts.RepoSummary.Changes, "\n  "))
	}

	diff := buildDraftDiff(s.Files)
	if realDiff, err := s.AW.ReadText("evidence/git.diff"); err == nil && strings.TrimSpace(realDiff) != "" {
		diff = realDiff
	}

	draft := &deepschema.PatchDraft{
		Summary:             summarize(s.Opts.Source),
		TouchedFiles:        s.Files,
		ImplementationNotes: notes,
		Constraints:         constraints,
		Diff:                diff,
	}
	s.Patch = draft

	if err := s.AW.WriteText("result/patch.diff", draft.Diff+"\n"); err != nil {
		return fmt.Errorf("write patch.diff: %w", err)
	}
	if err := writeWorkerRequest(s, "patcher", map[string]any{
		"source":        "source.md",
		"plan":          "plan.json",
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

	risks := detectSourceRisks(s.Opts.Source, s.Files)

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

// detectSourceRisks maps source keywords to relevant RiskItems.
func detectSourceRisks(source string, files []string) []deepschema.RiskItem {
	src := strings.ToLower(source)
	var risks []deepschema.RiskItem

	type rule struct {
		keywords []string
		title    string
		severity string
		detail   string
	}
	rules := []rule{
		{
			keywords: []string{"migration", "schema", "alter table"},
			title:    "Schema Migration Risk",
			severity: "high",
			detail:   "Schema or migration changes can cause data loss or incompatibility if not applied carefully.",
		},
		{
			keywords: []string{"auth", "login", "token", "jwt", "session"},
			title:    "Auth Regression Risk",
			severity: "high",
			detail:   "Auth changes affect security paths — verify no permission bypass introduced.",
		},
		{
			keywords: []string{"api", "endpoint", "rest", "handler", "route"},
			title:    "API Contract Risk",
			severity: "high",
			detail:   "API changes may break callers — check backwards compatibility.",
		},
		{
			keywords: []string{"goroutine", "mutex", "race", "concurrent"},
			title:    "Concurrency Risk",
			severity: "high",
			detail:   "Concurrent code changes can introduce data races — run with -race flag.",
		},
		{
			keywords: []string{"delete", "remove", "drop", "purge"},
			title:    "Destructive Operation Risk",
			severity: "medium",
			detail:   "Destructive operations are hard to reverse — confirm backups or dry-run first.",
		},
		{
			keywords: []string{"env", "config", "secret", "credential"},
			title:    "Config/Secret Exposure Risk",
			severity: "medium",
			detail:   "Config or secret changes can leak sensitive values — audit carefully.",
		},
		{
			keywords: []string{"cache", "redis", "invalidat"},
			title:    "Cache Invalidation Risk",
			severity: "medium",
			detail:   "Cache changes may serve stale data — verify invalidation logic.",
		},
	}

	for _, r := range rules {
		for _, kw := range r.keywords {
			if strings.Contains(src, kw) {
				risks = append(risks, deepschema.RiskItem{
					Title:      r.title,
					Severity:   r.severity,
					Detail:     r.detail,
					Confidence: "medium",
				})
				break
			}
		}
	}

	if len(risks) == 0 {
		risks = append(risks, deepschema.RiskItem{
			Title:      "Behavior Drift",
			Severity:   "high",
			Detail:     "The follow-up may change behavior beyond the user's requested fix if the source answer mixed diagnosis and implementation.",
			Confidence: "medium",
		})
	}

	if len(files) > 0 && files[0] != inspectChangedFiles {
		risks = append(risks, deepschema.RiskItem{
			Title:      "Local Context Risk",
			Severity:   "medium",
			Detail:     "Review the existing logic in " + files[0] + " before applying broad changes.",
			Confidence: "medium",
		})
	}

	if !strings.Contains(src, "test") && !strings.Contains(src, "spec") && !strings.Contains(src, "regression") {
		risks = append(risks, deepschema.RiskItem{
			Title:      "Test Gap Risk",
			Severity:   "medium",
			Detail:     "The patch may look plausible without adding a regression check for the user-visible failure path.",
			Confidence: "high",
		})
	}

	return risks
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

	cases := buildTestCases(s.Opts.Source, s.Files, s.Opts.RepoRoot)

	tp := &deepschema.TestPlan{
		TestCases:         cases,
		EdgeCases:         buildEdgeCases(s.Opts.Source),
		VerificationSteps: buildVerificationSteps(s.Opts.Source),
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

// buildTestCases generates targeted test cases based on source keywords.
func buildTestCases(source string, files []string, repoRoot string) []string {
	src := strings.ToLower(source)
	cases := []string{
		"Add or update one regression test that covers the primary failure path described in the source material.",
		"Verify the happy path still passes after the change.",
	}

	type rule struct {
		keywords []string
		testCase string
	}
	rules := []rule{
		{
			keywords: []string{"nil", "null", "pointer", "panic"},
			testCase: "Add a test that passes nil/zero-value input and asserts no panic.",
		},
		{
			keywords: []string{"error", "err", "failure"},
			testCase: "Add a test that forces the error path and asserts non-nil error with useful message.",
		},
		{
			keywords: []string{"timeout", "deadline", "context"},
			testCase: "Add a test that cancels context early and verifies clean exit.",
		},
		{
			keywords: []string{"auth", "token", "permission"},
			testCase: "Add a test with invalid credential and verify proper auth rejection.",
		},
		{
			keywords: []string{"loop", "range", "slice", "array"},
			testCase: "Add a test with empty and single-element collection for off-by-one.",
		},
		{
			keywords: []string{"goroutine", "race", "parallel"},
			testCase: "Run under `go test -race` and verify no data races.",
		},
	}

	for _, r := range rules {
		for _, kw := range r.keywords {
			if strings.Contains(src, kw) {
				cases = append(cases, r.testCase)
				break
			}
		}
	}

	if len(files) > 0 && files[0] != inspectChangedFiles {
		primary := files[0]
		testFile := findTestFile(repoRoot, primary)
		if testFile != "" {
			cases = append(cases,
				fmt.Sprintf("Add a regression test in %s that covers the primary failure path.", testFile))
		} else {
			cases = append(cases,
				fmt.Sprintf("Cover the behavior around %s with the narrowest useful scope.", primary))
		}
	}

	return cases
}

// buildEdgeCases generates edge cases based on source keywords.
func buildEdgeCases(source string) []string {
	src := strings.ToLower(source)
	cases := []string{
		"Empty or nil inputs that were mentioned or implied by the original issue.",
		"Boundary cases around existing conditionals, parsing, or branching logic.",
	}
	if strings.Contains(src, "concurrent") || strings.Contains(src, "race") || strings.Contains(src, "goroutine") {
		cases = append(cases, "Concurrent access with multiple goroutines calling the changed code simultaneously.")
	}
	if strings.Contains(src, "timeout") || strings.Contains(src, "deadline") {
		cases = append(cases, "Zero and negative timeout durations.")
	}
	return cases
}

// buildVerificationSteps generates verification steps based on source keywords.
func buildVerificationSteps(source string) []string {
	src := strings.ToLower(source)
	steps := []string{
		"Run the smallest targeted test command first.",
		"Re-run the original failing command if one was mentioned.",
		"Review logs or output formatting for regressions after the patch.",
	}
	if strings.Contains(src, "race") || strings.Contains(src, "goroutine") || strings.Contains(src, "concurrent") {
		steps = append(steps, "Run with `go test -race ./...` to detect data races.")
	}
	return steps
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
	if len(s.Patch.TouchedFiles) == 0 || (len(s.Patch.TouchedFiles) == 1 && s.Patch.TouchedFiles[0] == inspectChangedFiles) {
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
	if err := s.AW.WriteJSON("workers/"+name+"/request.json", req); err != nil {
		return fmt.Errorf("write %s request: %w", name, err)
	}
	return nil
}

func writeWorkerResult(s *State, name string, result any) error {
	if err := s.AW.WriteJSON("workers/"+name+"/result.json", result); err != nil {
		return fmt.Errorf("write %s result: %w", name, err)
	}
	return nil
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

// summarize extracts the first meaningful sentence or line from text.
// It strips blank lines, picks the first non-empty line as the lead,
// then appends a second line if present and the combined length is ≤220 runes.
func summarize(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return "Convert the source answer into a concrete implementation follow-up."
	}
	lines := strings.Split(text, "\n")
	var nonEmpty []string
	for _, l := range lines {
		if t := strings.TrimSpace(l); t != "" {
			nonEmpty = append(nonEmpty, t)
		}
	}
	if len(nonEmpty) == 0 {
		return "Convert the source answer into a concrete implementation follow-up."
	}
	result := nonEmpty[0]
	if len(nonEmpty) > 1 {
		combined := result + " " + nonEmpty[1]
		if len([]rune(combined)) <= 220 {
			result = combined
		}
	}
	runes := []rune(result)
	if len(runes) > 220 {
		return string(runes[:217]) + "..."
	}
	return result
}

func buildDraftDiff(files []string) string {
	if len(files) == 0 {
		s := inspectChangedFiles
		return "diff --git a/" + s + " b/" + s + "\n" +
			"--- a/" + s + "\n+++ b/" + s + "\n@@\n" +
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

// findTestFile returns the conventional *_test.go path for srcFile if it
// exists under repoRoot. Returns "" if not found or repoRoot is empty.
func findTestFile(repoRoot, srcFile string) string {
	if strings.TrimSpace(repoRoot) == "" {
		return ""
	}
	ext := filepath.Ext(srcFile)
	base := strings.TrimSuffix(srcFile, ext)
	candidate := filepath.Join(repoRoot, base+"_test"+ext)
	if _, err := os.Stat(candidate); err == nil {
		return base + "_test" + ext
	}
	return ""
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
