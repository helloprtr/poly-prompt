package llm

import (
	"fmt"
	"strings"

	"github.com/helloprtr/poly-prompt/internal/deep/schema"
)

// buildEnhancePrompt constructs the prompt sent to any LLM provider.
func buildEnhancePrompt(source string, bundle schema.PatchBundle, ruleBased string) string {
	diffHead := bundle.Diff
	if lines := strings.SplitN(bundle.Diff, "\n", 41); len(lines) > 40 {
		diffHead = strings.Join(lines[:40], "\n")
	}

	var topRisk string
	if len(bundle.Risks) > 0 {
		r := bundle.Risks[0]
		topRisk = r.Title
		if r.Detail != "" {
			topRisk += ": " + r.Detail
		}
	}

	var testCases string
	if len(bundle.TestPlan.TestCases) > 0 {
		testCases = strings.Join(bundle.TestPlan.TestCases, "; ")
	}

	return fmt.Sprintf(`You are a senior software engineer writing a focused implementation prompt for a coding AI.

Given this patch context, produce a concise, action-oriented prompt. Requirements:
- Line 1: Bold action title (e.g., "**Fix nil dereference in ApplyPolicy**")
- Section: Files to modify (exact paths from touched_files only)
- Section: Concrete code changes (specific, not general advice)
- Section: Top 1 risk and how to validate it
- Section: One regression test to add

Keep total length under 400 words. No markdown headers — use bold labels inline.

--- Source material ---
%s

--- Patch bundle ---
Summary: %s
Files: %s
Diff context: %s
Top risk: %s
Test cases: %s

--- Current rule-based prompt (improve this) ---
%s`,
		source,
		bundle.Summary,
		strings.Join(bundle.TouchedFiles, ", "),
		diffHead,
		topRisk,
		testCases,
		ruleBased,
	)
}
