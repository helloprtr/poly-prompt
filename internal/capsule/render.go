package capsule

import (
	"fmt"
	"strings"
)

// RenderResumePrompt generates a target-agnostic resume prompt from the capsule.
// The prompt is structured plain text suitable for pasting into any AI app.
// If drift has been detected, a warning section is prepended.
// Target is used to tailor tone (reserved for future LLM-enhanced rendering).
func RenderResumePrompt(c Capsule, target string, drift DriftReport) string {
	var b strings.Builder

	// Drift warning (prepended if any drift exists)
	if drift.HasDrift() {
		fmt.Fprintf(&b, "## ⚠ Repo Drift Detected\n")
		if drift.BranchChanged {
			fmt.Fprintf(&b, "- branch changed: %s → %s\n", drift.SavedBranch, drift.CurrentBranch)
		}
		if drift.SHAChanged {
			fmt.Fprintf(&b, "- commits since save: %s → %s\n", drift.SavedSHA, drift.CurrentSHA)
		}
		if drift.FilesChanged {
			fmt.Fprintf(&b, "- changed files differ from save point\n")
		}
		fmt.Fprintf(&b, "\nPlease review the changes above before continuing.\n\n---\n\n")
	}

	// Resume context
	fmt.Fprintf(&b, "## Resume Context\n\n")
	fmt.Fprintf(&b, "**Repo:** %s  **Branch:** %s  **SHA:** %s\n\n",
		c.Repo.Name, c.Repo.Branch, c.Repo.HeadSHA)

	if c.Work.OriginalRequest != "" {
		fmt.Fprintf(&b, "**What was being worked on:**\n%s\n\n", c.Work.OriginalRequest)
	}
	if c.Work.Summary != "" {
		fmt.Fprintf(&b, "**Progress summary:**\n%s\n\n", c.Work.Summary)
	}

	if len(c.Work.Todos) > 0 {
		fmt.Fprintf(&b, "**TODOs:**\n")
		for _, t := range c.Work.Todos {
			mark := "○"
			if t.Status == "completed" {
				mark = "✓"
			}
			fmt.Fprintf(&b, "- [%s] %s\n", mark, t.Title)
		}
		fmt.Fprintln(&b)
	}

	if len(c.Work.Decisions) > 0 {
		fmt.Fprintf(&b, "**Decisions already made:**\n")
		for _, d := range c.Work.Decisions {
			fmt.Fprintf(&b, "- %s\n", d)
		}
		fmt.Fprintln(&b)
	}

	if len(c.Work.OpenQuestions) > 0 {
		fmt.Fprintf(&b, "**Open questions:**\n")
		for _, q := range c.Work.OpenQuestions {
			fmt.Fprintf(&b, "- %s\n", q)
		}
		fmt.Fprintln(&b)
	}

	if len(c.Work.Risks) > 0 {
		fmt.Fprintf(&b, "**Risks:**\n")
		for _, r := range c.Work.Risks {
			fmt.Fprintf(&b, "- %s\n", r)
		}
		fmt.Fprintln(&b)
	}

	if c.Work.NextAction != "" {
		fmt.Fprintf(&b, "**Next recommended action:**\n%s\n\n", c.Work.NextAction)
	}

	if c.Session.ArtifactRoot != "" {
		fmt.Fprintf(&b, "**Artifacts to inspect:** `%s`\n\n", c.Session.ArtifactRoot)
	}

	if len(c.Repo.TouchedFiles) > 0 {
		fmt.Fprintf(&b, "**Touched files:**\n")
		for _, f := range c.Repo.TouchedFiles {
			fmt.Fprintf(&b, "- %s\n", f)
		}
		fmt.Fprintln(&b)
	}

	return b.String()
}
