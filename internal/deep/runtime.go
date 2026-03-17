package deep

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/helloprtr/poly-prompt/internal/deep/artifact"
	deepevent "github.com/helloprtr/poly-prompt/internal/deep/event"
	"github.com/helloprtr/poly-prompt/internal/deep/llm"
	deepplan "github.com/helloprtr/poly-prompt/internal/deep/plan"
	deeprun "github.com/helloprtr/poly-prompt/internal/deep/run"
	deepschema "github.com/helloprtr/poly-prompt/internal/deep/schema"
	"github.com/helloprtr/poly-prompt/internal/deep/worker"
	"github.com/helloprtr/poly-prompt/internal/history"
)

// ---------------------------------------------------------------------------
// Type aliases — callers (e.g. app.go) use the deep.* names unchanged.
// ---------------------------------------------------------------------------

type RunStatus = deeprun.Status
type TodoStatus = deepplan.TodoStatus
type EventType = deepevent.Type

type Run = deeprun.DeepRun
type WorkPlan = deepplan.WorkPlan
type TodoItem = deepplan.TodoItem
type WorkerSpec = deepplan.WorkerSpec

type PatchDraft = deepschema.PatchDraft
type RiskItem = deepschema.RiskItem
type RiskReport = deepschema.RiskReport
type TestPlan = deepschema.TestPlan
type PatchBundle = deepschema.PatchBundle

type Event = deepevent.Event
type Progress = deeprun.Progress
type Options = deeprun.Options
type Result = deeprun.Result

// ---------------------------------------------------------------------------
// Constant aliases — same names, wired to sub-package values.
// ---------------------------------------------------------------------------

const (
	RunStatusCreated              RunStatus = deeprun.StatusCreated
	RunStatusPlanning             RunStatus = deeprun.StatusPlanning
	RunStatusRunning              RunStatus = deeprun.StatusRunning
	RunStatusAwaitingApproval     RunStatus = deeprun.StatusAwaitingApproval
	RunStatusCompleted            RunStatus = deeprun.StatusCompleted
	RunStatusCompletedWithWarning RunStatus = deeprun.StatusCompletedWithWarnings
	RunStatusFailed               RunStatus = deeprun.StatusFailed

	TodoStatusPending    TodoStatus = deepplan.TodoPending
	TodoStatusInProgress TodoStatus = deepplan.TodoInProgress
	TodoStatusCompleted  TodoStatus = deepplan.TodoCompleted
	TodoStatusFailed     TodoStatus = deepplan.TodoFailed
	TodoStatusSkipped    TodoStatus = deepplan.TodoSkipped

	EventRunStarted        EventType = deepevent.RunStarted
	EventContextCompiled   EventType = deepevent.ContextCompiled
	EventPlanCreated       EventType = deepevent.PlanCreated
	EventTodoUpdated       EventType = deepevent.TodoUpdated
	EventWorkerStarted     EventType = deepevent.WorkerStarted
	EventWorkerCompleted   EventType = deepevent.WorkerCompleted
	EventArtifactReady     EventType = deepevent.ArtifactReady
	EventApprovalRequested EventType = deepevent.ApprovalRequested
	EventApprovalGranted   EventType = deepevent.ApprovalGranted
	EventDeliveryStarted   EventType = deepevent.DeliveryStarted
	EventDeliveryCompleted EventType = deepevent.DeliveryCompleted
	EventMemorySuggested   EventType = deepevent.MemorySuggested
	EventRunCompleted      EventType = deepevent.RunCompleted
	EventRunFailed         EventType = deepevent.RunFailed
	EventLLMEnhanceFailed  EventType = deepevent.LLMEnhanceFailed
)

// inspectChangedFiles is the sentinel placeholder used when no concrete file
// path can be extracted from the source text.
const inspectChangedFiles = "<inspect-changed-files>"

// fileRefRe matches relative file paths (e.g. "handlers/user.go").
var fileRefRe = regexp.MustCompile(`(?:[A-Za-z0-9_.-]+/)*[A-Za-z0-9_.-]+\.[A-Za-z0-9]+`)

// AppendEvent appends a structured event to the run's event log.
func AppendEvent(path string, e Event) error {
	return deepevent.Append(path, e)
}

// ---------------------------------------------------------------------------
// ExecutePatchRun — the main deep execution entry point.
// ---------------------------------------------------------------------------

// ExecutePatchRun runs the deep patch pipeline using the default worker graph.
func ExecutePatchRun(ctx context.Context, opts Options) (Result, error) {
	return executePatchRunWithGraph(ctx, opts, worker.NewPatchGraph, llm.New)
}

// executePatchRunWithGraph is the testable version that accepts a graph factory and
// an LLM factory, allowing tests to inject stub workers and mock enhancers.
func executePatchRunWithGraph(
	ctx context.Context,
	opts Options,
	graphFactory func() *worker.Graph,
	llmFactory func(provider, apiKey string) (llm.Enhancer, error),
) (Result, error) {
	switch strings.TrimSpace(opts.Action) {
	case "patch", "test", "debug", "refactor":
		// supported
	default:
		return Result{}, fmt.Errorf("deep execution supports: patch, test, debug, refactor; got %q", opts.Action)
	}

	now := time.Now().UTC()
	runID := fmt.Sprintf("%d", now.UnixNano())

	// ── Artifact directory ──────────────────────────────────────────────────
	aw, err := artifact.New(opts.RepoRoot, runID)
	if err != nil {
		return Result{}, err
	}
	if err := aw.Init([]string{"planner", "patcher", "critic", "tester", "reconciler"}); err != nil {
		return Result{}, err
	}

	// ── Initial run record ──────────────────────────────────────────────────
	run := Run{
		ID:              runID,
		Version:         1,
		Action:          strings.TrimSpace(opts.Action),
		Engine:          "deep",
		Status:          RunStatusCreated,
		ResultType:      resultTypeFor(opts.Action),
		TargetApp:       strings.TrimSpace(opts.TargetApp),
		DeliveryMode:    strings.TrimSpace(opts.DeliveryMode),
		SourceKind:      strings.TrimSpace(opts.SourceKind),
		ParentHistoryID: strings.TrimSpace(opts.ParentHistoryID),
		RepoRoot:        strings.TrimSpace(opts.RepoRoot),
		ArtifactRoot:    aw.Root,
		EventLogPath:    aw.EventLogPath(),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := aw.WriteManifest(run); err != nil {
		return Result{}, err
	}
	if err := AppendEvent(run.EventLogPath, Event{
		Type:      EventRunStarted,
		Timestamp: now,
		Data:      map[string]any{"run_id": run.ID, "target_app": run.TargetApp, "source_kind": run.SourceKind},
	}); err != nil {
		return Result{}, err
	}

	// ── Evidence files ──────────────────────────────────────────────────────
	if err := aw.WriteText("source.md", strings.TrimSpace(opts.Source)+"\n"); err != nil {
		return Result{}, err
	}
	if err := aw.WriteJSON("evidence/repo_context.json", opts.RepoSummary); err != nil {
		return Result{}, err
	}
	if opts.HistoryEntry != nil {
		if err := aw.WriteJSON("evidence/history.json", opts.HistoryEntry); err != nil {
			return Result{}, err
		}
	} else if err := aw.WriteJSON("evidence/history.json", map[string]any{}); err != nil {
		return Result{}, err
	}
	if err := aw.WriteJSON("evidence/memory.json", map[string]any{"protected_terms": opts.ProtectedTerms}); err != nil {
		return Result{}, err
	}
	if err := aw.WriteText("evidence/git.diff", gitDiff(opts.RepoRoot)); err != nil {
		return Result{}, err
	}
	if err := aw.WriteJSON("lineage.json", map[string]any{
		"parent_history_id": run.ParentHistoryID,
		"source_kind":       run.SourceKind,
		"target_app":        run.TargetApp,
	}); err != nil {
		return Result{}, err
	}
	if err := AppendEvent(run.EventLogPath, Event{
		Type:      EventContextCompiled,
		Timestamp: time.Now().UTC(),
		Data: map[string]any{"evidence_refs": []string{
			"source.md", "evidence/repo_context.json", "evidence/history.json",
			"evidence/memory.json", "evidence/git.diff",
		}},
	}); err != nil {
		return Result{}, err
	}

	// ── File references ─────────────────────────────────────────────────────
	files := extractFileRefs(opts.Source)
	if len(files) == 0 {
		files = inferFilesFromHistory(opts.HistoryEntry)
	}
	if len(files) == 0 {
		files = []string{inspectChangedFiles}
	}

	// ── Progress helper ─────────────────────────────────────────────────────
	emit := func(step string, idx int, msg string) {
		if opts.Progress != nil {
			opts.Progress(Progress{Step: step, Index: idx, Total: 5, Message: msg})
		}
	}

	// ── Planning phase ──────────────────────────────────────────────────────
	run.Status = RunStatusPlanning
	run.UpdatedAt = time.Now().UTC()
	if err := aw.WriteManifest(run); err != nil {
		return Result{}, err
	}
	emit("plan", 1, "capturing the run plan")

	// ── Graph execution ─────────────────────────────────────────────────────
	run.Status = RunStatusRunning
	run.UpdatedAt = time.Now().UTC()
	if err := aw.WriteManifest(run); err != nil {
		return Result{}, err
	}

	emit("patch", 2, "building a patch draft")
	emit("critique", 3, "reviewing risks")
	emit("tests", 4, "drafting verification steps")
	emit("reconcile", 5, "packaging the bundle")

	gr, err := graphFactory().Run(ctx, opts, aw, files)
	if err != nil {
		// Hard blocker failure — mark run as failed.
		run.Status = RunStatusFailed
		run.ErrorMessage = err.Error()
		run.UpdatedAt = time.Now().UTC()
		_ = aw.WriteManifest(run)
		return Result{}, err
	}

	// Attach the plan that the graph populated.
	if gr.Plan != nil {
		run.Plan = *gr.Plan
		if err := AppendEvent(run.EventLogPath, Event{
			Type:      EventPlanCreated,
			Timestamp: time.Now().UTC(),
			Data:      map[string]any{"todo_count": len(run.Plan.Todos)},
		}); err != nil {
			return Result{}, err
		}
	}

	// ── Finalise run ────────────────────────────────────────────────────────
	bundle := deepschema.PatchBundle{}
	if gr.Bundle != nil {
		bundle = *gr.Bundle
	}
	// Merge soft-blocker warnings into the bundle.
	bundle.Warnings = append(bundle.Warnings, gr.Warnings...)

	run.ResultRef = "result/patch_bundle.json"
	run.WarningCount = len(bundle.Warnings)
	if run.WarningCount > 0 {
		run.Status = RunStatusCompletedWithWarning
	} else {
		run.Status = RunStatusCompleted
	}
	completed := time.Now().UTC()
	run.UpdatedAt = completed
	run.CompletedAt = &completed
	if err := aw.WriteManifest(run); err != nil {
		return Result{}, err
	}
	// Re-persist bundle with merged warnings.
	if err := aw.WriteJSON("result/patch_bundle.json", bundle); err != nil {
		return Result{}, err
	}
	if err := aw.WriteText("result/summary.md", formatSummary(bundle)); err != nil {
		return Result{}, err
	}

	if err := AppendEvent(run.EventLogPath, Event{
		Type:      EventArtifactReady,
		Timestamp: completed,
		Data:      map[string]any{"path": "result/patch_bundle.json"},
	}); err != nil {
		return Result{}, err
	}
	if err := AppendEvent(run.EventLogPath, Event{
		Type:      EventRunCompleted,
		Timestamp: completed,
		Data:      map[string]any{"status": run.Status, "warning_count": run.WarningCount},
	}); err != nil {
		return Result{}, err
	}

	deliveryPrompt := buildDeliveryPrompt(run.Plan, bundle, opts.Source, opts.LLMProvider)
	llmEnhanced := false
	if opts.LLMProvider != "" && opts.LLMAPIKey != "" {
		enhancer, err := llmFactory(opts.LLMProvider, opts.LLMAPIKey)
		if err != nil {
			_ = AppendEvent(run.EventLogPath, Event{
				Type: EventLLMEnhanceFailed,
				Data: map[string]any{"provider": opts.LLMProvider, "error": err.Error(), "stage": "factory"},
			})
		} else if enriched, enhanceErr := enhancer.Enhance(ctx, opts.Source, bundle, deliveryPrompt); enhanceErr != nil {
			_ = AppendEvent(run.EventLogPath, Event{
				Type: EventLLMEnhanceFailed,
				Data: map[string]any{"provider": opts.LLMProvider, "error": enhanceErr.Error(), "stage": "enhance"},
			})
		} else {
			deliveryPrompt = enriched
			llmEnhanced = true
		}
	} else if opts.LLMProvider != "" {
		llmEnhanced = true
	}

	return Result{
		Run:            run,
		Bundle:         bundle,
		DeliveryPrompt: deliveryPrompt,
		LLMEnhanced:    llmEnhanced,
	}, nil
}

// ---------------------------------------------------------------------------
// Formatting helpers (used only by runtime.go)
// ---------------------------------------------------------------------------

func buildDeliveryPrompt(plan WorkPlan, bundle PatchBundle, source, provider string) string {
	switch provider {
	case "claude":
		return buildClaudePrompt(plan, bundle, source)
	case "gemini":
		return buildGeminiPrompt(plan, bundle, source)
	case "codex":
		return buildCodexPrompt(plan, bundle, source)
	default:
		return buildUniversalPrompt(plan, bundle, source)
	}
}

func buildClaudePrompt(plan WorkPlan, bundle PatchBundle, source string) string {
	topRisk := topRiskLine(bundle.Risks)
	lines := []string{
		"<role>",
		"You are an expert software engineer implementing a precise code change.",
		"Apply the structured patch bundle below as-is — do not expand scope.",
		"</role>",
		"",
		"<context>",
		"<source_intent>",
		strings.TrimSpace(source),
		"</source_intent>",
		"<patch_bundle>",
		"  <summary>" + bundle.Summary + "</summary>",
		"  <touched_files>",
	}
	for _, f := range bundle.TouchedFiles {
		lines = append(lines, "    <file>"+f+"</file>")
	}
	lines = append(lines,
		"  </touched_files>",
		"  <diff>"+bundle.Diff+"</diff>",
		"</patch_bundle>",
		"<plan>"+plan.Summary+"</plan>",
		"</context>",
		"",
		"<task>",
		"Apply the patch bundle to the repository.",
		"For each file in touched_files:",
		"  1. Read the current implementation.",
		"  2. Apply only the change described — do not refactor unrelated code.",
		"  3. Confirm the change compiles and matches the source intent.",
		"</task>",
		"",
		"<risks>",
		topRisk,
		"</risks>",
		"",
		"<constraints>",
		"<constraint>Which exact file and symbol should receive the first edit?</constraint>",
		"- Do not rename protected terms.",
		"- Do not change behavior outside the stated scope.",
		"</constraints>",
		"",
		"<validation>",
		"<test_cases>",
	)
	if len(bundle.TestPlan.TestCases) == 0 {
		lines = append(lines, "  <test>Add a regression test covering the primary failure path.</test>")
	} else {
		for _, tc := range bundle.TestPlan.TestCases {
			lines = append(lines, "  <test>"+tc+"</test>")
		}
	}
	lines = append(lines,
		"</test_cases>",
		"<verification_steps>",
		"  <step>Run the smallest targeted test command first.</step>",
		"</verification_steps>",
		"</validation>",
	)
	if len(bundle.Warnings) > 0 {
		lines = append(lines, "", "<warnings>", formatBullets(bundle.Warnings), "</warnings>")
	}
	return strings.Join(lines, "\n")
}

func buildGeminiPrompt(plan WorkPlan, bundle PatchBundle, source string) string {
	lines := []string{
		"## Role",
		"You are an expert software engineer implementing a precise code change.",
		"",
		"## Source Intent",
		strings.TrimSpace(source),
		"",
		"## Patch Bundle",
		"**Summary:** " + bundle.Summary,
		"**Files to modify:**",
		formatBullets(bundle.TouchedFiles),
		"",
		"**Diff context:**",
		bundle.Diff,
		"",
		"## Plan",
		plan.Summary,
		"",
		"## Task",
		"Apply the patch bundle file by file:",
		"1. Read the current implementation of each touched file.",
		"2. Apply only the described change — do not refactor unrelated code.",
		"3. Confirm the change compiles and matches the source intent.",
		"",
		"## Risks",
		formatRiskBullets(bundle.Risks),
		"",
		"## Validation",
		"**Test cases:**",
		formatBullets(bundle.TestPlan.TestCases),
		"",
		"**Verification steps:**",
		formatNumberedList(bundle.TestPlan.VerificationSteps),
	}
	if len(bundle.Warnings) > 0 {
		lines = append(lines, "", "## Warnings", formatBullets(bundle.Warnings))
	}
	return strings.Join(lines, "\n")
}

func buildCodexPrompt(plan WorkPlan, bundle PatchBundle, source string) string {
	lines := []string{
		"You are implementing a code change. Follow these instructions precisely.",
		"",
		"**Source intent:** " + strings.TrimSpace(source),
		"",
		"**Plan:** " + plan.Summary,
		"",
		"**Summary:** " + bundle.Summary,
		"",
		"**Files to modify:**",
	}
	for i, f := range bundle.TouchedFiles {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, f))
	}
	lines = append(lines,
		"",
		"**Diff context:**",
		"```diff",
		bundle.Diff,
		"```",
		"",
		"**Instructions:**",
		"1. Apply only the change described above.",
		"2. Do not modify unrelated code.",
		"3. Validate the change compiles and passes tests.",
		"",
		"**Risks:**",
		formatRiskBullets(bundle.Risks),
		"",
		"**Test cases:**",
		formatBullets(bundle.TestPlan.TestCases),
		"",
		"**Verify:** Run targeted tests after applying changes.",
	)
	if len(bundle.Warnings) > 0 {
		lines = append(lines, "", "**Warnings:**", formatBullets(bundle.Warnings))
	}
	return strings.Join(lines, "\n")
}

func buildUniversalPrompt(plan WorkPlan, bundle PatchBundle, source string) string {
	lines := []string{
		"## Goal",
		"Apply the following patch bundle precisely as described.",
		"",
		"## Source Intent",
		strings.TrimSpace(source),
		"",
		"## Patch Bundle Summary",
		bundle.Summary,
		"",
		"## Plan",
		plan.Summary,
		"",
		"## Files to Modify",
		formatBullets(bundle.TouchedFiles),
		"",
		"## Key Risks",
		formatRiskBullets(bundle.Risks),
		"",
		"## Implementation Steps",
		"1. Read the current implementation of each file listed above.",
		"2. Apply only the described change — do not refactor unrelated code.",
		"3. Confirm the change compiles and matches the source intent.",
		"",
		"## Tests Required",
		formatBullets(bundle.TestPlan.TestCases),
		"",
		"## Edge Cases",
		formatBullets(bundle.TestPlan.EdgeCases),
		"",
		"## Verify",
		formatNumberedList(bundle.TestPlan.VerificationSteps),
	}
	if len(bundle.Warnings) > 0 {
		lines = append(lines, "", "## Warnings", formatBullets(bundle.Warnings))
	}
	return strings.Join(lines, "\n")
}

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

func topRiskLine(risks []RiskItem) string {
	if len(risks) == 0 {
		return `<risk severity="medium">Behavior Drift: verify no scope expansion.</risk>`
	}
	r := risks[0]
	return fmt.Sprintf(`<risk severity=%q>%s: %s</risk>`, strings.ToLower(r.Severity), r.Title, r.Detail)
}

func formatNumberedList(values []string) string {
	if len(values) == 0 {
		return "1. Run the targeted test command."
	}
	lines := make([]string, 0, len(values))
	for _, v := range values {
		if t := strings.TrimSpace(v); t != "" {
			lines = append(lines, fmt.Sprintf("%d. %s", len(lines)+1, t))
		}
	}
	if len(lines) == 0 {
		return "1. Run the targeted test command."
	}
	return strings.Join(lines, "\n")
}

func formatSummary(bundle PatchBundle) string {
	lines := []string{
		"# Patch Bundle", "", "## Summary", bundle.Summary, "",
		"## Touched Files", formatBullets(bundle.TouchedFiles), "",
		"## Risks", formatRiskBullets(bundle.Risks), "",
		"## Test Plan", formatBullets(bundle.TestPlan.TestCases),
	}
	if len(bundle.Warnings) > 0 {
		lines = append(lines, "", "## Warnings", formatBullets(bundle.Warnings))
	}
	return strings.Join(lines, "\n") + "\n"
}

func formatBullets(values []string) string {
	if len(values) == 0 {
		return "- none"
	}
	lines := make([]string, 0, len(values))
	for _, v := range values {
		if t := strings.TrimSpace(v); t != "" {
			lines = append(lines, "- "+t)
		}
	}
	if len(lines) == 0 {
		return "- none"
	}
	return strings.Join(lines, "\n")
}

func formatRiskBullets(items []RiskItem) string {
	if len(items) == 0 {
		return "- none"
	}
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, fmt.Sprintf("- [%s] %s: %s", strings.ToUpper(item.Severity), item.Title, item.Detail))
	}
	return strings.Join(lines, "\n")
}

// ---------------------------------------------------------------------------
// Utility functions
// ---------------------------------------------------------------------------

func extractFileRefs(text string) []string {
	matches := fileRefRe.FindAllString(text, -1)
	out := make([]string, 0, len(matches))
	seen := map[string]bool{}
	for _, match := range matches {
		match = strings.Trim(match, " `\"'.,:;()[]{}")
		if match == "" || seen[match] {
			continue
		}
		seen[match] = true
		out = append(out, match)
	}
	sort.Strings(out)
	return out
}

func inferFilesFromHistory(entry *history.Entry) []string {
	if entry == nil {
		return nil
	}
	return extractFileRefs(strings.Join([]string{
		entry.Original, entry.Translated, entry.FinalPrompt,
	}, "\n"))
}

func gitDiff(repoRoot string) string {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		return ""
	}
	cmd := exec.Command("git", "diff", "--", ".")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil && len(out) == 0 {
		return ""
	}
	return strings.TrimSpace(string(out))
}
