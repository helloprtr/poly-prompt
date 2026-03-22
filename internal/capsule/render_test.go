package capsule_test

import (
	"strings"
	"testing"
	"time"

	"github.com/helloprtr/poly-prompt/internal/capsule"
)

func testCapsuleForRender() capsule.Capsule {
	now := time.Now().UTC()
	return capsule.Capsule{
		ID:        "cap_001",
		Label:     "auth refactor",
		Kind:      capsule.KindManual,
		CreatedAt: now,
		UpdatedAt: now,
		Repo: capsule.RepoState{
			Name:    "myrepo",
			Branch:  "fix/auth",
			HeadSHA: "abc1234",
		},
		Session: capsule.SessionState{
			TargetApp: "claude",
		},
		Work: capsule.WorkState{
			OriginalRequest: "implement JWT auth",
			NormalizedGoal:  "implement jwt auth",
			NextAction:      "add token refresh in auth.go",
			Summary:         "JWT base done.",
			Todos: []capsule.TodoItem{
				{ID: "a", Title: "Design auth", Status: "completed"},
				{ID: "b", Title: "Token refresh", Status: "pending"},
			},
			Decisions:     []string{"Use JWT"},
			OpenQuestions: []string{"Refresh interval?"},
			Risks:         []string{"No revocation"},
		},
	}
}

func TestRenderPromptContainsKeyFields(t *testing.T) {
	c := testCapsuleForRender()
	prompt := capsule.RenderResumePrompt(c, "claude", capsule.DriftReport{})

	checks := []string{
		"implement JWT auth", // original request
		"Token refresh",      // todo item
		"Use JWT",            // decision
		"Refresh interval?",  // open question
		"No revocation",      // risk
		"add token refresh",  // next action
		"fix/auth",           // branch
	}
	for _, s := range checks {
		if !strings.Contains(prompt, s) {
			t.Errorf("prompt missing %q", s)
		}
	}
}

func TestRenderPromptIncludesDriftWarning(t *testing.T) {
	c := testCapsuleForRender()
	drift := capsule.DriftReport{
		BranchChanged: true,
		SavedBranch:   "fix/auth",
		CurrentBranch: "main",
	}
	prompt := capsule.RenderResumePrompt(c, "claude", drift)

	if !strings.Contains(prompt, "drift") && !strings.Contains(prompt, "branch changed") {
		t.Error("prompt with drift should mention branch change")
	}
}

func TestRenderPromptNoDriftSection(t *testing.T) {
	c := testCapsuleForRender()
	prompt := capsule.RenderResumePrompt(c, "claude", capsule.DriftReport{})

	if strings.Contains(strings.ToLower(prompt), "drift") {
		t.Error("prompt without drift should not mention drift")
	}
}
