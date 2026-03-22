package session_test

import (
	"strings"
	"testing"
	"time"

	"github.com/helloprtr/poly-prompt/internal/session"
)

func TestBuildStartPrompt_ContainsGoal(t *testing.T) {
	s := session.Session{
		TaskGoal:    "인증 미들웨어 리팩토링",
		Files:       []string{"auth/*.ts"},
		Mode:        session.ModeEdit,
		Constraints: []string{"TypeScript"},
	}
	prompt := session.BuildStartPrompt(s)
	if !strings.Contains(prompt, "인증 미들웨어 리팩토링") {
		t.Error("start prompt missing task goal")
	}
	if !strings.Contains(prompt, "시작해주세요") {
		t.Error("start prompt should say '시작해주세요', not '이어서'")
	}
}

func TestBuildHandoffPrompt_BasicFields(t *testing.T) {
	s := session.Session{
		TaskGoal:    "인증 미들웨어 리팩토링",
		Files:       []string{"auth/*.ts"},
		Mode:        session.ModeEdit,
		Constraints: []string{"TypeScript", "테스트 포함"},
	}

	prompt := session.BuildHandoffPrompt(s, "", "")
	if !strings.Contains(prompt, "인증 미들웨어 리팩토링") {
		t.Error("prompt missing task goal")
	}
	if !strings.Contains(prompt, "auth/*.ts") {
		t.Error("prompt missing files")
	}
	if !strings.Contains(prompt, "TypeScript") {
		t.Error("prompt missing constraints")
	}
	if !strings.Contains(prompt, "이어서 작업해주세요") {
		t.Error("handoff prompt should say '이어서 작업해주세요'")
	}
}

func TestBuildHandoffPrompt_WithDiff(t *testing.T) {
	s := session.Session{TaskGoal: "fix bug", Mode: session.ModeFix}
	diff := "diff --git a/main.go b/main.go\n+added line"

	prompt := session.BuildHandoffPrompt(s, diff, "")
	if !strings.Contains(prompt, diff) {
		t.Error("prompt missing diff")
	}
}

func TestBuildHandoffPrompt_WithCheckpoints(t *testing.T) {
	s := session.Session{
		TaskGoal: "refactor",
		Mode:     session.ModeEdit,
		Checkpoints: []session.Checkpoint{
			{Note: "JWT done", At: time.Now()},
			{Note: "refresh WIP", At: time.Now()},
		},
	}
	prompt := session.BuildHandoffPrompt(s, "", "")
	if !strings.Contains(prompt, "JWT done") {
		t.Error("prompt missing checkpoint 1")
	}
	if !strings.Contains(prompt, "refresh WIP") {
		t.Error("prompt missing checkpoint 2")
	}
}

func TestBuildHandoffPrompt_EmptyDiffOmitsSection(t *testing.T) {
	s := session.Session{TaskGoal: "task", Mode: session.ModeEdit}
	prompt := session.BuildHandoffPrompt(s, "", "")
	if strings.Contains(prompt, "[코드 변화]") {
		t.Error("expected no code-change section when diff is empty")
	}
}

func TestBuildHandoffPrompt_WithLastResponse(t *testing.T) {
	s := session.Session{TaskGoal: "task", Mode: session.ModeEdit}
	prompt := session.BuildHandoffPrompt(s, "", "AI said: use interface{}")
	if !strings.Contains(prompt, "AI said: use interface{}") {
		t.Error("prompt missing last response")
	}
}
